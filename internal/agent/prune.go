package agent

import (
	"fmt"
	"strings"
	"time"

	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// Tool-result helpers serve first-visible bounding and summary fold input.
// Automatic prune/snip projections are gone; the public APIs are no-ops.
const (
	snippedMarker = "[snipped tool result — "
	prunedMarker  = "[elided tool result — "
	minPruneBytes = 1024
)

type toolResultMaintenanceMode int

const (
	toolResultSnip toolResultMaintenanceMode = iota
	toolResultPrune
)

// PruneStats reports one maintenance pass.
type PruneStats struct {
	Results    int
	SavedChars int
	Archive    string
	Mode       toolResultMaintenanceMode
	InputHash  string
	Force      bool
}

// PruneStaleToolResults elides stale tool results to short placeholders
// without a summarizer call — the /compress-fast rescue path used when the
// provider cannot host an AI compaction. Rebuilds the model-visible view in
// place (canonical untouched, projection updated) and reports the savings.
func (a *Agent) PruneStaleToolResults() (PruneStats, error) {
	return a.maintainStaleToolResults(toolResultPrune)
}

// maintainStaleToolResults applies the mode's maintenance view to the current
// model-visible messages and installs the elided result as a projection.
func (a *Agent) maintainStaleToolResults(mode toolResultMaintenanceMode) (PruneStats, error) {
	st := PruneStats{Mode: mode}
	if a.contextWindow <= 0 {
		return st, nil
	}
	visible := a.modelVisibleMessages()
	next, st := a.applyToolResultMaintenanceView(visible, mode)
	st.Force = true
	if st.Results == 0 {
		return st, nil
	}
	if err := a.installElidedProjection(next, st); err != nil {
		return PruneStats{Mode: mode}, err
	}
	return st, nil
}

// applyToolResultMaintenanceView rebuilds a copy of msgs with stale tool
// results elided/snipped; the original slice is never mutated.
func (a *Agent) applyToolResultMaintenanceView(msgs []provider.Message, mode toolResultMaintenanceMode) ([]provider.Message, PruneStats) {
	st := PruneStats{Mode: mode}
	if a.contextWindow <= 0 || len(msgs) == 0 {
		return msgs, st
	}
	next := append([]provider.Message(nil), msgs...)
	changed := false
	for i, m := range next {
		if !shouldMaintainToolResult(m, mode) {
			continue
		}
		replacement := rewriteToolResult(m, mode, "not archived", a.snipStrategyFor(m.Name))
		if replacement == m.Content {
			continue
		}
		st.SavedChars += len(m.Content) - len(replacement)
		next[i].Content = replacement
		st.Results++
		changed = true
	}
	if !changed {
		return msgs, st
	}
	return next, st
}

// installElidedProjection CAS-installs the elided view as the new projection
// under the current compaction lineage (no summarizer involved).
func (a *Agent) installElidedProjection(next []provider.Message, st PruneStats) error {
	// Collect read-only data before taking the lock: modelVisibleMessages
	// re-enters compactionMu, so it cannot run inside the locked section.
	canonical, version := a.Session().snapshotMessagesVersion()
	if len(canonical) == 0 {
		return nil
	}
	src := a.estimatedPromptTokens(a.modelVisibleMessages())
	dst := a.estimatedPromptTokens(provider.ModelMessages(next))
	outputHash := providerVisibleFingerprint(provider.ModelMessages(next))
	action := "snip"
	if st.Mode == toolResultPrune {
		action = "prune"
	}

	a.sess.compactionMu.Lock()
	defer a.sess.compactionMu.Unlock()
	state := a.sess.compactionState
	projVersion := state.Projection.ProjectionVersion + 1
	now := time.Now().UTC()
	state.Projection = ContextProjection{
		Messages:           provider.ModelMessages(next),
		TranscriptVersion:  version,
		ProjectionVersion:  projVersion,
		CoveredCount:       len(canonical),
		CoveredPrefixHash:  coveredPrefixHash(canonical, len(canonical)),
		SemanticPrefixHash: semanticPrefixHash(canonical, len(canonical)),
		SourceTokens:       src,
		ProjectionTokens:   dst,
		ViewOutputHash:     outputHash,
		CreatedAt:          now,
	}
	state.Generation++
	state.LastTrigger = CompactionTriggerPressure
	state.LastMode = CompactionModeSnip
	state.LastSourceTokens = src
	state.LastResultTokens = dst
	state.LastReceipt = &ContextMaintenanceReceipt{
		OperationID: fmt.Sprintf("elide-%d-%s", projVersion, outputHash), Status: "applied", Action: action,
		Trigger: CompactionTriggerPressure, SourceProjection: projVersion - 1, ProjectionVersion: projVersion,
		CoveredCount: len(canonical), CoveredPrefixHash: coveredPrefixHash(canonical, len(canonical)),
		OutputHash: outputHash, InputTokens: src, ResultTokens: dst,
		SavedTokens: max(0, src-dst), AffectedToolResults: st.Results, Archive: st.Archive,
		CacheBreak: true, CreatedAt: now,
	}
	state.UpdatedAt = now
	prev := a.sess.compactionState
	a.sess.compactionState = state
	if err := a.persistCompactionStateLocked(); err != nil {
		a.sess.compactionState = prev
		return err
	}
	a.emitContextMaintenance(state.LastReceipt)
	return nil
}

func shouldMaintainToolResult(m provider.Message, mode toolResultMaintenanceMode) bool {
	if m.LocalOnly || m.Role != provider.RoleTool {
		return false
	}
	if strings.HasPrefix(m.Content, prunedMarker) {
		return false
	}
	if mode == toolResultSnip {
		return len(m.Content) >= minPruneBytes && !strings.HasPrefix(m.Content, snippedMarker)
	}
	if strings.HasPrefix(m.Content, snippedMarker) {
		return true
	}
	return len(m.Content) >= minPruneBytes
}

func rewriteToolResult(m provider.Message, mode toolResultMaintenanceMode, archive string, strategy snipStrategy) string {
	if mode == toolResultPrune {
		return pruneToolResult(m, archive)
	}
	return snipToolResult(m, archive, strategy)
}

func pruneToolResult(m provider.Message, archive string) string {
	if archive == "" {
		archive = "not archived"
	}
	return fmt.Sprintf("%s%s, %d bytes archived to %s; re-run the tool if the data is needed again]",
		prunedMarker, m.Name, len(m.Content), archive)
}

// SnipStaleToolResults is a no-op: automatic snip projections are gone;
// the manual rescue path is PruneStaleToolResults (/compress-fast).
func (a *Agent) SnipStaleToolResults() (PruneStats, error) {
	return PruneStats{Mode: toolResultSnip}, nil
}

func snipToolResult(m provider.Message, archive string, strategy snipStrategy) string {
	if archive == "" {
		archive = "the canonical transcript"
	}
	lines := strings.Split(m.Content, "\n")
	if len(lines) <= strategy.head+strategy.tail {
		headChars := minInt(strategy.headChars, len(m.Content)/2)
		tailChars := minInt(strategy.tailChars, len(m.Content)/4)
		return fmt.Sprintf("%s%s, %d bytes; full original retained in %s; single large line truncated]\n%s\n[... %d bytes omitted ...]\n%s",
			snippedMarker, m.Name, len(m.Content), archive,
			firstRunes(m.Content, headChars),
			omittedBytes(m.Content, headChars, tailChars),
			lastRunes(m.Content, tailChars))
	}
	head := strings.Join(lines[:strategy.head], "\n")
	tail := strings.Join(lines[len(lines)-strategy.tail:], "\n")
	return fmt.Sprintf("%s%s, %d bytes; full original retained in %s; showing first %d lines and last %d lines]\n%s\n[... %d lines omitted ...]\n%s",
		snippedMarker, m.Name, len(m.Content), archive, strategy.head, strategy.tail,
		head, len(lines)-strategy.head-strategy.tail, tail)
}

type snipStrategy struct {
	head      int
	tail      int
	headChars int
	tailChars int
}

var (
	defaultReadOnlySnip      = snipStrategy{head: 80, tail: 12, headChars: 10000, tailChars: 2000}
	defaultSideEffectingSnip = snipStrategy{head: 40, tail: 40, headChars: 8000, tailChars: 8000}
)

func (a *Agent) snipStrategyFor(name string) snipStrategy {
	if a.svc.tools != nil {
		if t, ok := a.svc.tools.Get(name); ok {
			if h, ok := t.(tool.SnipHinter); ok {
				return snipStrategyFromHint(h.SnipHint())
			}
			if t.ReadOnly() {
				return defaultReadOnlySnip
			}
			return defaultSideEffectingSnip
		}
	}
	return defaultReadOnlySnip
}

func snipStrategyFromHint(h tool.SnipHint) snipStrategy {
	return snipStrategy{head: h.Head, tail: h.Tail, headChars: h.HeadChars, tailChars: h.TailChars}
}

func firstRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !isRuneBoundary(s, n) {
		n--
	}
	return s[:n]
}

func lastRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	start := len(s) - n
	for start < len(s) && !isRuneBoundary(s, start) {
		start++
	}
	return s[start:]
}

func omittedBytes(s string, head, tail int) int {
	omitted := len(s) - head - tail
	if omitted < 0 {
		return 0
	}
	return omitted
}

func isRuneBoundary(s string, i int) bool {
	return i == 0 || i == len(s) || (i > 0 && i < len(s) && (s[i]&0xc0) != 0x80)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
