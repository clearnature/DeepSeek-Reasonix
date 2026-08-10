package jobs

// P3 steer channel, T1: tests for the job pending-message queue
// (SendMessageForSession / DrainPendingMessages / ErrPendingQueueFull).

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
)

// releaseOnCleanup closes ch exactly once, when the test unwinds — safe to also
// close explicitly mid-test without a duplicate-close panic on early failure.
func releaseOnCleanup(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// TestSendMessageForSessionEnqueueAndDrain verifies a message sent to a running
// job is enqueued and then drained inside the job exactly once, via the same
// jobCtxKey pattern PublishEvidence uses.
func TestSendMessageForSessionEnqueueAndDrain(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	release := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(release) })
	got := make(chan string, 1)
	j := m.StartForSession("session-a", "task", "steer", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		if text, ok := DrainPendingMessages(ctx); ok {
			got <- text
		}
		return "done", nil
	})
	if err := m.SendMessageForSession("session-a", j.ID, "steer now"); err != nil {
		t.Fatalf("SendMessageForSession returned error: %v", err)
	}
	close(release)
	select {
	case text := <-got:
		if text != "steer now" {
			t.Fatalf("drained text = %q, want %q", text, "steer now")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("job did not drain the pending message")
	}
	if res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want done", res)
	}
}

// TestSendMessageForSessionRejectsCountLimit verifies the message-count bound:
// 16 messages fit, the 17th is rejected with ErrPendingQueueFull — rejection,
// never a silent drop.
func TestSendMessageForSessionRejectsCountLimit(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	block := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(block) })
	j := m.StartForSession("session-a", "task", "steer", func(context.Context, io.Writer) (string, error) {
		<-block
		return "done", nil
	})
	for i := 1; i <= maxPendingMessages; i++ {
		if err := m.SendMessageForSession("session-a", j.ID, "m"); err != nil {
			t.Fatalf("message %d/%d: SendMessageForSession returned error: %v", i, maxPendingMessages, err)
		}
	}
	if err := m.SendMessageForSession("session-a", j.ID, "m"); !errors.Is(err, ErrPendingQueueFull) {
		t.Fatalf("17th message error = %v, want ErrPendingQueueFull", err)
	}
	close(block)
	if res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want done", res)
	}
}

// TestSendMessageForSessionRejectsByteLimit verifies the byte bound: one
// oversized message is rejected even when message slots are still free.
func TestSendMessageForSessionRejectsByteLimit(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	block := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(block) })
	j := m.StartForSession("session-a", "task", "steer", func(context.Context, io.Writer) (string, error) {
		<-block
		return "done", nil
	})
	// Leave one slot free so the rejection below is provably the byte bound,
	// not the count bound.
	for i := 0; i < maxPendingMessages-1; i++ {
		if err := m.SendMessageForSession("session-a", j.ID, "m"); err != nil {
			t.Fatalf("small message %d: SendMessageForSession returned error: %v", i, err)
		}
	}
	big := strings.Repeat("x", maxPendingMessagesBytes+1)
	if err := m.SendMessageForSession("session-a", j.ID, big); !errors.Is(err, ErrPendingQueueFull) {
		t.Fatalf("oversized message error = %v, want ErrPendingQueueFull", err)
	}
	close(block)
	if res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want done", res)
	}
}

// TestSendMessageForSessionRejectsTerminal verifies only Running jobs accept
// messages; a terminal job is rejected with a descriptive error.
func TestSendMessageForSessionRejectsTerminal(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	j := m.StartForSession("session-a", "task", "steer", func(context.Context, io.Writer) (string, error) {
		return "done", nil
	})
	if res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want done", res)
	}
	err := m.SendMessageForSession("session-a", j.ID, "too late")
	if err == nil || !strings.Contains(err.Error(), "not running") {
		t.Fatalf("terminal SendMessageForSession error = %v, want not-running rejection", err)
	}
}

// TestDrainPendingMessagesOnePerCall verifies DrainPendingMessages pops one
// message per call in FIFO order, no-ops on an empty queue, and is a no-op for
// a context that carries no job (foreground agent / parent / planner).
func TestDrainPendingMessagesOnePerCall(t *testing.T) {
	if _, ok := DrainPendingMessages(context.Background()); ok {
		t.Fatal("plain ctx: DrainPendingMessages must be a no-op")
	}
	m := NewManager(event.Discard)
	defer m.Close()
	block := make(chan struct{})
	t.Cleanup(func() { releaseOnCleanup(block) })
	j := m.StartForSession("session-a", "task", "steer", func(context.Context, io.Writer) (string, error) {
		<-block
		return "done", nil
	})
	jctx := context.WithValue(context.Background(), jobCtxKey{}, j)
	for i := 0; i < 3; i++ {
		if err := m.SendMessageForSession("session-a", j.ID, "m"+string(rune('0'+i))); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	for i := 0; i < 3; i++ {
		want := "m" + string(rune('0'+i))
		text, ok := DrainPendingMessages(jctx)
		if !ok {
			t.Fatalf("drain %d: expected a message", i)
		}
		if text != want {
			t.Fatalf("drain %d text = %q, want %q (FIFO)", i, text, want)
		}
	}
	if _, ok := DrainPendingMessages(jctx); ok {
		t.Fatal("empty queue: DrainPendingMessages must return ok=false")
	}
	close(block)
	if res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want done", res)
	}
}
