package agent

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestIncrementalFoldPreservesUserTurnBoundariesForExplicitCompress is the
// SchemaV2 fusion guard: the incremental fold keeps logical user-turn
// boundaries in the projection sidecar (role coalescing happens only on the
// outbound copy), so the upstream explicit compress tool can anchor against
// the kept tail after an incremental fold.
func TestIncrementalFoldPreservesUserTurnBoundariesForExplicitCompress(t *testing.T) {
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: strings.Repeat("a ", 500)},
		{Role: provider.RoleAssistant, Content: "b"},
		{Role: provider.RoleUser, Content: "c"},
		{Role: provider.RoleAssistant, Content: "d"},
	}}
	ctx := context.Background()
	prov := &fakeProvider{reply: "s"}
	a := New(prov, tool.NewRegistry(), sess, Options{
		ContextWindow:          20000,
		RecentKeep:             2,
		ArchiveDir:             t.TempDir(),
		StrictAlternatingRoles: true, // force the V1 in-projection coalesce path if it regresses
	}, event.Discard)

	// First compaction installs a full-fold projection (manual degrades).
	if _, err := a.compactToProjection(ctx, CompactionTriggerManual, "", true); err != nil {
		t.Fatalf("first compact: %v", err)
	}
	proj1 := a.compactionState.Projection
	if len(proj1.Messages) == 0 || proj1.CoveredCount != len(sess.Messages) {
		t.Fatalf("first compaction malformed: %+v", proj1)
	}

	// Two consecutive user turns (no assistant between) exercise the V1/V2
	// fork: V1 coalesces them in-projection and erases the anchor; V2 keeps
	// both boundaries and coalesces only on the outbound copy.
	sess.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("x ", 5000)})
	sess.Add(provider.Message{Role: provider.RoleUser, Content: "keep-boundary-anchor"})

	prov.got = nil
	if _, err := a.compactToProjection(ctx, CompactionTriggerPressure, "", false); err != nil {
		t.Fatalf("incremental compact: %v", err)
	}
	proj2 := a.compactionState.Projection
	for i := range proj1.Messages {
		if !reflect.DeepEqual(proj2.Messages[i], proj1.Messages[i]) {
			t.Fatalf("incremental fold changed prior projection bytes at %d", i)
		}
	}
	// Prove the incremental path actually ran: the summarizer must see only the
	// appended segment, never the covered history (a degraded full re-fold
	// would re-submit it).
	if prov.got == nil {
		t.Fatal("incremental fold never reached the summarizer")
	}
	var joined []string
	for _, m := range prov.got {
		joined = append(joined, m.Content)
	}
	if strings.Contains(strings.Join(joined, "|"), strings.Repeat("a ", 500)) {
		t.Fatalf("incremental fold re-folded covered history: %q", strings.Join(joined, "|")[:160])
	}

	// V2: the incremental fold must not coalesce the appended kept tail — the
	// explicit compress anchor below relies on the logical user-turn boundary
	// of the appended messages surviving in the projection sidecar.
	foundAnchor := false
	for _, m := range proj2.Messages {
		if m.Role == provider.RoleUser && m.Content == "keep-boundary-anchor" {
			foundAnchor = true
		}
	}
	if !foundAnchor {
		t.Fatalf("incremental fold lost the appended user turn boundary: %+v", proj2.Messages)
	}

	// The explicit compress tool must resolve an anchor in the kept tail.
	got, err := a.CompressContext(ctx, tool.CompressRequest{
		Direction: "before", Anchor: "keep-boundary-anchor",
	})
	if err != nil {
		t.Fatalf("CompressContext after incremental fold: %v", err)
	}
	if got.Status != "ok" {
		t.Fatalf("explicit compress could not anchor after incremental fold: %+v", got)
	}
}
