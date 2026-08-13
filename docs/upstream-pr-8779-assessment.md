# 上游 PR #8779 评估记录（2026-08-13）

> 决策：**不应用**（用户 2026-08-13 决定，只记录）。本文是移植前的评估备忘，非承诺。

## PR 概要

- 仓库：`esengine/DeepSeek-Reasonix`，PR #8779，作者 SivanCola（collaborator）
- 标题：`fix(provider): follow DeepSeek's official 384K output ceiling / 官方 DeepSeek 取消 16/32/64K 自动档，改走 384K`
- 状态：未合入（评估时点）

## 问题（PR 所述）

官方 DeepSeek V4 Pro 高思考在 ~128KB 隐藏推理（~32K 估算 tokens）后被客户端**取消 stream**——用户看到两个英文警告、无答案，即使上下文几乎为空。

根因两个：
1. 自造 16/32/64K 自动 `max_output_tokens` 阶梯 < 官方文档 384K 输出上限（thinking depth 应只由 effort 控制）
2. 客户端在存储的 reasoning > 128KB 时取消 provider stream

## 修复内容（17 文件）

| 文件 | 改动 |
|---|---|
| `internal/provider/provider.go` | 新增 `DeepSeekMaxOutputTokens = 384_000`；`AutoOutputBudget`（16/32/64K）改为非 DeepSeek 专用 |
| `internal/provider/openai/openai.go` | 官方 DeepSeek `max_output_tokens=0` → 省略字段（删 autoMaxOutput 逻辑）|
| `internal/provider/responses/responses.go` + `vendor.go` | 官方 DeepSeek 省略 `max_output_tokens`（vendor default=0）；**MiMo 保持 16K/32K 阶梯** |
| `internal/provider/anthropic/anthropic.go` | 官方 DeepSeek Anthropic → `max_tokens=384000`（必填字段）；原生 Anthropic 保持 16K |
| `internal/agent/agent.go` + `run_loop.go` | `defaultReasoningByteLimit` 128KB → **8MiB**；超限 `reasoning.Reset()` 继续，**不再取消 stream / 不再 `errReasoningByteLimitExceeded`**；消除 finish-reason 重复警告 |
| `internal/agent/cancel_test.go` | 测试改为"byte guard 不取消 stream"（finite reasoning → text 继续）|
| 文档 | GUIDE / GUIDE.zh-CN / SPEC / SPEC.zh-CN / research/cache-aware-compaction-design.md / config 注释 / render.go |
| `tools/repolint/baseline.json` | file-size +10、test-file-size +2 |

## 与本地 dev/clearnature 的差异与风险

| 项 | 本地现状 | PR 后 | 风险 |
|---|---|---|---|
| deepseek-flash/pro（`kind=anthropic`，api.deepseek.com/anthropic）| 0 → 16/32/64K 阶梯 | 384000 | 输出上限 64K→384K，**长输出成本风险**（按实际 completion 计费）|
| deepseek-responses | vendor 默认（本地 vendorTable deepseek 条目）| 省略字段 | 见"共享窗口交互" |
| agent 128KB reasoning 取消 | 取消 stream + 报错 | 8MiB 缓冲继续 | **纯修复，无风险** |
| MiMo | 本地 `defaultMaxOutputTokens: 128000`（记忆：MiMo 配置）| PR 不动 mimo 条目 → 保持 | 兼容 ✓ |

**⚠️ 本地特有交互点（全量应用前必须补）**：
本地共享窗口输出预算 `effectiveOutputBudget`（`internal/agent/output_budget.go`）在 `req.MaxTokens==0` 时走 `budget <= 0` 分支**直接返回不裁剪**。DeepSeek 省略字段后（服务端 384K），**输入接近窗口时输出可能溢出 400**——PR 未覆盖本地这条共享窗口逻辑。

其他本地差异（PR 未考虑）：检索系统、遥测层（status/tpc/prefix_hash/est）、本地 MiMo summary 隔离（`summaryMode: "none"`）、`maybePredictOverflow`/hard ceiling 路径。

## 决策与待办

- 2026-08-13 用户决定：**不应用，只记录**。
- 若日后应用，推荐**分两步**：
  1. 先移植 agent 的 reasoning 修复（128KB → 8MiB 缓冲继续，不取消 stream + cancel_test 重写）——纯修复无成本风险；
  2. 再评估 384K 阶梯：先补 `effectiveOutputBudget` 对"省略字段 + 共享窗口"的溢出保护（`est + DeepSeekMaxOutputTokens > window` 时裁剪或拒绝），并确认成本影响后再动 openai/responses/anthropic。
- 移植时注意 repolint baseline 的 carry-forward 预算（PR 已列 +10/+2）。
