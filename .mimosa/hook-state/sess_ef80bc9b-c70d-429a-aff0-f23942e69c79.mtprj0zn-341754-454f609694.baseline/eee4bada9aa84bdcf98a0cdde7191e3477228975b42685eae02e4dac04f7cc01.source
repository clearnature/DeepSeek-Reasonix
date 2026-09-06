package jobs

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/evidence"
)

// TestStartForegroundSuppressesCompletionEnvelope: a job started via
// StartForegroundForSession must NOT deliver the P1 <background-job-result>
// envelope nor the closing Notice on completion — the foreground run-loop
// already surfaced the result in its tool card and claims it afterwards via
// ClaimForegroundResult.
func TestStartForegroundSuppressesCompletionEnvelope(t *testing.T) {
	sink := &recordingSink{}
	m := NewManager(sink)
	defer m.Close()
	j := m.StartForegroundForSession("session-a", "task", "fg", func(ctx context.Context, _ io.Writer) (string, error) {
		return "FG-RESULT", nil
	})
	<-j.done

	if note := m.DrainCompletedNoteForSession("session-a"); note != "" {
		t.Fatalf("foreground job leaked a completion note: %q", note)
	}
	for _, text := range sink.texts() {
		if strings.Contains(text, "background task finished") || strings.Contains(text, "background task failed") {
			t.Fatalf("foreground job emitted a closing Notice: %q", text)
		}
	}
	fr, ok := m.ClaimForegroundResult("session-a", j.ID)
	if !ok {
		t.Fatal("ClaimForegroundResult did not find the finished foreground job")
	}
	if fr.Status != Done {
		t.Fatalf("claimed status = %q, want done", fr.Status)
	}
	if fr.Result != "FG-RESULT" {
		t.Fatalf("claimed result = %q, want %q", fr.Result, "FG-RESULT")
	}
}

// TestNormalJobStillDeliversEnvelopeAfterForeground: the P1 envelope path for
// ordinary StartForSession jobs must be untouched when a foreground job lives
// alongside them in the same session.
func TestNormalJobStillDeliversEnvelopeAfterForeground(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	bg := m.StartForSession("session-a", "task", "bg", func(context.Context, io.Writer) (string, error) {
		return "BG-RESULT", nil
	})
	fg := m.StartForegroundForSession("session-a", "task", "fg", func(context.Context, io.Writer) (string, error) {
		return "FG-RESULT", nil
	})
	<-bg.done
	<-fg.done

	note := m.DrainCompletedNoteForSession("session-a")
	if !strings.Contains(note, bg.ID) || !strings.Contains(note, "<background-job-result") {
		t.Fatalf("normal job envelope missing while a foreground job was present: %q", note)
	}
	if strings.Contains(note, fg.ID) {
		t.Fatalf("foreground job leaked into the completion note: %q", note)
	}
	if _, ok := m.ClaimForegroundResult("session-a", fg.ID); !ok {
		t.Fatal("foreground result unclaimable")
	}
}

// TestClaimForegroundResultWaitsForTerminal: ClaimForegroundResult blocks until
// the run goroutine finishes, then returns the terminal outcome.
func TestClaimForegroundResultWaitsForTerminal(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	release := make(chan struct{})
	j := m.StartForegroundForSession("session-a", "task", "fg", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "LATE-RESULT", nil
	})
	got := make(chan ForegroundResult, 1)
	go func() {
		if fr, ok := m.ClaimForegroundResult("session-a", j.ID); ok {
			got <- fr
		}
	}()
	select {
	case <-got:
		t.Fatal("ClaimForegroundResult returned before the job finished")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case fr := <-got:
		if fr.Result != "LATE-RESULT" {
			t.Fatalf("claimed result = %q, want %q", fr.Result, "LATE-RESULT")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ClaimForegroundResult did not return after the job finished")
	}
}

// TestClaimForegroundResultBridgesEvidenceLikeCollectBackgroundEvidence: the
// claim carries the same ready-gated, non-consuming evidence lease the
// background collectBackgroundEvidence path takes, so the foreground run-loop
// can merge the job's mutation receipts into the parent ledger and commit them
// through the standard gates.
func TestClaimForegroundResultBridgesEvidenceLikeCollectBackgroundEvidence(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	j := m.StartForegroundForSession("session-a", "task", "writer", func(ctx context.Context, _ io.Writer) (string, error) {
		PublishEvidence(ctx, evidence.ChildEvidenceSummary{Receipts: []evidence.Receipt{{
			ToolName: "write_file", Success: true, Mutation: true, Paths: []string{"fg.go"},
		}}})
		return "WROTE", nil
	})
	fr, ok := m.ClaimForegroundResult("session-a", j.ID)
	if !ok {
		t.Fatal("ClaimForegroundResult did not find the finished foreground job")
	}
	if !fr.Ready {
		t.Fatal("claimed evidence not ready after done")
	}
	if !fr.Evidence.HasMutation() {
		t.Fatalf("claimed evidence lost mutation receipts: %+v", fr.Evidence)
	}
	// Provisional lease, exactly like collectBackgroundEvidence: it does not
	// consume, so a re-lease still sees the receipts until a commit.
	if again, ready := m.TryLeaseEvidenceForSession("session-a", j.ID); !ready || !again.HasMutation() {
		t.Fatalf("re-lease after claim = %v/%+v, want the same receipts", ready, again)
	}
	m.CommitEvidenceForSession("session-a", j.ID)
	if after := m.LeaseEvidenceForSession("session-a", j.ID); len(after.Receipts) != 0 {
		t.Fatalf("committed evidence still leasable after claim: %+v", after)
	}
	// A second claim stays idempotent on the result and never resurrects
	// committed evidence.
	fr2, ok2 := m.ClaimForegroundResult("session-a", j.ID)
	if !ok2 || fr2.Result != "WROTE" {
		t.Fatalf("second claim = %+v/%v, want the same result", fr2, ok2)
	}
	if len(fr2.Evidence.Receipts) != 0 {
		t.Fatalf("second claim resurrected committed evidence: %+v", fr2.Evidence)
	}
}

