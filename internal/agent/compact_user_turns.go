package agent

import (
	"fmt"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// userTurnRetention is what one fold could and could not hold verbatim. A
// dropped turn is the loss compaction cannot undo — the workspace still holds
// the code a constraint governs, nothing holds the constraint — so it is
// counted and surfaced rather than absorbed.
type userTurnRetention struct {
	Kept          int
	Dropped       int
	DroppedTokens int
}

// keepUserTurns protects the user's own words from summarizer judgement: a
// constraint stated mid-session is unrecoverable once a digest drops it. The
// projection is the only view ever sent again, so it stays complete: the
// budget scales with the room the checkpoint ceiling leaves, shrinking only
// when the window is genuinely short on space (automatic pressure folds).
func (a *Agent) keepUserTurns(region []provider.Message, keep []bool, projBase, projCap int) userTurnRetention {
	budget := a.keptUserTurnsBudget(projBase, projCap)
	var ret userTurnRetention
	// Oldest-first: the recent tail already covers the newest turns verbatim,
	// and an old turn has survived more folds than a new one.
	for i, m := range region {
		if m.Role != provider.RoleUser || m.LocalOnly || isCompactionSummary(m) {
			continue
		}
		if keep[i] {
			// Already held by the keep policy — a [[keep]] marker, which is the
			// documented way past the size budget below.
			ret.Kept++
			continue
		}
		cost := fixedTokenEstimate(m)
		if cost > maxKeptUserTurnTokens || cost > budget {
			ret.Dropped++
			ret.DroppedTokens += cost
			continue
		}
		keep[i] = true
		budget -= cost
		ret.Kept++
	}
	return ret
}

// keptUserTurnsBudget caps what user turns may spend of the checkpoint.
// Unbounded hoisting padded candidates past the acceptance ceiling; a static
// 8192 shed 96 small turns (~128k tokens) on a 1M window (2026-08-12 manual
// /compact). The budget is dynamic: a window-proportional floor covers small
// windows, and real ceiling room keeps every small user turn verbatim.
func (a *Agent) keptUserTurnsBudget(projBase, projCap int) int {
	base := keptUserTurnsBudgetTokens
	if a.contextWindow > 0 {
		base = min(keptUserTurnsBudgetTokens, int(float64(a.contextWindow)*keptUserTurnsWindowFrac))
	}
	// Real ceiling room is the budget: keep every small user turn that fits,
	// so the sent view stays complete; near-full windows stay conservative.
	if projCap > 0 && projBase > 0 {
		if room := projCap - projBase; room > base {
			return min(room, maxKeptUserTurnsBudgetTokens)
		}
	}
	return base
}

// noticeDroppedUserTurns reports the turns the budget could not hold. Without
// it the drop is invisible: the projection reads as complete, and the escape
// hatch is only useful to someone told it exists at the moment it is needed.
func (a *Agent) noticeDroppedUserTurns(ret userTurnRetention) {
	if ret.Dropped == 0 {
		return
	}
	a.svc.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
		Text: fmt.Sprintf("%s of yours %s too large to keep whole and now survive only through the summary. Prefix a turn with [[keep]] to hold it verbatim.",
			pluralTurns(ret.Dropped), wereOrWas(ret.Dropped)),
		Detail: fmt.Sprintf("compaction dropped %d user turn(s) (~%d tokens) past the retention budget of %d",
			ret.Dropped, ret.DroppedTokens, a.keptUserTurnsBudget(0, 0))})
}

func pluralTurns(n int) string {
	if n == 1 {
		return "1 earlier message"
	}
	return fmt.Sprintf("%d earlier messages", n)
}

func wereOrWas(n int) string {
	if n == 1 {
		return "was"
	}
	return "were"
}
