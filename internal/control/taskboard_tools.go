package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/tool"
)

// TaskBoard tools wrap the claimable queue (qwen task_create / task_list /
// task_update on the model surface). Visible to teammates too so a member can
// self-claim with task_update owner=itself — unlike the leader-only team tool.

// TaskCreate publishes an unassigned board item for self-claiming.
type taskCreateTool struct{ ts *agent.TeammateStore }

// NewTaskCreateTool wraps a TeammateStore as the task_create tool.
func NewTaskCreateTool(ts *agent.TeammateStore) tool.Tool { return &taskCreateTool{ts: ts} }

func (t *taskCreateTool) Name() string { return "task_create" }
func (t *taskCreateTool) Description() string {
	return "Publish a new task on the team task board for a teammate to claim. The task has no owner: any teammate can claim it with task_update (owner=<its name>) on its next round. Use for work you want the team to pick up autonomously (qwen TaskCreate analog)."
}
func (t *taskCreateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"prompt":{"description":"The task to publish. Be specific about the deliverable; a claiming teammate sees only this.","type":"string"}},"required":["prompt"],"type":"object"}`)
}
func (t *taskCreateTool) ReadOnly() bool { return false }
func (t *taskCreateTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("task_create: %w", err)
	}
	id, err := t.ts.BoardCreate(p.Prompt)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("task %q published on the board — teammates can claim it with task_update", id), nil
}

// TaskList shows the claimable board.
type taskListTool struct{ ts *agent.TeammateStore }

// NewTaskListTool wraps a TeammateStore as the task_list tool.
func NewTaskListTool(ts *agent.TeammateStore) tool.Tool { return &taskListTool{ts: ts} }

func (t *taskListTool) Name() string { return "task_list" }
func (t *taskListTool) Description() string {
	return "List the team task board. Teammates use this to find unclaimed (owner empty, status open) work to take with task_update owner=<its name>; the leader uses it to see what is claimed/running/done. Returns each task's id, prompt, owner and status."
}
func (t *taskListTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *taskListTool) ReadOnly() bool          { return false }
func (t *taskListTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	items := t.ts.BoardList()
	if len(items) == 0 {
		return "task board is empty — publish work with task_create", nil
	}
	var b strings.Builder
	for _, v := range items {
		fmt.Fprintf(&b, "- %s [%s] owner=%s: %s\n", v.ID, v.Status, v.Owner, v.Prompt)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// TaskUpdate claims or completes a board item.
type taskUpdateTool struct{ ts *agent.TeammateStore }

// NewTaskUpdateTool wraps a TeammateStore as the task_update tool.
func NewTaskUpdateTool(ts *agent.TeammateStore) tool.Tool { return &taskUpdateTool{ts: ts} }

func (t *taskUpdateTool) Name() string { return "task_update" }
func (t *taskUpdateTool) Description() string {
	return "Claim an open board task by setting its owner to your name (teammate self-assign), or complete one you own (set status=done). Claiming requires the task to be unowned; completion requires you to be its owner. qwen TaskUpdate analog."
}
func (t *taskUpdateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"task_id":{"description":"Board task id from task_list (e.g. board-...).","type":"string"},"owner":{"description":"The claiming teammate's name, required when claiming.","type":"string"},"status":{"description":"done to complete an owned task (or open to release it).","type":"string"}},"required":["task_id"],"type":"object"}`)
}
func (t *taskUpdateTool) ReadOnly() bool { return false }
func (t *taskUpdateTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		TaskID string `json:"task_id"`
		Owner  string `json:"owner"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("task_update: %w", err)
	}
	switch strings.TrimSpace(p.Status) {
	case agent.BoardDone:
		if err := t.ts.BoardUpdate(p.TaskID, p.Owner, agent.BoardDone); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q marked done", p.TaskID), nil
	case agent.BoardOpen:
		if err := t.ts.BoardUpdate(p.TaskID, p.Owner, agent.BoardOpen); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q released to open", p.TaskID), nil
	default:
		if err := t.ts.BoardClaim(p.TaskID, p.Owner); err != nil {
			return "", err
		}
		return fmt.Sprintf("task %q claimed by %q", p.TaskID, p.Owner), nil
	}
}
