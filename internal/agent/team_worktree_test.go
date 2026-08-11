package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo initializes a throwaway git repository for worktree tests.
func initGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return string(out)
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "main.txt"), []byte("main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-q", "-m", "initial")
	return root
}

// TestWorktreeCreateRemoveIsolation verifies the D1 branch-parallel core:
// a worktree teammate's checkout is a real git worktree on a dedicated
// branch, writes there do not touch the main checkout, and removal cleans
// both the worktree and its branch.
func TestWorktreeCreateRemoveIsolation(t *testing.T) {
	ctx := context.Background()
	root := initGitRepo(t)

	path, branch, err := createTeammateWorktree(ctx, root, "alpha")
	if err != nil {
		t.Fatalf("createTeammateWorktree: %v", err)
	}
	defer removeTeammateWorktree(ctx, root, "alpha")

	if branch != "team-alpha" {
		t.Fatalf("branch = %q, want team-alpha", branch)
	}
	if !strings.HasSuffix(path, filepath.Join(".reasonix", "worktrees", "alpha")) {
		t.Fatalf("worktree path = %q, want under .reasonix/worktrees/alpha", path)
	}

	// Writes in the worktree must not appear in the main checkout.
	if err := os.WriteFile(filepath.Join(path, "alpha.txt"), []byte("alpha work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "alpha.txt")); !os.IsNotExist(err) {
		t.Fatalf("main checkout should not see worktree writes; alpha.txt exists")
	}

	// The dedicated branch exists and is checked out in the worktree.
	br := exec.Command("git", "-C", path, "branch", "--show-current")
	out, _ := br.Output()
	if strings.TrimSpace(string(out)) != "team-alpha" {
		t.Fatalf("worktree branch = %q, want team-alpha", strings.TrimSpace(string(out)))
	}

	// Changes detection: dirty worktree reports true.
	changed, err := worktreeHasChanges(ctx, path, "HEAD")
	if err != nil {
		t.Fatalf("worktreeHasChanges: %v", err)
	}
	if !changed {
		t.Fatal("dirty worktree should report changes")
	}
}

// TestWorktreeIdempotentResume verifies re-creating the same teammate's
// worktree is a no-op (deterministic path, CCB resume fast-path).
func TestWorktreeIdempotentResume(t *testing.T) {
	ctx := context.Background()
	root := initGitRepo(t)

	p1, b1, err := createTeammateWorktree(ctx, root, "beta")
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	defer removeTeammateWorktree(ctx, root, "beta")
	p2, b2, err := createTeammateWorktree(ctx, root, "beta")
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if p1 != p2 || b1 != b2 {
		t.Fatalf("resume changed path/branch: %q/%q vs %q/%q", p1, b1, p2, b2)
	}
}

// TestSanitizeWorktreeName guards the branch-name sanitizer against ref
// collisions and path traversal (CCB isSafeRefName analog).
func TestSanitizeWorktreeName(t *testing.T) {
	cases := map[string]string{
		"alpha":    "alpha",
		"dev one":  "dev-one",
		"../evil":  "evil", // traversal + leading dots stripped
		"a/b":      "a-b",
		"名前":       "teammate", // all-dash would be an invalid ref; fallback
		"!!":       "teammate", // degenerate fallback
		"ok_1-x.y": "ok_1-x.y",
	}
	for in, want := range cases {
		if got := sanitizeWorktreeName(in); got != want {
			t.Errorf("sanitizeWorktreeName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestWorktreeCleanupRemovesWhenUnchanged verifies the D1 auto-cleanup happy
// path: a worktree the teammate left untouched is removed after the task
// (kept=false) — both the worktree directory and its dedicated branch are
// gone.
func TestWorktreeCleanupRemovesWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	root := initGitRepo(t)

	path, _, err := createTeammateWorktree(ctx, root, "gamma")
	if err != nil {
		t.Fatalf("createTeammateWorktree: %v", err)
	}

	kept, err := cleanupTeammateWorktreeIfNeeded(ctx, root, "gamma")
	if err != nil {
		t.Fatalf("cleanupTeammateWorktreeIfNeeded: %v", err)
	}
	if kept {
		t.Fatal("unchanged worktree should be removed (kept=false)")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("worktree directory %q should be gone, stat err = %v", path, statErr)
	}
	out, err := exec.Command("git", "-C", root, "branch", "--list", "team-gamma").Output()
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("branch team-gamma should be deleted, still listed: %q", strings.TrimSpace(string(out)))
	}
}

// TestWorktreeCleanupKeepsWhenChanged verifies the D1 auto-cleanup keep path:
// a worktree that diverged from the workspace (dirty files) is preserved
// (kept=true) with its directory and branch intact for manual review/merge.
func TestWorktreeCleanupKeepsWhenChanged(t *testing.T) {
	ctx := context.Background()
	root := initGitRepo(t)

	path, _, err := createTeammateWorktree(ctx, root, "delta")
	if err != nil {
		t.Fatalf("createTeammateWorktree: %v", err)
	}
	defer removeTeammateWorktree(ctx, root, "delta")

	if err := os.WriteFile(filepath.Join(path, "delta.txt"), []byte("delta work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	kept, err := cleanupTeammateWorktreeIfNeeded(ctx, root, "delta")
	if err != nil {
		t.Fatalf("cleanupTeammateWorktreeIfNeeded: %v", err)
	}
	if !kept {
		t.Fatal("changed worktree should be kept (kept=true)")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("worktree directory %q should still exist, stat err = %v", path, statErr)
	}
	out, err := exec.Command("git", "-C", root, "branch", "--list", "team-delta").Output()
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Fatal("branch team-delta should be preserved when the worktree is kept")
	}
}

// TestWorktreeCleanupFailClosed verifies cleanup never deletes on a git
// error: when the change probe cannot run (repository metadata destroyed),
// the worktree is kept (kept=true) and the error surfaced — fail-closed, so
// an unverifiable state is never cleaned by accident.
func TestWorktreeCleanupFailClosed(t *testing.T) {
	ctx := context.Background()
	root := initGitRepo(t)

	path, _, err := createTeammateWorktree(ctx, root, "echo")
	if err != nil {
		t.Fatalf("createTeammateWorktree: %v", err)
	}

	// Break the repository so the workspace HEAD probe must fail; cleanup
	// must keep the worktree rather than delete it on an unverifiable state.
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}

	kept, err := cleanupTeammateWorktreeIfNeeded(ctx, root, "echo")
	if err == nil {
		t.Fatal("cleanupTeammateWorktreeIfNeeded should fail closed on a git error")
	}
	if !kept {
		t.Fatal("fail-closed cleanup must keep the worktree (kept=true)")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("worktree directory %q should be preserved on git error, stat err = %v", path, statErr)
	}
}
