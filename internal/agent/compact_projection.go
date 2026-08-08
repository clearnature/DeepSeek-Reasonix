package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	promptCacheKey    string
}

func (a *Agent) snapshotExplicitCompression() explicitCompressionSnapshot {
	canonical, version := a.session.snapshotMessagesVersion()
	cacheKey := a.currentPromptCacheKey()
	state := a.compactionState
	visible := canonical
	if projectionValid(state, canonical, version, cacheKey) {
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
	archive      string
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
	plan, ok := a.planVisibleCompression(snap, direction, anchorIndex, preview)
	if !ok {
		return plan.result, nil
	}
	result := plan.result

	a.sink.Emit(event.Event{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: trigger}})
	prepared, reason, err := a.prepareVisibleCompression(ctx, trigger, plan.fold, instructions)
	if err != nil {
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	if reason != "" {
		a.emitCompactionAborted(trigger)
		result.Reason = reason
		return result, nil
	}

	summary, mode, usage, providerReqID, err := a.runCompactionSummary(ctx, prepared.fold, prepared.instructions)
	tele := compactionTelemetryFromSummary(trigger, a.CacheState(), result.SourceTokens, mode, usage, providerReqID)
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
	projectionTokens := estimateMessagesTokens(a.providerProjectionMessages(projection))
	tele.ProjectionTokens = projectionTokens
	result.Messages = len(plan.fold)
	result.ProjectionTokens = projectionTokens
	result.Mode = mode
	if projectionTokens >= result.SourceTokens {
		result.Reason = "compressed context would not be smaller"
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return result, nil
	}

	if err := a.installVisibleCompression(snap, trigger, mode, summary, projection, result.SourceTokens, projectionTokens, usage); err != nil {
		if errors.Is(err, errCompressStaleContext) {
			tele.Error = err.Error()
			a.emitCompactionTelemetry(tele)
		}
		a.emitCompactionAborted(trigger)
		return tool.CompressResult{}, err
	}
	a.session.NoteContentRewrite("compact_" + trigger)
	a.emitCompactionTelemetry(tele)
	a.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{
		Trigger: trigger, Messages: len(plan.fold), Summary: summary, Archive: prepared.archive,
	}})
	result.Status = "ok"
	result.Reason = ""
	return result, nil
}

