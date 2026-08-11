package main

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/control"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// teamBlockingRunner keeps a turn observably running until its context is
// cancelled, so tests can exercise the turn-admission barrier.
type teamBlockingRunner struct {
	started chan struct{}
}

func (r *teamBlockingRunner) Run(ctx context.Context, _ string) error {
	close(r.started)
	<-ctx.Done()
	return ctx.Err()
}

type noticeSink struct {
	mu   sync.Mutex
	text []string
}

func (s *noticeSink) Emit(e event.Event) {
	if e.Kind == event.Notice {
		s.mu.Lock()
		s.text = append(s.text, e.Text)
		s.mu.Unlock()
	}
}
func (s *noticeSink) Close() {}

func (s *noticeSink) has(sub string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.text {
		if strings.Contains(t, sub) {
			return true
		}
	}
	return false
}

// TestSubmitToTabTeamCommandBypassesTurnAdmission pins the "/team-add did
// nothing" fix: team commands run inline in the controller, so the desktop
// submit path must not gate them on the turn-admission barrier (busy turn
// used to reject with ErrTurnRunning and the assignment silently never
// happened).
func TestSubmitToTabTeamCommandBypassesTurnAdmission(t *testing.T) {
	isolateDesktopUserDirs(t)

	// The foreground turn is serviced by teamBlockingRunner (never the
	// provider), and /team-create only registers a teammate — the task tool
	// is never asked to execute here, so nil providers are safe.
	executor := agent.New(nil, tool.NewRegistry(), agent.NewSession("sys"), agent.Options{}, event.Discard)
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := agent.NewTaskTool(nil, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil)
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	sink := &noticeSink{}
	runner := &teamBlockingRunner{started: make(chan struct{})}

	ctrl := control.New(control.Options{
		Runner:       runner,
		Executor:     executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: "sys",
		SessionDir:   t.TempDir(),
	})
	defer ctrl.Close()

	app := NewApp()
	app.setTestCtrl(ctrl, "deepseek/test")

	// Kick off a foreground turn that blocks in the runner: the tab is now
	// observably running.
	go app.SubmitToTab("test", "hello")
	<-runner.started
	deadline := time.Now().Add(5 * time.Second)
	for !ctrl.RuntimeStatus().Running && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !ctrl.RuntimeStatus().Running {
		t.Fatal("foreground turn did not enter running state")
	}

	// The team command must go through despite the busy turn.
	if err := app.SubmitToTab("test", "/team-create alpha"); err != nil {
		t.Fatalf("SubmitToTab /team-create during running turn = %v, want nil (ErrTurnRunning regression)", err)
	}
	if !sink.has("created") || !sink.has("alpha") {
		t.Fatalf("no teammate-created notice emitted; sink=%v", sink.text)
	}
}
