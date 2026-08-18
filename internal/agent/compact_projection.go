package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

	snap := a.snapshotExplicitCompression()
	matches := make([]int, 0, 2)
	for i, msg := range snap.visible {
		if !compressAnchorCandidate(msg) {
			continue
		}
		if strings.Contains(UserMessageText(msg), anchor) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return tool.CompressResult{}, fmt.Errorf("compress: anchor did not match any current user message; retry with an exact excerpt from a visible user turn")
	}
	if len(matches) > 1 {
		return tool.CompressResult{}, fmt.Errorf("compress: anchor matched %d user messages; retry with a longer unique excerpt", len(matches))
	}

	return a.compressVisibleRange(ctx, snap, CompactionTriggerTool, direction, matches[0], anchorPreview(UserMessageText(snap.visible[matches[0]])), focus)
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
	} else if degraded, ok := a.modelVisibleDegraded(state, canonical); ok {
		visible = degraded
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

<<<<<<< HEAD
	start := time.Now()
	res, err := a.foldToSummary(ctx, snap.visible[:plan.firstFold], prepared.fold, prepared.instructions)
=======
	res, err := a.foldToSummaryMode(ctx, prepared.fold, prepared.instructions, prepared.inputMode)
>>>>>>> origin/main-v2
	summary := res.Text
	tele := compactionTelemetryFromSummary(trigger, a.CacheState(), result.SourceTokens, a.decisionEstimateTokens(), res)
	tele.ElapsedMs = time.Since(start).Milliseconds()
	if err != nil {
		tele.Error = err.Error()
		tele.Status = "failed"
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	summary, err = a.interceptCompactionComplete(ctx, summary)
	if err != nil {
		tele.Error = err.Error()
		tele.Status = "failed"
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
	tele.Status = CompactionStatusInstalled
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

func compactionTelemetryFromSummary(trigger, cacheState string, sourceTokens, estTokens int, res foldSummary) CompactionTelemetry {
	tele := CompactionTelemetry{
		Trigger: trigger, CacheState: cacheState, Mode: res.Mode,
		EstTokens:         estTokens,
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
// mustFree marks the fold the caller cannot proceed without.
func (a *Agent) compactToProjection(ctx context.Context, trigger, instructions string, force, mustFree bool) (outcome CompactionOutcome, err error) {
	a.sess.compactionRunMu.Lock()
	defer a.sess.compactionRunMu.Unlock()
<<<<<<< HEAD
	a.markCompactionInflight()
	defer a.clearCompactionInflight()
=======
	return a.compactToProjectionLocked(ctx, trigger, instructions, force, mustFree)
}

func (a *Agent) compactToProjectionLocked(ctx context.Context, trigger, instructions string, force, mustFree bool) (CompactionOutcome, error) {
>>>>>>> origin/main-v2
	activeTurn := a.activeTurnCreatedAt.Load()
	canonical, transcriptVersion := a.sess.conversation.snapshotMessagesVersion()
	// Silent exits (Noop/aborted) must still land in the stats file: a fold
	// that found nothing is the "compacted but nothing happened" case that
	// was invisible (user-observed 2026-08-09). Success paths emit inside.
	emitted := false
	emit := func(t CompactionTelemetry) { a.emitCompactionTelemetry(t); emitted = true }
	defer func() {
		if outcome == CompactionNoop && !emitted {
			emit(a.silentCompactionTelemetry(trigger, canonical, err))
		}
	}()
	a.sess.compactionMu.Lock()
	stateSnapshot := a.sess.compactionState
	startProjectionVersion := a.sess.compactionState.Projection.ProjectionVersion
	startGeneration := a.sess.compactionState.Generation
	a.sess.compactionMu.Unlock()
	msgs, onProjection := a.visibleInputForFold(stateSnapshot, canonical, transcriptVersion)
<<<<<<< HEAD
	viewInputHash := providerVisibleFingerprint(provider.ModelMessages(msgs))
	if a.sameTurnCompactionBlocked(activeTurn, trigger, mustFree, stateSnapshot, viewInputHash) {
		return CompactionNoop, nil
	}
	if trigger != CompactionTriggerManual && stateSnapshot.LastReceipt != nil && stateSnapshot.LastReceipt.Status == "applied" && stateSnapshot.LastReceipt.Action == "summary" && stateSnapshot.LastReceipt.InputHash == viewInputHash {
		return CompactionNoop, nil
	}
=======
	viewInputHash := providerVisibleFingerprint(modelInputMessages(msgs))
>>>>>>> origin/main-v2
	head, start, ok := a.planFoldRegion(msgs, force)
	if !ok {
		return CompactionNoop, nil
	}
<<<<<<< HEAD
	// start indexes the working view; covered is a canonical index. On a live
	// projection the view is frozen body + canonical[prior:], so a boundary
	// inside the body covers the prior range and past it maps offset-for-offset.
	covered := start
	var bodySuffix []provider.Message
	if onProjection {
		body := len(stateSnapshot.Projection.Messages)
		prior := stateSnapshot.Projection.CoveredCount
		if start < body {
			covered = prior
			// The unfolded remainder of the old body stays verbatim in the new
			// body; it has no canonical tail to splice from.
			bodySuffix = msgs[start:body]
		} else {
			covered = prior + (start - body)
		}
	}
	kept, fold, retention := a.partitionWithBudget(msgs, head, start, force)
	if len(fold) == 0 || (!force && !foldEconomics(fold)) {
=======
	_, preliminaryFold, _ := a.partitionFoldForProjection(msgs[head:start])
	if len(preliminaryFold) == 0 || (!force && !foldEconomics(preliminaryFold)) {
>>>>>>> origin/main-v2
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
<<<<<<< HEAD
=======
	if mustFree {
		start = a.maximumSafeSummaryPrefixEnd(msgs, head, start, instructions)
		if start <= head {
			a.emitCompactionAborted(trigger)
			return CompactionNoop, fmt.Errorf("%w: no balanced prefix leaves enough room for a summary response", errCheckpointRejected)
		}
	}

	covered, bodySuffix := projectionCoverageForFold(stateSnapshot, msgs, start, onProjection)
	kept, fold, retention := a.partitionFoldForProjection(msgs[head:start])
	if len(fold) == 0 {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, nil
	}
	originalFoldHash := providerVisibleFingerprint(modelInputMessages(fold))
	var err error
>>>>>>> origin/main-v2
	fold, instructions, err = a.interceptCompactionPrepare(ctx, fold, instructions)
	if err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
	if len(fold) == 0 {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, nil
	}

<<<<<<< HEAD
	sourceTokens := a.estimatedPromptTokens(msgs)
	res, tele, err := a.foldOrDegrade(ctx, trigger, mustFree, msgs[:head], fold, instructions, sourceTokens)
=======
	sourceTokens := a.estimatedVisibleRequestTokens(msgs)
	inputMode := SummaryInputCachePrefix
	if providerVisibleFingerprint(modelInputMessages(fold)) != originalFoldHash {
		inputMode = SummaryInputExtensionRewritten
	}
	res, tele, err := a.foldSummaryWithTelemetry(ctx, trigger, fold, instructions, sourceTokens, inputMode)
>>>>>>> origin/main-v2
	if err != nil {
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
	if res.Mode == CompactionModeDegraded {
		// A mechanical fold must keep user turns verbatim regardless of how
		// the fold was triggered: "through the summary" never existed.
		kept = a.keepDegradedUserTurnsVerbatim(msgs, head, start, kept, fold, res.Text, &retention)
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
<<<<<<< HEAD
	projTokens := a.estimatedPromptTokens(spliced)
	fixedPrefixTokens = a.estimatedPromptTokens(msgs[:head])
=======
	projTokens := a.estimatedVisibleRequestTokens(spliced)
>>>>>>> origin/main-v2
	tele.ProjectionTokens = projTokens
	tele.UserTurnsKept, tele.UserTurnsDropped = retention.Kept, retention.Dropped
	tele.Status = CompactionStatusInstalled
	a.emitCompactionTelemetry(tele)
	if err := a.acceptCheckpointCandidate(trigger, sourceTokens, projTokens); err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}
<<<<<<< HEAD
	viewOutputHash := providerVisibleFingerprint(provider.ModelMessages(spliced))
=======
	viewOutputHash := providerVisibleFingerprint(modelInputMessages(spliced))
>>>>>>> origin/main-v2
	_, err = a.commitSummaryProjection(summaryProjectionCommit{
		canonical: canonical, fold: fold, projected: projMsgs, result: res,
		transcriptVersion: transcriptVersion, projectionVersion: startProjectionVersion,
		generation: startGeneration, activeTurn: activeTurn, trigger: trigger,
		summary: summary, inputHash: viewInputHash, outputHash: viewOutputHash,
		sourceTokens: sourceTokens, projectionTokens: projTokens, covered: covered,
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

<<<<<<< HEAD
// visibleInputForFold prefers the prior projection + new history over full
// canonical. The second return reports whether the projection was used, so
// fold boundaries can be translated back to canonical indices.
func (a *Agent) visibleInputForFold(state CompactionState, canonical []provider.Message, transcriptVersion uint64) ([]provider.Message, bool) {
	if a.projectionUsable(state, canonical, transcriptVersion) {
		if projected := modelVisibleFromProjection(state.Projection, canonical); len(projected) > 0 {
			return projected, true
		}
	}
	if degraded, ok := a.modelVisibleDegraded(state, canonical); ok {
		return degraded, true
	}
	return canonical, false
}

=======
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

// visibleInputForFold prefers the prior projection + new history over full
// canonical. The second return reports whether the projection was used, so
// fold boundaries can be translated back to canonical indices.
func (a *Agent) visibleInputForFold(state CompactionState, canonical []provider.Message, transcriptVersion uint64) ([]provider.Message, bool) {
	if projectionValid(state, canonical, a.currentPromptCacheKey()) {
		if projected := modelVisibleFromProjection(state.Projection, canonical); len(projected) > 0 {
			return projected, true
		}
	}
	return canonical, false
}

>>>>>>> origin/main-v2
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
	if active := a.activeTurnStart(msgs); active >= head && active < start {
		start = active
	}
	return head, start, start > head
}

<<<<<<< HEAD
// partitionWithBudget partitions the fold region against the real projection
// room, then trims the summarizer request to the shared window. The projection
// is the only view ever sent again, so user turns keep their text verbatim for
// as long as the ceiling allows.
func (a *Agent) partitionWithBudget(msgs []provider.Message, head, start int, force bool) (kept, fold []provider.Message, retention userTurnRetention) {
	projBase := a.estimatedPromptTokens(msgs[:head]) + a.estimatedPromptTokens(msgs[start:]) + summaryHeadroomTokens
	projCap := a.compactTrigger()
	if !force {
		projCap = a.checkpointCeiling()
	}
	kept, fold, retention = a.partitionFoldForProjection(msgs[head:start], projBase, projCap)
	// The summarizer request = prefix + fold + system prompt must stay inside
	// the shared window: overflow fold messages keep their text verbatim.
	kept, fold = a.keepFoldWithinSummaryBudget(msgs[:head], kept, fold, projBase, projCap)
	return kept, fold, retention
}

func (a *Agent) partitionFoldForProjection(region []provider.Message, projBase, projCap int) (kept, fold []provider.Message, retention userTurnRetention) {
	policyKeep, retention := a.keepIndexes(region, projBase, projCap)
	for i, m := range region {
		switch {
		case m.LocalOnly: // display-only output never reaches a provider
		case isCompactionSummary(m):
			// Always merge prior digests into the single next summary.
			fold = append(fold, m)
		case policyKeep[i]:
			kept = append(kept, a.keptForProjection(m))
		default:
			fold = append(fold, m)
=======
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
	maxPromptTokens := a.hardInputCeiling()
	if policy.WindowMode == provider.ContextWindowShared {
		maxPromptTokens = window - outputBudgetReserve - 256
	}
	if maxPromptTokens <= 0 {
		return head
	}
	fits := func(candidate int) bool {
		request := a.summaryRequest(msgs[head:candidate], instructions)
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
>>>>>>> origin/main-v2
		}
	}
	return kept, fold, retention
}

// runCompactionSummary uses the single local summarizer path for every provider.
// prefix is the main-request prefix (msgs[:head]) reused so the summarizer
// request hits the provider prefix cache instead of paying full price.
func (a *Agent) runCompactionSummary(ctx context.Context, prefix, fold []provider.Message, instructions string) (summary, mode string, usage *provider.Usage, providerReqID string, err error) {
	summary, usage, err = a.summarizeOnce(ctx, prefix, fold, instructions)
	if err != nil {
		return "", CompactionModeSummarized, usage, "", err
	}
	return summary, CompactionModeSummarized, usage, "", nil
}

// keepDegradedUserTurnsVerbatim keeps user turns verbatim when a mechanical
// fold had no real summary — they must not survive "through the summary" that
// never existed. The candidate stays under the trigger ceiling or it would be
// rejected and re-trigger every turn; newest first, as many as fit.
func (a *Agent) keepDegradedUserTurnsVerbatim(msgs []provider.Message, head, start int, kept, fold []provider.Message, summary string, retention *userTurnRetention) []provider.Message {
	cap := a.compactTrigger()
	proj := estimateMessagesTokens(msgs[:head]) + estimateMessagesTokens(msgs[start:]) + estimateTextTokens(summary)
	for _, m := range kept {
		proj += estimateMessagesTokens([]provider.Message{m})
	}
	for _, m := range fold {
		if m.Role != provider.RoleUser || m.LocalOnly || isCompactionSummary(m) {
			continue
		}
		extra := estimateMessagesTokens([]provider.Message{a.keptForProjection(m)})
		if cap > 0 && proj+extra >= cap {
			break
		}
		kept = append(kept, a.keptForProjection(m))
		proj += extra
		if retention != nil && retention.Dropped > 0 {
			// A mechanical fold keeps the turn verbatim; it is no longer a
			// summary-only survivor, so the dropped bookkeeping must follow.
			retention.Dropped--
			retention.DroppedTokens -= fixedTokenEstimate(m)
			if retention.DroppedTokens < 0 {
				retention.DroppedTokens = 0
			}
		}
	}
	return kept
}
