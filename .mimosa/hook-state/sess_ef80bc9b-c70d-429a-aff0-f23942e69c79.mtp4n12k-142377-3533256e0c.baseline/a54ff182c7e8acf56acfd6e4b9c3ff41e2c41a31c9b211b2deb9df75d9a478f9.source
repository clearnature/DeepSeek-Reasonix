# Repository Guidelines

Reasonix is a Go-based AI coding agent: a transport-agnostic Go kernel under `internal/` with three frontends — the CLI/TUI (`cmd/reasonix`), an HTTP/SSE server (`internal/serve`), and a Wails desktop app (`desktop/`) — plus an SDK (`sdk/`) and benchmark harnesses (`benchmarks/`). Default branch: `main-v2`. Module: `reasonix` (Go 1.25+).

## Architecture and Key Invariants

- One `control.Controller` implements behavior; frontends are thin. Add behavior to the controller, not a frontend.
- Layering is enforced by `tools/repolint/layers.go`: only frontends and hosts may import `control`, and utility packages import nothing under `reasonix/`.
- Cache-first: the system-prompt prefix (base prompt + tools + memory) must stay byte-stable across turns to keep the provider prefix cache warm. Never mutate it mid-session.
- Durable project rules live in `REASONIX.md`, which is loaded into every session; keep it concise. Long package explanations belong in `doc.go`.

## Build, Test, and Development Commands

- `make build` — build `bin/reasonix` and the plugin example
- `make test` — run `go test ./...`
- `make vet` / `make lint` — run `go vet` and the repo layer lint (`go run ./tools/repolint`)
- `make lint-cross` — golangci-lint across OS/build-tag combinations
- `make fmt` — apply `gofmt -w`
- `make hooks` — install `.githooks` (pre-push runs `go vet`)
- `make desktop-test` / `sdk-test` — test the `desktop/` and `sdk/go` modules

For isolated development, run `REASONIX_HOME=/tmp/reasonix-dev go run ./cmd/reasonix` so no state touches a real install.

## Coding Conventions

- `gofmt` and `go vet` are enforced by CI; run both before pushing.
- Wrap errors with `%w` (`fmt.Errorf("...: %w", err)`); library code never calls `os.Exit` or prints to stdout/stderr — only frontends decide exit codes and user-facing output.
- Exported identifiers must have doc comments. Default comments: none — write one only when the why is non-obvious.
- Prefer table-driven tests; performance features need an effect test at their final boundary (see the `internal/boot/effect_test.go` pattern).

## Commit and Pull Request Guidelines

- Use [Conventional Commits](https://www.conventionalcommits.org/): `feat(agent): ...`, `fix(cli): ...`, `test(event): ...`, `docs: ...`
- Branch from `main-v2` and open PRs against `main-v2`; PRs must pass `go test ./...` and leave `gofmt -l .` clean.
- Cache-sensitive changes (system prompt, memory prefix, skill index, tool schemas, compaction, MCP/tool registration) require `Cache-impact:`, `Cache-guard:`, and `System-prompt-review:` lines in the PR body.

## Definition of Done

Verify before claiming completion: `make test`, `make lint`, and `gofmt -l .` all pass; for performance work, the boundary effect test demonstrates the actual behavior change.

## Agent-Specific Instructions

- The GitNexus code-intelligence workflow (impact analysis before edits, `detect_changes()` before commit) is preserved in `AGENTS.md.gitnexus.bak`.
- Update `REASONIX.md` when durable project rules change; keep AGENTS.md a concise pointer.
