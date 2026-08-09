package agent

import (
	"context"
	"fmt"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// minOutputBudget is the smallest output allowance a shared-window request may
// keep; below it a request is rejected locally instead of sent to 400.
const minOutputBudget = 8 * 1024

// forceThreshold is the prompt high-water mark that forces compaction. On
// shared-window providers it never exceeds window - output budget - reserve,
// so a request below it cannot be rejected for exceeding the model context
// length; independent-ceiling providers keep the plain ratio mark.
func (a *Agent) forceThreshold() int {
	force := int(float64(a.contextWindow) * a.compactForceRatio)
	if sharesContextWindow(a.prov) {
		budget := a.outputBudget
		if a.maxOutputTokens > 0 {
			budget = a.maxOutputTokens
		}
		// Only shrink the force mark when a real shared window exists (window
		// larger than the output budget + reserve); tiny windows go negative
		// and would force a fold every turn.
		if budget > 0 && a.contextWindow > budget+minOutputBudget {
			budgetAware := a.contextWindow - budget - minOutputBudget
			if budgetAware < force {
				force = budgetAware
			}
		}
	}
	return force
}

// MaybeCompactOnResume folds a resumed session only when the model-visible
// prompt already leaves no room for output. Everything else keeps the warm
// cached prefix (deferred-compaction policy); the canonical transcript is
// never rewritten.
func (a *Agent) MaybeCompactOnResume(ctx context.Context) {
	if a == nil || a.session == nil || a.contextWindow <= 0 {
		return
	}
	if !sharesContextWindow(a.prov) {
		return
	}
	// Model-visible shape, not the canonical transcript: a large history with
	// a small valid projection sends exactly projection + tail, so the gate
	// must not fold a cached prefix on canonical bulk it never transmits.
	msgs := a.modelVisibleMessages()
	// Real-shape estimate without the overflow-guard safety factor: the resume
	// gate must not fold a warm cached prefix on an estimate miss; under-
	// estimating only defers compaction to the request path.
	est := estimateMessagesTokens(provider.ModelMessages(msgs))
	// The prompt alone already leaves no room for output: any request would be
	// rejected regardless of cache state. Compact unconditionally.
	if est >= a.contextWindow-minOutputBudget-outputBudgetReserve {
		if err := a.CompactNow(ctx, ""); err == nil {
			a.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
				"resumed session prompt ~%d tokens est. exceeds the shared context window's input allowance — compacted before first send", est)})
		}
	}
}

// maybePredictOverflow emits a record-only notice when the estimated prompt
// leaves less than minOutputBudget of headroom for the output: the next turn
// may overflow the shared context window. Compaction is not triggered — this
// is a diagnostic signal, not a pressure gate.
func (a *Agent) maybePredictOverflow(est, maxTokens int) {
	if a == nil || a.sink == nil || a.contextWindow <= 0 || maxTokens <= 0 {
		return
	}
	headroom := a.contextWindow - est - maxTokens
	if headroom < minOutputBudget {
		a.sink.Emit(event.Event{
			Kind:  event.Notice,
			Level: event.LevelInfo,
			Text:  "context window nearly full",
			Detail: fmt.Sprintf("estimated prompt %d tokens, max output %d, headroom %d — next turn may overflow",
				est, maxTokens, max(headroom, 0)),
		})
	}
}
