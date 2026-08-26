package control

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"reasonix/internal/event"
)

type turnEventGateRunner struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (r *turnEventGateRunner) Run(context.Context, string) error {
	r.calls.Add(1)
	close(r.started)
	<-r.release
	return nil
}

func TestTurnAdmissionIsDurableBeforeRunnerStarts(t *testing.T) {
	dir := t.TempDir()
	runner := &turnEventGateRunner{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan event.Event, 1)
	c := New(Options{
		Runner: runner,
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.TurnDone {
				done <- e
			}
		}),
		SessionDir: dir, SessionPath: filepath.Join(dir, "session.jsonl"),
	})
	t.Cleanup(c.Close)

	c.Submit("run")
	select {
	case <-runner.started:
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not start")
	}
	records, err := c.TurnEventsAfter(0)
	if err != nil {
		t.Fatalf("TurnEventsAfter: %v", err)
	}
	if len(records) < 2 || records[0].Status != event.TurnQueued || records[1].Kind != "turn_started" || records[1].Status != event.TurnInProgress {
		t.Fatalf("admission prefix = %+v, want queued then durable in_progress start", records)
	}
	close(runner.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not finish")
	}
	records, err = c.TurnEventsAfter(0)
	if err != nil {
		t.Fatalf("TurnEventsAfter terminal: %v", err)
	}
	started := 0
	for _, record := range records {
		if record.Kind == "turn_started" {
			started++
		}
	}
	if started != 1 {
		t.Fatalf("turn_started records = %d, want exactly one", started)
	}
}

func TestTurnAdmissionLedgerFailureProceedsWithoutDurability(t *testing.T) {
	dir := t.TempDir()
	blockedParent := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blockedParent, []byte("block"), 0o600); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	runner := &turnEventGateRunner{started: make(chan struct{}), release: make(chan struct{})}
	done := make(chan event.Event, 1)
	c := New(Options{
		Runner: runner,
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.TurnDone {
				done <- e
			}
		}),
		SessionDir: dir, SessionPath: filepath.Join(blockedParent, "session.jsonl"),
	})
	t.Cleanup(c.Close)

	c.Submit("should proceed without durability")
	// With non-fatal ledger errors, the turn should proceed.
	// Wait for the runner to start, then release it.
	select {
	case <-runner.started:
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not start")
	}
	close(runner.release)
	select {
	case terminal := <-done:
		// Turn should succeed — ledger errors are non-fatal.
		if terminal.Err != nil {
			t.Fatalf("terminal error = %v, want success (ledger errors are non-fatal)", terminal.Err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("turn did not complete")
	}
	if got := runner.calls.Load(); got != 1 {
		t.Fatalf("runner calls = %d, want provider ran once", got)
	}
}

func TestAsyncStreamLedgerFailureCancelsTurnWithoutPublishingChunk(t *testing.T) {
	root := filepath.Join(t.TempDir(), "session-dir")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	cancelled := make(chan struct{})
	done := make(chan event.Event, 1)
	var publishedText atomic.Int32
	c := New(Options{
		Sink: event.FuncSink(func(e event.Event) {
			if e.Kind == event.Text {
				publishedText.Add(1)
			}
			if e.Kind == event.TurnDone {
				done <- e
			}
		}),
		SessionDir: root, SessionPath: filepath.Join(root, "session.jsonl"),
	})
	t.Cleanup(c.Close)

	c.runGuarded(func(ctx context.Context) error {
		close(started)
		<-release
		c.sink.Emit(event.Event{Kind: event.Text, Text: "must stay behind the WAL"})
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	})
	<-started
	ledger := c.turnEventLedger()
	if ledger == nil {
		t.Fatal("controller did not open a turn ledger")
	}
	if err := ledger.Close(); err != nil {
		t.Fatalf("close active WAL handle: %v", err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("remove temporary ledger directory: %v", err)
	}
	if err := os.WriteFile(root, []byte("block future WAL opens"), 0o600); err != nil {
		t.Fatalf("install WAL blocker: %v", err)
	}
	close(release)

	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("asynchronous stream persistence failure did not cancel the turn")
	}
	select {
	case terminal := <-done:
		if terminal.Status != event.TurnFailed || terminal.Err == nil {
			t.Fatalf("control-plane terminal = %+v, want explicit storage failure", terminal)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("storage failure did not release frontend running state")
	}
	if got := publishedText.Load(); got != 0 {
		t.Fatalf("published text chunks = %d, want none before durable append", got)
	}
	if err := c.turnEventLedgerError(); err == nil {
		t.Fatal("controller accepted new work after the ledger was poisoned")
	}
}
