package jobs

import (
	"context"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"reasonix/internal/event"
)

// TestSnapshotsDoNotConsumeOutputForSession locks the P2 HIGH red line: a
// panel snapshot must never consume readOffset (or resultRead). If it did,
// OutputForSession right after a snapshot would return nothing and the model
// would lose streamed output to the panel.
func TestSnapshotsDoNotConsumeOutputForSession(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	release := make(chan struct{})
	defer close(release)
	j := m.StartForSession("panel-sess", "bash", "panel", func(ctx context.Context, w io.Writer) (string, error) {
		if _, err := w.Write([]byte("first-chunk\n")); err != nil {
			t.Errorf("write: %v", err)
		}
		<-release
		return "", ctx.Err()
	})

	// Wait until the write has landed. Polling the snapshot is itself the
	// probe: it must be non-consuming or the assertion below would fail.
	waitFor(t, func() bool {
		snaps := m.JobSnapshotsForSession("panel-sess")
		return len(snaps) == 1 && strings.Contains(snaps[0].Tail, "first-chunk")
	})

	// Snapshot ran (in waitFor). OutputForSession must still return the full
	// streamed output — the snapshot must not have advanced readOffset.
	text1, status, ok := m.OutputForSession("panel-sess", j.ID)
	if !ok || status != Running {
		t.Fatalf("OutputForSession = ok:%v status:%q, want running", ok, status)
	}
	if !strings.Contains(text1, "first-chunk") {
		t.Errorf("Output after snapshot = %q, want full %q (snapshot consumed readOffset?)", text1, "first-chunk")
	}

	// A second snapshot still sees the full tail (non-consuming), while a
	// second Output — readOffset now at EOF — returns nothing new.
	snaps := m.JobSnapshotsForSession("panel-sess")
	if len(snaps) != 1 || !strings.Contains(snaps[0].Tail, "first-chunk") {
		t.Errorf("second snapshot = %+v, want full tail preserved", snaps)
	}
	if text2, _, ok2 := m.OutputForSession("panel-sess", j.ID); ok2 && text2 != "" {
		t.Errorf("second Output after EOF = %q, want empty (offset was not consumed by the snapshot)", text2)
	}
}

// TestSnapshotTailBoundedRuneSafe locks the P2 tail contract: 4KiB, rune-safe,
// with a [truncated…] marker when anything was cut, and no truncation for
// small output.
func TestSnapshotTailBoundedRuneSafe(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	big := strings.Repeat("界", 5000) // 15000 bytes, well over the 4KiB budget
	jBig := m.StartForSession("s", "bash", "big", func(_ context.Context, w io.Writer) (string, error) {
		if _, err := w.Write([]byte(big)); err != nil {
			t.Errorf("write: %v", err)
		}
		return "", nil
	})
	jSmall := m.StartForSession("s", "bash", "small", func(_ context.Context, w io.Writer) (string, error) {
		if _, err := w.Write([]byte("hello")); err != nil {
			t.Errorf("write: %v", err)
		}
		return "", nil
	})
	if res := m.WaitForSession(context.Background(), "s", []string{jBig.ID, jSmall.ID}, 5); len(res) != 2 {
		t.Fatalf("want 2 results, got %d", len(res))
	}

	byID := map[string]JobSnapshot{}
	for _, s := range m.JobSnapshotsForSession("s") {
		byID[s.ID] = s
	}
	bigSnap, ok := byID[jBig.ID]
	if !ok {
		t.Fatal("big job missing from snapshot")
	}
	if len(bigSnap.Tail) > resultSnapshotMaxBytes {
		t.Errorf("big tail length = %d, want <= %d", len(bigSnap.Tail), resultSnapshotMaxBytes)
	}
	if !strings.HasSuffix(bigSnap.Tail, truncatedMarker) {
		t.Errorf("big tail does not end with %q: %q", truncatedMarker, bigSnap.Tail)
	}
	if !utf8.ValidString(bigSnap.Tail) {
		t.Errorf("big tail is not valid UTF-8 (must be rune-safe): %q", bigSnap.Tail)
	}

	smallSnap, ok := byID[jSmall.ID]
	if !ok {
		t.Fatal("small job missing from snapshot")
	}
	if smallSnap.Tail != "hello" {
		t.Errorf("small tail = %q, want untruncated %q", smallSnap.Tail, "hello")
	}
}

