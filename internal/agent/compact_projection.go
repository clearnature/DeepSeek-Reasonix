package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

const (
	maxCompressAnchorBytes = 512
	maxCompressFocusBytes  = 2000
)

var errCompressStaleContext = errors.New("compress: conversation changed while compression was running; retry with the current context")

// CompressContext implements the context-bound compress tool. It resolves the
// anchor against the current model-visible view and installs a projection only;
// the canonical transcript and checkpoint lineage remain untouched.
func (a *Agent) CompressContext(ctx context.Context, req tool.CompressRequest) (tool.CompressResult, error) {
	direction := strings.TrimSpace(req.Direction)
	anchor := strings.TrimSpace(req.Anchor)
	focus := strings.TrimSpace(req.Focus)
	if direction != "before" && direction != "after" {
		return tool.CompressResult{}, fmt.Errorf("compress: direction must be before or after")
	}
	if anchor == "" {
		return tool.CompressResult{}, fmt.Errorf("compress: anchor must not be empty")
	}
	if len(anchor) > maxCompressAnchorBytes {
		return tool.CompressResult{}, fmt.Errorf("compress: anchor exceeds %d bytes", maxCompressAnchorBytes)
	}
	if len(focus) > maxCompressFocusBytes {
		return tool.CompressResult{}, fmt.Errorf("compress: focus exceeds %d bytes", maxCompressFocusBytes)
	}

	var last tool.CompressResult
	lastOK := false
	lastSource := 0
	for pass := 0; pass < maxCompressPasses; pass++ {
		snap := a.snapshotExplicitCompression()
		matches := findCompressAnchors(snap.visible, anchor)
		if len(matches) == 0 {
			if pass == 0 {
				return tool.CompressResult{}, fmt.Errorf("compress: anchor did not match any current user message; retry with an exact excerpt from a visible user turn")
			}
			return last, nil
		}
		if len(matches) > 1 {
			if pass == 0 {
				return tool.CompressResult{}, fmt.Errorf("compress: anchor matched %d user messages; retry with a longer unique excerpt", len(matches))
			}
			return last, nil
		}
		result, err := a.compressVisibleRange(ctx, snap, CompactionTriggerTool, direction, matches[0], anchorPreview(UserMessageText(snap.visible[matches[0]])), focus)
		if err != nil {
			return result, err
		}
		if result.Status != "ok" || result.Messages == 0 {
			// A later pass found nothing more to fold (or the projection would
			// not shrink); report the last successful pass, not the noop.
			if lastOK {
				return last, nil
			}
			return result, nil
		}
		// Converge: stop once a pass no longer folds meaningfully more content
		// (the remaining fold is only previously folded summaries).
		if pass > 0 && lastSource-result.SourceTokens < minCompressSavings {
			if lastOK {
				return last, nil
			}
			return result, nil
		}
		last = result
		lastOK = true
		lastSource = result.SourceTokens
	}
	return last, nil
}

// maxCompressPasses bounds how many summary passes one compress invocation may
// run; each pass is one summarize request folding the next prefix region.
const maxCompressPasses = 6

// minCompressSavings is the token floor below which another pass is not worth
// its summarize round trip.
const minCompressSavings = 10_000

func findCompressAnchors(visible []provider.Message, anchor string) []int {
	matches := make([]int, 0, 2)
	for i, msg := range visible {
		if !compressAnchorCandidate(msg) {
			continue
		}
		if strings.Contains(UserMessageText(msg), anchor) {
			matches = append(matches, i)
		}
	}
	return matches
}

type explicitCompressionSnapshot struct {
	canonical         []provider.Message
	visible           []provider.Message
	transcriptVersion uint64
	coveredHash       string
	projectionVersion uint64
	generation        uint64
	promptCacheKey    string
}

func (a *Agent) snapshotExplicitCompression() explicitCompressionSnapshot {
	canonical, version := a.sess.conversation.snapshotMessagesVersion()
	cacheKey := a.currentPromptCacheKey()
	a.sess.compactionMu.Lock()
	state := a.sess.compactionState
	a.sess.compactionMu.Unlock()
	visible := canonical
	if projectionValid(state, canonical, cacheKey) {
		if projected := modelVisibleFromProjection(state.Projection, canonical); len(projected) > 0 {
			visible = projected
		}
	}
	return explicitCompressionSnapshot{
		canonical:         canonical,
		visible:           compressionVisibleMessages(visible),
		transcriptVersion: version,
		coveredHash:       coveredPrefixHash(canonical, len(canonical)),
		projectionVersion: state.Projection.ProjectionVersion,
		generation:        state.Generation,
		promptCacheKey:    cacheKey,
	}
}