func (a *Agent) planVisibleCompression(snap explicitCompressionSnapshot, direction string, anchorIndex int, preview string) (visibleCompressionPlan, bool) {
	sourceTokens := estimateMessagesTokens(snap.visible)
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

func (a *Agent) prepareVisibleCompression(ctx context.Context, trigger string, fold []provider.Message, instructions string) (preparedVisibleCompression, string, error) {
	if a.hooks != nil {
		if hookInstructions := a.hooks.PreCompact(ctx, trigger); hookInstructions != "" {
			if instructions != "" {
				instructions += "\n"
			}
			instructions += hookInstructions
		}
	}
	preparedFold, preparedInstructions, err := a.interceptCompactionPrepare(ctx, fold, instructions)
	if err != nil {
		return preparedVisibleCompression{}, "", err
	}
	preparedFold = provider.ModelMessages(preparedFold)
	if len(preparedFold) == 0 {
		return preparedVisibleCompression{}, "compaction hook removed the selected range", nil
	}
	prepared := preparedVisibleCompression{fold: preparedFold, instructions: preparedInstructions}
	if a.archiveDir == "" {
		return prepared, "", nil
	}
	prepared.archive, err = archiveMessages(a.archiveDir, preparedFold)
	if err != nil {
		return preparedVisibleCompression{}, "", fmt.Errorf("archive: %w", err)
	}
	return prepared, "", nil
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
	return provider.ModelMessages(projection)
}

func (a *Agent) installVisibleCompression(snap explicitCompressionSnapshot, trigger, mode, summary string, projection []provider.Message, sourceTokens, projectionTokens int, usage *provider.Usage) error {
	current, currentVersion := a.session.snapshotMessagesVersion()
	if currentVersion != snap.transcriptVersion || len(current) != len(snap.canonical) ||
		coveredPrefixHash(current, len(current)) != snap.coveredHash ||
		a.compactionState.Projection.ProjectionVersion != snap.projectionVersion {
		return errCompressStaleContext
	}
	now := time.Now().UTC()
	state := CompactionState{
		SchemaVersion:     compactionStateSchemaCurrent,
		TranscriptVersion: snap.transcriptVersion,
		Projection: ContextProjection{
			Messages:          projection,
			TranscriptVersion: snap.transcriptVersion,
			ProjectionVersion: snap.projectionVersion + 1,
			CoveredCount:      len(snap.canonical),
			CoveredPrefixHash: snap.coveredHash,
			SummaryHash:       summaryContentHash(summary),
			SourceTokens:      sourceTokens,
			ProjectionTokens:  projectionTokens,
			CreatedAt:         now,
		},
		PromptCacheKey:   snap.promptCacheKey,
		LastCacheState:   a.CacheState(),
		LastTrigger:      trigger,
		LastMode:         mode,
		LastSourceTokens: sourceTokens,
		LastResultTokens: projectionTokens,
		UpdatedAt:        now,
	}
	if a.pricing != nil && usage != nil {
		state.LastCompactionCost = a.pricing.Cost(usage)
	}
	if err := a.installProjection(state); err != nil {
		return fmt.Errorf("persist projection: %w", err)
	}
	return nil
}

func compactionTelemetryFromSummary(trigger, cacheState string, sourceTokens int, mode string, usage *provider.Usage, providerReqID string) CompactionTelemetry {
	tele := CompactionTelemetry{
		Trigger: trigger, CacheState: cacheState, Mode: mode,
		Native: mode == CompactionModeNative, SourceTokens: sourceTokens,
		ProviderRequestID: providerReqID,
	}
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
	_, err := a.compactToProjection(ctx, trigger, instructions, force)
	return err
}

// compactToProjection summarizes the older middle of the session into a model-
// visible projection. The canonical transcript is never rewritten. force
// bypasses the fold-economics skip. CompactionNoop means no projection was
// installed (nothing to fold); callers at the force threshold must treat that
// as a hard failure rather than sending the oversized canonical prompt.
func (a *Agent) compactToProjection(ctx context.Context, trigger, instructions string, force bool) (CompactionOutcome, error) {
	canonical, transcriptVersion := a.session.snapshotMessagesVersion()
	// Fold against the current model-visible view when a projection is valid:
	// an installed prune/snip projection already collapsed stale tool results.
	// Coverage still maps to the canonical transcript below.
	msgs := canonical
	if st := a.compactionState; projectionValid(st, canonical, transcriptVersion, a.currentPromptCacheKey()) {
		if visible := modelVisibleFromProjection(st.Projection, canonical); len(visible) > 0 {
			msgs = visible
		}
	}
	// Incremental fold: with a valid projection, fold only the appended
	// messages and append the digest — prior bytes keep hitting; otherwise a
	// full re-fold (digest merge) is a rare prefix rewrite.
	if outcome, err := a.tryIncrementalFold(ctx, trigger, instructions, force, canonical, transcriptVersion); err != nil || outcome != CompactionNoop {
		return outcome, err
	}
	head, start, kept, fold, ok := a.planFold(msgs, force)
	if !ok {
		return CompactionNoop, nil
	}

	a.sink.Emit(event.Event{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: trigger}})

	if a.hooks != nil {
		if hookInstr := a.hooks.PreCompact(ctx, trigger); hookInstr != "" {
			if instructions != "" {
				instructions += "\n"
			}
			instructions += hookInstr
		}
	}

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

	archived := ""
	if a.archiveDir != "" {
		path, aerr := archiveMessages(a.archiveDir, fold)
		if aerr != nil {
			a.emitCompactionAborted(trigger)
			return CompactionNoop, fmt.Errorf("archive: %w", aerr)
		}
		archived = path
	}

	sourceTokens := estimateMessagesTokens(provider.ModelMessages(msgs))
	summary, mode, usage, providerReqID, err := a.runCompactionSummary(ctx, fold, instructions)
	tele := compactionTelemetryFromSummary(trigger, a.CacheState(), sourceTokens, mode, usage, providerReqID)
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}

	summary, err = a.interceptCompactionComplete(ctx, summary)
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}

	early := a.fixedEarlyUserTurns(msgs, head)
	projMsgs := make([]provider.Message, 0, head+len(early)+1+len(kept)+len(msgs)-start)
	projMsgs = append(projMsgs, msgs[:head]...)
	projMsgs = append(projMsgs, early...)
	projMsgs = append(projMsgs, formatSummaryMessage(summary))
	projMsgs = append(projMsgs, kept...)
	projMsgs = append(projMsgs, msgs[start:]...)
	projMsgs = provider.ModelMessages(projMsgs)

	projTokens := estimateMessagesTokens(a.providerProjectionMessages(projMsgs))
	tele.ProjectionTokens = projTokens
	a.emitCompactionTelemetry(tele)

	projVersion := a.compactionState.Projection.ProjectionVersion + 1
	st := a.buildCompactionState(projMsgs, canonical, transcriptVersion, summary, mode, usage, sourceTokens, projTokens, projVersion, trigger)
	if err := a.installProjection(st); err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, fmt.Errorf("persist projection: %w", err)
	}
	a.session.NoteContentRewrite("compact_" + trigger)

	a.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{
		Trigger: trigger, Messages: len(fold), Summary: summary, Archive: archived,
	}})
	return CompactionInstalled, nil
}

