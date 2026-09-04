package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestTeamSnapshotPersistsAndRestores covers the P10 crash snapshot: a store
// with a snapshot path persists teammates/grants/approvals on mutation, and
// a fresh store on the same path restores them (running state reset to idle).
func TestTeamSnapshotPersistsAndRestores(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "team-state.json")

	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm)
	ts.SetSnapshotPath(path)
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Grant("alpha", WritePathSet{Paths: []string{"/ws/alpha"}}); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if err := ts.RequestApproval("alpha", "req-x", "plan y"); err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	ts.Close()

	// The snapshot is written asynchronously; wait for the file to appear.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("snapshot file missing after mutations")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Fresh store on the same path restores teammates/grants/approvals.
	ts2 := NewTeammateStore(testTaskToolForTeam(t), jm)
	ts2.SetSnapshotPath(path)
	defer ts2.Close()

	var alpha *Teammate
	for i := range ts2.List() {
		tm := &ts2.List()[i]
		if tm.Name == "alpha" {
			alpha = tm
		}
	}
	if alpha == nil {
		t.Fatalf("alpha not restored from snapshot; roster=%+v", ts2.List())
	}
	if alpha.State != TeammateIdle {
		t.Fatalf("restored alpha state = %s, want idle (running jobs died with the process)", alpha.State)
	}
	if p := ts2.grantedPaths("alpha"); len(p.Paths) != 1 || p.Paths[0] != "/ws/alpha" {
		t.Fatalf("grant not restored: %+v", p)
	}
	if len(ts2.PendingApprovals()) != 1 || ts2.PendingApprovals()[0].RequestID != "req-x" {
		t.Fatalf("approvals not restored: %+v", ts2.PendingApprovals())
	}
}

// TestTeamStallAbortKillsStalled verifies the P10 stall worker path: a
// teammate whose running job is stalled past the abort threshold is killed
// and returned to idle.
func TestTeamStallAbortKillsStalled(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, t.TempDir())
	ts.SetStallAbort(10 * time.Second)
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

	jobID, err := ts.Assign(ctx, "alpha", "long task")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	// Simulate the jobs layer marking the job stalled, then run the check.
	jm.MarkStalledForTest(jobID)
	ts.checkStalled()
	st := TeammateRunning
	for _, tm := range ts.List() {
		if tm.Name == "alpha" {
			st = tm.State
		}
	}
	if st != TeammateIdle {
		t.Fatalf("alpha state after stall abort = %s, want idle", st)
	}
}

// TestTeamRosterViewProjection covers the P12 non-consuming roster view: it
// reflects identity/role/state/write posture without advancing lifecycle.
func TestTeamRosterViewProjection(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm, t.TempDir())
	defer ts.Close()
	ts.SetWorkspaceRoot(t.TempDir())

	if err := ts.Create("alpha", "coder"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.GrantWorktree("alpha"); err != nil {
		t.Fatalf("GrantWorktree: %v", err)
	}
	roster := ts.Roster()
	if len(roster) != 1 {
		t.Fatalf("roster = %+v, want 1 entry", roster)
	}
	r := roster[0]
	if r.Name != "alpha" || r.Role != "coder" || r.State != string(TeammateIdle) {
		t.Fatalf("roster entry = %+v, want alpha/coder/idle", r)
	}
	if !r.Worktree || !r.Token {
		t.Fatalf("worktree teammate should report worktree+token posture; got %+v", r)
	}
}