func compressionVisibleMessages(msgs []provider.Message) []provider.Message {
	out := make([]provider.Message, 0, len(msgs)+1)
	for _, msg := range msgs {
		if !msg.LocalOnly {
			summary, user, split := splitLegacyCoalescedSummary(msg)
			if split {
				out = append(out, summary, user)
			} else {
				out = append(out, msg)
			}
		}
	}
	return out
}

// Older schema-v1 sidecars may have persisted a strict-role merge of the
// summary and its following user turn. Split that legacy shape for range
// planning; new sidecars keep the logical messages separate and coalesce only
// on the provider request copy.
func splitLegacyCoalescedSummary(msg provider.Message) (provider.Message, provider.Message, bool) {
	if !isCompactionSummary(msg) {
		return provider.Message{}, provider.Message{}, false
	}
	separator := summaryTagClose + "\n\n"
	i := strings.Index(msg.Content, separator)
	if i < 0 || i+len(separator) >= len(msg.Content) {
		return provider.Message{}, provider.Message{}, false
	}
	summary := msg
	summary.Content = msg.Content[:i+len(summaryTagClose)]
	summary.RawContent = ""
	summary.Images = nil
	summary.ToolCalls = nil
	summary.ResponsesItems = nil
	summary.ServerSearch = nil
	summary.CreatedAt = 0
	user := msg
	user.Content = msg.Content[i+len(separator):]
	user.RawContent = ""
	return summary, user, true
}

func compressAnchorCandidate(msg provider.Message) bool {
	if msg.Role != provider.RoleUser || msg.LocalOnly || isCompactionSummary(msg) {
		return false
	}
	return IsUserAuthoredTurn(UserMessageText(msg))
}

func anchorPreview(text string) string {
	return truncatePreview(previewProse(text))
}

type visibleCompressionPlan struct {
	result    tool.CompressResult
	foldMask  []bool
	fold      []provider.Message
	firstFold int
}

type preparedVisibleCompression struct {
	fold         []provider.Message
	instructions string
	inputMode    string
}

func (a *Agent) compressVisibleRange(
	ctx context.Context,
	snap explicitCompressionSnapshot,
	trigger string,
	direction string,
	anchorIndex int,
	preview string,
	instructions string,
) (tool.CompressResult, error) {
	a.sess.compactionRunMu.Lock()
	defer a.sess.compactionRunMu.Unlock()
	if !a.explicitCompressionSnapshotCurrent(snap) {
		return tool.CompressResult{}, errCompressStaleContext
	}
	plan, ok := a.planVisibleCompression(snap, direction, anchorIndex, preview)
	if !ok {
		return plan.result, nil
	}
	result := plan.result
	inputMode := SummaryInputNonPrefix
	if direction == "before" && foldMatchesVisiblePrefix(snap.visible, plan.fold) {
		inputMode = SummaryInputCachePrefix
	}

	a.svc.sink.Emit(event.Event{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: trigger}})
	prepared, reason, err := a.prepareVisibleCompression(ctx, trigger, plan.fold, instructions, inputMode)
	if err != nil {
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	if reason != "" {
		a.emitCompactionAborted(trigger)
		result.Reason = reason
		return result, nil
	}

	res, err := a.foldToSummaryMode(ctx, snap.visible[0:plan.firstFold], prepared.fold, prepared.instructions, prepared.inputMode)
	summary := res.Text
	tele := compactionTelemetryFromSummary(trigger, a.CacheState(), result.SourceTokens, res)
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	summary, err = a.interceptCompactionComplete(ctx, summary)
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}

	projection := buildVisibleCompressionProjection(snap.visible, plan, summary)
	projectionTokens := a.estimatedVisibleRequestTokens(projection)
	tele.ProjectionTokens = projectionTokens
	result.Messages = len(plan.fold)
	result.ProjectionTokens = projectionTokens
	result.Mode = res.Mode
	if projectionTokens >= result.SourceTokens {
		result.Reason = "compressed context would not be smaller"
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return result, nil
	}

	inputHash := providerVisibleFingerprint(modelInputMessages(snap.visible))
	outputHash := providerVisibleFingerprint(projection)
	state, err := a.commitSummaryProjection(summaryProjectionCommit{
		canonical: snap.canonical, fold: prepared.fold, projected: projection, result: res,
		transcriptVersion: snap.transcriptVersion, projectionVersion: snap.projectionVersion, generation: snap.generation,
		activeTurn: a.activeTurnCreatedAt.Load(), trigger: trigger, summary: summary,
		inputHash: inputHash, outputHash: outputHash, sourceTokens: result.SourceTokens, projectionTokens: projectionTokens,
		covered: len(snap.canonical),
	})
	if err != nil {
		if errors.Is(err, errCompressStaleContext) {
			tele.Error = err.Error()
			a.emitCompactionTelemetry(tele)
		}
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	a.emitCompactionTelemetry(tele)
	a.svc.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{
		Trigger: trigger, Messages: len(plan.fold), Summary: summary, Archive: state.LastReceipt.Archive,
	}})
	result.Status = "ok"
	result.Reason = ""
	return result, nil
}

