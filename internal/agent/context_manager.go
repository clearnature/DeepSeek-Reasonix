package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// compactionProgress is how compaction is faring in this session: whether a
// fold stopped reducing, how many ran back to back, and which retries already
// ran in the active turn. The fields are cleared together on lineage resets.
type compactionProgress struct {
<<<<<<< HEAD
	stuck       bool // a fold landed above the trigger, so pressure retries are pointless
	consecutive int  // back-to-back folds since one last helped
=======
	stuck          bool   // a fold landed above the trigger, so the same-view pressure retry is pointless
	stuckInputHash string // provider-visible view covered by stuck; changed input may retry
	consecutive    int    // back-to-back folds since one last helped
>>>>>>> origin/main-v2
	// failedTurn backs off changed-view retries within one active tool loop.
	// A later user turn may retry, while hard-ceiling recovery bypasses it.
	failedTurn atomic.Int64
	// lastTurn stops the post-turn observer and the pre-send preflight from
	// paying for two summaries during one active tool loop.
	lastTurn atomic.Int64
}

// ContextManager is the sole owner of provider-visible context maintenance.
// Canonical session messages are immutable inputs; Prepare evolves only the
// durable projection and returns the exact visible view for one sampling round.
type ContextManager struct {
	agent *Agent
}

// ContextPreparePolicy describes one maintenance transaction.
type ContextPreparePolicy struct {
	Trigger      string
	Instructions string
	Force        bool
	// ObservedInputTokens is used by compatibility harnesses that invoke the
	// old post-turn shim directly. Production Prepare estimates the current view
	// from its calibrated final request shape.
	ObservedInputTokens int
}

// PreparedContext is the frozen result of a successful Prepare transaction.
type PreparedContext struct {
	Messages          []provider.Message
	InputTokens       int
	ProjectionVersion uint64
}

func (a *Agent) contextManager() ContextManager { return ContextManager{agent: a} }

// PrepareContext is the public automatic-maintenance entry used by smoke tools
// and controllers that need a one-shot Prepare without sampling.
func (a *Agent) PrepareContext(ctx context.Context) error {
	_, err := a.contextManager().Prepare(ctx, ContextPreparePolicy{Trigger: CompactionTriggerPressure})
	return err
}

// ObserveUsage is retained as a compatibility hook. Usage observations never
// mutate the provider-visible checkpoint.
func (m ContextManager) ObserveUsage(u *provider.Usage) {
	_ = u
}

// Prepare is the sole automatic maintenance entry. Below compact_ratio it does
// nothing. At or above the trigger it runs one single-flight prune/summary
// transaction, with at most two successful summary attempts under pressure.
func (m ContextManager) Prepare(ctx context.Context, policy ContextPreparePolicy) (PreparedContext, error) {
	if policy.Trigger == "" {
		policy.Trigger = CompactionTriggerPressure
	}
	if m.agent == nil {
		return PreparedContext{}, nil
	}
	m.agent.sess.compactionRunMu.Lock()
	defer m.agent.sess.compactionRunMu.Unlock()
	return m.prepareOnce(ctx, policy)
}

func (m ContextManager) prepareOnce(ctx context.Context, policy ContextPreparePolicy) (PreparedContext, error) {
	a := m.agent
	if a == nil || a.sess.conversation == nil {
		return PreparedContext{}, nil
	}
	visible := a.modelVisibleMessages()
	// Threshold uses the stable pre-interceptor request shape (messages + tools
	// + role projection); interceptor expansion past hard still overflows.
	est := a.estimatedVisibleRequestTokens(visible)
	prepared := PreparedContext{
		Messages:          append([]provider.Message(nil), visible...),
		InputTokens:       est,
		ProjectionVersion: a.currentProjectionVersion(),
	}
	if a.contextWindow <= 0 || len(visible) == 0 {
		return prepared, nil
	}
	fold := a.compactTrigger()
	hard := a.hardInputCeiling()
	if policy.ObservedInputTokens > 0 {
		est = policy.ObservedInputTokens
		prepared.InputTokens = est
	}
	inputHash := a.contextMaintenanceInputHash(visible)
<<<<<<< HEAD
	// Receipts back off sub-critical retries only: at a physical recovery point
	// (overflow, or a view at/above the hard ceiling) the fold must still run —
	// degradeFoldSummary guarantees mustFree progress.
=======
	// Receipts back off sub-critical retries only. Physical overflow may retry
	// maintenance once, but a failed summary never fabricates fallback content.
>>>>>>> origin/main-v2
	if blocked, _ := a.contextMaintenanceBlocked(inputHash); blocked && policy.Trigger != CompactionTriggerManual &&
		policy.Trigger != CompactionTriggerOverflow && est < hard {
		return prepared, nil
	}
	if est < fold {
		a.sess.compaction.consecutive = 0
		a.sess.compaction.stuck = false
<<<<<<< HEAD
		a.sess.compaction.failedTurn.Store(0)
	}
=======
		a.sess.compaction.stuckInputHash = ""
		a.sess.compaction.failedTurn.Store(0)
	}
	if a.sess.compaction.stuck && a.sess.compaction.stuckInputHash != inputHash {
		// The previous projection could not reclaim enough from its exact view,
		// but newly appended messages create a new fold boundary and may retry.
		a.sess.compaction.stuck = false
		a.sess.compaction.stuckInputHash = ""
		a.sess.compaction.consecutive = 0
	}