// tryIncrementalFold runs the incremental path when a valid projection covers
// the earlier history. Returns CompactionNoop when no incremental fold applies.
func (a *Agent) tryIncrementalFold(ctx context.Context, trigger, instructions string, force bool, msgs []provider.Message, transcriptVersion uint64) (CompactionOutcome, error) {
	base, head, start, ok := a.incrementalFoldTarget(msgs, transcriptVersion, trigger)
	if !ok {
		return CompactionNoop, nil
	}
	outcome, err := a.compactIncremental(ctx, trigger, instructions, force, base, msgs, transcriptVersion, head, start)
	if err != nil || outcome != CompactionInstalled {
		return outcome, err
	}
	// Chain invariant: an incremental step must strictly advance coverage. A
	// projection that does not move is a stale chain link — fail closed rather
	// than keep riding it (Sovereign chainAssoc analog: every step well-defined).
	if got := a.compactionState.Projection.CoveredCount; got <= base.CoveredCount {
		return CompactionNoop, fmt.Errorf("incremental fold did not advance coverage (%d → %d)", base.CoveredCount, got)
	}
	// The incremental fold keeps prior bytes (prefix hit) but may leave the
	// projection too close to the trigger; converge with one full re-fold (the
	// recursive call takes the full path — the new projection covers all).
	_, _, high := a.compactThresholds()
	if a.estimatedPromptTokens(a.compactionState.Projection.Messages) >= high-a.compactionTailBudget() {
		return a.compactToProjection(ctx, trigger, instructions, force)
	}
	return CompactionInstalled, nil
}

// incrementalFoldTarget picks the incremental path (append to the existing
// projection) over a full re-fold: a valid projection with un-covered messages
// under the projection budget, not a manual/overflow trigger, and no trailing
// user run that appending would coalesce across.
func (a *Agent) incrementalFoldTarget(msgs []provider.Message, transcriptVersion uint64, trigger string) (baseProj ContextProjection, head, start int, ok bool) {
	st := a.compactionState
	if !projectionValid(st, msgs, transcriptVersion, a.currentPromptCacheKey()) {
		return ContextProjection{}, 0, 0, false
	}
	covered := st.Projection.CoveredCount
	if covered <= 0 || covered >= len(msgs) {
		return ContextProjection{}, 0, 0, false
	}
	if a.estimatedPromptTokens(st.Projection.Messages) >= a.projectionCompactBudget() {
		return ContextProjection{}, 0, 0, false
	}
	if trigger == CompactionTriggerManual || trigger == CompactionTriggerOverflow {
		return ContextProjection{}, 0, 0, false
	}
	if a.strictAlternatingRoles && len(st.Projection.Messages) > 0 &&
		st.Projection.Messages[len(st.Projection.Messages)-1].Role == provider.RoleUser {
		return ContextProjection{}, 0, 0, false
	}
	head, start, ok = a.incrementalFoldRange(msgs, covered)
	if !ok {
		return ContextProjection{}, 0, 0, false
	}
	return st.Projection, head, start, true
}

