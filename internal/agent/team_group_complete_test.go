package agent

import (
	"context"
	"strings"
	"sync"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// TestGroupCompletedEmitsNoticeOnLastDone locks in checkGroupCompleted: when
// the last teammate job reaches Done, the store must emit one group-complete
// notice (system feedback, not leader polling). No notice before the final
// member completes.
func TestGroupCompletedEmitsNoticeOnLastDone(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	var mu sync.Mutex
	var notices []string
	sink := event.FuncSink(func(e event.Event) {
		if e.Kind == event.Notice {
			mu.Lock()
			notices = append(notices, e.Text)
			mu.Unlock()
		}
	})
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	ts.SetSink(sink)
	if err := ts.CreateGroup("g1"); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := ts.Create("a", "worker"); err != nil {
		t.Fatalf("Create a: %v", err)
	}
	if err := ts.Create("b", "worker"); err != nil {
		t.Fatalf("Create b: %v", err)
	}
	ctx := teamAssignCtx(jm)

	j1, err := ts.Assign(ctx, "a", "first task")
	if err != nil {
		t.Fatalf("Assign a: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{j1}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job a = %+v, want Done", res)
	}
	mu.Lock()
	early := strings.Join(notices, " ")
	mu.Unlock()
	if strings.Contains(early, "all 2 dispatched") {
		t.Fatalf("two-task completion claimed after only one done: %q", early)
	}

	j2, err := ts.Assign(ctx, "b", "second task")
	if err != nil {
		t.Fatalf("Assign b: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{j2}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("job b = %+v, want Done", res)
	}
	mu.Lock()
	got := strings.Join(notices, " ")
	mu.Unlock()
	if !strings.Contains(got, "all 2 dispatched task(s) completed") {
		t.Fatalf("two-task completion notice missing after last done: %q", got)
	}
}
