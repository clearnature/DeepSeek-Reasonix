package agent

import (
	"context"
	"sync"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// TestCompletionCallbackFiresOnDone locks the digest hook: a terminal
// teammate job must invoke the completion callback (registered via
// SetCompletionCallback) with the member name and Done, outside the store
// lock, so the leader digest round is notified without polling.
func TestCompletionCallbackFiresOnDone(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	var mu sync.Mutex
	var gotName string
	var gotStatus jobs.Status
	called := make(chan struct{}, 1)
	ts.SetCompletionCallback(func(name string, status jobs.Status) {
		mu.Lock()
		gotName, gotStatus = name, status
		mu.Unlock()
		called <- struct{}{}
	})
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)
	job, err := ts.Assign(ctx, "alpha", "first task")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{job}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job = %+v, want Done", res)
	}
	select {
	case <-called:
	case <-ctx.Done():
		t.Fatal("completion callback not fired after job Done")
	}
	mu.Lock()
	defer mu.Unlock()
	if gotName != "alpha" || gotStatus != jobs.Done {
		t.Fatalf("callback = (%q, %v), want (alpha, Done)", gotName, gotStatus)
	}
}
