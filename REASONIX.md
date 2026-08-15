# Reasonix project memory

This file is loaded into every session's system prompt (the cache-stable prefix),
so keep it concise and durable — it is the project's standing instructions to the
agent. It is the Reasonix analog of Claude Code's CLAUDE.md.

## Conventions

- Go kernel under `internal/`; each package owns one concern. A package's long
  explanation belongs in its `doc.go`, not spread across implementation files.
- One transport-agnostic `control.Controller` sits behind every frontend (chat
  TUI, HTTP/SSE serve, Wails desktop). Add behavior to the controller, not a
  frontend, so all three inherit it.
- Layering (enforced): utility packages import nothing under `reasonix/`; only
  the frontends `cli`, `serve`, `acp`, `bot`, `botruntime`, `boot` and the hosts
  `cmd/`, `desktop/` may import `control`; nothing below a frontend may import
  one. The declared sets live in `tools/repolint/layers.go`.
- Subagent delegation keeps five concepts apart: a profile says how a worker
  thinks, `TaskSpec` what this call wants, `CapabilityGrant` what it may touch,
  `ContextRequest` what it starts from, `SchedulerPolicy` when it runs. Put a
  field in whichever member decides its value — profiles carry ceilings, never
  per-call values. `internal/agent/profile_boundary_test.go` enforces it.
- Cache-first: the system-prompt prefix (base prompt + tools + memory) must stay
  byte-stable across turns so DeepSeek's automatic prefix cache stays warm. Never
  mutate it mid-session — ride the turn tail instead (see `control.Compose`).
- Performance features land with an effect test at their final boundary
  (`internal/boot/effect_test.go` pattern): assert what actually reaches the
  provider request, frontend sink, or trajectory through the real `boot.Build`
  assembly. Component correctness is not system effectiveness.
- A mutex- or atomic-guarded struct is ratcheted on its **scalar** field count
  (`struct-state`), not its total: independent flags multiply into states no
  type records as legal. Fixing a boundary case by adding one more `bool` is
  the move this blocks — group by lifetime into a named sub-state instead
  (`agent.perTurnState` is the pattern), which costs one field and removes the
  whole product.

## Comments

Default is none — the code is the truth. Write one only when the **why** is
non-obvious: a hidden constraint, a workaround anchored to something verifiable,
an invariant the type system cannot express, or an external-protocol quirk.

- Declaration doc: ≤15 lines. Package comment: ≤8 lines, or ≤40 in a `doc.go`.
- Every other comment: ≤3 lines. Struct-field and trailing `//`: 1 line.
- Never: restatements of the code, phase/stage narrative, incident or
  conversation history, section banners, commented-out code, `@param` lists.
- `TODO(#nnn):` and `HACK(#nnn):` need the issue anchor. `FIXME` is banned.
- One responsibility per file; 800 lines is the ceiling.

`go run ./tools/repolint` enforces all of it against a ratchet baseline: recorded
debt is tolerated, anything new fails CI. Never widen the baseline to land a
change — fix the code. `-update` exists for carrying debt through a rename or an
extraction, and that diff must be justified in the PR.

## Memory

- Standing instructions are hierarchical: committed/shared `REASONIX.md`,
  `AGENTS.md`, and `CLAUDE.md`; personal `*.local.md` variants; matching files in
  ancestor directories; and user-global files under the memory state root
  (`REASONIX_STATE_HOME`, otherwise `REASONIX_HOME`, otherwise `~/.reasonix` on
  macOS/Linux or `%APPDATA%\reasonix` on Windows). All distinct supported files
  in a directory load; `AGENTS.md` is not merely a fallback.
- `@path` on its own line imports another file's contents.
- `#<note>` in chat quick-adds an always-on instruction. The `remember` tool
  instead saves a fallible background fact (frontmatter file + `MEMORY.md`
  index). Fact `type` classifies content; independent `scope` controls whether it
  is project-only (the default) or explicitly global. The index loads into the
  stable prefix on the next session; global user/feedback bodies also load as
  lower-priority compatibility guidance. The current turn receives a tail note.

## Notes
- 8031 和 # 8024 是什么工具，提供上下文的精度计算和价格成本核算的吗？

## Pre-push CI simulation

Run these **before every commit** to catch the fastest CI failures locally:

