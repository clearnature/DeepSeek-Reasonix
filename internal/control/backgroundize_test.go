package control

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// P4 foreground→background, T3: Controller.Backgroundize, /background slash,
// and ForegroundTask state. The signal is stamped into the turn context by
// spawnGuardedTurn/RunTurn; Backgroundize requests it, the run loop consumes.
// Tests cover the controller surface; the run-loop handoff lives in agent (T2).

// backgroundizeSignalRunner captures the backgroundize signal from the turn
// context and can block the turn so a test can drive /background mid-turn.
// closing got publishes the captured signal (happens-before via close).
type backgroundizeSignalRunner struct {
	session     *agent.Session
	got         chan struct{}
	release     chan struct{} // nil = never block
	releaseOnce sync.Once

	mu     sync.Mutex
	signal *agent.BackgroundizeSignal
}

func (r *backgroundizeSignalRunner) Run(ctx context.Context, input string) error {
	if r.session != nil {
		r.session.Add(provider.Message{Role: provider.RoleUser, Content: input})
	}
	r.mu.Lock()
	if r.signal == nil {
		r.signal = agent.BackgroundizeSignalFromContext(ctx)
	}
	first := r.signal != nil
	r.mu.Unlock()
	if first && r.got != nil {
		select {
		case <-r.got:
		default:
			close(r.got)
		}
	}
	if r.release != nil {
		<-r.release
	}
	return nil
}

func (r *backgroundizeSignalRunner) captured() *agent.BackgroundizeSignal {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.signal
}

func (r *backgroundizeSignalRunner) releaseNow() {
	if r.release != nil {
		r.releaseOnce.Do(func() { close(r.release) })
	}
}

func newBackgroundizeSignalRunner(block bool) *backgroundizeSignalRunner {
	r := &backgroundizeSignalRunner{
		session: agent.NewSession("sys"),
		got:     make(chan struct{}),
	}
	if block {
		r.release = make(chan struct{})
	}
	return r
}

// TestBackgroundizeFailsClosedWithoutForegroundTurn pins the fail-closed
// posture: with no foreground turn there is nothing to hand off, so
// Backgroundize errors (mirroring P3 SendTaskMessage) and the state stays
// idle.
func TestBackgroundizeFailsClosedWithoutForegroundTurn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	sess := agent.NewSession("sys")
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	c := New(Options{Runner: appendingRunner{session: sess}, Executor: exec, SessionDir: dir, SessionPath: path, Label: "test"})
	t.Cleanup(func() { c.Close() })

	if err := c.Backgroundize(); !errors.Is(err, errNoForegroundTaskToBackgroundize) {
		t.Fatalf("Backgroundize() = %v, want %v", err, errNoForegroundTaskToBackgroundize)
	}
	if st := c.ForegroundTaskState(); st != ForegroundTaskIdle {
		t.Fatalf("ForegroundTaskState() = %q, want %q", st, ForegroundTaskIdle)
	}
	if st := c.RuntimeStatus().ForegroundTask; st != ForegroundTaskIdle {
		t.Fatalf("RuntimeStatus().ForegroundTask = %q, want %q", st, ForegroundTaskIdle)
	}
}

// TestBackgroundizeRequestsSignalOnForegroundTurn drives the full async path:
// the turn context carries the signal, Backgroundize() requests it, and the
// state transitions running → backgroundize_requested.
func TestBackgroundizeRequestsSignalOnForegroundTurn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	sess := agent.NewSession("sys")
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	runner := newBackgroundizeSignalRunner(true)
	c := New(Options{Runner: runner, Executor: exec, SessionDir: dir, SessionPath: path, Label: "test"})
	t.Cleanup(func() { c.Close() })
	t.Cleanup(runner.releaseNow)

	c.Send("run a foreground task")
	select {
	case <-runner.got:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not start with a backgroundize signal in context")
	}
	if st := c.ForegroundTaskState(); st != ForegroundTaskRunning {
		t.Fatalf("ForegroundTaskState() = %q, want %q", st, ForegroundTaskRunning)
	}

	if err := c.Backgroundize(); err != nil {
		t.Fatalf("Backgroundize() = %v, want nil", err)
	}
	sig := runner.captured()
	if sig == nil || !sig.Requested() {
		t.Fatalf("foreground signal requested = %v, want true", sig)
	}
	if st := c.ForegroundTaskState(); st != ForegroundTaskBackgroundizeRequested {
		t.Fatalf("ForegroundTaskState() = %q, want %q", st, ForegroundTaskBackgroundizeRequested)
	}
	if st := c.RuntimeStatus().ForegroundTask; st != ForegroundTaskBackgroundizeRequested {
		t.Fatalf("RuntimeStatus().ForegroundTask = %q, want %q", st, ForegroundTaskBackgroundizeRequested)
	}
}

