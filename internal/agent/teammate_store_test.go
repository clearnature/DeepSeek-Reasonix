package agent

import (
	"context"
	"os"
	"path/filepath"
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

// TestTeammatePostMailPersistsAndFlushes is the P6.1 enhancement 2 e2e: mail
// posted while the teammate is idle lands on disk, and the next assignment
// flushes it into the job's P3 steer queue before the first turn.
func TestTeammatePostMailPersistsAndFlushes(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	root := t.TempDir()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm, root)
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.PostMail("alpha", "check the schema first"); err != nil {
		t.Fatalf("PostMail: %v", err)
	}
	// Idle teammate: mail sits on disk (survives restart).
	dir := filepath.Join(root, "alpha", "inbox")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("inbox entries = %v err=%v, want 1 persisted mail", len(entries), err)
	}
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	ctx = WithForkSource(ctx, newAgentForForkSource())
	jobID, err := ts.Assign(ctx, "alpha", "summarize the findings")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	// The mail was flushed into the job's steer queue (drainable once running).
	if err := jm.SendMessageForSession("leader-session", jobID, "first steer"); err != nil {
		t.Fatalf("steer into teammate job: %v", err)
	}
	// Mail file is consumed after flush.
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("inbox not drained, %d entries left", len(entries))
	}
	// Posting to an unknown teammate fails.
	if err := ts.PostMail("ghost", "hi"); err == nil {
		t.Fatal("PostMail unknown teammate accepted")
	}
}

// newAgentForForkSource returns a minimal *Agent for WithForkSource (P5 fork
// capture). captureForkPrefix only needs a session; nothing provider-bound.
func newAgentForForkSource() *Agent {
	return New(&mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
}

// TestTeammateDependencyGate is the P6.1 enhancement 3 e2e: a dependent
// assignment is refused while its prerequisite job runs, then accepted once the
// prerequisite reaches a terminal state.
func TestTeammateDependencyGate(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	ctx = WithForkSource(ctx, newAgentForForkSource())
	first, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("first Assign: %v", err)
	}
	t.Logf("alpha first jobID=%s", first)
	// beta depends on alpha's still-running first job → refused by the gate
	// (beta is idle, so the running-teammate guard does not mask this).
	if _, err := ts.Assign(ctx, "beta", "second step", first); err == nil ||
		!strings.Contains(err.Error(), "blocked by unfinished dependency") {
		t.Fatalf("dependent Assign while prerequisite running = %v, want blocked", err)
	}
	t.Log("gate refused while running — OK")
	// Wait for the first job to finish, then beta's dependent assignment passes.
	jm.WaitForSession(context.Background(), "leader-session", []string{first}, 5)
	t.Log("first job finished — OK")
	if _, err := ts.Assign(ctx, "beta", "second step", first); err != nil {
		t.Fatalf("dependent Assign after prerequisite done: %v", err)
	}
	if tasks := ts.Tasks(); len(tasks) != 2 {
		t.Fatalf("Tasks() = %d entries, want 2 tracked", len(tasks))
	}
}
