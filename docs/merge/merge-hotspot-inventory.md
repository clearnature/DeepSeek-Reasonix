# Merge Hotspot Inventory（合并热区清单）

> 生成：2026-09-07（北京时间）。数据：dev/develop HEAD（8ada459db）vs `origin/main-v2`
> 快照 diff。目的：每次 merge 上游前**定向复查**，替代盲扫 2681 个上游活跃文件。

## 诊断方法

```
本地 diff（internal/）= 145 文件、+17957 / −173   ← 纯追加式演进，几乎不改上游代码
上游近 60 天活跃    = 2681 个 internal 文件          ← 上游高频
交集（热区）        = 47 个文件                      ← 未来 merge 冲突的真正来源
其余 98 个本地 diff 文件 = 本地独占（上游无此路径）  ← merge 零冲突，勿复查
```

## 热区 47 文件 + 本地增量规模（deleted>0 = 本地改过上游行 = 真叠加点）

| 文件 | 本地增量 |
|---|---|
| internal/jobs/jobs.go | +547 / −39 |
| internal/agent/compact_projection.go | +156 / −16 |
| internal/agent/planner_text_fallback.go | +98 / −0 |
| internal/stats/record.go | +82 / −0 |
| internal/provider/responses/responses.go | +67 / −15 |
| internal/agent/compact.go | +66 / −14 |
| internal/agent/projection.go | +66 / −3 |
| internal/config/config.go | +50 / −0 |
| internal/control/projection_bind.go | +41 / −0 |
| internal/agent/preflight.go | +35 / −0 |
| internal/stats/query.go | +35 / −0 |
| internal/provider/responses/vendor.go | +34 / −9 |
| internal/agent/compact_fold_input.go | +32 / −9 |
| internal/agent/sessionstate.go | +32 / −0 |
| internal/agent/subagent_store.go | +31 / −0 |
| internal/agent/agent.go | +25 / −0 |
| internal/agent/compact_turn_guard.go | +25 / −0 |
| internal/agent/profile_spec.go | +21 / −0 |
| internal/control/controller.go | +19 / −11 |
| internal/tool/contract_test.go | +18 / −15 |
| internal/agent/preview.go | +17 / −0 |
| internal/provider/provider.go | +15 / −0 |
| internal/agent/compact_commit.go | +14 / −0 |
| internal/agent/output_budget.go | +13 / −0 |
| internal/agent/session.go | +10 / −1 |
| internal/jobs/jobs_test.go | +267 / −0 |
| internal/stats/recorder.go | +244 / −0 |
| internal/agent/compact_chunked_policy_test.go | +1 / −1 |
| internal/agent/compact_fold_input_test.go | +9 / −9 |
| internal/agent/compact_overflow_prefix_test.go | +1 / −1 |
| internal/agent/compact_test.go | +5 / −3 |
| internal/agent/context_receipt.go | +4 / −6 |
| internal/agent/output_budget_test.go | +4 / −4 |
| internal/agent/profile_boundary_test.go | +8 / −2 |
| internal/agent/projection_valid_test.go | +1 / −1 |
| internal/agent/session_content.go | +6 / −2 |
| internal/agent/session_extract.go | +1 / −1 |
| internal/agent/session_extract_test.go | +1 / −1 |
| internal/agent/sessionstate_test.go | +6 / −0 |
| internal/agent/tool_result_capability_test.go | +1 / −1 |
| internal/i18n/i18n.go | +1 / −0 |
| internal/i18n/messages_en.go | +1 / −0 |
| internal/i18n/messages_zh.go | +1 / −0 |
| internal/i18n/messages_zh_tw.go | +1 / −0 |
| internal/netclient/netclient_test.go | +8 / −0 |
| internal/plugin/transport_http_test.go | +0 / −9 |
| internal/provider/responses/output_budget.go | +5 / −0 |

## 分类与复查指引

### A 类｜compaction/投影系 —— merge 重点逐块复查
本地冻结逻辑（summaryFoldPlan / LastWire* / main_request_freeze 的投影绑定）住在上游
每版必改文件里。本地增量多为**追加方法/分支**（git 三路自动可解），但
`deleted>0` 行是本地改过上游逻辑的**真叠加点**，上游重写后必手动：

- compact_projection.go（−16）、compact.go（−14）、projection.go（−3）、
  compact_fold_input.go（−9）、context_receipt.go（−6）、preflight.go、
  sessionstate.go、projection_bind.go（本地独有绑定层）、session.go（−1）

复查动作：merge 后 `git diff origin/main-v2 -- <file>` 逐块确认冻结逻辑仍在、
投影哈希对齐机制（coveredPrefixHash/realign）未被上游新语义顶替。

### B 类｜纯追加键/字典/字段 —— 自动合并后抽查即可
本地增量是追加 key/字段（追加区与上游改动区不重叠，冲突率≈0）：

- i18n 全家（messages_*.go 各 +1）、stats（recorder/record/query）、
  config.go（+50）、provider.go、output_budget.go、profile_spec.go、
  subagent_store.go、agent.go（+25）

复查动作：merge 后 `go build ./...` + `go test ./internal/i18n/ ./internal/stats/`
绿即可，无需逐块 diff。

### C 类｜jobs 双端大功能 —— 冲突率最高单点
jobs.go（+547/−39）本地与上游都在持续加功能（teammate 完成事件体系）。
−39 行是真叠加点。上游每动 jobs 必手动合并。

复查动作：merge 后跑 jobs 全家测试
（`go test ./internal/jobs/ ./internal/agent/ -run 'Team|Job'`）。

### 测试文件（+1/−1 类）
多为快照/断言微调，冲突无害；以 `go test ./internal/agent/ ./internal/control/` 全绿为准。

## 用法（merge 前 checklist）

1. `git fetch origin` 后先跑本文档的诊断命令刷新交集（上游活跃面是滚动窗口）
2. 按 A 类逐文件 diff 复查 → B/C 类按测试门禁
3. 合并纪律仍适用：gofmt 全量 → build → 核心包测试 → repolint → 遥测四方审计