func foldMatchesVisiblePrefix(visible, fold []provider.Message) bool {
	head := 0
	if len(visible) > 0 && visible[0].Role == provider.RoleSystem {
		head = 1
	}
	if len(fold) == 0 || head+len(fold) > len(visible) {
		return false
	}
	return providerVisibleFingerprint(modelInputMessages(fold)) ==
		providerVisibleFingerprint(modelInputMessages(visible[head:head+len(fold)]))
}

func (a *Agent) explicitCompressionSnapshotCurrent(snap explicitCompressionSnapshot) bool {
	current, version := a.sess.conversation.snapshotMessagesVersion()
	a.sess.compactionMu.Lock()
	projectionVersion := a.sess.compactionState.Projection.ProjectionVersion
	generation := a.sess.compactionState.Generation
	a.sess.compactionMu.Unlock()
	return version == snap.transcriptVersion && len(current) == len(snap.canonical) &&
		coveredPrefixHash(current, len(current)) == snap.coveredHash &&
		projectionVersion == snap.projectionVersion && generation == snap.generation &&
		a.currentPromptCacheKey() == snap.promptCacheKey
}

func (a *Agent) planVisibleCompression(snap explicitCompressionSnapshot, direction string, anchorIndex int, preview string) (visibleCompressionPlan, bool) {
	sourceTokens := a.estimatedVisibleRequestTokens(snap.visible)
	plan := visibleCompressionPlan{result: tool.CompressResult{
		Status:           "noop",
		Direction:        direction,
		Anchor:           preview,
		SourceTokens:     sourceTokens,
		ProjectionTokens: sourceTokens,
	}}
	if anchorIndex < 0 || anchorIndex >= len(snap.visible) {
		plan.result.Reason = "anchor is no longer present in the model context"
		return plan, false
	}
	head := 0
	if len(snap.visible) > 0 && snap.visible[0].Role == provider.RoleSystem {
		head = 1
	}
	completedEnd := len(snap.visible)
	if active := a.activeTurnStart(snap.visible); active >= 0 {
		completedEnd = active
	}
	start, end := head, anchorIndex
	if direction == "after" {
		start, end = anchorIndex, completedEnd
	}
	if start < head {
		start = head
	}
	if end > completedEnd {
		end = completedEnd
	}
	if start >= end {
		plan.result.Reason = "selected range is empty"
		return plan, false
	}

	plan.foldMask = make([]bool, len(snap.visible))
	plan.firstFold = len(snap.visible)
	for i, msg := range snap.visible {
		selected := i >= start && i < end
		mergeSummary := i < completedEnd && isCompactionSummary(msg)
		if msg.Role == provider.RoleSystem || i < head || (!selected && !mergeSummary) {
			continue
		}
		plan.foldMask[i] = true
		plan.fold = append(plan.fold, msg)
		if i < plan.firstFold {
			plan.firstFold = i
		}
	}
	if len(plan.fold) == 0 {
		plan.result.Reason = "selected range has no model-visible messages"
		return plan, false
	}
	return plan, true
}

