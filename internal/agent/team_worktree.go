package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"reasonix/internal/gitcmd"
)

// D1 branch-parallel isolation (CCB worktree analog): a worktree-granted
// teammate works in its own git worktree on a dedicated branch, so parallel
// writers never touch the main checkout — the whole-workspace write-claim
// serialization is bypassed by physical isolation instead of narrowed
// semantics. The worktree path doubles as the token's WritePathSet (tools
// bind to it), so no extra cwd-override machinery is needed.

const (
	worktreeBranchPrefix = "team-"
	worktreeDirName      = ".reasonix/worktrees"
)

// teammateWorktreePath returns the deterministic worktree directory for a
// teammate: <workspace>/.reasonix/worktrees/<name>. Deterministic slug keeps
// creation idempotent and cleanup resumable (CCB worktree.ts pattern).
func teammateWorktreePath(workspaceRoot, name string) string {
	return filepath.Join(workspaceRoot, worktreeDirName, name)
}

// createTeammateWorktree creates a git worktree for the teammate on a
// dedicated branch (team-<name>) rooted at the current branch (fallback
// HEAD). Fail-closed: any git error aborts the assignment — the teammate
// never silently falls back to the shared checkout.
func createTeammateWorktree(ctx context.Context, workspaceRoot, name string) (path, branch string, err error) {
	path = teammateWorktreePath(workspaceRoot, name)
	branch = worktreeBranchPrefix + sanitizeWorktreeName(name)
	if _, statErr := os.Stat(filepath.Join(path, ".git")); statErr == nil {
		return path, branch, nil // already exists (idempotent resume)
	}
	base, baseErr := currentBranch(ctx, workspaceRoot)
	if baseErr != nil || base == "" {
		base = "HEAD"
	}
	cmd := gitcmd.Command(ctx, workspaceRoot, "worktree", "add", "-B", branch, path, base)
	// -B reuses an orphan branch from a previous run (CCB: avoids one
	// `git branch -D`); GIT_TERMINAL_PROMPT=0 keeps git from hanging on
	// missing refs.
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		// TOCTOU: a concurrent creator may have won the race — if the
		// worktree now exists, treat it as our own (idempotent resume).
		if _, statErr := os.Stat(filepath.Join(path, ".git")); statErr == nil {
			return path, branch, nil
		}
		return "", "", fmt.Errorf("create worktree for %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return path, branch, nil
}

// removeTeammateWorktree deletes the worktree and its branch. Errors are
// returned so the caller can surface them (cleanup never silently loses a
// checkout the user may want to inspect).
func removeTeammateWorktree(ctx context.Context, workspaceRoot, name string) error {
	path := teammateWorktreePath(workspaceRoot, name)
	branch := worktreeBranchPrefix + sanitizeWorktreeName(name)
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return nil // nothing to clean
	}
	rm := gitcmd.Command(ctx, workspaceRoot, "worktree", "remove", "--force", path)
	if out, err := rm.CombinedOutput(); err != nil {
		return fmt.Errorf("remove worktree for %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	br := gitcmd.Command(ctx, workspaceRoot, "branch", "-D", branch)
	_ = br.Run() // branch may already be gone; removal of the worktree is the hard part
	return nil
}

// cleanupTeammateWorktreeIfNeeded implements CCB cleanupWorktreeIfNeeded:
// after a teammate's task completes, the dedicated worktree is auto-removed
// (worktree + branch) iff the teammate left it untouched — clean status and
// no commits ahead of the workspace. If the worktree diverged (dirty files
// or new commits) it is kept and the caller reports its path for manual
// review/merge. Fail-closed: any git error (including change-detection
// failures) is treated as "has changes" — the worktree is never removed by
// accident; only a confirmed-clean worktree gets deleted.
func cleanupTeammateWorktreeIfNeeded(ctx context.Context, workspaceRoot, name string) (kept bool, err error) {
	path := teammateWorktreePath(workspaceRoot, name)
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		return false, nil // nothing to clean
	}
	// Baseline is the workspace's current HEAD: a worktree whose branch is an
	// ancestor of it and whose status is clean carried no new work. Fail-closed
	// if the baseline cannot be resolved — keep, never delete.
	head := gitcmd.Command(ctx, workspaceRoot, "rev-parse", "HEAD")
	headOut, err := head.Output()
	if err != nil {
		return true, fmt.Errorf("resolve baseline HEAD for %q: %w", name, err)
	}
	changed, err := worktreeHasChanges(ctx, path, strings.TrimSpace(string(headOut)))
	if err != nil {
		return true, err // fail-closed: detection error keeps the worktree
	}
	if changed {
		return true, nil // diverged — keep for manual merge
	}
	if err := removeTeammateWorktree(ctx, workspaceRoot, name); err != nil {
		return true, err // removal failed — the worktree still exists
	}
	return false, nil
}

// worktreeHasChanges reports whether the teammate's worktree diverged from
// the commit it was created at (CCB hasWorktreeChanges: dirty status OR new
// commits, fail-closed on git errors).
func worktreeHasChanges(ctx context.Context, worktreePath, headCommit string) (bool, error) {
	status := gitcmd.Command(ctx, worktreePath, "status", "--porcelain")
	out, err := status.Output()
	if err != nil {
		return true, fmt.Errorf("worktree status: %w", err) // fail-closed
	}
	if len(strings.TrimSpace(string(out))) > 0 {
		return true, nil
	}
	commits := gitcmd.Command(ctx, worktreePath, "rev-list", "--count", headCommit+"..HEAD")
	co, err := commits.Output()
	if err != nil {
		return true, fmt.Errorf("worktree rev-list: %w", err)
	}
	return strings.TrimSpace(string(co)) != "0", nil
}

// currentBranch returns the checked-out branch of the workspace ("" if
// detached). Used as the worktree base so teammates branch from where the
// leader actually is.
func currentBranch(ctx context.Context, workspaceRoot string) (string, error) {
	cmd := gitcmd.Command(ctx, workspaceRoot, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// sanitizeWorktreeName makes a teammate name safe as a git branch segment
// (CCB validateWorktreeSlug / isSafeRefName): drop everything outside
// [A-Za-z0-9._-] so ref D/F conflicts and path traversal are impossible.
func sanitizeWorktreeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		out = "teammate"
	}
	return out
}