>>>>>>> origin/main-v2
	if a.sess.compaction.stuck && policy.Trigger == CompactionTriggerPressure && est < hard {
		return prepared, nil
	}
	// One user trigger. Overflow is a one-shot physical recovery path only.
	forceFold := policy.Force || policy.Trigger == CompactionTriggerManual || policy.Trigger == CompactionTriggerOverflow || est >= hard
	if est < fold && !forceFold {
		return prepared, nil
	}
	if est > a.contextWindow && a.sess.cacheState != "" && a.svc.sink != nil {
		// Resume replay past the window: the C1 gate let it through warm, but
		// it must fold before the first send. Land the decision for diagnosis.
		a.svc.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: "resume telemetry",
			Detail: fmt.Sprintf("state=%s idle_min=0 decision=compact-first est=%d", a.sess.cacheState, est)})
	}

	if policy.Trigger == CompactionTriggerPressure || policy.Trigger == CompactionTriggerOverflow {
		applied, err := a.pruneToolResultsToProjectionLocked(policy.Trigger)
		if err != nil {
			return PreparedContext{}, err
		}
		if applied {
			prepared = m.currentPrepared()
			est = prepared.InputTokens
			inputHash = a.contextMaintenanceInputHash(prepared.Messages)
			if (policy.Trigger == CompactionTriggerPressure && est < fold) ||
				(policy.Trigger == CompactionTriggerOverflow && est < hard) {
				return prepared, nil
			}
		}
	}

	return m.foldContext(ctx, prepared, policy, inputHash, est, fold, hard, forceFold)
}

func (m ContextManager) foldContext(ctx context.Context, prepared PreparedContext, policy ContextPreparePolicy, inputHash string, est, fold, hard int, forceFold bool) (PreparedContext, error) {
	a := m.agent
	maxSummaries := 1
	if policy.Trigger == CompactionTriggerPressure {
		maxSummaries = 2
	}
	result := prepared
	for range maxSummaries {
		mustFree := policy.Trigger != CompactionTriggerManual && (policy.Trigger == CompactionTriggerOverflow || result.InputTokens >= hard)
		outcome, err := a.compactToProjectionLocked(ctx, policy.Trigger, policy.Instructions, forceFold, mustFree)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return PreparedContext{}, err
			}
			if errors.Is(err, errCompressStaleContext) && policy.Trigger != CompactionTriggerManual {
				reason := "context changed during summary; automatic retry blocked for this generation"
				a.recordContextMaintenanceBlocked(inputHash, policy.Trigger, "summary", reason)
				if policy.Trigger == CompactionTriggerOverflow || result.InputTokens >= hard {
					return PreparedContext{}, fmt.Errorf("%w: %s", ErrCompactionRequired, reason)
				}
				return m.currentPrepared(), nil
			}
			status := "failed"
			if errors.Is(err, errSummaryOutputTruncated) || errors.Is(err, errCheckpointRejected) {
				status = "blocked"
			}
			reason := fmt.Sprintf("context summary failed: %v", err)
			a.recordContextMaintenanceOutcome(inputHash, policy.Trigger, "summary", status, reason)
			if policy.Trigger == CompactionTriggerManual {
				return PreparedContext{}, err
			}
			latest := m.currentPrepared()
			if policy.Trigger == CompactionTriggerOverflow || latest.InputTokens >= hard {
				return PreparedContext{}, fmt.Errorf("%w: %w", ErrCompactionRequired, err)
			}
			return latest, nil
		}
		if outcome == CompactionNoop {
			reason := "context is above the maintenance threshold but no foldable region remains"
			a.recordContextMaintenanceBlocked(inputHash, policy.Trigger, "summary", reason)
			latest := m.currentPrepared()
			if policy.Trigger == CompactionTriggerOverflow || policy.Force || latest.InputTokens >= hard {
				return PreparedContext{}, fmt.Errorf("%w: %s", ErrCompactionRequired, reason)
			}
			return latest, nil
		}

		result = m.currentPrepared()
		if policy.Trigger == CompactionTriggerManual || result.InputTokens < fold ||
			(policy.Trigger == CompactionTriggerOverflow && result.InputTokens < hard) {
			a.sess.compaction.stuck = false
			a.sess.compaction.stuckInputHash = ""
			a.sess.compaction.consecutive = 0
			a.sess.compaction.failedTurn.Store(0)
			return result, nil
		}
		forceFold = false
		inputHash = a.contextMaintenanceInputHash(result.Messages)
	}

