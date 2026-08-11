package agent

import (
	"context"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestPlanApprovalLifecycle covers the P9 gate: request → pending → approve
// → consumed; duplicate request rejected; verdict reachable via Approve.
func TestPlanApprovalLifecycle(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, t.TempDir())
	defer ts.Close()

	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Submit a plan for approval.
	if err := ts.RequestApproval("alpha", "req-1", "implement quicksort"); err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	pending := ts.PendingApprovals()
	if len(pending) != 1 || pending[0].RequestID != "req-1" || pending[0].Teammate != "alpha" {
		t.Fatalf("PendingApprovals = %+v, want one req-1 from alpha", pending)
	}

	// Duplicate request id is rejected.
	if err := ts.RequestApproval("alpha", "req-1", "again"); err == nil {
		t.Fatal("duplicate request id should be rejected")
	}

	// Approve consumes the request.
	if err := ts.Approve("req-1", true, "leader-session"); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if len(ts.PendingApprovals()) != 0 {
		t.Fatalf("after approve, pending should be empty; got %+v", ts.PendingApprovals())
	}

	// Unknown / consumed request id errors.
	if err := ts.Approve("req-1", true, "leader-session"); err == nil {
		t.Fatal("approving a consumed request should fail")
	}
	if err := ts.Approve("nope", true, "leader-session"); err == nil {
		t.Fatal("approving an unknown request should fail")
	}
}

// TestPlanApprovalVerdictReachesTeammate verifies the verdict path: a
// teammate submits a plan while its job runs, and Approve(deny) returns
// cleanly with the request consumed (the verdict itself rides the P3 steer
// queue — best effort, covered by jobs.SendMessageForSession).
func TestPlanApprovalVerdictReachesTeammate(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm, t.TempDir())
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

	if _, err := ts.Assign(ctx, "alpha", "plan the work"); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := ts.RequestApproval("alpha", "req-2", "plan: implement x"); err != nil {
		t.Fatalf("RequestApproval: %v", err)
	}
	if err := ts.Approve("req-2", false, "leader-session"); err != nil {
		t.Fatalf("Approve(deny): %v", err)
	}
	if len(ts.PendingApprovals()) != 0 {
		t.Fatalf("request should be consumed after approve; got %+v", ts.PendingApprovals())
	}
}