// incrementalFoldRange returns the canonical range [head,start) a fold should
// cover when a projection already covers the earlier history: only the messages
// appended since that projection, aligned off any tool message so the fold never
// begins with an orphan tool result. ok is false when there is nothing new.
func (a *Agent) incrementalFoldRange(msgs []provider.Message, covered int) (head, start int, ok bool) {
	head = covered
	for head >= 0 && head < len(msgs) && msgs[head].Role == provider.RoleTool {
		head--
	}
	// A tool result at the boundary belongs to a pre-covered turn; folding it
	// would duplicate base content, so degrade and let the caller use full.
	if head < covered {
		return covered, covered, false
	}
	start = tailStart(msgs, head, a.compactionTailBudget(), a.tokPerChar(), a.tailFloor())
	if start <= head {
		return head, start, false
	}
	return head, start, true
}

// projectionCompactBudget is the projection-size ceiling before a wholesale
// re-fold (digest merge): below it the incremental path extends the projection,
// at or above it only the full path applies. Half the window keeps the view
// comfortably under the 80% fold trigger even with a fresh tail.
func (a *Agent) projectionCompactBudget() int {
	if a.contextWindow <= 0 {
		return 0
	}
	return int(float64(a.contextWindow) * 0.5)
}

// buildCompactionState assembles the sidecar payload shared by the full and
// incremental fold paths.
func (a *Agent) buildCompactionState(projMsgs []provider.Message, msgs []provider.Message, transcriptVersion uint64, summary, mode string, usage *provider.Usage, sourceTokens, projTokens int, projVersion uint64, trigger string) CompactionState {
	st := CompactionState{
		SchemaVersion:     compactionStateSchemaCurrent,
		TranscriptVersion: transcriptVersion,
		Projection: ContextProjection{
			Messages:          projMsgs,
			TranscriptVersion: transcriptVersion,
			ProjectionVersion: projVersion,
			CoveredCount:      len(msgs),
			CoveredPrefixHash: coveredPrefixHash(msgs, len(msgs)),
			SummaryHash:       summaryContentHash(summary),
			SourceTokens:      sourceTokens,
			ProjectionTokens:  projTokens,
			CreatedAt:         time.Now().UTC(),
		},
		PromptCacheKey:   a.currentPromptCacheKey(),
		LastCacheState:   a.CacheState(),
		LastTrigger:      trigger,
		LastMode:         mode,
		LastSourceTokens: sourceTokens,
		LastResultTokens: projTokens,
		UpdatedAt:        time.Now().UTC(),
	}
	if a.pricing != nil && usage != nil {
		st.LastCompactionCost = a.pricing.Cost(usage)
	}
	return st
}