<<<<<<< HEAD
	result := m.currentPrepared()
	if result.InputTokens < fold {
		// A fold that landed under the trigger proves compaction reduces again;
		// a stale stuck latch must not suppress the next pressure round.
		a.sess.compaction.stuck = false
		a.sess.compaction.consecutive = 0
		a.sess.compaction.failedTurn.Store(0)
	}
	if policy.Trigger == CompactionTriggerManual {
		return result, nil
	}
	if result.InputTokens >= fold {
		reason := fmt.Sprintf("summary result remains above fold trigger (%d >= %d)", result.InputTokens, fold)
		a.recordContextMaintenanceBlocked(a.contextMaintenanceInputHash(result.Messages), policy.Trigger, "summary", reason)
		a.sess.compaction.stuck = true
		a.sess.compaction.consecutive++
		if policy.Trigger == CompactionTriggerOverflow || result.InputTokens >= hard {
			return PreparedContext{}, fmt.Errorf("%w: %s", ErrCompactionRequired, reason)
		}
		slog.Info("agent: context maintenance paused below hard ceiling", "reason", reason)
=======
	reason := fmt.Sprintf("summary result remains above fold trigger after %d attempts (%d >= %d)", maxSummaries, result.InputTokens, fold)
	blockedInputHash := a.contextMaintenanceInputHash(result.Messages)
	a.recordContextMaintenanceBlocked(blockedInputHash, policy.Trigger, "summary", reason)
	a.sess.compaction.stuck = true
	a.sess.compaction.stuckInputHash = blockedInputHash
	a.sess.compaction.consecutive += maxSummaries
	if policy.Trigger == CompactionTriggerOverflow || result.InputTokens >= hard {
		return PreparedContext{}, fmt.Errorf("%w: %s", ErrCompactionRequired, reason)
>>>>>>> origin/main-v2
	}
	slog.Info("agent: context maintenance paused below hard ceiling", "reason", reason)
	return result, nil
}

func (m ContextManager) currentPrepared() PreparedContext {
	if m.agent == nil {
		return PreparedContext{}
	}
	visible := m.agent.modelVisibleMessages()
	return PreparedContext{
		Messages:          append([]provider.Message(nil), visible...),
		InputTokens:       m.agent.estimatedVisibleRequestTokens(visible),
		ProjectionVersion: m.agent.currentProjectionVersion(),
	}
}

// estimatedVisibleRequestTokens sizes the pre-interceptor sampling shape:
// ModelMessages + role projection + tool schemas. Extension interceptors are
// intentionally omitted here (see prepareOnce) to avoid double side effects.
// decisionEstimateTokens is the estimate that crosses the fold trigger in
// Prepare; recorded so a pass can be audited against the actual prompt.
func (a *Agent) decisionEstimateTokens() int {
	return a.estimatedVisibleRequestTokens(a.modelVisibleMessages())
}

func (a *Agent) estimatedVisibleRequestTokens(visible []provider.Message) int {
	if a == nil {
		return 0
	}
	msgs := a.normalizeModelRequestMessages(visible)
	var tools []provider.ToolSchema
	if a.svc.tools != nil {
		tools = a.svc.tools.Schemas()
	}
	req := provider.Request{
		Messages:    msgs,
		Tools:       tools,
		MaxTokens:   a.maxOutputTokens,
		Temperature: provider.OptionalTemperature(a.temperature),
	}
	shape := a.requestCalibrationShape(req)
	if calibrated, ok := a.calibratedPromptTokens(shape); ok {
		return calibrated
	}
	// No calibration yet (fresh fork or resume first turn): the 0.25 wire-char
	// fallback under-sizes CJK (0.25 × 3-byte runes = phantom 0.75/rune). Use
	// official ratios (CJK 0.6, ASCII 0.3); 1.0 mis-triggered 8/13 (316K est
	// ≥ 850K).
	if containsCJKText(msgs) {
		return officialMessagesTokens(msgs)
	}
	return a.estimatedRequestTokens(req)
}

// containsCJKText reports whether any visible message carries CJK runes, the
// only case where the conservative 1-rune-per-token estimate is appropriate
// before provider calibration exists.
func containsCJKText(msgs []provider.Message) bool {
	for _, m := range msgs {
		for _, r := range m.Content {
			if isCJKRune(r) {
				return true
			}
		}
		for _, tc := range m.ToolCalls {
			for _, r := range tc.Arguments {
				if isCJKRune(r) {
					return true
				}
			}
		}
	}
	return false
}
