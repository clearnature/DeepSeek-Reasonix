package agent

import (
	"context"
	"fmt"
	"strings"

	"reasonix/internal/worktree"
)

// worktreeSession records where a teammate works in isolation — the Reasonix
// adaptation of qwen's worktree session sidecar: a background sandbox job
// cannot switch cwd, so the session binds the write token to the worktree path.

// EnterWorktree creates an isolated git worktree for the teammate and binds
// its write token to that path (qwen enter_worktree analog). managedRoot is
// the durable worktree storage (config.DeliveryWorktreeDir()).
func (ts *TeammateStore) EnterWorktree(ctx context.Context, name, managedRoot string) (path, branch string, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", fmt.Errorf("enter_worktree: name is required")
	}
	ts.mu.Lock()
	root := ts.workspaceRoot
	tm, ok := ts.teammates[name]
	ts.mu.Unlock()
	if !ok {
		return "", "", fmt.Errorf("teammate %q not found", name)
	}
	if root == "" {
		return "", "", fmt.Errorf("enter_worktree: workspace root is unavailable")
	}
	if tm.Worktree {
		return "", "", fmt.Errorf("teammate %q is already in a worktree — exit_worktree first", name)
	}
	res, err := worktree.Create(ctx, root, managedRoot)
	if err != nil {
		return "", "", err
	}
	// The worktree lives in managed storage outside the workspace, so it is
	// granted directly rather than through NormalizeWritePaths (which rejects
	// out-of-workspace paths by design).
	ws := WritePathSet{Paths: []string{res.WorktreeRoot}}
	if err := ts.Grant(name, ws); err != nil {
		_ = worktree.RollbackCreate(ctx, res)
		return "", "", err
	}
	ts.mu.Lock()
	tm.Worktree = true
	tm.WorktreeRoot = res.WorktreeRoot
	tm.WorktreeBranch = res.Branch
	ts.mu.Unlock()
	ts.saveSnapshot()
	return res.WorktreeRoot, res.Branch, nil
}

// ExitWorktree leaves the teammate's worktree. remove=true tears down the
// worktree and its branch (qwen ExitWorktree remove); remove=false keeps them
// for later use (keep) and only drops the write token.
func (ts *TeammateStore) ExitWorktree(ctx context.Context, name string, remove bool) (string, error) {
	name = strings.TrimSpace(name)
	ts.mu.Lock()
	tm, ok := ts.teammates[name]
	if !ok {
		ts.mu.Unlock()
		return "", fmt.Errorf("teammate %q not found", name)
	}
	root := ts.workspaceRoot
	wtRoot, branch := tm.WorktreeRoot, tm.WorktreeBranch
	tm.Worktree = false
	tm.WorktreeRoot = ""
	tm.WorktreeBranch = ""
	ts.mu.Unlock()
	if err := ts.Revoke(name); err != nil {
		return "", err
	}
	ts.saveSnapshot()
	if !remove || wtRoot == "" {
		return fmt.Sprintf("teammate %q left the worktree (kept %s on branch %s)", name, wtRoot, branch), nil
	}
	if err := worktree.Remove(ctx, root, wtRoot, branch); err != nil {
		return "", err
	}
	return fmt.Sprintf("teammate %q left the worktree — removed %s and branch %s", name, wtRoot, branch), nil
}
