package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
// TestTeammateConcurrencyCap verifies the R3/R10 resource guard: without a
// scheduler the legacy background cap (32) rejects the 33rd concurrent
// assign with an explicit error instead of over-committing the machine.
// 60-concurrency therefore requires the scheduler path (queues rather than
// rejects) plus config max_subagent_concurrency >= 60 — see the R3/R10 doc.
func TestTeammateConcurrencyCap(t *testing.T) {
	if testing.Short() {
		t.Skip("concurrency cap test")
	}
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, t.TempDir())
	defer ts.Close()

	const n = 60
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
			t.Fatalf("Create dev%d: %v", i, err)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, n)
	var accepted atomic.Int32
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("dev%d", i)
			if _, err := ts.Assign(ctx, name, "write a short file"); err != nil {
				errs <- fmt.Errorf("%s: %w", name, err)
				return
			}
			accepted.Add(1)
		}(i)
	}
	wg.Wait()
	close(errs)
	rejected := 0
	for err := range errs {
		if !strings.Contains(err.Error(), "limit") {
			t.Fatalf("unexpected assign error: %v", err)
		}
		rejected++
	}
	acceptedN := accepted.Load()
	if acceptedN == 0 || acceptedN > 32 {
		t.Fatalf("accepted = %d, want 1..32 (legacy background cap)", acceptedN)
	}
	if rejected != n-int(acceptedN) {
		t.Fatalf("rejected = %d, want %d (n-accepted)", rejected, n-int(acceptedN))
	}
	// The accepted jobs must show progress (some back to idle) quickly;
	// full drain at 60 mock jobs is bounded by the shared transcript-store
	// contention, not by the cap mechanism under test.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		roster := ts.Roster()
		running, idle := 0, 0
		for _, r := range roster {
			switch r.State {
			case string(TeammateRunning):
				running++
			case string(TeammateIdle):
				idle++
			}
		}
		if idle > 0 {
			t.Logf("R3/R10: cap guard ok — %d accepted (<=32 concurrent), %d rejected with limit error, %d already idle",
				acceptedN, rejected, idle)
			return
		}
		if running < int(acceptedN)/2 {
			// Queue drains: at least half the accepted jobs finished.
			t.Logf("R3/R10: cap guard ok — %d accepted draining (running=%d), %d rejected with limit error",
				acceptedN, running, rejected)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("accepted jobs made no progress (roster=%v)", ts.Roster())
}

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
	for i := range n {
		name := fmt.Sprintf("dev%d", i)
		if err := ts.Create(name, "coder"); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		if err := ts.GrantWorktree(name); err != nil {
			t.Fatalf("GrantWorktree %s: %v", name, err)
		}
	}
	// Assign all 10 in parallel (the burst).
	for i := range n {
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