// compactIncremental folds only the canonical messages appended since the last
// projection, then appends the new digest to the existing projection messages.
// The prior projection bytes are untouched (the server prefix keeps hitting);
// only the newly added segment is re-shaped. Canonical history is never
// rewritten; dropped originals are archived as in the full path.
func (a *Agent) compactIncremental(ctx context.Context, trigger, instructions string, force bool, baseProj ContextProjection, msgs []provider.Message, transcriptVersion uint64, head, start int) (CompactionOutcome, error) {
	region := msgs[head:start]
	kept, fold := a.partitionFoldForProjectionIncremental(region)
	if len(fold) == 0 {
		return CompactionNoop, nil
	}
	// Same bound as the full path: a huge appended block must not make one
	// summarize call blow past summaryTimeout or overflow the window.
	headTokens := estimateMessagesTokens(provider.ModelMessages(msgs[:head]))
	tailTokens := estimateMessagesTokens(provider.ModelMessages(msgs[start:]))
	fold, kept = a.fitFoldToWindow(fold, kept, headTokens, tailTokens)
	if len(fold) == 0 {
		return CompactionNoop, nil
	}
	if !force && !foldEconomics(fold) {
		return CompactionNoop, nil
	}

	a.sink.Emit(event.Event{Kind: event.CompactionStarted, Compaction: event.Compaction{Trigger: trigger}})

	if a.hooks != nil {
		if hookInstr := a.hooks.PreCompact(ctx, trigger); hookInstr != "" {
			if instructions != "" {
				instructions += "\n"
			}
			instructions += hookInstr
		}
	}

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

	archived := ""
	if a.archiveDir != "" {
		path, aerr := archiveMessages(a.archiveDir, fold)
		if aerr != nil {
			a.emitCompactionAborted(trigger)
			return CompactionNoop, fmt.Errorf("archive: %w", aerr)
		}
		archived = path
	}

	sourceTokens := estimateMessagesTokens(provider.ModelMessages(msgs))
	summary, mode, usage, providerReqID, err := a.runCompactionSummary(ctx, fold, instructions)
	tele := CompactionTelemetry{
		Trigger:           trigger,
		CacheState:        a.CacheState(),
		Mode:              mode,
		Native:            mode == CompactionModeNative,
		SourceTokens:      sourceTokens,
		ProviderRequestID: providerReqID,
	}
	if usage != nil {
		tele.InputTokens = usage.PromptTokens
		tele.OutputTokens = usage.CompletionTokens
		tele.CacheHitTokens = usage.CacheHitTokens
		tele.CacheMissTokens = usage.CacheMissTokens
		tele.CacheWriteTokens = usage.CacheWriteTokens
		tele.RequestCount = usage.RequestCount
		if tele.RequestCount <= 0 {
			tele.RequestCount = 1
		}
	}
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}

	summary, err = a.interceptCompactionComplete(ctx, summary)
	if err != nil {
		tele.Error = err.Error()
		a.emitCompactionTelemetry(tele)
		a.emitCompactionAborted(trigger)
		return CompactionNoop, err
	}

	// Append-only rebuild: prior bytes verbatim; coalescing only the added
	// segment (the caller degraded to a full fold when the boundary touches a
	// trailing user run).
	base := append([]provider.Message(nil), baseProj.Messages...)
	added := append([]provider.Message{formatSummaryMessage(summary)}, kept...)
	added = append(added, msgs[start:]...)
	added = provider.ModelMessages(added)
	if a.strictAlternatingRoles {
		added = coalesceProjectionUserRuns(added)
	}
	projMsgs := append(base, added...)

	projTokens := estimateMessagesTokens(projMsgs)
	tele.ProjectionTokens = projTokens
	a.emitCompactionTelemetry(tele)

	projVersion := baseProj.ProjectionVersion + 1
	st := a.buildCompactionState(projMsgs, msgs, transcriptVersion, summary, mode, usage, sourceTokens, projTokens, projVersion, trigger)
	if err := a.installProjection(st); err != nil {
		a.emitCompactionAborted(trigger)
		return CompactionNoop, fmt.Errorf("persist projection: %w", err)
	}
	a.session.NoteContentRewrite("compact_" + trigger)

	a.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{
		Trigger: trigger, Messages: len(fold), Summary: summary, Archive: archived,
	}})
	return CompactionInstalled, nil
}

// partitionFoldForProjection is like partitionFold but prior digests join the
// fold so A1 rolling merge produces a single latest summary. Fixed early user
// turns are excluded from both kept and fold — the caller re-inserts them from
// the full transcript so their bytes stay position-stable.
func (a *Agent) partitionFoldForProjection(region []provider.Message) (kept, fold []provider.Message) {
	return a.partitionFoldForProjectionMode(region, true)
}