func (a *Agent) prepareVisibleCompression(ctx context.Context, trigger string, fold []provider.Message, instructions, inputMode string) (preparedVisibleCompression, string, error) {
	if a.svc.hooks != nil {
		if hookInstructions := a.svc.hooks.PreCompact(ctx, trigger); hookInstructions != "" {
			if instructions != "" {
				instructions += "\n"
			}
			instructions += hookInstructions
		}
	}
	originalHash := providerVisibleFingerprint(modelInputMessages(fold))
	preparedFold, preparedInstructions, err := a.interceptCompactionPrepare(ctx, fold, instructions)
	if err != nil {
		return preparedVisibleCompression{}, "", err
	}
	preparedFold = modelInputMessages(preparedFold)
	if len(preparedFold) == 0 {
		return preparedVisibleCompression{}, "compaction hook removed the selected range", nil
	}
	if providerVisibleFingerprint(modelInputMessages(preparedFold)) != originalHash {
		inputMode = SummaryInputExtensionRewritten
	}
	return preparedVisibleCompression{fold: preparedFold, instructions: preparedInstructions, inputMode: inputMode}, "", nil
}

func buildVisibleCompressionProjection(visible []provider.Message, plan visibleCompressionPlan, summary string) []provider.Message {
	projection := make([]provider.Message, 0, len(visible)-len(plan.fold)+1)
	for i, msg := range visible {
		if i == plan.firstFold {
			projection = append(projection, formatSummaryMessage(summary))
		}
		if !plan.foldMask[i] {
			projection = append(projection, msg)
		}
	}
	return provider.ProjectionMessages(projection)
}

func compactionTelemetryFromSummary(trigger, cacheState string, sourceTokens int, res foldSummary) CompactionTelemetry {
	tele := CompactionTelemetry{
		Trigger: trigger, CacheState: cacheState, Mode: res.Mode,
		SourceTokens:      sourceTokens,
		ProviderRequestID: res.RequestID,
		FoldTokens:        res.FoldTokens,
		Spans:             1, // one application-layer summary request per transaction
		SummaryInputMode:  res.InputMode,
	}
	usage := res.Usage
	if usage == nil {
		return tele
	}
	tele.InputTokens = usage.PromptTokens
	tele.OutputTokens = usage.CompletionTokens
	tele.CacheHitTokens = usage.CacheHitTokens
	tele.CacheMissTokens = usage.CacheMissTokens
	tele.CacheWriteTokens = usage.CacheWriteTokens
	tele.RequestCount = usage.RequestCount
	if tele.RequestCount <= 0 {
		tele.RequestCount = 1
	}
	return tele
}

// compact writes a context projection; trigger stays "auto"/"manual" for UI cards.
func (a *Agent) compact(ctx context.Context, trigger, instructions string, force bool) error {
	_, err := a.compactToProjection(ctx, trigger, instructions, force, false)
	return err
}

// compactToProjection installs one content-driven summary checkpoint:
// stable prefix + one structured digest + recent verbatim tail.
// The canonical transcript is never rewritten. CompactionNoop means nothing
// was foldable; callers at physical overflow must treat that as hard failure.
// mustFree is kept for caller semantics (overflow paths cannot proceed without
// a fold); the fold input budget is now enforced unconditionally inside.
func (a *Agent) compactToProjection(ctx context.Context, trigger, instructions string, force, mustFree bool) (CompactionOutcome, error) {
	a.sess.compactionRunMu.Lock()
	defer a.sess.compactionRunMu.Unlock()
	return a.compactToProjectionLocked(ctx, trigger, instructions, force, mustFree)
}

