package boot

// Blank imports register every provider kind at build time so any caller
// assembling through boot (CLI, desktop, harnesses, tests) gets all kinds
// regardless of its own imports. Orchestration (team/task/subagent) must stay
// protocol-agnostic: it must not depend on a caller remembering to import a
// provider package, and the config may carry any supported kind (e.g.
// anthropic-compatible endpoints). Without this, a missing kind surfaces as
// "unknown kind" only when that provider is first resolved.
import (
	_ "reasonix/internal/provider/anthropic"
	_ "reasonix/internal/provider/openai"
	_ "reasonix/internal/provider/responses"
)
