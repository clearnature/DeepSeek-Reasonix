package agent

import (
	"context"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

func testTaskToolForTeam(t *testing.T) *TaskTool {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "done"},
		{Type: provider.ChunkDone},
	}}
	task := NewTaskTool(sub, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	return task
}

// TestTeammateStoreCreateListRemove covers the registry lifecycle.
func TestTeammateStoreCreateListRemove(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	if err := ts.Create("alpha", "researcher"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Create("alpha", ""); err == nil {
		t.Fatal("duplicate Create accepted")
	}
	if err := ts.Create("", "x"); err == nil {
		t.Fatal("empty name Create accepted")
	}
	list := ts.List()
	if len(list) != 1 || list[0].Name != "alpha" || list[0].State != TeammateIdle {
		t.Fatalf("List = %+v, want one idle alpha", list)
	}
	if err := ts.Remove("alpha"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := ts.Status("alpha"); ok {
		t.Fatal("alpha still present after Remove")
	}
}

// TestTeammateStoreAssignRejectsUnknownAndRunning pins the two guard rails:
// unknown teammate and double-assignment while running.
func TestTeammateStoreAssignRejectsUnknownAndRunning(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	if _, err := ts.Assign(context.Background(), "ghost", "work"); err == nil ||
		!strings.Contains(err.Error(), "unknown teammate") {
		t.Fatalf("Assign unknown = %v, want unknown-teammate error", err)
	}
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.mu.Lock()
	ts.teammates["alpha"].State = TeammateRunning
	ts.mu.Unlock()
	if _, err := ts.Assign(context.Background(), "alpha", "more work"); err == nil ||
		!strings.Contains(err.Error(), "is running") {
		t.Fatalf("Assign running = %v, want running error", err)
	}
}

// TestTeammateAssignStartsBackgroundJobWithEnvelope is the P6 e2e: a teammate
// assignment starts a background job whose result rides the P1 envelope back
// (non-silent fork), unlike P5 fire-and-forget.
func TestTeammateAssignStartsBackgroundJobWithEnvelope(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm)
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
	jobID, err := ts.Assign(ctx, "alpha", "summarize the findings")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if jobID == "" {
		t.Fatal("Assign returned empty job id")
	}
	res := jm.WaitForSession(context.Background(), "leader-session", []string{jobID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("teammate job = %+v, want one Done", res)
	}
	// Non-silent: the P1 envelope carries the teammate's result back.
	note := jm.DrainCompletedNoteForSession("leader-session")
	if !strings.Contains(note, `task_id="`+jobID+`"`) {
		t.Fatalf("envelope missing teammate job: %q", note)
	}
	if list := ts.List(); len(list) != 1 || list[0].State != TeammateIdle {
		t.Fatalf("alpha state after completion = %+v, want idle (lazy sync)", list)
	}
}
