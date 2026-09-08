package control

import (
	"context"
	"encoding/json"
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// TestTaskBoardToolRoundTrip drives create -> list -> claim -> complete
// through the model tool Execute paths, locking the qwen-aligned board tools
// against drift (claim then owner-only complete).
func TestTaskBoardToolRoundTrip(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	prov := &smokeProvider{}
	ts := agent.NewTeammateStore(
		agent.NewTaskTool(prov, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "t", nil, 0, "", "", nil),
		jm, t.TempDir())
	t.Cleanup(ts.Close)

	id, err := ts.BoardCreate("research X")
	if err != nil {
		t.Fatalf("BoardCreate: %v", err)
	}

	create := NewTaskCreateTool(ts)
	if out, err := create.Execute(context.Background(), json.RawMessage(`{"prompt":"research Y"}`)); err != nil || out == "" {
		t.Fatalf("task_create: %v / %q", err, out)
	}

	list := NewTaskListTool(ts)
	listOut, _ := list.Execute(context.Background(), json.RawMessage(`{}`))
	if listOut == "task board is empty" {
		t.Fatalf("task_list empty after create: %q", listOut)
	}

	update := NewTaskUpdateTool(ts)
	claim, err := update.Execute(context.Background(), json.RawMessage(`{"task_id":"`+id+`","owner":"alpha"}`))
	if err != nil {
		t.Fatalf("task_update claim: %v", err)
	}
	if claim == "" {
		t.Fatal("task_update claim empty output")
	}
	// Non-owner complete rejected.
	if _, err := update.Execute(context.Background(), json.RawMessage(`{"task_id":"`+id+`","owner":"beta","status":"done"}`)); err == nil {
		t.Fatal("non-owner complete should be rejected")
	}
	// Owner completes.
	if _, err := update.Execute(context.Background(), json.RawMessage(`{"task_id":"`+id+`","owner":"alpha","status":"done"}`)); err != nil {
		t.Fatalf("task_update owner complete: %v", err)
	}
}
