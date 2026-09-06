package jobs

import (
	"context"
	"io"
	"strings"
	"testing"

	"reasonix/internal/event"
)

// TestStartSilentSuppressesCompletionEnvelopeAndNotice: a silent (P5 fork)
// job's completion must never auto-deliver a P1 <background-job-result>
// envelope nor emit the closing Notice — fire-and-forget semantics. The result
// stays pollable via wait / Output; only the automatic delivery is suppressed.
func TestStartSilentSuppressesCompletionEnvelopeAndNotice(t *testing.T) {
	sink := &recordingSink{}
	m := NewManager(sink)
	defer m.Close()

	j := m.StartSilentForSession("session-a", "task", "fork", func(ctx context.Context, _ io.Writer) (string, error) {
		return "FORK-ANSWER", nil
	})

	res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != Done || res[0].Output != "FORK-ANSWER" {
		t.Fatalf("wait = %+v, want done with FORK-ANSWER", res)
	}

	// No automatic completion envelope, no closing Notice, no dropped-message
	// warning — the whole terminal announcement is suppressed.
	for _, text := range sink.texts() {
		if strings.Contains(text, "background task finished") || strings.Contains(text, "dropped on completion") {
			t.Fatalf("silent job emitted a suppressed Notice: %q", text)
		}
	}
	if note := m.DrainCompletedNoteForSession("session-a"); note != "" {
		t.Fatalf("silent job leaked a completion envelope: %q", note)
	}

	// The result is still retrievable on demand — wait and Output stay intact.
	if text, status, ok := m.OutputForSession("session-a", j.ID); !ok || status != Done || text != "FORK-ANSWER" {
		t.Fatalf("Output = %q/%v/%v, want FORK-ANSWER/done/ok", text, status, ok)
	}
}

// TestStartSilentKeepsSteerDuringRun: P3 steer and bash_output remain usable on
// a silent job while it runs — silence only affects the terminal auto-delivery.
func TestStartSilentKeepsSteerDuringRun(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	release := make(chan struct{})
	j := m.StartSilentForSession("session-a", "task", "fork", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "done", nil
	})
	if err := m.SendMessageForSession("session-a", j.ID, "steer me"); err != nil {
		t.Fatalf("SendMessageForSession on silent job while running: %v", err)
	}
	close(release)
	res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != Done {
		t.Fatalf("wait after steer = %+v", res)
	}
}

// TestStartSilentStillFiresTaskRecorder: the task lifecycle hook still sees the
// terminal transition exactly like a normal job — silence suppresses only the
// user-visible envelope, never monitoring.
func TestStartSilentStillFiresTaskRecorder(t *testing.T) {
	rec := &recordingRecorder{}
	m := NewManager(event.Discard, WithTaskRecorder(rec))
	defer m.Close()

	j := m.StartSilentForSession("session-a", "task", "fork", func(ctx context.Context, _ io.Writer) (string, error) {
		return "answer", nil
	})
	res := m.WaitForSession(context.Background(), "session-a", []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != Done {
		t.Fatalf("wait = %+v", res)
	}
	starts, dones, status := rec.snapshot()
	if len(starts) != 1 || starts[0] != j.ID+"|task|fork" {
		t.Fatalf("starts = %v, want [%s]", starts, j.ID+"|task|fork")
	}
	if len(dones) != 1 || dones[0] != j.ID || len(status) != 1 || status[0] != Done {
		t.Fatalf("dones = %v status = %v, want [%s] [done]", dones, status, j.ID)
	}
}

// TestStartSilentUnscopedAndNormalJobStillDelivers: StartSilent (unscoped)
// keeps the same silence; a regular StartForSession job still delivers its
// envelope, proving the flag is per-job and not manager-wide.
func TestStartSilentUnscopedAndNormalJobStillDelivers(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	js := m.StartSilent("task", "fork", func(context.Context, io.Writer) (string, error) {
		return "silent", nil
	})
	jn := m.StartForSession("session-a", "task", "normal", func(context.Context, io.Writer) (string, error) {
		return "loud", nil
	})

	rs := m.Wait(context.Background(), []string{js.ID}, 5)
	if len(rs) != 1 || rs[0].Output != "silent" {
		t.Fatalf("silent wait = %+v", rs)
	}
	rn := m.WaitForSession(context.Background(), "session-a", []string{jn.ID}, 5)
	if len(rn) != 1 || rn[0].Output != "loud" {
		t.Fatalf("normal wait = %+v", rn)
	}

	if note := m.DrainCompletedNoteForSession("session-a"); !strings.Contains(note, `task_id="`+jn.ID+`"`) {
		t.Fatalf("normal job lost its envelope: %q", note)
	}
	if note := m.DrainCompletedNoteForSession(""); strings.Contains(note, `task_id="`+js.ID+`"`) {
		t.Fatalf("silent fork job leaked its envelope: %q", note)
	}
}
