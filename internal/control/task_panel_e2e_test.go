package control

// P2 task panel, T6: e2e composition. T1 jobs snapshot API + T2
// Controller.JobSnapshots/CancelJob cover the panel chain: running job with
// non-consuming tail, session-scoped cancel to killed, stalled short window.

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

// waitForTaskPanelCond polls cond until it holds or a deadline passes, keeping
// the e2e steps tolerant of goroutine scheduling.
func waitForTaskPanelCond(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition never became true within 5s")
}

// TestTaskPanelEndToEndRunningTailThenCancel walks the complete P2 panel path:
// a real background task job starts and streams a probe line; the panel snapshot
// (Controller.JobSnapshots) shows it running with the tail visible; CancelJob
// flips the snapshot to killed. Polling the snapshot is itself the non-consumption
// probe — the tail must stay visible across polls.
func TestTaskPanelEndToEndRunningTailThenCancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	jm := jobs.NewManager(event.Discard)
	t.Cleanup(jm.Close)
	c := New(Options{
		Executor:    agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard),
		SessionDir:  dir,
		SessionPath: path,
		Label:       "panel-e2e",
		Jobs:        jm,
	})
	t.Cleanup(c.Close)

	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	j := jm.StartForSession(agent.BranchID(path), "task", "panel-e2e",
		func(jobCtx context.Context, w io.Writer) (string, error) {
			if _, err := w.Write([]byte("panel-tail-probe-42\n")); err != nil {
				t.Errorf("job write: %v", err)
			}
			<-release
			return "", jobCtx.Err()
		})

	// Snapshot surfaces the running job with its tail; repeated polling must not
	// consume it (P2 red line, locked by TestSnapshotsDoNotConsumeOutputForSession).
	waitForTaskPanelCond(t, func() bool {
		for _, s := range c.JobSnapshots() {
			if s.ID == j.ID && s.Status == string(jobs.Running) &&
				strings.Contains(s.Tail, "panel-tail-probe-42") {
				return true
			}
		}
		return false
	})

	// CancelJob (session-scoped kill) flips the snapshot to killed.
	if !c.CancelJob(j.ID) {
		t.Fatalf("CancelJob(%s) = false, want true for a running job owned by this session", j.ID)
	}
	waitForTaskPanelCond(t, func() bool {
		for _, s := range c.JobSnapshots() {
			if s.ID == j.ID && s.Status == string(jobs.Killed) {
				return true
			}
		}
		return false
	})
}

// TestTaskPanelEndToEndStalledWarning drives stalled through the real
// monitorStalled loop with a short WithStalledWarningAfter window: a task job
// that stays silent past the window is flagged, and the panel snapshot reports
// Stalled=true while the job is still running (terminal beats stalled, so the
// status must stay running for the stalled decoration to apply).
func TestTaskPanelEndToEndStalledWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	jm := jobs.NewManager(event.Discard, jobs.WithStalledWarningAfter(60*time.Millisecond))
	t.Cleanup(jm.Close)
	c := New(Options{
		Executor:    agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard),
		SessionDir:  dir,
		SessionPath: path,
		Label:       "panel-stalled",
		Jobs:        jm,
	})
	t.Cleanup(c.Close)

	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	j := jm.StartForSession(agent.BranchID(path), "task", "panel-stalled",
		func(jobCtx context.Context, _ io.Writer) (string, error) {
			<-release
			return "", jobCtx.Err()
		})

	// The job emits nothing, so after the 60ms window monitorStalled flags it;
	// the snapshot must report Stalled=true while the status stays running.
	waitForTaskPanelCond(t, func() bool {
		for _, s := range c.JobSnapshots() {
			if s.ID == j.ID && s.Stalled && s.Status == string(jobs.Running) {
				return true
			}
		}
		return false
	})
}
