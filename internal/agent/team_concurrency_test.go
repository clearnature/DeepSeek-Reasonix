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

// TestTeammateConcurrencyBurst verifies the P13 concurrency mechanism: 10
// worktree-granted teammates assigned at once all reach running together and
// all complete — the scheduler accepts a 6-teammate burst (no session cap
// tripping) and every teammate gets a dedicated worktree.
func TestTeammateConcurrencyBurst(t *testing.T) {
	if testing.Short() {
		t.Skip("concurrency burst test")
	}
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, t.TempDir())
	defer ts.Close()
	root := initGitRepo(t)
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

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("dev%d", i)
		if err := ts.Create(name, "coder"); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		if err := ts.GrantWorktree(name); err != nil {
			t.Fatalf("GrantWorktree %s: %v", name, err)
		}
	}
	// Assign all 10 in parallel (the burst).
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("dev%d", i)
			if _, err := ts.Assign(ctx, name, "write a short file"); err != nil {
				errs <- fmt.Errorf("%s: %w", name, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent assign: %v", err)
	}

	// All 10 must be running together.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		roster := ts.Roster()
		running := 0
		for _, r := range roster {
			if r.State == string(TeammateRunning) {
				running++
			}
		}
		if running == n {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	running := 0
	for _, r := range ts.Roster() {
		if r.State == string(TeammateRunning) {
			running++
		}
	}
	if running != n {
		t.Fatalf("running = %d, want %d — the burst did not all start together", running, n)
	}

	// All complete back to idle.
	deadline = time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		idle := 0
		for _, r := range ts.Roster() {
			if r.State == string(TeammateIdle) {
				idle++
			}
		}
		if idle == n {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("only %d/%d completed within 60s", idleCount(ts), n)
}

func idleCount(ts *TeammateStore) int {
	c := 0
	for _, r := range ts.Roster() {
		if r.State == string(TeammateIdle) {
			c++
		}
	}
	return c
}