// TestClaimForegroundResultReturnsKilledJobOutcome: when a foreground job is
// killed mid-run and its run goroutine unwinds, the claim returns the Killed
// status with whatever partial result the run func produced.
func TestClaimForegroundResultReturnsKilledJobOutcome(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	release := make(chan struct{})
	j := m.StartForegroundForSession("session-a", "task", "fg", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "partial", nil
	})
	if !m.KillForSession("session-a", j.ID) {
		t.Fatal("KillForSession did not find the foreground job")
	}
	close(release)
	fr, ok := m.ClaimForegroundResult("session-a", j.ID)
	if !ok {
		t.Fatal("ClaimForegroundResult did not find the killed foreground job")
	}
	if fr.Status != Killed {
		t.Fatalf("claimed status = %q, want killed", fr.Status)
	}
	if fr.Result != "partial" {
		t.Fatalf("claimed result = %q, want %q", fr.Result, "partial")
	}
}

// TestClaimForegroundResultUnknownJob: claiming an unknown job is a clean miss.
func TestClaimForegroundResultUnknownJob(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	if _, ok := m.ClaimForegroundResult("session-a", "no-such-job"); ok {
		t.Fatal("unknown job claimed as a foreground result")
	}
}

// TestStartForegroundSuppressesDroppedMessageNotice: while foregroundClaimPending
// is set, recordCompletion suppresses the whole closing Notice block — including
// the P3 dropped-message warning — because the run-loop is no longer draining
// steer messages and the outcome is claimed, not announced.
func TestStartForegroundSuppressesDroppedMessageNotice(t *testing.T) {
	sink := &recordingSink{}
	m := NewManager(sink)
	defer m.Close()
	release := make(chan struct{})
	j := m.StartForegroundForSession("session-a", "task", "fg", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "done", nil
	})
	if err := m.SendMessageForSession("session-a", j.ID, "steer me"); err != nil {
		t.Fatalf("SendMessageForSession while running: %v", err)
	}
	close(release)
	<-j.done
	for _, text := range sink.texts() {
		if strings.Contains(text, "dropped on completion") || strings.Contains(text, "background task finished") {
			t.Fatalf("foreground job emitted a suppressed Notice: %q", text)
		}
	}
	if note := m.DrainCompletedNoteForSession("session-a"); note != "" {
		t.Fatalf("foreground job leaked a note: %q", note)
	}
	if _, ok := m.ClaimForegroundResult("session-a", j.ID); !ok {
		t.Fatal("foreground result unclaimable")
	}
}

// TestStartForegroundUnscopedClaim: StartForeground (no parent session) keeps
// the same foreground semantics, and the unscoped claim resolves by id.
func TestStartForegroundUnscopedClaim(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	j := m.StartForeground("task", "fg", func(context.Context, io.Writer) (string, error) {
		return "UNSCOPED", nil
	})
	fr, ok := m.ClaimForegroundResult("", j.ID)
	if !ok {
		t.Fatal("unscoped foreground job unclaimable")
	}
	if fr.Result != "UNSCOPED" {
		t.Fatalf("claimed result = %q, want %q", fr.Result, "UNSCOPED")
	}
}

// TestClaimForegroundResultKeepsOutputReadable: claiming a foreground result
// must not consume the job's output — a later OutputForSession (the bash_output
// equivalent) still surfaces the answer.
func TestClaimForegroundResultKeepsOutputReadable(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	j := m.StartForegroundForSession("session-a", "task", "fg", func(context.Context, io.Writer) (string, error) {
		return "VISIBLE", nil
	})
	fr, ok := m.ClaimForegroundResult("session-a", j.ID)
	if !ok || fr.Result != "VISIBLE" {
		t.Fatalf("claim = %+v/%v, want VISIBLE", fr, ok)
	}
	text, _, ok := m.OutputForSession("session-a", j.ID)
	if !ok || !strings.Contains(text, "VISIBLE") {
		t.Fatalf("output after claim = %q/%v, want VISIBLE readable", text, ok)
	}
}
