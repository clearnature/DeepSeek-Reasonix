package control

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// P3 steer channel, T4: /task-message slash command tests. Parses <job_id>
// <text>, forwards to Controller.SendTaskMessage (jobs queue), reports via
// Notice — the /tree /branch /rewind management-verb pattern.

// releaseOnCleanup closes ch exactly once when the test unwinds, so a failed
// assertion cannot leave the job closure blocked on <-release and hang
// Controller.Close's job teardown.
func releaseOnCleanup(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// newTaskMessageController builds a Controller with a session-scoped jobs
// manager and a Notice-capturing sink.
func newTaskMessageController(t *testing.T, dir, path string) (*Controller, *jobs.Manager, *[]string) {
	t.Helper()
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
	t.Cleanup(func() { c.Close() })
	return c, jm, &notices
}

// TestTaskMessageSlashQueuesForRunningJob drives the full command path: the
// user submits "/task-message <id> <text>", the controller forwards to jobs,
// and the running task job drains the exact text (internal spaces preserved).
func TestTaskMessageSlashQueuesForRunningJob(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	c, jm, notices := newTaskMessageController(t, dir, path)

	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	got := make(chan string, 1)
	j := jm.StartForSession(agent.BranchID(path), "task", "steer", func(jobCtx context.Context, _ io.Writer) (string, error) {
		<-release
		if text, ok := jobs.DrainPendingMessages(jobCtx); ok {
			got <- text
		}
		return "done", nil
	})

	c.Submit("/task-message " + j.ID + " use plan B and keep diffs small")
	joined := strings.Join(*notices, "\n")
	if !strings.Contains(joined, "message queued for background job "+j.ID) {
		t.Fatalf("notices = %v, want a queued confirmation for %s", *notices, j.ID)
	}

	close(release)
	select {
	case text := <-got:
		if text != "use plan B and keep diffs small" {
			t.Fatalf("drained text = %q, want the full text with spaces preserved", text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("job did not drain the queued message")
	}
}

// TestTaskMessageSlashUsageError pins argument validation: missing text or a
// bare command must surface the usage notice, not crash or queue anything.
func TestTaskMessageSlashUsageError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	c, _, notices := newTaskMessageController(t, dir, path)

	c.Submit("/task-message")
	if !strings.Contains(strings.Join(*notices, "\n"), "usage: /task-message <job_id> <text>") {
		t.Fatalf("bare command notices = %v, want usage", *notices)
	}
	c.Submit("/task-message task-1")
	if !strings.Contains(strings.Join(*notices, "\n"), "usage: /task-message <job_id> <text>") {
		t.Fatalf("missing-text notices = %v, want usage", *notices)
	}
}

// TestTaskMessageSlashRejectsUnknownAndTerminal pins the failure surface: an
// unknown job and a finished job are reported as descriptive Notices and
// nothing is queued.
func TestTaskMessageSlashRejectsUnknownAndTerminal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	c, jm, notices := newTaskMessageController(t, dir, path)

	c.Submit("/task-message task-99 hello")
	if !strings.Contains(strings.Join(*notices, "\n"), "unknown job") {
		t.Fatalf("unknown-job notices = %v, want unknown-job error", *notices)
	}

	j := jm.StartForSession(agent.BranchID(path), "task", "done-fast", func(context.Context, io.Writer) (string, error) {
		return "done", nil
	})
	if res := jm.WaitForSession(context.Background(), agent.BranchID(path), []string{j.ID}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job result = %+v, want done", res)
	}
	c.Submit("/task-message " + j.ID + " too late")
	if !strings.Contains(strings.Join(*notices, "\n"), "not running") {
		t.Fatalf("terminal-job notices = %v, want not-running error", *notices)
	}
}

// TestTaskMessageSlashWithoutJobs fails closed when background jobs are
// disabled (nil Jobs option).
func TestTaskMessageSlashWithoutJobs(t *testing.T) {
	var notices []string
	c := New(Options{
		Executor: agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard),
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.Notice {
				notices = append(notices, e.Text)
			}
		}),
	})
	defer c.Close()

	c.Submit("/task-message task-1 hello")
	if !strings.Contains(strings.Join(notices, "\n"), "background jobs are disabled") {
		t.Fatalf("notices = %v, want disabled-jobs error", notices)
	}
}

// TestSplitTaskMessage unit-tests the argument splitter: the first field is
// the job id, the rest is the text with internal spacing preserved.
func TestSplitTaskMessage(t *testing.T) {
	tests := []struct {
		args  string
		jobID string
		text  string
		ok    bool
	}{
		{"task-1 use plan B", "task-1", "use plan B", true},
		{"  task-2  hello world  ", "task-2", "hello world", true},
		{"task-3", "", "", false},
		{"", "", "", false},
		{"   ", "", "", false},
	}
	for _, tt := range tests {
		jobID, text, err := splitTaskMessage(tt.args)
		if tt.ok {
			if err != nil || jobID != tt.jobID || text != tt.text {
				t.Fatalf("splitTaskMessage(%q) = (%q, %q, %v), want (%q, %q, nil)", tt.args, jobID, text, err, tt.jobID, tt.text)
			}
		} else if err == nil {
			t.Fatalf("splitTaskMessage(%q) = (%q, %q, nil), want an error", tt.args, jobID, text)
		}
	}
}
