package agent

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	summary.Origin = provider.MessageOriginHost
	summary.Content = msg.Content[:i+len(summaryTagClose)]
	summary.RawContent = ""
	summary.Images = nil
	summary.ToolCalls = nil
	summary.ResponsesItems = nil
	summary.ServerSearch = nil
	summary.CreatedAt = 0
	user := msg
	// The legacy coalesced record did not retain the following turn's
	// provenance. Empty keeps old-session fallback available instead of
	// asserting that an old host continuation was user-authored.
	user.Origin = ""
	user.Content = msg.Content[i+len(separator):]
	user.RawContent = ""
	return summary, user, true
}

func compressAnchorCandidate(msg provider.Message) bool {
	if msg.Role != provider.RoleUser || msg.LocalOnly || isCompactionSummary(msg) {
		return false
	}
	return IsUserAuthoredTurnMessage(msg)
}

func anchorPreview(text string) string {
	return truncatePreview(previewProse(text))
}

type visibleCompressionPlan struct {
	result    tool.CompressResult
	foldMask  []bool
	dropMask  []bool
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

	res, err := a.foldToSummaryMode(ctx, nil, prepared.fold, prepared.instructions, prepared.inputMode)
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
	projection, pinnedCheckpoint, err := rebasePinnedContextProjection(projection, snap.canonical, len(snap.canonical))
	if err != nil {
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	projectionTokens := a.estimatedVisibleRequestTokens(projection)
	tele.ProjectionTokens = projectionTokens
	result.Messages = len(plan.fold)
	result.ProjectionTokens = projectionTokens
	result.Mode = res.Mode
	if projectionTokens >= result.SourceTokens {
		if pinnedCheckpoint {
			result.Reason = "pinned-context-too-large: checkpoint prevents compaction from reducing context"
			a.emitCompactionTelemetry(tele)
			a.emitCompactionAborted(trigger)
			return result, nil
		}
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
	plan.dropMask = make([]bool, len(snap.visible))
	plan.firstFold = len(snap.visible)
	latestContext := latestSessionContextIndex(snap.visible)
	for i, msg := range snap.visible {
		selected := i >= start && i < end
		mergeSummary := i < completedEnd && isCompactionSummary(msg)
		if isSessionContextMessage(msg) {
			// Context never enters the summarizer. Once an older snapshot falls
			// inside the explicitly compressed range, remove it from the
			// projection; the latest valid snapshot remains byte-identical.
			plan.dropMask[i] = selected && i != latestContext
			continue
		}
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
	filteredFold, removedPinned := withoutPinnedContextRevisions(fold)
	if len(filteredFold) == 0 {
		return preparedVisibleCompression{}, "selected range contains no summarizable messages", nil
	}
	if removedPinned {
		inputMode = SummaryInputNonPrefix
	}
	originalHash := providerVisibleFingerprint(modelInputMessages(filteredFold))
	preparedFold, preparedInstructions, err := a.interceptCompactionPrepare(ctx, filteredFold, instructions)
	if err != nil {
		return preparedVisibleCompression{}, "", err
	}
	preparedFold = modelInputMessages(preparedFold)
	if len(preparedFold) == 0 {
		return preparedVisibleCompression{}, "compaction hook removed the selected range", nil
	}
	if !removedPinned && providerVisibleFingerprint(modelInputMessages(preparedFold)) != originalHash {
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
		if !plan.foldMask[i] && (len(plan.dropMask) <= i || !plan.dropMask[i]) {
			projection = append(projection, msg)
		}
	}
	return projectionMessagesPreservingPinnedContext(projection)
}

func compactionTelemetryFromSummary(trigger, cacheState string, sourceTokens int, res foldSummary) CompactionTelemetry {
	tele := CompactionTelemetry{
		Trigger: trigger, CacheState: cacheState, Mode: res.Mode,
		SourceTokens:      sourceTokens,
		ProviderRequestID: res.RequestID,
		FoldTokens:        res.FoldTokens,
		Spans:             res.Spans,
		SummaryInputMode:  res.InputMode,
	}
	if tele.Spans <= 0 {
		tele.Spans = 1
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

// foldSummaryWithChunkedFallback retries summary size failures through the
// resilient fragment/tree-reduce path used for over-length sessions.
func (a *Agent) foldSummaryWithChunkedFallback(ctx context.Context, trigger string, prefix, fold []provider.Message, instructions string, sourceTokens int, inputMode string) (foldSummary, CompactionTelemetry, error) {
	res, tele, err := a.foldSummaryWithTelemetry(ctx, trigger, prefix, fold, instructions, sourceTokens, inputMode)
	if err == nil || (!errors.Is(err, errSummaryOutputTruncated) && !errors.Is(err, ErrCompactionRequired)) {
		return res, tele, err
	}
	chunked, chunkedErr := a.chunkedFoldSummary(ctx, prefix, fold, instructions, nil)
	chunked.Usage = mergeSamplingUsage(res.Usage, chunked.Usage)
	chunked.Spans += res.Spans
	if chunked.FoldTokens <= 0 {
		chunked.FoldTokens = res.FoldTokens
	}
	if chunked.RequestID == "" {
		chunked.RequestID = res.RequestID
	}
	if chunkedErr != nil {
		tele = compactionTelemetryFromSummary(trigger, a.CacheState(), sourceTokens, chunked)
		tele.Error = fmt.Sprintf("%v (chunked fallback: %v)", err, chunkedErr)
		return chunked, tele, chunkedErr
	}
	return chunked, compactionTelemetryFromSummary(trigger, a.CacheState(), sourceTokens, chunked), nil
}

// compact writes a context projection; trigger stays "auto"/"manual" for UI cards.
func (a *Agent) summarizeFold(ctx context.Context, trigger string, prefix, fold []provider.Message, instructions string, sourceTokens int, inputMode string, allowChunked bool) (foldSummary, CompactionTelemetry, error) {
	if allowChunked {
		return a.foldSummaryWithChunkedFallback(ctx, trigger, prefix, fold, instructions, sourceTokens, inputMode)
	}
	return a.foldSummaryWithTelemetry(ctx, trigger, prefix, fold, instructions, sourceTokens, inputMode)
}

func (a *Agent) compactToProjectionLocked(ctx context.Context, trigger, instructions string, force, mustFree, allowChunked bool) (CompactionOutcome, error) {
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
	latestContext := latestSessionContextIndex(msgs)
	_, preliminaryFold, _ := a.partitionFoldForProjectionAt(msgs[head:start], head, latestContext)
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
	// Cap every automatic summary input (#9572), including pressure folds after
	// projection invalidation. mustFree also covers the over-ceiling manual rescue
	// merged in #9474; ordinary manual compaction keeps its requested range.
	if mustFree || trigger != CompactionTriggerManual {
		start = a.maximumSafeSummaryPrefixEnd(msgs, head, start, instructions)
		if start <= head {
			a.emitCompactionAborted(trigger)
			return CompactionNoop, fmt.Errorf("%w: no balanced prefix leaves enough room for a summary response", errCheckpointRejected)
		}
	}

	covered, bodySuffix := projectionCoverageForFold(stateSnapshot, msgs, start, onProjection)
	regionHadPinnedRevision := containsPinnedContextRevision(msgs[head:start])
	kept, fold, retention := a.partitionFoldForProjectionAt(msgs[head:start], head, latestContext)
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
	if mustFree || trigger != CompactionTriggerManual {
		if err := a.validateSafeSummaryRequest(fold, instructions); err != nil {
			a.emitCompactionAborted(trigger)
			return CompactionNoop, err
		}
	}

	sourceTokens := a.estimatedVisibleRequestTokens(msgs)
	inputMode := SummaryInputCachePrefix
	if regionHadPinnedRevision {
		inputMode = SummaryInputNonPrefix
	} else if providerVisibleFingerprint(modelInputMessages(fold)) != originalFoldHash {
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
	res, tele, err := a.summarizeFold(ctx, trigger, summaryPrefix, foldExtra, instructions, sourceTokens, inputMode, allowChunked)
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
		projMsgs = append(projMsgs, projectionMessagesPreservingPinnedContext(bodySuffix)...)
	}
	tele.UserTurnsKept, tele.UserTurnsDropped = retention.Kept, retention.Dropped
	projMsgs, spliced, projTokens, err := a.preparePinnedCheckpointCandidate(trigger, projMsgs, canonical, covered, sourceTokens, &tele)
	if err != nil {
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

func (a *Agent) preparePinnedCheckpointCandidate(
	trigger string,
	projection, canonical []provider.Message,
	covered, sourceTokens int,
	tele *CompactionTelemetry,
) ([]provider.Message, []provider.Message, int, error) {
	projection, pinnedCheckpoint, err := rebasePinnedContextProjection(projection, canonical, covered)
	if err != nil {
		return nil, nil, 0, err
	}
	spliced := append(append([]provider.Message(nil), projection...), canonical[covered:]...)
	projectionTokens := a.estimatedVisibleRequestTokens(spliced)
	tele.ProjectionTokens = projectionTokens
	a.emitCompactionTelemetry(*tele)
	if err := a.acceptCheckpointCandidate(trigger, sourceTokens, projectionTokens); err != nil {
		if pinnedCheckpoint {
			return nil, nil, 0, fmt.Errorf("pinned-context-too-large: checkpoint prevents compaction acceptance: %w", err)
		}
		return nil, nil, 0, err
	}
	return projection, spliced, projectionTokens, nil
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

func checkpointProjectionMessages(msgs []provider.Message, head int, kept []provider.Message, summary string) []provider.Message {
	projMsgs := make([]provider.Message, 0, head+1+len(kept))
	projMsgs = append(projMsgs, msgs[:head]...)
	projMsgs = append(projMsgs, kept...)
	projMsgs = append(projMsgs, formatSummaryMessage(summary))
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

// maximumSafeSummaryPrefixEnd returns the largest balanced contiguous prefix
// whose exact summary request leaves the collector's minimum output budget.
// The remaining middle and tail stay verbatim in the projection.
func (a *Agent) maximumSafeSummaryPrefixEnd(msgs []provider.Message, head, end int, instructions string) int {
	if head < 0 || end <= head || end > len(msgs) {
		return end
	}
	maxPromptTokens, enforce := a.safeSummaryPromptTokenLimit()
	if !enforce {
		return end
	}
	if maxPromptTokens <= 0 {
		return head
	}
	fits := func(candidate int) bool {
		fold, _ := withoutPinnedContextRevisions(msgs[head:candidate])
		request := a.summaryRequest(msgs[:head], fold, instructions)
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

// safeSummaryPromptTokenLimit is shared by prefix planning and the final
// post-extension guard. Unknown gateways conservatively honor the configured
// or learned window; explicitly independent providers retain the full fold.
func (a *Agent) safeSummaryPromptTokenLimit() (int, bool) {
	window := a.effectiveContextWindow()
	if window <= 0 || contextBudgetPolicyOf(a.svc.prov).WindowMode == provider.ContextWindowIndependent {
		return 0, false
	}
	return window - a.summaryOutputBudget() - protocolReserveTokens, true
}

func (a *Agent) validateSafeSummaryRequest(fold []provider.Message, instructions string) error {
	maxPromptTokens, enforce := a.safeSummaryPromptTokenLimit()
	if !enforce {
		return nil
	}
	requestTokens := a.estimatedRequestTokens(a.summaryRequest(nil, fold, instructions))
	if maxPromptTokens <= 0 || requestTokens > maxPromptTokens {
		return fmt.Errorf("%w: prepared summary request (%d tokens) exceeds safe prompt budget (%d)",
			errCheckpointRejected, requestTokens, maxPromptTokens)
	}
	return nil
}

// summaryFoldPlan selects the cache-aligned summary request shape for the
// fold msgs[head:start]: the frozen main-request bytes when they cover the
// head, otherwise the live view (when it fits the admissible ceiling),
// otherwise the verbatim head + fold. The anchors name the fold region inside
// the replayed bytes so the summarizer summarizes only that segment.
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
		// For estimation, use only the fold region as prefix (matching v1.36.0).
		// The actual summaryFoldPlan may use all visible messages for cache
		// alignment, but the estimate should reflect the minimal request shape.
		return a.summaryRequest(msgs[head:candidate], nil, instructions+anchors)
	}
	return a.summaryRequest(msgs[:head], msgs[head:candidate], instructions)
}

// summaryMaxPromptTokens is the admissible summarizer input ceiling, shared by
// planning (maximumSafeSummaryPrefixEnd), the replay-fits check, and send-time
// admission so a planned request is never rejected after it is selected.
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
		return window - outputBudgetReserve - protocolReserveTokens
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
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= maxAnchor {
		return text
	}
	return text[:maxAnchor]
}

type userTurnRetention struct {
	Kept    int
	Dropped int
}

func (a *Agent) partitionFoldForProjection(region []provider.Message) (kept, fold []provider.Message, retention userTurnRetention) {
	return a.partitionFoldForProjectionAt(region, 0, latestSessionContextIndex(region))
}

func (a *Agent) partitionFoldForProjectionAt(region []provider.Message, offset, latestContext int) (kept, fold []provider.Message, retention userTurnRetention) {
	for i, m := range region {
		if m.LocalOnly || IsPinnedContextRevision(m) {
			continue
		}
		if isSessionContextMessage(m) {
			if offset+i == latestContext {
				kept = append(kept, m)
			}
			continue
		}
		fold = append(fold, m)
		if IsUserAuthoredTurnMessage(m) {
			retention.Dropped++
		}
	}
	return kept, fold, retention
}

func latestSessionContextIndex(messages []provider.Message) int {
	for i := range slices.Backward(messages) {
		if isSessionContextMessage(messages[i]) {
			return i
		}
	}
	return -1
}

// runCompactionSummary uses the single local summarizer path for every provider.
func (a *Agent) runCompactionSummary(ctx context.Context, prefix, fold []provider.Message, instructions string) (summary, mode string, usage *provider.Usage, providerReqID string, err error) {
	summary, usage, err = a.summarizeOnce(ctx, prefix, fold, instructions)
	if err != nil {
		return "", CompactionModeSummarized, usage, "", err
	}
	return summary, CompactionModeSummarized, usage, "", nil
}