// TestBackgroundizeIdempotent pins the one-shot signal semantics: repeated
// requests all succeed at the controller surface and collapse to a single
// handoff (Requested stays true, never regresses).
func TestBackgroundizeIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	sess := agent.NewSession("sys")
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	runner := newBackgroundizeSignalRunner(true)
	c := New(Options{Runner: runner, Executor: exec, SessionDir: dir, SessionPath: path, Label: "test"})
	t.Cleanup(func() { c.Close() })
	t.Cleanup(runner.releaseNow)

	c.Send("run")
	select {
	case <-runner.got:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not start")
	}
	for i := range 3 {
		if err := c.Backgroundize(); err != nil {
			t.Fatalf("Backgroundize() #%d = %v, want nil", i+1, err)
		}
	}
	if sig := runner.captured(); sig == nil || !sig.Requested() {
		t.Fatalf("signal after repeated requests = %v, want requested", sig)
	}
	runner.releaseNow()
	waitIdle(t, c)
}

// TestBackgroundizeSignalReachesRunTurnContext covers the synchronous
// RunTurn path: a blocking transport (e.g. ACP) gets the same signal stamp and
// can request the handoff mid-turn.
func TestBackgroundizeSignalReachesRunTurnContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	sess := agent.NewSession("sys")
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	runner := newBackgroundizeSignalRunner(true)
	c := New(Options{Runner: runner, Executor: exec, SessionDir: dir, SessionPath: path, Label: "test"})
	t.Cleanup(func() { c.Close() })
	t.Cleanup(runner.releaseNow)

	done := make(chan error, 1)
	go func() { done <- c.RunTurn(context.Background(), "sync foreground task") }()
	select {
	case <-runner.got:
	case <-time.After(5 * time.Second):
		t.Fatal("synchronous turn did not start")
	}
	if err := c.Backgroundize(); err != nil {
		t.Fatalf("Backgroundize() = %v, want nil", err)
	}
	if sig := runner.captured(); sig == nil || !sig.Requested() {
		t.Fatalf("synchronous turn signal requested = %v, want true", sig)
	}
	runner.releaseNow()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunTurn returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunTurn did not finish after release")
	}
}

// TestForegroundTaskStateLifecycle walks the full state machine through a
// real turn: idle → running → backgroundize_requested → idle after the turn
// completes.
func TestForegroundTaskStateLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	sess := agent.NewSession("sys")
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	runner := newBackgroundizeSignalRunner(true)
	c := New(Options{Runner: runner, Executor: exec, SessionDir: dir, SessionPath: path, Label: "test"})
	t.Cleanup(func() { c.Close() })
	t.Cleanup(runner.releaseNow)

	if st := c.ForegroundTaskState(); st != ForegroundTaskIdle {
		t.Fatalf("before turn: ForegroundTaskState() = %q, want idle", st)
	}

	c.Send("run")
	select {
	case <-runner.got:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not start")
	}
	if st := c.ForegroundTaskState(); st != ForegroundTaskRunning {
		t.Fatalf("during turn: ForegroundTaskState() = %q, want running", st)
	}

	if err := c.Backgroundize(); err != nil {
		t.Fatalf("Backgroundize() = %v, want nil", err)
	}
	if st := c.ForegroundTaskState(); st != ForegroundTaskBackgroundizeRequested {
		t.Fatalf("after request: ForegroundTaskState() = %q, want %q", st, ForegroundTaskBackgroundizeRequested)
	}

	runner.releaseNow()
	waitIdle(t, c)
	if st := c.ForegroundTaskState(); st != ForegroundTaskIdle {
		t.Fatalf("after turn: ForegroundTaskState() = %q, want idle", st)
	}
}

// TestBackgroundSlashFailsClosedWithoutForegroundTask pins the /background
// slash surface: with no foreground turn it emits a descriptive Notice and
// does nothing else.
func TestBackgroundSlashFailsClosedWithoutForegroundTask(t *testing.T) {
	var notices []string
	c := New(Options{
		Executor: agent.New(nil, nil, agent.NewSession(""), agent.Options{}, event.Discard),
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.Notice {
				notices = append(notices, e.Text)
			}
		}),
	})
	t.Cleanup(func() { c.Close() })

	c.Submit("/background")
	if !strings.Contains(strings.Join(notices, "\n"), "no foreground task is running") {
		t.Fatalf("notices = %v, want fail-closed error notice", notices)
	}
	if st := c.ForegroundTaskState(); st != ForegroundTaskIdle {
		t.Fatalf("ForegroundTaskState() = %q, want idle", st)
	}
}

// TestBackgroundSlashRequestsMidTurn drives the slash command while a
// foreground turn is busy: the request must go through (the inline command
// path does not park), land on the turn's signal, and confirm via Notice.
func TestBackgroundSlashRequestsMidTurn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	sess := agent.NewSession("sys")
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	runner := newBackgroundizeSignalRunner(true)
	var notices []string
	c := New(Options{
		Runner:      runner,
		Executor:    exec,
		SessionDir:  dir,
		SessionPath: path,
		Label:       "test",
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.Notice {
				notices = append(notices, e.Text)
			}
		}),
	})
	t.Cleanup(func() { c.Close() })
	t.Cleanup(runner.releaseNow)

	c.Send("run")
	select {
	case <-runner.got:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not start")
	}

	c.Submit("/background")
	if !strings.Contains(strings.Join(notices, "\n"), "backgroundize requested") {
		t.Fatalf("notices = %v, want backgroundize confirmation", notices)
	}
	if sig := runner.captured(); sig == nil || !sig.Requested() {
		t.Fatalf("signal after /background = %v, want requested", sig)
	}
	runner.releaseNow()
	waitIdle(t, c)
}
