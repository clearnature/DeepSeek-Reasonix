package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// Legacy snip helpers still support compatibility storage. Their public APIs
// are no-ops; pressure-time Harness pruning uses the rune-based policy below.
const (
	snippedMarker = "[snipped tool result — "
	prunedMarker  = "[elided tool result — "
	minPruneBytes = 1024

	toolPruneThresholdRunes = 8192
	toolPruneHeadRunes      = 4096
	toolPruneTailRunes      = 1024
	toolPruneMarker         = "[... tool result middle pruned ...]"
)

func pruneToolResultContent(content string) (string, bool) {
	if utf8.RuneCountInString(content) <= toolPruneThresholdRunes {
		return content, false
	}
	headEnd := byteOffsetAfterRunes(content, toolPruneHeadRunes)
	tailStart := byteOffsetBeforeLastRunes(content, toolPruneTailRunes)
	var pruned strings.Builder
	pruned.Grow(headEnd + len(toolPruneMarker) + len(content) - tailStart)
	pruned.WriteString(content[:headEnd])
	pruned.WriteString(toolPruneMarker)
	pruned.WriteString(content[tailStart:])
	return pruned.String(), true
}

func byteOffsetAfterRunes(content string, count int) int {
	if count <= 0 {
		return 0
	}
	seen := 0
	for offset := range content {
		if seen == count {
			return offset
		}
		seen++
	}
	return len(content)
}

func byteOffsetBeforeLastRunes(content string, count int) int {
	offset := len(content)
	for range count {
		if offset == 0 {
			return 0
		}
		_, size := utf8.DecodeLastRuneInString(content[:offset])
		offset -= size
	}
	return offset
}

// pruneToolResultsToProjectionLocked installs a durable, model-visible prune
// projection. The caller owns compactionRunMu for the whole maintenance run;
// canonical storage, including RawContent, is never modified.
func (a *Agent) pruneToolResultsToProjectionLocked(trigger string) (bool, error) {
	canonical, transcriptVersion := a.sess.conversation.snapshotMessagesVersion()
	a.sess.compactionMu.Lock()
	stateSnapshot := a.sess.compactionState
	a.sess.compactionMu.Unlock()
	visible, _ := a.visibleInputForFold(stateSnapshot, canonical, transcriptVersion)
	projected := append([]provider.Message(nil), visible...)
	affected := 0
	for i := range projected {
		if projected[i].Role != provider.RoleTool {
			continue
		}
		source := projected[i].Content
		if projected[i].ProviderContent != "" {
			source = projected[i].ProviderContent
		}
		if pruned, changed := pruneToolResultContent(source); changed {
			projected[i].Content = pruned
			projected[i].RawContent = ""
			projected[i].ProviderContent = ""
			affected++
		}
	}
	if affected == 0 {
		return false, nil
	}
	projected = provider.ProjectionMessages(projected)
	sourceTokens := a.estimatedVisibleRequestTokens(visible)
	resultTokens := a.estimatedVisibleRequestTokens(projected)
	inputHash := a.contextMaintenanceInputHash(modelInputMessages(visible))
	outputHash := providerVisibleFingerprint(modelInputMessages(projected))
	projectionVersion := stateSnapshot.Projection.ProjectionVersion + 1
	now := time.Now().UTC()
	coveredHash := coveredPrefixHash(canonical, len(canonical))
	receipt := &ContextMaintenanceReceipt{
		OperationID: fmt.Sprintf("prune-%d-%s", projectionVersion, outputHash), Status: "applied", Action: "prune",
		Trigger: trigger, SourceProjection: stateSnapshot.Projection.ProjectionVersion, ProjectionVersion: projectionVersion,
		CoveredCount: len(canonical), CoveredPrefixHash: coveredHash, InputHash: inputHash, OutputHash: outputHash,
		InputTokens: sourceTokens, ResultTokens: resultTokens, SavedTokens: max(0, sourceTokens-resultTokens),
		AffectedToolResults: affected, CacheBreak: true, CreatedAt: now,
	}
	next := stateSnapshot
	next.SchemaVersion = compactionStateSchemaCurrent
	next.TranscriptVersion = transcriptVersion
	next.Generation++
	next.PromptCacheKey = a.currentPromptCacheKey()
	next.Projection = ContextProjection{
		Messages: projected, TranscriptVersion: transcriptVersion, ProjectionVersion: projectionVersion,
		CoveredCount: len(canonical), CoveredPrefixHash: coveredHash, SourceTokens: sourceTokens,
		ProjectionTokens: resultTokens, ViewInputHash: inputHash, ViewOutputHash: outputHash, CreatedAt: now,
	}
	next.LastReceipt = receipt
	next.UpdatedAt = now

	a.sess.compactionMu.Lock()
	current, currentVersion := a.sess.conversation.snapshotMessagesVersion()
	if currentVersion != transcriptVersion || len(current) != len(canonical) ||
		coveredPrefixHash(current, len(current)) != coveredHash ||
		a.sess.compactionState.Projection.ProjectionVersion != stateSnapshot.Projection.ProjectionVersion ||
		a.sess.compactionState.Generation != stateSnapshot.Generation {
		a.sess.compactionMu.Unlock()
		return false, errCompressStaleContext
	}
	previous := a.sess.compactionState
	a.sess.compactionState = next
	if err := a.persistCompactionStateLocked(); err != nil {
		a.sess.compactionState = previous
		a.sess.compactionMu.Unlock()
		if errors.Is(err, errCompressStaleContext) {
			return false, err
		}
		return false, fmt.Errorf("persist prune projection: %w", err)
	}
	a.sess.checkpointState = "applied"
	a.sess.compactionMu.Unlock()
	a.emitContextMaintenance(receipt)
	return true, nil
}

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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
