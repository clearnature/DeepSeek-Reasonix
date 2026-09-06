package builtin

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// P3 steer channel, T3: send_message tool tests — queues a steer for a running
// background task job, visible only with a job manager on ctx (parent agent),
// hidden from sub-agent contexts (jobs.WithoutManager).

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

// TestSendMessageQueuesForRunningJob verifies the happy path: a running task
// job accepts a message, and the job drains it via DrainPendingMessages with
// the same jobCtxKey the background agent's run loop uses.
func TestSendMessageQueuesForRunningJob(t *testing.T) {
	m := jobs.NewManager(event.Discard)
	defer m.Close()
	ctx := jobs.WithManager(context.Background(), m)
	ctx = jobs.WithSession(ctx, "session-a")

	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	got := make(chan string, 1)
	j := m.StartForSession("session-a", "task", "steer", func(jobCtx context.Context, _ io.Writer) (string, error) {
		<-release
		if text, ok := jobs.DrainPendingMessages(jobCtx); ok {
			got <- text
		}
		return "done", nil
	})

	out, err := (sendMessage{}).Execute(ctx, argsJSON(t, map[string]any{"job_id": j.ID, "text": "use plan B"}))
	if err != nil {
		t.Fatalf("send_message: %v", err)
	}
	if !strings.Contains(out, "queued") {
		t.Fatalf("send_message output = %q, want a queued confirmation", out)
	}

	close(release)
	select {
	case text := <-got:
		if text != "use plan B" {
			t.Fatalf("drained text = %q, want %q", text, "use plan B")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("job did not drain the queued message")
	}
}

// TestSendMessageRejectsUnknownAndTerminal pins the failure paths: an unknown
// job and a finished job are rejected with descriptive errors, never accepted.
func TestSendMessageRejectsUnknownAndTerminal(t *testing.T) {
	m := jobs.NewManager(event.Discard)
	defer m.Close()
	ctx := jobs.WithManager(context.Background(), m)
	ctx = jobs.WithSession(ctx, "session-a")

	if _, err := (sendMessage{}).Execute(ctx, argsJSON(t, map[string]any{"job_id": "task-99", "text": "hi"})); err == nil {
		t.Fatal("unknown job: send_message must error")
	} else if !strings.Contains(err.Error(), "unknown job") {
		t.Fatalf("unknown job error = %v, want descriptive message", err)
	}

	j := m.StartForSession("session-a", "task", "done-fast", func(context.Context, io.Writer) (string, error) {
		return "done", nil
	})
	if res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job result = %+v, want done", res)
	}
	if _, err := (sendMessage{}).Execute(ctx, argsJSON(t, map[string]any{"job_id": j.ID, "text": "too late"})); err == nil {
		t.Fatal("terminal job: send_message must error")
	} else if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("terminal job error = %v, want not-running rejection", err)
	}
}

// TestSendMessageValidatesArgs covers required-argument and no-manager errors.
func TestSendMessageValidatesArgs(t *testing.T) {
	m := jobs.NewManager(event.Discard)
	defer m.Close()
	ctx := jobs.WithManager(context.Background(), m)

	if _, err := (sendMessage{}).Execute(ctx, argsJSON(t, map[string]any{"text": "no id"})); err == nil {
		t.Fatal("missing job_id must error")
	}
	if _, err := (sendMessage{}).Execute(ctx, argsJSON(t, map[string]any{"job_id": "task-1"})); err == nil {
		t.Fatal("missing text must error")
	}
	if _, err := (sendMessage{}).Execute(context.Background(), argsJSON(t, map[string]any{"job_id": "task-1", "text": "hi"})); err == nil {
		t.Fatal("send_message without a manager must error")
	}
}

// TestSendMessageVisibility pins the bgjobs.go visibility contract: the tool
// is visible only on a context with a job manager (parent agent) and hidden
// without one (sub-agent / planner contexts clear the manager).
func TestSendMessageVisibility(t *testing.T) {
	sm := sendMessage{}
	if sm.ProviderVisible(context.Background()) {
		t.Fatal("send_message must be hidden without a job manager (sub-agent context)")
	}
	if !sm.ProviderVisible(jobs.WithManager(context.Background(), &jobs.Manager{})) {
		t.Fatal("send_message must be visible to the parent agent's job-manager context")
	}
	if sm.ReadOnly() {
		t.Fatal("send_message queues a message and must not be classified read-only")
	}
}
