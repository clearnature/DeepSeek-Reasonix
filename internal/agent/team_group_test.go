package agent

import (
	"fmt"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

func newGroupTestStore(t *testing.T) *TeammateStore {
	t.Helper()
	jm := jobs.NewManager(event.Discard)
	t.Cleanup(jm.Close)
	ts := NewTeammateStore(nil, jm, t.TempDir())
	t.Cleanup(ts.Close)
	return ts
}

// TestGroupLifecycleCreateRejectDelete locks in the qwen-aligned group
// semantics: one active named group (singleton), a second create is refused
// until delete clears members + name, and delete dissolves the whole team.
func TestGroupLifecycleCreateRejectDelete(t *testing.T) {
	ts := newGroupTestStore(t)
	if name := ts.Name(); name != "" {
		t.Fatalf("fresh store name = %q, want empty (unnamed mode)", name)
	}
	if err := ts.CreateGroup("writers"); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := ts.CreateGroup("second"); err == nil || !strings.Contains(err.Error(), "already active") {
		t.Fatalf("second CreateGroup err = %v, want singleton refusal", err)
	}
	if err := ts.Create("alice", "researcher"); err != nil {
		t.Fatalf("Create member: %v", err)
	}
	if err := ts.DeleteGroup(); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if name := ts.Name(); name != "" {
		t.Fatalf("name after delete = %q, want empty", name)
	}
	if _, ok := ts.Status("alice"); ok {
		t.Fatal("member alice survived DeleteGroup")
	}
	// Group name is reusable after delete.
	if err := ts.CreateGroup("writers"); err != nil {
		t.Fatalf("recreate after delete: %v", err)
	}
}

// TestMemberCapRejectsAtMaxTeamTeammates locks in the qwen MAX_TEAMMATES
// analog: the 11th member create is refused.
func TestMemberCapRejectsAtMaxTeamTeammates(t *testing.T) {
	ts := newGroupTestStore(t)
	for i := range MaxTeamTeammates {
		if err := ts.Create(fmt.Sprintf("m%d", i), "coder"); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	if err := ts.Create("overflow", "coder"); err == nil || !strings.Contains(err.Error(), "maximum teammates") {
		t.Fatalf("overflow create err = %v, want cap refusal", err)
	}
}
