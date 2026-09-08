package control

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// TestTeamToolBoardAndSchedulerActions locks the converged surface: the board
// and scheduling capabilities are reachable through the single team tool
// (qwen tool-surface convergence), with the same arbitration as the legacy
// standalone tools.
func TestTeamToolBoardAndSchedulerActions(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := agent.NewTeammateStore(
		agent.NewTaskTool(&smokeProvider{}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "t", nil, 0, "", "", nil),
		jm, t.TempDir())
	t.Cleanup(ts.Close)
	tt := NewTeamLeaderTool(ts, t.TempDir())
	exec := func(args string) (string, error) {
		return tt.Execute(context.Background(), json.RawMessage(args))
	}

	out, err := exec(`{"action":"task_create","prompt":"research X"}`)
	if err != nil || !strings.Contains(out, "published") {
		t.Fatalf("team task_create: %v / %q", err, out)
	}
	list, err := exec(`{"action":"task_list"}`)
	if err != nil || !strings.Contains(list, "research X") {
		t.Fatalf("team task_list: %v / %q", err, list)
	}
	id := strings.Fields(strings.SplitN(list, "]", 2)[0])[1]
	if _, err := exec(`{"action":"task_update","task_id":"` + id + `","owner":"alpha"}`); err != nil {
		t.Fatalf("team task_update claim: %v", err)
	}
	if _, err := exec(`{"action":"task_update","task_id":"` + id + `","owner":"alpha","status":"done"}`); err != nil {
		t.Fatalf("team task_update done: %v", err)
	}
	// Scheduler family through the same tool.
	if _, err := exec(`{"action":"cron_create","schedule_seconds":3600,"prompt":"sweep"}`); err != nil {
		t.Fatalf("team cron_create: %v", err)
	}
	cl, _ := exec(`{"action":"cron_list"}`)
	if !strings.Contains(cl, "sweep") {
		t.Fatalf("team cron_list: %q", cl)
	}
	cid := strings.Fields(strings.SplitN(cl, "]", 2)[0])[1]
	if _, err := exec(`{"action":"cron_delete","id":"` + cid + `"}`); err != nil {
		t.Fatalf("team cron_delete: %v", err)
	}
	// exit_worktree remove requires confirm (qwen guard).
	if _, err := exec(`{"action":"exit_worktree","name":"alpha","disposition":"remove"}`); err == nil {
		t.Fatal("exit_worktree remove without confirm should be rejected")
	}
}