```bash
gofmt -w .                          # catches gofmt (saves ~13s CI)
go vet ./...                        # catches vet warnings (saves ~52s CI/lint)
make lint                           # golangci-lint at CI's pin + repolint
go test ./internal/tool/builtin/ ./internal/boot/  # catches tool/boot test breaks
```

## 合并后必做遥测四方审计

合并涉及遥测文件（`internal/agent/*telemetry*.go`、`internal/stats/*`）后，
**先做四方字段对照再提交**：`CompactionTelemetry` 结构 ↔ `emit` 的 detail
键 ↔ `recordCompaction`/`setCompactionInt` 解析 ↔ `CompactionRecord` JSON
tag——任一环缺 = 断链，一次补完（禁止零敲碎打）。失败路径必须落盘
（`status=failed` + `err_type=`，禁止 return 吞 notice）。详见
`docs/telemetry-audit-20260813.md`。

`make lint` runs both gates CI runs, at the version in `.golangci-version`;
`make lint-install` installs it. Do not skip it: a `modernize` finding never
shows up in `go vet`, and the CI round trip that catches it instead costs ten
minutes.

After **merging upstream into dev**, also run the desktop boundary:

```bash
cd desktop && CI=true ~/go/bin/wails build -tags webkit2_41 -ldflags "-X main.version=$(date +%Y%m%d-%H%M)"
```

Wails re-runs `tsc` + `check-bundle-budget.mjs`, which catches frontend
breakage (RecoveryLineageDialog, new panels) and bundle-budget drift that Go
tests never see. Calibrate `check-bundle-budget.mjs` to the dev baseline when
it fails after a merge (upstream budgets exclude dev frontend increments:
Virtuoso/team panels/locale copy).

## Import cycle rule

Before importing a new internal package from a non-test file, verify the target package's **test files** aren't already importing back to you:

```
# BAD: agent(_test.go) → tool/builtin(sessions.go) → agent  → setup failed
```

Use `go test ./path/to/target/` to detect cycles **before** pushing. A `[setup failed]` message means a cycle exists.

## PR hygiene

- **One force-push per round of review feedback.** Multiple force-pushes destroy review history and confuse reviewers.
- **Keep the PR diff minimal.** Only the files relevant to the PR's purpose — no stray changes from other branches.
- **Amend, don't add commits, for review feedback** — keeps the commit history clean.

## Cache-impact PR metadata

When PR changes touch files under `internal/boot/`, `internal/tool/`, `internal/provider/`, or other cache-sensitive paths (listed in `scripts/check-cache-impact.sh`), the PR body MUST include these lines at the end:

```
Cache-impact: <none|low|medium|high> — <reason>
Cache-guard: <focused guard test/command or existing guard rationale>
```

If the PR also touches files under `internal/config/`, `internal/memory/`, `internal/outputstyle/`, `internal/skill/`, or `internal/boot/`, add:

```
System-prompt-review: <reviewer/approval note>
```

Values `n/a`, `none`, `todo`, `tbd` are rejected — use a descriptive reason instead.

<!-- MENTAL-SEAL:START -->

## 🎯 核心目标（由 Mental Seal · 思想钢印 管理）

> 以下目标在每次会话启动时自动注入 AI 的 system prompt，永不被压缩遗忘。

- **前缀缓存是第一原则**：任何发送侧改动（序列化/字段/顺序/重放）先问「会不会改变前缀字节？」——会 = 破坏服务端缓存 = 用户付全价，否决或必须有缓存收益补偿。
- **B2 小 turns 保留窗口必须按「消息位置固定」实现**（保留 `[head, head+N]` 区间），禁止「最新 N 条」动态保留——否则每次 compaction 前缀漂移、服务端缓存反复打穿。
- **C1 重放门控：打开历史会话/recovery 恢复时按缓存窗口分路**：窗口内（`cacheColdAfter()`/TTL 内，deepseek/mimo 24h、dashscope 5m）**原样暖重放**——前缀命中缓存是便宜的知识恢复（不压缩）；窗口外才 `maybeColdResumePrune` 裁剪旧 tool results——把「赌缓存」变成「窗口内确定命中 + 窗口外小前缀」。

<!-- MENTAL-SEAL:END -->