func (a *Agent) compactToProjectionLocked(ctx context.Context, trigger, instructions string, force, mustFree bool) (CompactionOutcome, error) {
	activeTurn := a.activeTurnCreatedAt.Load()
	canonical, transcriptVersion := a.sess.conversation.snapshotMessagesVersion()
	a.sess.compactionMu.Lock()
	stateSnapshot := a.sess.compactionState
	startProjectionVersion := a.sess.compactionState.Projection.ProjectionVersion
	startGeneration := a.sess.compactionState.Generation
	a.sess.compactionMu.Unlock()
	msgs, onProjection := a.visibleInputForFold(stateSnapshot, canonical, transcriptVersion)
	viewInputHash := providerVisibleFingerprint(modelInputMessages(msgs))
	head, start, ok := a.planFoldRegion(msgs, force)
	if !ok {
		return CompactionNoop, nil
	}
	_, preliminaryFold, _ := a.partitionFoldForProjection(msgs[head:start])
	if len(preliminaryFold) == 0 || (!force && !foldEconomics(preliminaryFold)) {
		return CompactionNoop, nil
	}
	fixedPrefixTokens := a.estimatedVisibleRequestTokens(msgs[:head])
	if a.contextWindow > 0 && fixedPrefixTokens >= a.compactTrigger() {
		return CompactionNoop, fmt.Errorf("%w: fixed prefix (%d tokens) already exceeds trigger (%d)", errCheckpointRejected, fixedPrefixTokens, a.compactTrigger())
	}

	a.svc.sink.Emit(event.Event{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: trigger}})
	if a.svc.hooks != nil {
		if hookInstr := a.svc.hooks.PreCompact(ctx, trigger); hookInstr != "" {
			if instructions != "" {
				instructions += "\n"
			}
			instructions += hookInstr
		}
	}
	// Fold input is bounded by the window's physical input ceiling on every
	// path — not only overflow-triggered compactions. A manual compaction with
	// an under-estimated input would otherwise send an oversized fold and get
	// a provider 400 ("maximum context length exceeded"), e.g. a 1.15M fold
	// against a 1,048,576-token model (2026-08-30). maximumSafeSummaryPrefixEnd
	// trims the fold region when the summary request would exceed the budget;
	// normal-size folds are untouched (fits(end) short-circuits).
	start = a.maximumSafeSummaryPrefixEnd(msgs, head, start, instructions)
	if start <= head {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, fmt.Errorf("%w: no balanced prefix leaves enough room for a summary response", errCheckpointRejected)
	}

	covered, bodySuffix := projectionCoverageForFold(stateSnapshot, msgs, start, onProjection)
	kept, fold, retention := a.partitionFoldForProjection(msgs[head:start])
	if len(fold) == 0 {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, nil
	}
	originalFoldHash := providerVisibleFingerprint(modelInputMessages(fold))
	var err error
	fold, instructions, err = a.interceptCompactionPrepare(ctx, fold, instructions)
	if err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
	if len(fold) == 0 {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, nil
	}

	sourceTokens := a.estimatedVisibleRequestTokens(msgs)
	inputMode := SummaryInputCachePrefix
	if providerVisibleFingerprint(modelInputMessages(fold)) != originalFoldHash {
		inputMode = SummaryInputExtensionRewritten
	}
	summaryPrefix, foldExtra, foldAnchors := a.summaryFoldPlan(msgs, head, start)
	if providerVisibleFingerprint(modelInputMessages(fold)) != originalFoldHash {
		// An extension rewrote the fold: the rewritten bytes must be sent and
		// read literally — anchor locating inside the frozen prefix would
		// summarize the pre-rewrite text. Quality wins over cache here.
		summaryPrefix = msgs[:head]
		foldExtra = fold
		foldAnchors = ""
	}
	if foldAnchors != "" {
		instructions += foldAnchors
	}
	res, tele, err := a.foldSummaryWithTelemetry(ctx, trigger, summaryPrefix, foldExtra, instructions, sourceTokens, inputMode)
	if err != nil {
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
	summary, err := a.interceptCompactionComplete(ctx, res.Text)
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}

	// The projection body freezes only prefix + digest + kept messages; the
	// verbatim tail splices live from canonical[start:] so tail-side rewrites
	// (rewind truncation, snips) stay visible without rebuilding the fold.
	projMsgs := checkpointProjectionMessages(msgs, head, kept, summary)
	if len(bodySuffix) > 0 {
		projMsgs = append(projMsgs, provider.ProjectionMessages(bodySuffix)...)
	}
	spliced := append(append([]provider.Message(nil), projMsgs...), canonical[covered:]...)
	projTokens := a.estimatedVisibleRequestTokens(spliced)
	tele.ProjectionTokens = projTokens
	tele.UserTurnsKept, tele.UserTurnsDropped = retention.Kept, retention.Dropped
	a.emitCompactionTelemetry(tele)
	if err := a.acceptCheckpointCandidate(trigger, sourceTokens, projTokens); err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
	viewOutputHash := providerVisibleFingerprint(modelInputMessages(spliced))
	_, err = a.commitSummaryProjection(summaryProjectionCommit{
		canonical: canonical, fold: fold, projected: projMsgs, result: res,
		transcriptVersion: transcriptVersion, projectionVersion: startProjectionVersion,
		generation: startGeneration, activeTurn: activeTurn, trigger: trigger,
		summary: summary, inputHash: viewInputHash, outputHash: viewOutputHash,
		sourceTokens: sourceTokens, projectionTokens: projTokens, covered: covered,
		// Persist the wire form (normalized): a resumed process re-normalizes
		// the restored bytes inside summaryRequest, so they must already match
		// what this request actually sent — the raw view would diverge. The
		// tools ride along as the same cached unit (system+tools+messages).
		wirePrefix: a.normalizeModelRequestMessages(summaryPrefix),
		wireTools:  a.summaryRequestToolsForCommit(summaryPrefix),
	})
	if err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
	a.svc.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{
		Trigger: trigger, Messages: len(fold), Summary: summary,
	}})
	return CompactionInstalled, nil
}

