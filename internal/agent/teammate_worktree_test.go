package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// gitRepo prepares a minimal git repository with one commit.
func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "init")
	return dir
}

// TestEnterExitWorktreeKeepThenRemove locks the qwen enter/exit pair: enter
// creates a worktree and binds the teammate's write token there; exit keep
// leaves it on disk; a second enter+exit remove tears the worktree and branch
// down.
func TestEnterExitWorktreeKeepThenRemove(t *testing.T) {
	repo := gitRepo(t)
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	ts.SetWorkspaceRoot(repo)
	t.Cleanup(ts.Close)
	if err := ts.Create("alpha", "researcher"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	managed := t.TempDir()

	path, branch, err := ts.EnterWorktree(context.Background(), "alpha", managed)
	if err != nil {
		t.Fatalf("EnterWorktree: %v", err)
	}
	if !strings.HasPrefix(branch, "reasonix/") {
		t.Fatalf("branch = %q, want reasonix/*", branch)
	}
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		t.Fatalf("worktree dir %s missing: %v", path, err)
	}
	ts.mu.Lock()
	granted := ts.grants["alpha"].Paths
	ts.mu.Unlock()
	if len(granted) != 1 || granted[0] != path {
		t.Fatalf("write token = %v, want [%s]", granted, path)
	}
	// Double enter refused until exit.
	if _, _, err := ts.EnterWorktree(context.Background(), "alpha", managed); err == nil {
		t.Fatal("second enter without exit should be refused")
	}
	// keep: worktree survives, token cleared.
	if _, err := ts.ExitWorktree(context.Background(), "alpha", false); err != nil {
		t.Fatalf("ExitWorktree keep: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("keep should preserve worktree: %v", err)
	}
	// remove: worktree and branch gone.
	path2, branch2, err := ts.EnterWorktree(context.Background(), "alpha", managed)
	if err != nil {
		t.Fatalf("re-enter: %v", err)
	}
	if _, err := ts.ExitWorktree(context.Background(), "alpha", true); err != nil {
		t.Fatalf("ExitWorktree remove: %v", err)
	}
	if _, err := os.Stat(path2); !os.IsNotExist(err) {
		t.Fatalf("remove should delete worktree %s (err=%v)", path2, err)
	}
	out, _ := exec.Command("git", "-C", repo, "branch", "--list", branch2).Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("branch %s should be deleted, got %q", branch2, out)
	}
}
