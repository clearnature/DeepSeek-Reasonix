package control

import (
	"strings"
	"testing"
)

func TestDreamConsolidationPromptPhases(t *testing.T) {
	c := &Controller{}
	p := dreamConsolidationPrompt(c, "")
	for _, phase := range []string{"Phase 1 — Orient", "Phase 2 — Gather", "Phase 3 — Consolidate", "Phase 4 — Prune"} {
		if !strings.Contains(p, phase) {
			t.Fatalf("dream prompt missing %q", phase)
		}
	}
	if !strings.Contains(p, "compaction-digest") {
		t.Fatalf("dream prompt should reference the P2 compaction-digest knowledge cache tier")
	}
	if !strings.Contains(p, "MEMORY.md") {
		t.Fatalf("dream prompt should reference the MEMORY.md index")
	}
	// Extra user context is appended.
	p2 := dreamConsolidationPrompt(c, "focus on the cache work")
	if !strings.Contains(p2, "focus on the cache work") {
		t.Fatalf("extra context not appended")
	}
}
