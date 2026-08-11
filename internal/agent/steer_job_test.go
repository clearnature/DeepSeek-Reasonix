package agent

// P3 steer channel, T2: run-loop injection tests. They drive a real
// StartForSession job closure context (the only place jobCtxKey exists) through
// Agent.Run, asserting pending job messages inject as one-per-round steers.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"reasonix/internal/agent/testutil"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// releaseOnCleanup closes ch exactly once when the test unwinds, so a failed
// assertion cannot leave the job closure blocked on <-release and hang
// Manager.Close.
func releaseOnCleanup(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// p3NoopTool is a read-only no-op tool so the mock provider can script a
// first tool-call round and a second final round, giving the run loop two
// per-iteration boundaries to drain job messages on.
type p3NoopTool struct{}

func (p3NoopTool) Name() string        { return "p3_noop" }
func (p3NoopTool) Description() string { return "does nothing" }
func (p3NoopTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}
func (p3NoopTool) ReadOnly() bool { return true }
func (p3NoopTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "ok", nil
}

// steerTexts returns every session message recognized as a mid-turn steer,
// in session order, with the wrapper stripped.
func steerTexts(t *testing.T, sess *Session) []string {
	t.Helper()
	var out []string
	for _, m := range sess.Messages {
		if text, ok := SteerText(m.Content); ok {
			out = append(out, text)
		}
	}
	return out
}

// TestBackgroundTaskDrainsJobMessage is the core P3 path: a background task
// job's closure context carries jobCtxKey, so messages queued by the parent
// are injected into the sub-agent session tail as user steers — exactly one
// per tool round, in FIFO order — and surfaced as event.Steer. The mock
// provider's request tail proves the steer reached the model request.
func TestBackgroundTaskDrainsJobMessage(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	var steers []event.Event
	sink := event.FuncSink(func(e event.Event) {
		if e.Kind == event.Steer {
			steers = append(steers, e)
		}
	})
	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	var sess *Session
	var mp *testutil.MockProvider
	j := jm.StartForSession("session-a", "task", "steer", func(jobCtx context.Context, _ io.Writer) (string, error) {
		<-release
		mp = testutil.NewMock("m",
			testutil.Turn{ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "p3_noop", Arguments: `{}`}}},
			testutil.Turn{Text: "done"},
		)
		reg := tool.NewRegistry()
		reg.Add(p3NoopTool{})
		a := New(mp, reg, NewSession(""), Options{}, sink)
		sess = a.Session()
		if runErr := a.Run(jobCtx, "do the task"); runErr != nil {
			return "ok", runErr
		}
		// Both queued messages were consumed one per round; the queue must be
		// empty when the run finishes.
		if text, ok := jobs.DrainPendingMessages(jobCtx); ok {
			return "ok", fmt.Errorf("unexpected leftover job steer %q after the run", text)
		}
		return "ok", nil
	})
	for _, text := range []string{"use plan B", "and keep diffs small"} {
		if err := jm.SendMessageForSession("session-a", j.ID, text); err != nil {
			t.Fatalf("SendMessageForSession(%q): %v", text, err)
		}
	}
	close(release)
	res := jm.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job result = %+v, want done", res)
	}
	if sess == nil || mp == nil {
		t.Fatal("job closure did not run the agent")
	}

	// One message per tool round, FIFO, wrapper stripped by SteerText.
	got := steerTexts(t, sess)
	if len(got) != 2 || got[0] != "use plan B" || got[1] != "and keep diffs small" {
		t.Fatalf("session steers = %q, want [use plan B and keep diffs small] one per round", got)
	}
	// The first round must have injected only the first message: request 1's
	// tail is the first steer, request 2's tail is the second.
	reqs := mp.Requests()
	if len(reqs) != 2 {
		t.Fatalf("mock provider calls = %d, want 2", len(reqs))
	}
	for i, want := range []string{"use plan B", "and keep diffs small"} {
		msgs := reqs[i].Messages
		if len(msgs) == 0 {
			t.Fatalf("request %d has no messages", i)
		}
		if text, ok := SteerText(msgs[len(msgs)-1].Content); !ok || text != want {
			t.Fatalf("request %d tail = %q (steer=%v), want %q", i, msgs[len(msgs)-1].Content, ok, want)
		}
	}
	// event.Steer is emitted for each injected message, carrying the raw text.
	if len(steers) != 2 || steers[0].Text != "use plan B" || steers[1].Text != "and keep diffs small" {
		t.Fatalf("Steer events = %+v, want two with raw text", steers)
	}
}

// TestParentAgentIgnoresJobMessages pins the no-op defence: a context without
// jobCtxKey (foreground agent / front-desk subagent / planner) must never
// inject a job message — no session steer appears and no Steer event fires.
func TestParentAgentIgnoresJobMessages(t *testing.T) {
	mp := testutil.NewMock("m", testutil.Turn{Text: "done"})
	var steers []event.Event
	sink := event.FuncSink(func(e event.Event) {
		if e.Kind == event.Steer {
			steers = append(steers, e)
		}
	})
	a := New(mp, tool.NewRegistry(), NewSession(""), Options{}, sink)
	if err := a.Run(context.Background(), "hi"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, m := range a.Session().Messages {
		if _, ok := SteerText(m.Content); ok {
			t.Fatalf("parent agent injected a job steer: %q", m.Content)
		}
	}
	if len(steers) != 0 {
		t.Fatalf("parent agent emitted Steer events: %+v", steers)
	}
}

// TestDrainPendingMessagesImport sanity-checks that the P3 helper is reachable
// from the agent package with the expected contract (belt-and-braces for the
// import surface used by the run loop).
func TestDrainPendingMessagesImport(t *testing.T) {
	if _, ok := jobs.DrainPendingMessages(context.Background()); ok {
		t.Fatal("plain context must be a no-op")
	}
	if strings.TrimSpace(jobs.ErrPendingQueueFull.Error()) == "" {
		t.Fatal("ErrPendingQueueFull must carry a descriptive message")
	}
}