// TestSnapshotStalledOnlyWhenRunning locks the P2 stalled contract: stalled
// only decorates running jobs; a terminal job is never stalled even if the
// flag is still set (terminal priority).
func TestSnapshotStalledOnlyWhenRunning(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	release := make(chan struct{})
	defer close(release)
	j := m.StartForSession("s", "bash", "stalled", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "", ctx.Err()
	})

	if snaps := m.JobSnapshotsForSession("s"); snaps[0].Stalled {
		t.Error("fresh running job must not be stalled")
	}

	// Simulate monitorStalled flagging the job as stalled.
	j.mu.Lock()
	j.stalled = true
	j.mu.Unlock()

	if snaps := m.JobSnapshotsForSession("s"); !snaps[0].Stalled {
		t.Error("running job flagged stalled must report Stalled=true")
	}

	// Terminal beats stalled: flip to Done with the flag still set.
	j.mu.Lock()
	j.status = Done
	j.mu.Unlock()

	snaps := m.JobSnapshotsForSession("s")
	if snaps[0].Stalled {
		t.Error("terminal job must not report Stalled even with the flag set")
	}
	if snaps[0].Status != string(Done) {
		t.Errorf("status = %q, want done", snaps[0].Status)
	}
}

// TestSnapshotIncludesTerminalInterruptedTombstone locks the P2 contract that
// the panel list carries terminal jobs (including interrupted tombstones), not
// just running ones.
func TestSnapshotIncludesTerminalInterruptedTombstone(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	// Real terminal job.
	doneJ := m.StartForSession("s", "task", "term", func(_ context.Context, _ io.Writer) (string, error) {
		return "terminal-answer", nil
	})
	if res := m.WaitForSession(context.Background(), "s", []string{doneJ.ID}, 5); len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want one done", res)
	}

	// Interrupted tombstone in the same shape loadSessionArtifacts builds for a
	// repaired abandoned Running record.
	m.mu.Lock()
	tomb := &Job{
		ID:         "task-900",
		Kind:       "task",
		Label:      "tomb",
		SessionID:  "s",
		status:     Interrupted,
		activityAt: nowMs(),
		done:       make(chan struct{}),
		tombstone:  true,
	}
	close(tomb.done)
	key := jobKey("s", tomb.ID)
	m.jobs[key] = tomb
	m.order = append(m.order, key)
	m.mu.Unlock()

	snaps := m.JobSnapshotsForSession("s")
	if len(snaps) != 2 {
		t.Fatalf("snapshots = %d, want 2 (terminal + tombstone)", len(snaps))
	}
	byID := map[string]JobSnapshot{}
	for _, s := range snaps {
		byID[s.ID] = s
	}
	ds, ok := byID[doneJ.ID]
	if !ok {
		t.Fatal("done job missing from snapshot")
	}
	if ds.Status != string(Done) {
		t.Errorf("done job status = %q, want done", ds.Status)
	}
	if ds.Interrupted {
		t.Error("done job must not report Interrupted")
	}
	if !strings.Contains(ds.Tail, "terminal-answer") {
		t.Errorf("done job tail = %q, want terminal-answer", ds.Tail)
	}

	ts, ok := byID[tomb.ID]
	if !ok {
		t.Fatal("tombstone job missing from snapshot")
	}
	if !ts.Interrupted {
		t.Error("tombstone job must report Interrupted=true")
	}
	if ts.Status != string(Interrupted) {
		t.Errorf("tombstone status = %q, want interrupted", ts.Status)
	}
	if ts.Session != "s" || ts.Kind != "task" || ts.Label != "tomb" {
		t.Errorf("tombstone identity = %+v, want session s / kind task / label tomb", ts)
	}
}

// TestSnapshotSessionFiltering locks the P2 session boundary: snapshots only
// surface jobs owned by the requested session; empty parentSession preserves
// the legacy unscoped behavior.
func TestSnapshotSessionFiltering(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	release := make(chan struct{})
	defer close(release)
	jA := m.StartForSession("sess-a", "bash", "a", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "", ctx.Err()
	})
	jB := m.StartForSession("sess-b", "bash", "b", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "", ctx.Err()
	})

	sa := m.JobSnapshotsForSession("sess-a")
	if len(sa) != 1 || sa[0].ID != jA.ID {
		t.Fatalf("session-a snapshots = %+v, want only %s", sa, jA.ID)
	}
	sb := m.JobSnapshotsForSession("sess-b")
	if len(sb) != 1 || sb[0].ID != jB.ID {
		t.Fatalf("session-b snapshots = %+v, want only %s", sb, jB.ID)
	}
	if all := m.JobSnapshotsForSession(""); len(all) != 2 {
		t.Fatalf("unscoped snapshots = %d, want 2", len(all))
	}
}
