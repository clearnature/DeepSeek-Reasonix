package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/tool"
)

// task_stop: qwen's task_stop on the model surface — interrupt a teammate's
// in-flight work and release any board item it had claimed back to open, so
// the task does not silently stay owned by a stopped member.

type taskStopTool struct{ ts *agent.TeammateStore }

// NewTaskStopTool wraps a TeammateStore as the task_stop tool.
func NewTaskStopTool(ts *agent.TeammateStore) tool.Tool { return &taskStopTool{ts: ts} }

func (t *taskStopTool) Name() string { return "task_stop" }
func (t *taskStopTool) Description() string {
	return "Stop a teammate's current work (kills its running job) and release any board task it had claimed back to open. Use when a member is stuck, off-track, or its work is no longer wanted. The teammate stays registered and can be assigned or claim again. qwen task_stop analog."
}
func (t *taskStopTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"name":{"description":"The teammate whose work should stop.","type":"string"}},"required":["name"],"type":"object"}`)
}
func (t *taskStopTool) ReadOnly() bool { return false }
func (t *taskStopTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("task_stop: %w", err)
	}
	name := strings.TrimSpace(p.Name)
	if err := t.ts.TeamStop(name); err != nil {
		return "", err
	}
	released := t.ts.ReleaseBoardTasks(name)
	return fmt.Sprintf("stopped %q (job killed, member idle); %d claimed board task(s) released", name, released), nil
}
