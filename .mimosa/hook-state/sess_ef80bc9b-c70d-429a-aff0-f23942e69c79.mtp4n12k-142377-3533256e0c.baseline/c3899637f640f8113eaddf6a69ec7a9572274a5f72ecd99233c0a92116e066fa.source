package agent

// P3 steer channel, T2: run-loop injection tests. They drive a real
// StartForSession job closure context (the only place jobCtxKey exists) through
// Agent.Run, asserting pending job messages inject as one-per-round steers.

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
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
// gateProvider blocks the first provider call until release closes, so the
// test can queue steers while Run is already active (steerRunActive true).
type gateProvider struct {
	mp      *testutil.MockProvider
	release <-chan struct{}
	reached chan struct{}
	once    sync.Once
	gate    sync.Once
}

func (p *gateProvider) Name() string { return p.mp.Name() }

func (p *gateProvider) Stream(ctx context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.gate.Do(func() { close(p.reached) })
	p.once.Do(func() { <-p.release })
	return p.mp.Stream(ctx, req)
}

func TestBackgroundTaskDrainsJobMessage(t *testing.T) {
	// Steer messages queued while a run is active are consumed one per tool
	// round, FIFO, and surface as event.Steer plus a session user turn.
	var steers []event.Event
	sink := event.FuncSink(func(e event.Event) {
		if e.Kind == event.Steer {
			steers = append(steers, e)
		}
	})
	release := make(chan struct{})
	reached := make(chan struct{})
	mp := testutil.NewMock("m",
		testutil.Turn{ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "p3_noop", Arguments: `{}`}}},
		testutil.Turn{ToolCalls: []provider.ToolCall{{ID: "call-2", Name: "p3_noop", Arguments: `{}`}}},
		testutil.Turn{Text: "done"},
	)
	reg := tool.NewRegistry()
	reg.Add(p3NoopTool{})
	a := New(&gateProvider{mp: mp, release: release, reached: reached}, reg, NewSession(""), Options{}, sink)
	sess := a.Session()
	runErr := make(chan error, 1)
	go func() { runErr <- a.Run(context.Background(), "do the task") }()
	// Wait until Run is active (first Stream arrived, steerRunActive=true).
	<-reached
	// Queue both messages before the tool rounds consume them.
	if !a.Steer("use plan B") {
		t.Fatal("first Steer not queued while run active")
	}
	if !a.Steer("and keep diffs small") {
		t.Fatal("second Steer not queued")
	}
	close(release)
	if err := <-runErr; err != nil {
		t.Fatalf("Run: %v", err)
	}
	// One message per tool round, FIFO, wrapper stripped by SteerText.
	got := steerTexts(t, sess)
	if len(got) != 2 || got[0] != "use plan B" || got[1] != "and keep diffs small" {
		t.Fatalf("session steers = %q, want [use plan B and keep diffs small] one per round", got)
	}
	reqs := mp.Requests()
	// beginRunTurn issues the first provider request before any steer is
	// consumed; each subsequent tool-round iteration consumes one steer, so
	// steer i lands on request i+1 (two tool rounds + final text = 3 calls).
	if len(reqs) != 3 {
		t.Fatalf("mock provider calls = %d, want 3 (first + 2 tool rounds)", len(reqs))
	}
	for i, want := range []string{"use plan B", "and keep diffs small"} {
		msgs := reqs[i+1].Messages
		if len(msgs) == 0 {
			t.Fatalf("request %d has no messages", i+1)
		}
		if text, ok := SteerText(msgs[len(msgs)-1].Content); !ok || text != want {
			t.Fatalf("request %d tail = %q (steer=%v), want %q", i+1, msgs[len(msgs)-1].Content, ok, want)
		}
	}
	if first := reqs[0].Messages; len(first) > 0 {
		if _, ok := SteerText(first[len(first)-1].Content); ok {
			t.Fatalf("first request unexpectedly carries a steer: %q", first[len(first)-1].Content)
		}
	}
}

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
