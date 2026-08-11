package agent

import (
	"context"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestTeammateTokenGrantRevoke covers the D1 write-token lifecycle: Grant
// issues a token, grantedPaths returns it, Revoke removes it (back to
// read-only), and unknown teammates are rejected.
func TestTeammateTokenGrantRevoke(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm)
	defer ts.Close()

	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// No token yet: read-only default.
	if p := ts.grantedPaths("alpha"); !p.Empty() {
		t.Fatalf("fresh teammate should have no token, got %+v", p)
	}

	// Grant a token over two paths.
	tok := WritePathSet{Paths: []string{"/ws/a", "/ws/b"}}
	if err := ts.Grant("alpha", tok); err != nil {
		t.Fatalf("Grant: %v", err)
	}
	got := ts.grantedPaths("alpha")
	if len(got.Paths) != 2 || got.Paths[0] != "/ws/a" || got.Paths[1] != "/ws/b" {
		t.Fatalf("grantedPaths after Grant = %+v, want the token paths", got)
	}

	// Revoke returns to read-only.
	if err := ts.Revoke("alpha"); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if p := ts.grantedPaths("alpha"); !p.Empty() {
		t.Fatalf("after Revoke token should be empty, got %+v", p)
	}

	// Unknown teammate rejected on both Grant and Revoke.
	if err := ts.Grant("ghost", tok); err == nil {
		t.Fatal("Grant on unknown teammate should fail")
	}
	if err := ts.Revoke("ghost"); err == nil {
		t.Fatal("Revoke on unknown teammate should fail")
	}
}

// TestTeammateAssignWithTokenRunsRestrictedWriter verifies that a granted
// teammate is assigned with a precise write claim (not the whole workspace):
// the fork runs (no no-tools error) and completes with an envelope.
func TestTeammateAssignWithTokenRunsRestrictedWriter(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm)
	defer ts.Close()

	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Grant("alpha", WritePathSet{Paths: []string{"/ws/alpha-work"}}); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	parent := New(&mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "parent done"},
		{Type: provider.ChunkDone},
	}}, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	ctx = WithForkSource(ctx, parent)

	jobID, err := ts.Assign(ctx, "alpha", "do the thing")
	if err != nil {
		t.Fatalf("Assign (granted teammate) = %v, want nil — token must unlock a restricted-writer fork", err)
	}
	if jobID == "" {
		t.Fatal("Assign returned empty job id")
	}
	// Teammate must transition to running (the fork job started).
	st := TeammateIdle
	for _, tm := range ts.List() {
		if tm.Name == "alpha" {
			st = tm.State
		}
	}
	if st != TeammateRunning {
		t.Fatalf("alpha state = %s, want running (fork started)", st)
	}
}