// partitionFoldForProjectionIncremental is the incremental-path variant: the
// fixed early-window skip is disabled because the fold region starts at the
// covered boundary — there is no early user prefix to re-insert, and skipping
// would silently drop freshly appended small user turns from the projection.
func (a *Agent) partitionFoldForProjectionIncremental(region []provider.Message) (kept, fold []provider.Message) {
	return a.partitionFoldForProjectionMode(region, false)
}

func (a *Agent) partitionFoldForProjectionMode(region []provider.Message, skipEarly bool) (kept, fold []provider.Message) {
	policyKeep := keepIndexes(region, a.keepPolicy)
	earlySeen := 0
	const maxEarly = 3
	for i, m := range region {
		if m.LocalOnly {
			continue
		}
		// Skip the fixed early small user turns — they are re-added from the
		// full transcript after the summary so the prefix stays byte-stable.
		if skipEarly && m.Role == provider.RoleUser && !isCompactionSummary(m) && a.fixedPinnableUserTurn(m) && earlySeen < maxEarly {
			earlySeen++
			continue
		}
		if isCompactionSummary(m) {
			fold = append(fold, m)
			continue
		}
		if policyKeep[i] {
			kept = append(kept, m)
			continue
		}
		if m.Role == provider.RoleUser && a.fixedPinnableUserTurn(m) {
			// Additional small user turns beyond the fixed early window fold so
			// the projection does not grow unbounded with every user fact.
			fold = append(fold, m)
			continue
		}
		fold = append(fold, m)
	}
	return kept, fold
}

// planFold picks the fold region and applies bounded folding: when the fold
// alone cannot fit the shared window (an invalidated sidecar rebuild starts
// from canonical, possibly far over the window), the newer tail drops back
// into kept instead of failing — later turns fold it.
func (a *Agent) planFold(msgs []provider.Message, force bool) (head, start int, kept, fold []provider.Message, ok bool) {
	head, start, ok = a.planCompaction(msgs, minCompactMessages)
	if !ok {
		head, start, ok = a.planCompaction(msgs, 1)
	}
	if !ok {
		return 0, 0, nil, nil, false
	}
	if active := a.activeTurnStart(msgs); active >= head && active < start {
		start = active
		if start <= head {
			return 0, 0, nil, nil, false
		}
	}
	region := msgs[head:start]
	kept, fold = a.partitionFoldForProjection(region)
	headTokens := estimateMessagesTokens(provider.ModelMessages(msgs[:head]))
	tailTokens := estimateMessagesTokens(provider.ModelMessages(msgs[start:]))
	fold, kept = a.fitFoldToWindow(fold, kept, headTokens, tailTokens)
	if len(fold) == 0 {
		return 0, 0, nil, nil, false
	}
	if !force && !foldEconomics(fold) {
		return 0, 0, nil, nil, false
	}
	return head, start, kept, fold, true
}

// maxCompactFoldTokens caps one summarizer fold so a single call finishes
// within summaryTimeout. The effective cap is the smaller of this and the
// projection budget (window - head - tail - kept - summary).
const maxCompactFoldTokens = 600_000

// summaryTokensBudget reserves space for the distilled summary inside the new
// projection, so the fold budget keeps the projection inside the window.
const summaryTokensBudget = 12_000

