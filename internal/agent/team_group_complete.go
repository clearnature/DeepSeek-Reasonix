package agent

import (
	"fmt"
	"log/slog"

	"reasonix/internal/event"
)

// checkGroupCompleted runs after a job completion flips a member to idle and
// emits one group-done notice when the whole team has finished: every member
// idle and no pending/blocked task remains. This makes batch completion a
// deterministic system event for the leader (any model, regardless of
// capability, is told the whole batch finished and can deliver results) —
// it does not rely on the leader polling status or remembering to check.
// The transition is naturally idempotent: it fires only on the edge into
// the all-complete state.
func (ts *TeammateStore) checkGroupCompleted(flipped string) {
	if flipped == "" {
		return
	}
	ts.mu.Lock()
	total := 0
	pending := 0
	busy := 0
	for _, tm := range ts.teammates {
		if tm.State == TeammateRunning {
			busy++
		}
	}
	for _, t := range ts.tasks {
		if t == nil {
			continue
		}
		total++
		if !isTerminalStatus(t.Status) {
			pending++
		}
	}
	name := ts.name
	ts.mu.Unlock()
	if total == 0 || busy != 0 || pending != 0 {
		return
	}
	ts.mu.Lock()
	sink := ts.sink
	ts.mu.Unlock()
	if sink == nil {
		return
	}
	group := name
	if group == "" {
		group = "team"
	}
	sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
		Text: fmt.Sprintf("team group %q: all %d dispatched task(s) completed — results delivered, ready to summarize", group, total)})
	slog.Info("team group completed", "group", name, "tasks", total)
}