// projectionCoverageForFold maps a working-view boundary to canonical
// coverage. A suffix inside an existing frozen body remains in the new body
// because it has no corresponding canonical tail to splice from.
func projectionCoverageForFold(state CompactionState, msgs []provider.Message, start int, onProjection bool) (int, []provider.Message) {
	if !onProjection {
		return start, nil
	}
	body := len(state.Projection.Messages)
	prior := state.Projection.CoveredCount
	if start < body {
		return prior, msgs[start:body]
	}
	return prior + (start - body), nil
}

// visibleInputForFold resolves the same model-visible view ordinary sampling
// uses (see visibleMessagesWithFlag), so summary requests share the sampling
// request's byte prefix and keep the provider prefix cache warm. The second
// return reports whether the projection view was used, so fold boundaries can
// be translated back to canonical indices.
func (a *Agent) visibleInputForFold(state CompactionState, canonical []provider.Message, transcriptVersion uint64) ([]provider.Message, bool) {
	return a.visibleMessagesWithFlag(state, canonical, a.currentPromptCacheKey())
}

func checkpointProjectionMessages(msgs []provider.Message, head int, kept []provider.Message, summary string) []provider.Message {
	projMsgs := make([]provider.Message, 0, head+1+len(kept))
	projMsgs = append(projMsgs, msgs[:head]...)
	projMsgs = append(projMsgs, formatSummaryMessage(summary))
	projMsgs = append(projMsgs, kept...)
	return provider.ProjectionMessages(projMsgs)
}

// acceptCheckpointCandidate requires real savings and, for automatic
// maintenance, a result below the physical input ceiling.
func (a *Agent) acceptCheckpointCandidate(trigger string, sourceTokens, candidateTokens int) error {
	if candidateTokens >= sourceTokens {
		return fmt.Errorf("%w: candidate would not reduce tokens (%d >= %d)", errCheckpointRejected, candidateTokens, sourceTokens)
	}
	hard := a.hardInputCeiling()
	if trigger != CompactionTriggerManual && hard > 0 && candidateTokens >= hard {
		return fmt.Errorf("%w: candidate %d still at or above physical ceiling %d", errCheckpointRejected, candidateTokens, hard)
	}
	return nil
}

// planFoldRegion returns [head:start] to fold; force shrinks the recent tail.
func (a *Agent) planFoldRegion(msgs []provider.Message, force bool) (head, start int, ok bool) {
	head, start, ok = a.planCompaction(msgs, minCompactMessages, force)
	if !ok {
		head, start, ok = a.planCompaction(msgs, 1, force)
	}
	if !ok {
		return head, start, false
	}
	// Resume-first fold: this process has no frozen main-request bytes, so
	// the summarizer prefix cannot hit the parent's cached unit and only the
	// system prefix survives (2026-08-31 00:26:19: hit=16896 of 86355).
	// Aligning the fold end to the full view keeps the fold maximal; the
	// next in-process compaction aligns to a fresh unit instead.
	if a.sess.checkpointState == "restored" {
		start = len(msgs)
	}
	if active := a.activeTurnStart(msgs); active >= head && active < start {
		start = active
	}
	return head, start, start > head
}