// fitFoldToWindow bounds the fold so the rebuilt projection (head + summary +
// kept + tail) fits the shared window: fold at least foldTokens - avail but no
// more than maxCompactFoldTokens per call; the deferred tail folds later.
func (a *Agent) fitFoldToWindow(fold, kept []provider.Message, headTokens, tailTokens int) ([]provider.Message, []provider.Message) {
	if a.contextWindow <= minOutputBudget || !sharesContextWindow(a.prov) || len(fold) == 0 {
		return fold, kept
	}
	keptTokens := estimateMessagesTokens(provider.ModelMessages(kept))
	foldTokens := estimateMessagesTokens(provider.ModelMessages(fold))
	avail := a.contextWindow - minOutputBudget - headTokens - tailTokens - keptTokens - summaryTokensBudget
	minFold := foldTokens - avail
	if minFold <= 0 {
		return fold, kept // the projection already fits; nothing to trim
	}
	if minFold > maxCompactFoldTokens {
		minFold = maxCompactFoldTokens // single-round limit; the rest folds later
	}
	// Move the newer tail back into kept until the remaining fold fits the
	// projection budget; when even maxCompactFoldTokens cannot reach it, fold
	// the cap and defer the rest to later turns (window may exceed this round).
	sizes := make([]int, len(fold))
	remaining := foldTokens
	cut := len(fold)
	for i := len(fold) - 1; i >= 0; i-- {
		sizes[i] = estimateMessagesTokens([]provider.Message{fold[i]})
		remaining -= sizes[i]
		cut = i
		if remaining <= avail {
			break
		}
	}
	if cut == len(fold) || cut == 0 {
		return fold, kept // nothing to move, or a single message exceeds the budget
	}
	if folded := foldTokens - remaining; folded > maxCompactFoldTokens {
		for i := cut - 1; i >= 0; i-- {
			sizes[i] = estimateMessagesTokens([]provider.Message{fold[i]})
			remaining -= sizes[i]
			cut = i
			if foldTokens-remaining <= maxCompactFoldTokens || cut == 0 {
				break
			}
		}
	}
	movedMsgs := append([]provider.Message(nil), fold[cut:]...)
	return fold[:cut], append(movedMsgs, kept...)
}

// runCompactionSummary tries native compaction first, then summarizeWithRetry.
// On total failure it returns the error without a mechanical marker.
func (a *Agent) runCompactionSummary(ctx context.Context, fold []provider.Message, instructions string) (summary, mode string, usage *provider.Usage, providerReqID string, err error) {
	if nc, ok := provider.AsNativeCompactor(a.prov); ok {
		maxOut := 0
		if cc, ok := provider.AsCompactionCapabler(a.prov); ok {
			caps := cc.CompactionCapabilities()
			maxOut = caps.CompactionOutputTokens
			if maxOut <= 0 {
				maxOut = caps.MaxOutputTokens
			}
		}
		res, nerr := nc.Compact(ctx, provider.CompactionRequest{
			Messages:        fold,
			Instructions:    instructions,
			MaxOutputTokens: maxOut,
			PromptCacheKey:  promptCacheKey(a.workspaceID, BranchID(a.sessionPath), a.modelRef),
			SessionID:       BranchID(a.sessionPath),
		})
		if nerr == nil && res.Valid() {
			if res.Summary != "" {
				return res.Summary, CompactionModeNative, res.Usage, res.ProviderRequestID, nil
			}
			// Provider returned a full projection; extract summary text if present,
			// otherwise render the projection as the digest body.
			if s := extractLatestSummary(res.Projection); s != "" {
				return s, CompactionModeNative, res.Usage, res.ProviderRequestID, nil
			}
			return renderTranscript(res.Projection), CompactionModeNative, res.Usage, res.ProviderRequestID, nil
		}
		if nerr != nil && !errors.Is(nerr, provider.ErrCompactionUnsupported) {
			// Hard native failure: still try ordinary summarize fallback.
			a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: "Native compaction unavailable; using summary fallback.", Detail: nerr.Error()})
		}
	}
	summary, usage, err = a.summarizeWithRetry(ctx, fold, instructions)
	if err != nil {
		return "", CompactionModeSummarized, usage, "", err
	}
	return summary, CompactionModeSummarized, usage, "", nil
}

// snipToProjection builds a projection that only snips stale tool results.
func (a *Agent) snipToProjection(ctx context.Context) error {
	_ = ctx
	msgs, _ := a.session.snapshotMessagesVersion()
	snipped, st := a.applyToolResultMaintenanceView(msgs, toolResultSnip)
	if st.Results == 0 {
		return nil
	}
	ratio := a.tokPerChar()
	saved := int(float64(st.SavedChars) * ratio)
	a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
		"snipped %d stale tool results (~%d tokens est.) before compaction", st.Results, saved)})
	return a.installPruneProjection(snipped, st)
}

