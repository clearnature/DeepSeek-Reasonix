package control

import (
	"strings"
)

// dreamConsolidationPrompt builds the four-phase memory-consolidation
// instruction for the /dream slash command (P1). The prompt points the model
// at the project memory directory, the MEMORY.md index, and the shared
// knowledge cache, then walks Orient → Gather → Consolidate → Prune.
func dreamConsolidationPrompt(c *Controller, extra string) string {
	var b strings.Builder
	b.WriteString(`# Dream: Memory Consolidation

You are performing a dream — a reflective pass that distills what this session
and recent compactions have learned into durable, well-organized memories so
that future sessions can orient quickly.

## Phase 1 — Orient

- List the project memory directory and the global memory directory.
- Read the memory index (MEMORY.md) to understand what is already recorded.
- Skim existing memory files so you improve them rather than create duplicates.

## Phase 2 — Gather

- Review the most recent compaction digests: the knowledge cache holds entries
  with tier "compaction-digest" distilled from this session's rolling summary.
  Retrieve them (semantic recall) to see what the session learned.
- Look for new information worth persisting that is not already recorded.

## Phase 3 — Consolidate

- Use the remember tool to write or update memory files. Follow the existing
  memory format (frontmatter: title/description/type/scope; body with Why and
  How to apply).
- Merge near-duplicate entries instead of creating new ones.
- Convert relative dates to absolute dates.
- If a new fact contradicts an old memory, fix the old one (forget + rewrite).

## Phase 4 — Prune and index

- Keep the MEMORY.md index lean: one line per entry, one-line hook, under
  150 characters.
- Remove index lines for entries that were deleted or superseded.
- Resolve contradictions between two files by keeping the correct one.

Memory index discipline: MEMORY.md is an index, not a dump. Each line:
` + "`" + `- [Title](file.md) — one-line hook` + "`" + `
Keep memory files under 25KB; split oversized files by topic.
`)
	if extra != "" {
		b.WriteString("\n## Additional context from user\n\n")
		b.WriteString(extra)
	}
	return b.String()
}