// summaryFoldPlan splits the summarizer request into the byte prefix that
// matches the provider-cached unit and the extra messages that must be sent.
// With frozen main-request bytes, the whole main request is the prefix (it is
// the unit the server cached) and the fold is NOT re-sent — the fold segment
// already lives inside that prefix, so the instruction locates it by verbatim
// anchor excerpts instead. A fresh resume has no frozen bytes, but the live
// view still byte-matches the parent process's last request while the cache
// is warm — replaying the whole view + tail instruction is the C1 shape
// (warm replay, not new bytes). Only an over-window view falls back to the
// old cropped shape. The second return is the fold tail that extends beyond
// the prefix; anchors describe the fold region for the summarizer instruction.
func (a *Agent) summaryFoldPlan(msgs []provider.Message, head, start int) (prefix, extra []provider.Message, anchors string) {
	fold := msgs[head:start]
	if saved := a.savedMainRequest(); saved != nil && len(saved.messages) > 0 {
		region := fold
		if start > len(saved.messages) {
			extra = msgs[max(head, len(saved.messages)):start]
			region = append(append([]provider.Message(nil), fold...), extra...)
		}
		return saved.messages, extra, foldAnchorInstruction(region)
	}
	if a.summaryViewReplayFits(msgs) {
		return msgs, nil, foldAnchorInstruction(fold)
	}
	return msgs[:head], msgs[head:start], ""
}

// summaryMaxPromptTokens is the admissible summarizer input ceiling, shared by
// planning (maximumSafeSummaryPrefixEnd), the replay-fits check, and send-time
// admission so a planned request is never rejected after selection.
func (a *Agent) summaryMaxPromptTokens() int {
	window := a.effectiveContextWindow()
	if window <= 0 {
		return 0
	}
	policy := contextBudgetPolicyOf(a.svc.prov)
	if policy.WindowMode == provider.ContextWindowUnknown {
		// A learned overflow makes an unknown gateway shared-window. Otherwise
		// preserve the request because the configured window may be an estimate.
		if a.lastAdmission().ObservedWindow <= 0 {
			return a.hardInputCeiling()
		}
		policy.WindowMode = provider.ContextWindowShared
	}
	if policy.WindowMode == provider.ContextWindowShared {
		return window - outputBudgetReserve - 256
	}
	return a.hardInputCeiling()
}

// summaryViewReplayFits reports whether the whole live view can be replayed
// as the summarizer prefix (view + instruction within the admissible input
// ceiling). Over-ceiling views must crop instead, at the cost of the prefix
// cache match.
func (a *Agent) summaryViewReplayFits(msgs []provider.Message) bool {
	maxPromptTokens := a.summaryMaxPromptTokens()
	if maxPromptTokens <= 0 {
		return true
	}
	return a.estimatedRequestTokens(a.summaryRequest(msgs, nil, "")) <= maxPromptTokens
}

// foldAnchorInstruction names the fold region by verbatim excerpts so the
// summarizer can locate it inside the already-sent main-request bytes. The end
// excerpt is taken from the whole region (fold + extras) so new turns beyond
// the frozen prefix stay inside the summarized range.
func foldAnchorInstruction(region []provider.Message) string {
	startAnchor, endAnchor := "", ""
	for _, m := range region {
		if text := strings.TrimSpace(m.Content); text != "" {
			if startAnchor == "" {
				startAnchor = foldAnchor(text)
			}
			endAnchor = foldAnchor(text)
		}
	}
	if startAnchor == "" {
		return ""
	}
	return fmt.Sprintf("\n\nThe conversation above contains a segment to summarize. It starts with the excerpt %q and ends with the excerpt %q (both appear verbatim in the conversation above). Summarize ONLY that segment; leave everything else untouched.", startAnchor, endAnchor)
}

// foldAnchor truncates a message's content to a stable locating excerpt.
func foldAnchor(text string) string {
	const maxAnchor = 160
	if len(text) <= maxAnchor {
		return text
	}
	return text[:maxAnchor]
}

// summaryFoldEstimate builds the same request shape summaryFoldPlan would
// produce for a candidate fold end, for budget checks (see
// maximumSafeSummaryPrefixEnd). With frozen main-request bytes the whole saved
// request is the prefix; otherwise the whole view replays when it fits.
func (a *Agent) summaryFoldEstimate(msgs []provider.Message, head, candidate int, instructions string) provider.Request {
	if saved := a.savedMainRequest(); saved != nil && len(saved.messages) > 0 {
		var extra []provider.Message
		if start := max(head, len(saved.messages)); start < candidate && candidate <= len(msgs) {
			extra = msgs[start:candidate]
		}
		anchors := ""
		if region := msgs[head:candidate]; len(region) > 0 && candidate <= len(msgs) {
			anchors = foldAnchorInstruction(append(append([]provider.Message(nil), region...), extra...))
		}
		return a.summaryRequest(saved.messages, extra, instructions+anchors)
	}
	if a.summaryViewReplayFits(msgs) {
		anchors := ""
		if region := msgs[head:candidate]; len(region) > 0 {
			anchors = foldAnchorInstruction(region)
		}
		return a.summaryRequest(msgs, nil, instructions+anchors)
	}
	return a.summaryRequest(msgs[:head], msgs[head:candidate], instructions)
}

