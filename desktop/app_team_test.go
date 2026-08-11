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

// TestSubmitToTabSlashCommandsBypassTurnAdmission pins the "/team-add did
// nothing" / "Team Planner 无法使用" fixes: slash commands are controller
// management verbs, so the desktop submit path must not gate them on the
// turn-admission barrier (a busy turn used to drop them with ErrTurnRunning).
func TestSubmitToTabSlashCommandsBypassTurnAdmission(t *testing.T) {
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

	// Every user-typed slash command must go through despite the busy turn.
	// wantNotice asserts the command visibly took effect (not silently
	// dropped); empty means "accepted without ErrTurnRunning" is enough (the
	// command's own async path reports its outcome).
	tests := []struct {
		name       string
		input      string
		wantNotice string
	}{
		{"team-create", "/team-create alpha", "created"},
		{"context", "/context", ""},
		{"retrieve-info", "/retrieve_info 温州天气", ""},
		{"team-planner", "/team-planner 规划一个快速排序", ""},
		{"compact", "/compact", ""},
		{"compress-fast", "/compress-fast", ""},
	}
	for _, tt := range tests {
		if err := app.SubmitToTab("test", tt.input); err != nil {
			t.Fatalf("SubmitToTab %q during running turn = %v, want nil (ErrTurnRunning regression)", tt.input, err)
		}
		if tt.wantNotice != "" && !sink.has(tt.wantNotice) {
			t.Fatalf("command %q: no notice containing %q; sink=%v", tt.input, tt.wantNotice, sink.text)
		}
	}
	// Give the async command goroutines a beat to finish their (safe) error
	// paths before the controller closes under them.
	time.Sleep(100 * time.Millisecond)
}
