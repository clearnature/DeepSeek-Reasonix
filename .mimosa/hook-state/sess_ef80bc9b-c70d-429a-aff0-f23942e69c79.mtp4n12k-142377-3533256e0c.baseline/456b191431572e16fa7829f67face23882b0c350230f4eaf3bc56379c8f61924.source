package agent

import (
	"context"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestTeammateRefPersistsAcrossAssignments pins datamodel §4.3: the first
// (fork) assignment records the NEW transcript ref back on the teammate, so a
// later assignment continues the same transcript instead of re-forking the
// leader prefix (which would drop the cache-warm prefix). The ref must be
// stable across rounds: fork produces a fresh ref, continue keeps it.
func TestTeammateRefPersistsAcrossAssignments(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	parent := New(&mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "parent done"},
		{Type: provider.ChunkDone},
	}}, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	ctx = WithForkSource(ctx, parent)

	// First round: fork, so a fresh transcript ref must be recorded.
	job1, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("Assign #1: %v", err)
	}
	if job1 == "" {
		t.Fatal("Assign #1 returned empty job id")
	}
	ref1 := teammateRefOf(t, ts, "alpha")
	if ref1 == "" {
		t.Fatal("first (fork) assignment left tm.Ref empty — continue would re-fork")
	}
	waitTeammateIdle(t, ts, "alpha", 5*time.Second)

	// Second round: continue the same transcript; the ref must not change.
	job2, err := ts.Assign(ctx, "alpha", "second step")
	if err != nil {
		t.Fatalf("Assign #2: %v", err)
	}
	if job2 == "" {
		t.Fatal("Assign #2 returned empty job id")
	}
	if ref2 := teammateRefOf(t, ts, "alpha"); ref2 != ref1 {
		t.Fatalf("second assignment changed ref %q -> %q (continue broken, re-forked)", ref1, ref2)
	}
	waitTeammateIdle(t, ts, "alpha", 5*time.Second)
}

// teammateRefOf returns the recorded transcript ref of a teammate.
func teammateRefOf(t *testing.T, ts *TeammateStore, name string) string {
	t.Helper()
	for _, tm := range ts.List() {
		if tm.Name == name {
			return tm.Ref
		}
	}
	t.Fatalf("teammate %q not found", name)
	return ""
}

// waitTeammateIdle polls until the teammate's state is idle (completion event
// flips it asynchronously) or the deadline expires.
func waitTeammateIdle(t *testing.T, ts *TeammateStore, name string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if tm, ok := ts.Status(name); ok && tm.State == TeammateIdle {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("teammate %q not idle within %s (last state via Status)", name, timeout)
}

// TestTeammatePostMailPersistsAndFlushes is the P6.1 enhancement 2 e2e: mail
// posted while the teammate is idle lands on disk, and the next assignment
// flushes it into the job's P3 steer queue before the first turn.
