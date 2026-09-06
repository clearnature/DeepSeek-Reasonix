package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestTeammateRaceGrantAssignInterleave exercises the P13 race-audit window:
// concurrent GrantWorktree/SetWorkspaceRoot (writers) against Assign (reader
// of tm.Worktree/workspaceRoot) plus concurrent first-fork worktree creation
// (TOCTOU). Run under -race; any data race in the old code fails this test.
func TestTeammateRaceGrantAssignInterleave(t *testing.T) {
	if testing.Short() {
		t.Skip("race interleave test")
	}
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	root := initGitRepo(t)
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, t.TempDir())
	defer ts.Close()
	ts.SetWorkspaceRoot(root)

	const n = 6
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	parent := New(&mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "parent done"},
		{Type: provider.ChunkDone},
	}}, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	ctx = WithForkSource(ctx, parent)

	for i := range n {
		if err := ts.Create(fmt.Sprintf("dev%d", i), "coder"); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	// Interleave: one goroutine flips worktree grants while others assign.
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("dev%d", i)
			if i%2 == 0 {
				_ = ts.GrantWorktree(name)
			}
			_, _ = ts.Assign(ctx, name, "write a file")
		}(i)
	}
	wg.Wait()

	// All should eventually complete; assert no deadlock by reaching here and
	// every teammate back to idle (mock provider resolves fast).
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		all := true
		for _, r := range ts.Roster() {
			if r.State != string(TeammateIdle) {
				all = false
			}
		}
		if all {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("not all teammates completed after race interleave")
}