// maximumSafeSummaryPrefixEnd returns the largest balanced contiguous prefix
// whose exact summary request leaves the collector's minimum output budget.
// The remaining middle and tail stay verbatim in the projection.
func (a *Agent) maximumSafeSummaryPrefixEnd(msgs []provider.Message, head, end int, instructions string) int {
	window := a.effectiveContextWindow()
	if window <= 0 || head < 0 || end <= head || end > len(msgs) {
		return end
	}
	policy := contextBudgetPolicyOf(a.svc.prov)
	if policy.WindowMode == provider.ContextWindowUnknown {
		// A learned overflow makes an unknown gateway shared-window. Otherwise
		// preserve the request because the configured window may be an estimate.
		if a.lastAdmission().ObservedWindow <= 0 {
			return end
		}
		policy.WindowMode = provider.ContextWindowShared
	}
	maxPromptTokens := a.summaryMaxPromptTokens()
	// The fold is a canonical subset the server already accepted this session
	// (ObservedPrompt), so folding all at once cannot overflow; the estimate is
	// inflated for replayed history. Without an observed ceiling, truncation stands.
	if obs := a.lastAdmission().ObservedPrompt; obs > maxPromptTokens {
		maxPromptTokens = obs
		if all := a.estimatedRequestTokens(a.summaryFoldEstimate(msgs, head, end, instructions)); all > maxPromptTokens {
			maxPromptTokens = all
		}
	}
	if maxPromptTokens <= 0 {
		return head
	}
	fits := func(candidate int) bool {
		request := a.summaryFoldEstimate(msgs, head, candidate, instructions)
		return a.estimatedRequestTokens(request) <= maxPromptTokens
	}
	if fits(end) {
		return end
	}

	low, high, best := head+1, end-1, head
	for low <= high {
		mid := low + (high-low)/2
		if fits(mid) {
			best = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	// A tail beginning with a tool result would split it from the assistant
	// tool-call message. Move the fold boundary back across the whole result
	// group; the assistant call and all of its results then remain together.
	for best > head && best < len(msgs) && msgs[best].Role == provider.RoleTool {
		best--
	}
	return best
}

type userTurnRetention struct {
	Kept    int
	Dropped int
}

func (a *Agent) partitionFoldForProjection(region []provider.Message) (kept, fold []provider.Message, retention userTurnRetention) {
	for _, m := range region {
		if m.LocalOnly {
			continue
		}
		fold = append(fold, m)
		if m.Role == provider.RoleUser && !isCompactionSummary(m) {
			retention.Dropped++
		}
	}
	return kept, fold, retention
}

// runCompactionSummary uses the single local summarizer path for every provider.
func (a *Agent) runCompactionSummary(ctx context.Context, prefix, fold []provider.Message, instructions string) (summary, mode string, usage *provider.Usage, providerReqID string, err error) {
	summary, usage, err = a.summarizeOnce(ctx, prefix, fold, instructions)
	if err != nil {
		return "", CompactionModeSummarized, usage, "", err
	}
	return summary, CompactionModeSummarized, usage, "", nil
}

// foldSummaryWithChunkedFallback retries summary size failures through the
// chunked fallback path. This is a stub that delegates to foldToSummary.
func (a *Agent) foldSummaryWithChunkedFallback(ctx context.Context, trigger string, fold []provider.Message, instructions string, sourceTokens int, inputMode string) (foldSummary, CompactionTelemetry, error) {
	res, err := a.foldToSummary(ctx, nil, fold, instructions)
	if err != nil {
		return foldSummary{}, CompactionTelemetry{}, err
	}
	tele := compactionTelemetryFromSummary(trigger, a.CacheState(), sourceTokens, res)
	return res, tele, nil
}
