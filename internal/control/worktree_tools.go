package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/config"
	"reasonix/internal/tool"
)

// Worktree tools: qwen enter_worktree / exit_worktree on the model surface.
// qwen switches agent cwd (single process); a Reasonix teammate is a background
// sandbox job, so enter binds the write token to the worktree path instead.

// enterWorktreeTool creates an isolated worktree for a teammate.
type enterWorktreeTool struct{ ts *agent.TeammateStore }

// NewEnterWorktreeTool wraps a TeammateStore as the enter_worktree tool.
func NewEnterWorktreeTool(ts *agent.TeammateStore) tool.Tool { return &enterWorktreeTool{ts: ts} }

func (t *enterWorktreeTool) Name() string { return "enter_worktree" }
func (t *enterWorktreeTool) Description() string {
	return "Create an isolated git worktree for a teammate and route its writes there: a new branch is created from the current checkout and the teammate's write token is bound to the worktree path, so every later edit stays inside it. The worktree persists until exit_worktree. Only call this when worktree isolation is explicitly wanted (parallel writers, experiments) — not for ordinary edits. qwen enter_worktree analog."
}
func (t *enterWorktreeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"name":{"description":"The teammate that will work inside the new worktree.","type":"string"}},"required":["name"],"type":"object"}`)
}
func (t *enterWorktreeTool) ReadOnly() bool { return false }
func (t *enterWorktreeTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("enter_worktree: %w", err)
	}
	path, branch, err := t.ts.EnterWorktree(ctx, p.Name, config.DeliveryWorktreeDir())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("teammate %q entered worktree %s on branch %s — its writes are bound there until exit_worktree", strings.TrimSpace(p.Name), path, branch), nil
}

// exitWorktreeTool leaves a teammate's worktree (keep or remove).
type exitWorktreeTool struct{ ts *agent.TeammateStore }

// NewExitWorktreeTool wraps a TeammateStore as the exit_worktree tool.
func NewExitWorktreeTool(ts *agent.TeammateStore) tool.Tool { return &exitWorktreeTool{ts: ts} }

func (t *exitWorktreeTool) Name() string { return "exit_worktree" }
func (t *exitWorktreeTool) Description() string {
	return "Leave a teammate's worktree created by enter_worktree. action=keep leaves the checkout and branch on disk for later use; action=remove deletes both (requires confirm=true, mirroring qwen ExitWorktree). The teammate's write token returns to read-only."
}
func (t *exitWorktreeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"name":{"description":"The teammate leaving its worktree.","type":"string"},"action":{"description":"keep (default) or remove.","type":"string"},"confirm":{"description":"Required true when action=remove deletes the worktree and branch.","type":"boolean"}},"required":["name"],"type":"object"}`)
}
func (t *exitWorktreeTool) ReadOnly() bool { return false }
func (t *exitWorktreeTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Name    string `json:"name"`
		Action  string `json:"action"`
		Confirm bool   `json:"confirm"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("exit_worktree: %w", err)
	}
	action := strings.TrimSpace(p.Action)
	if action == "" {
		action = "keep"
	}
	remove := false
	switch action {
	case "keep":
	case "remove":
		if !p.Confirm {
			return "", fmt.Errorf("exit_worktree: action=remove deletes the worktree and branch; pass confirm=true")
		}
		remove = true
	default:
		return "", fmt.Errorf("exit_worktree: unknown action %q (keep|remove)", action)
	}
	return t.ts.ExitWorktree(ctx, p.Name, remove)
}