// installPruneProjection stores a projection whose messages are a snipped/pruned
// view of the canonical transcript (no summarizer call).
func (a *Agent) installPruneProjection(view []provider.Message, st PruneStats) error {
	// Let the response-side cache-break detector know the prefix intentionally
	// shrank (snip/prune) — without this, hit-token drops would be misreported.
	a.session.NoteContentRewrite("snip")
	msgs, version := a.session.snapshotMessagesVersion()
	view = provider.ModelMessages(view)
	src := estimateMessagesTokens(provider.ModelMessages(msgs))
	dst := estimateMessagesTokens(view)
	projVersion := a.compactionState.Projection.ProjectionVersion + 1
	state := CompactionState{
		SchemaVersion:     compactionStateSchemaCurrent,
		TranscriptVersion: version,
		Projection: ContextProjection{
			Messages:          view,
			TranscriptVersion: version,
			ProjectionVersion: projVersion,
			CoveredCount:      len(msgs),
			CoveredPrefixHash: coveredPrefixHash(msgs, len(msgs)),
			SourceTokens:      src,
			ProjectionTokens:  dst,
			CreatedAt:         time.Now().UTC(),
		},
		PromptCacheKey:   a.currentPromptCacheKey(),
		LastCacheState:   a.CacheState(),
		LastTrigger:      CompactionTriggerSnip,
		LastMode:         CompactionModeSnip,
		LastSourceTokens: src,
		LastResultTokens: dst,
		UpdatedAt:        time.Now().UTC(),
	}
	_ = st
	return a.installProjection(state)
}

// emitCompactionTelemetry records structured compaction observability without
// logging sensitive transcript content. The slog record persists to the
// session's log file (CLI binds the default logger to ~/.reasonix/logs), so a
// later cache-miss window can be attributed to a specific compaction.
func (a *Agent) emitCompactionTelemetry(t CompactionTelemetry) {
	attrs := []any{
		"trigger", t.Trigger, "mode", t.Mode, "cache", t.CacheState,
		"src", t.SourceTokens, "proj", t.ProjectionTokens,
		"in", t.InputTokens, "out", t.OutputTokens,
		"hit", t.CacheHitTokens, "miss", t.CacheMissTokens,
		"write", t.CacheWriteTokens, "reqs", t.RequestCount,
	}
	if t.ProviderRequestID != "" {
		attrs = append(attrs, "provider_request_id", t.ProviderRequestID)
	}
	if t.Error != "" {
		slog.Warn("agent: compaction", append(attrs, "err", t.Error)...)
	} else {
		slog.Info("agent: compaction", attrs...)
	}
	detail := fmt.Sprintf("trigger=%s mode=%s cache=%s src=%d proj=%d in=%d out=%d hit=%d miss=%d write=%d reqs=%d",
		t.Trigger, t.Mode, t.CacheState, t.SourceTokens, t.ProjectionTokens,
		t.InputTokens, t.OutputTokens, t.CacheHitTokens, t.CacheMissTokens, t.CacheWriteTokens, t.RequestCount)
	if t.ProviderRequestID != "" {
		detail += " provider_request_id=" + t.ProviderRequestID
	}
	if t.Error != "" {
		detail += " err_type=" + t.Error
	}
	level := event.LevelInfo
	text := "compaction telemetry"
	if t.Error != "" {
		level = event.LevelWarn
		text = "compaction failed"
	}
	a.sink.Emit(event.Event{Kind: event.Notice, Level: level, Text: text, Detail: detail})
}

// emitCompactionAborted resolves a "compacting…" placeholder when a pass fails
// after the Started event: a Done with no summary tells a frontend to drop the
// placeholder. The caller still surfaces the reason (a Notice), so this carries
// no text of its own.
func (a *Agent) emitCompactionAborted(trigger string) {
	a.sink.Emit(event.Event{Kind: event.CompactionDone, Compaction: event.Compaction{Trigger: trigger}})
}
