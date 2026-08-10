package jobs

import (
	"context"
	"io"
	"sync"
	"testing"

	"reasonix/internal/event"
)

// doneObserverCapture records completion-observer invocations for assertions.
// The capture is mutex-guarded because observers fire on the job's own
// goroutine while the test reads from the waiting goroutine.
type doneObserverCapture struct {
	mu    sync.Mutex
	calls []string // "id|status"
}

func (c *doneObserverCapture) observer(id string, st Status, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, id+"|"+string(st))
}

func (c *doneObserverCapture) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.calls...)
}

// TestJobDoneObserverFiresOnCompletion: a real job that runs to completion
// fires the observer exactly once, with the terminal id and status. The
// observer is guaranteed to have run by the time Wait returns, because
// recordCompletion (which fires it) precedes close(j.done).
func TestJobDoneObserverFiresOnCompletion(t *testing.T) {
	cap := &doneObserverCapture{}
	m := NewManager(event.Discard, WithJobDoneObserver(cap.observer))
	defer m.Close()

	j := m.Start("task", "demo", func(ctx context.Context, out io.Writer) (string, error) {
		return "answer", nil
	})
	if res := m.Wait(context.Background(), []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("wait = %+v", res)
	}

	calls := cap.snapshot()
	if len(calls) != 1 || calls[0] != j.ID+"|done" {
		t.Fatalf("observer calls = %v, want [%s|done]", calls, j.ID)
	}
}

// TestJobDoneObserverMultipleObserversAllFire: every registered observer runs
// for the same terminal job — the mechanism is a fan-out, not a singleton.
func TestJobDoneObserverMultipleObserversAllFire(t *testing.T) {
	cap1 := &doneObserverCapture{}
	cap2 := &doneObserverCapture{}
	m := NewManager(event.Discard, WithJobDoneObserver(cap1.observer), WithJobDoneObserver(cap2.observer))
	defer m.Close()

	j := m.Start("task", "demo", func(ctx context.Context, out io.Writer) (string, error) {
		return "ok", nil
	})
	m.Wait(context.Background(), []string{j.ID}, 5)

	for name, c := range map[string]*doneObserverCapture{"first": cap1, "second": cap2} {
		if calls := c.snapshot(); len(calls) != 1 || calls[0] != j.ID+"|done" {
			t.Fatalf("%s observer calls = %v, want [%s|done]", name, calls, j.ID)
		}
	}
}

// TestJobDoneObserverNilIsNoop: a nil observer neither fires nor panics —
// both as an Option (ignored at registration) and via SetJobDoneObserver(nil)
// (clears any previously installed observers).
func TestJobDoneObserverNilIsNoop(t *testing.T) {
	m := NewManager(event.Discard, WithJobDoneObserver(nil))
	defer m.Close()

	j := m.Start("bash", "echo", func(ctx context.Context, out io.Writer) (string, error) {
		return "", nil
	})
	if res := m.Wait(context.Background(), []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("wait with nil Option observer = %+v", res)
	}

	// SetJobDoneObserver(nil) clears: an observer registered before the clear
	// must not fire for jobs completing afterwards.
	cap := &doneObserverCapture{}
	m2 := NewManager(event.Discard, WithJobDoneObserver(cap.observer))
	m2.SetJobDoneObserver(nil)
	defer m2.Close()
	j2 := m2.Start("task", "cleared", func(ctx context.Context, out io.Writer) (string, error) {
		return "ok", nil
	})
	m2.Wait(context.Background(), []string{j2.ID}, 5)
	if calls := cap.snapshot(); len(calls) != 0 {
		t.Fatalf("cleared observer still fired: %v", calls)
	}
}

// TestJobDoneObserverPanicDoesNotBreakPipeline: a panicking observer is
// recovered per-callback, so it neither breaks the job teardown (Wait still
// returns the terminal result) nor prevents sibling observers from running.
func TestJobDoneObserverPanicDoesNotBreakPipeline(t *testing.T) {
	cap := &doneObserverCapture{}
	m := NewManager(event.Discard,
		WithJobDoneObserver(func(id string, st Status, err error) { panic("observer boom") }),
		WithJobDoneObserver(cap.observer),
	)
	defer m.Close()

	j := m.Start("task", "demo", func(ctx context.Context, out io.Writer) (string, error) {
		return "answer", nil
	})
	res := m.Wait(context.Background(), []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != Done || res[0].Output != "answer" {
		t.Fatalf("wait after panicking observer = %+v, want done/answer", res)
	}
	if calls := cap.snapshot(); len(calls) != 1 || calls[0] != j.ID+"|done" {
		t.Fatalf("sibling observer after panic = %v, want [%s|done]", calls, j.ID)
	}
}

// TestSetJobDoneObserverAfterConstruction: SetJobDoneObserver installs the
// observer after construction (the boot/controller assembly pattern), and
// jobs started afterwards trigger it.
func TestSetJobDoneObserverAfterConstruction(t *testing.T) {
	cap := &doneObserverCapture{}
	m := NewManager(event.Discard)
	m.SetJobDoneObserver(cap.observer)
	defer m.Close()

	j := m.Start("bash", "echo", func(ctx context.Context, out io.Writer) (string, error) {
		return "", nil
	})
	m.Wait(context.Background(), []string{j.ID}, 5)

	if calls := cap.snapshot(); len(calls) != 1 || calls[0] != j.ID+"|done" {
		t.Fatalf("observer calls = %v, want [%s|done]", calls, j.ID)
	}
}

// TestJobDoneObserverNotFiredAfterClose: the observer lifecycle mirrors the
// completed queue / Notice semantics — once the manager is closed no further
// completion events occur, so no further calls fire (a pre-close completion
// still counts, and an observer installed after Close receives nothing).
func TestJobDoneObserverNotFiredAfterClose(t *testing.T) {
	cap := &doneObserverCapture{}
	m := NewManager(event.Discard, WithJobDoneObserver(cap.observer))

	j := m.Start("task", "demo", func(ctx context.Context, out io.Writer) (string, error) {
		return "ok", nil
	})
	m.Wait(context.Background(), []string{j.ID}, 5)
	m.Close()

	if calls := cap.snapshot(); len(calls) != 1 || calls[0] != j.ID+"|done" {
		t.Fatalf("observer calls after close = %v, want only the pre-close [%s|done]", calls, j.ID)
	}

	var after int
	m.SetJobDoneObserver(func(id string, st Status, err error) { after++ })
	if after != 0 {
		t.Fatalf("observer installed after Close fired %d time(s), want 0", after)
	}
}
