package control

// P3 steer channel, T5: end-to-end + cache regression. /task-message (T4)
// forwards to the jobs queue (T1); the background agent's next tool round
// receives it as a steer; the parent compose stays untouched.

import (
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/agent/testutil"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
	_ "reasonix/internal/tool/builtin" // register the real built-ins (incl. send_message)
)

// e2eNoopTool is a read-only no-op so the background task's mock provider can
// script one tool round followed by a final round, giving the run loop two
// per-iteration boundaries to drain job messages on.
type e2eNoopTool struct{}

func (e2eNoopTool) Name() string        { return "e2e_noop" }
func (e2eNoopTool) Description() string { return "does nothing" }
func (e2eNoopTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}
func (e2eNoopTool) ReadOnly() bool { return true }
func (e2eNoopTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "ok", nil
}

// TestTaskMessageEndToEndDeliversToBackgroundJob drives the complete P3 path
// through the Controller: /task-message queues the steer, and the running
// background task job — running a real agent with a real run loop — receives
// it as a user message on its next tool round (request tail).
func TestTaskMessageEndToEndDeliversToBackgroundJob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	jm := jobs.NewManager(event.Discard)
	var notices []string
	c := New(Options{
		Executor:    agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard),
		SessionDir:  dir,
		SessionPath: path,
		Label:       "test",
		Jobs:        jm,
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.Notice {
				notices = append(notices, e.Text)
			}
		}),
	})
	defer c.Close()

	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	var mp *testutil.MockProvider
	var runErr error
	j := jm.StartForSession(agent.BranchID(path), "task", "e2e", func(jobCtx context.Context, _ io.Writer) (string, error) {
		<-release
		mp = testutil.NewMock("m",
			testutil.Turn{ToolCalls: []provider.ToolCall{{ID: "call-1", Name: "e2e_noop", Arguments: `{}`}}},
			testutil.Turn{Text: "done"},
		)
		reg := tool.NewRegistry()
		reg.Add(e2eNoopTool{})
		a := agent.New(mp, reg, agent.NewSession(""), agent.Options{}, event.Discard)
		runErr = a.Run(jobCtx, "do the task")
		return "ok", runErr
	})

	c.Submit("/task-message " + j.ID + " switch to plan B")
	if !strings.Contains(strings.Join(notices, "\n"), "message queued") {
		t.Fatalf("notices = %v, want a queued confirmation", notices)
	}
	close(release)
	if res := jm.WaitForSession(context.Background(), agent.BranchID(path), []string{j.ID}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job result = %+v, want done", res)
	}
	if runErr != nil {
		t.Fatalf("background agent run: %v", runErr)
	}
	if mp == nil {
		t.Fatal("job closure did not run the agent")
	}

	// The background agent's first model request tail is the steered message.
	reqs := mp.Requests()
	if len(reqs) != 2 {
		t.Fatalf("mock provider calls = %d, want 2", len(reqs))
	}
	first := reqs[0].Messages
	if len(first) == 0 {
		t.Fatal("request 1 has no messages")
	}
	if text, ok := agent.SteerText(first[len(first)-1].Content); !ok || text != "switch to plan B" {
		t.Fatalf("request 1 tail = %q (steer=%v), want the queued message", first[len(first)-1].Content, ok)
	}
}

// TestTaskMessageParentComposeUnchanged pins the cache regression: /task-message
// is a management command, so the parent session stays byte-identical (no
// steer, no new user message) and the job queue keeps its message for the job
// — the parent neither injects nor consumes.
func TestTaskMessageParentComposeUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	jm := jobs.NewManager(event.Discard)
	parentSess := agent.NewSession("")
	var notices []string
	c := New(Options{
		Executor:    agent.New(nil, nil, parentSess, agent.Options{}, event.Discard),
		SessionDir:  dir,
		SessionPath: path,
		Label:       "test",
		Jobs:        jm,
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.Notice {
				notices = append(notices, e.Text)
			}
		}),
	})
	defer c.Close()

	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	drained := make(chan string, 1)
	j := jm.StartForSession(agent.BranchID(path), "task", "steer", func(jobCtx context.Context, _ io.Writer) (string, error) {
		<-release
		if text, ok := jobs.DrainPendingMessages(jobCtx); ok {
			drained <- text
		}
		return "done", nil
	})

	before := len(parentSess.Messages)
	c.Submit("/task-message " + j.ID + " stay queued")
	if !strings.Contains(strings.Join(notices, "\n"), "message queued") {
		t.Fatalf("notices = %v, want a queued confirmation", notices)
	}
	// Parent compose: no new user message, no steer, unchanged length.
	if got := len(parentSess.Messages); got != before {
		t.Fatalf("parent session length %d -> %d after /task-message, want unchanged", before, got)
	}
	for _, m := range parentSess.Messages {
		if _, ok := agent.SteerText(m.Content); ok {
			t.Fatalf("parent session contains a steer: %q", m.Content)
		}
	}

	// The message stayed queued for the job: the parent never consumed it.
	close(release)
	select {
	case text := <-drained:
		if text != "stay queued" {
			t.Fatalf("job drained %q, want %q", text, "stay queued")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("job did not drain the message the parent left queued")
	}
}
