# 上游 main-v2 合并决策记录（2026-09-02）

## 合并概要

| 项目 | 值 |
|---|---|
| 合并日期 | 2026-09-02 |
| 上游版本 | v1.35.0（main-v2 HEAD: 7e09e6f69） |
| 目标分支 | dev/develop |
| 合并方式 | 按主题 cherry-pick（非全量 merge） |
| 提交总数 | 276 个（含 43 个 cherry-pick + 修复提交） |
| 严格保护区 | 全部不合并 |

## 合并主题

### ✅ 已合并（43 个提交）

| 主题 | 提交数 | PR | 风险 | 冲突 |
|---|---|---|---|---|
| security | 3 | #9599, #9596, #9602 | 🟢 低 | 手动解 1 处 |
| frontend 稳定性 | 5 | #9570, #9567, #9585, #9584, #9645 | 🟢 低 | 从 main-v2 接受最终状态 |
| SSH/CLI | 5 | #9634, #9641, #9633, #9700 | 🟢 低 | 手动解 5 处 |
| release/docs | 8 | #9607, #9609, #9667, #9670 | 🟢 低 | 手动解 8 处 |
| deps | 3 | #9549, #9574, #6795, #7824 | 🟢 低 | 手动解 1 处 |
| 测试 | 7 | #9608, #9591, #9630, #9643 | 🟢 低 | 手动解 2 处 |
| session recovery | 7 | #9618, #9591, #9474, #9587 | 🟡 中 | 手动解 3 处 |
| 修复提交 | 5 | — | — | 冲突标记清理 |

### ⛔ 未合并（严格保护区）

| 主题 | PR | 风险 | 原因 |
|---|---|---|---|
| MCP 2026 Apps | #9547 | 🔴 高 | 不稳定，改核心路径，删除 interceptToolBefore |
| Billing | — | 🔴 高 | 删除峰谷定价（peak.go、RateCard.Peak*字段） |
| 压缩/投影核心 | #9082, #9632 | 🔴 高 | 删除大量我们的功能（1929 deletions） |
| Responses API | — | 🔴 高 | 删除大量我们的功能（6414 deletions） |
| agent completion | #9654 | 🟡 中 | 改 internal/agent/ 保护区文件 |

## 风险评估

### 已合并主题风险

| 主题 | 风险点 | 缓解措施 |
|---|---|---|
| security | serve DNS 重绑定、preview 越界读、git clean 过滤器 | 已 cherry-pick，无冲突 |
| frontend 稳定性 | bundle budget、markdown 渲染、scroll 稳定性 | 从 main-v2 接受最终状态，保留我们的自定义 |
| SSH/CLI | SSH sync output、native mouse、preset surface | 手动解决冲突，保留我们的 i18n 字段 |
| release/docs | release notes、contributor credits | 手动解决冲突，接受上游版本 |
| deps | go group 11 updates、jsdom bump | 手动解决冲突，接受上游版本 |
| 测试 | headless checkpoint、session history、MCP routing | 手动解决冲突，接受上游版本 |
| session recovery | path identity、failed session archive | 手动解决冲突，保留我们的 recovery_gc.go |

### 未合并主题风险

| 主题 | 风险点 | 缓解措施 |
|---|---|---|
| MCP 2026 Apps | 改 execute_one.go（核心工具执行路径）、删除 interceptToolBefore | ⛔ 不合并（等上游稳定） |
| Billing | 删除 peak.go（峰谷定价）、RateCard.Peak*字段 | ⛔ 不合并（保留我们的峰谷定价） |
| 压缩/投影核心 | 删除 compact_ratio_price_test、compact_sidecar_fresh_test、compact_turn_guard.go | ⛔ 不合并（保留我们的功能） |
| Responses API | 删除 verification.go 等（6414 deletions） | ⛔ 不合并（保留我们的功能） |
| agent completion | 改 internal/agent/ 保护区文件 | ⏸ 推迟（等上游稳定） |

## 验证结果

| 验证项 | 状态 | 详情 |
|---|---|---|
| go build | ✅ 通过 | 编译成功，无错误 |
| go test ./internal/agent/ | ✅ 通过 | 29s，全部通过 |
| go test ./internal/control/ | ✅ 通过 | 11s，全部通过 |
| go vet | ✅ 通过 | 无警告 |
| repolint | ✅ 通过 | 1407 baselined findings |

## 冲突解决记录

### 1. security: MCP Streamable HTTP (#9602)

- **冲突文件**：`internal/plugin/sdk_session.go`
- **冲突原因**：上游用 `invokeManaged` 替代 `invokeSDKMethod`
- **解决方案**：保留上游版本（`invokeManaged`），后续添加 `sdk_legacy_elicitation.go`

### 2. frontend 稳定性（批量）

- **冲突文件**：`package.json`、`useTranscriptReaderExtentStability.ts`、`useTranscriptScrollArbiter.ts`、`check-bundle-budget.mjs`
- **冲突原因**：这些文件是线性序列，逐个 cherry-pick 造成级联冲突
- **解决方案**：从 main-v2 checkout 最终状态，保留我们的自定义（MCPInteractionCard 删除、rowForAnchor 简化）

### 3. SSH/CLI: i18n 字段

- **冲突文件**：`internal/i18n/messages_en.go`、`internal/i18n/messages_zh.go`、`internal/i18n/messages_zh_tw.go`
- **冲突原因**：上游添加了 CmdTeam* 字段，我们的 Messages 结构体里没有
- **解决方案**：手动添加 CmdTeam* 字段到 Messages 结构体和所有 locale 文件

### 4. session recovery: desktop/tabs.go

- **冲突文件**：`desktop/tabs.go`
- **冲突原因**：上游改了 tabs.go，我们的版本有冲突标记残留
- **解决方案**：从 main-v2 checkout 最终状态，更新 repolint 基线

### 5. control: checkpoint prompt

- **冲突文件**：`internal/control/controller.go`、`internal/control/input.go`
- **冲突原因**：上游 cherry-pick `31b8cf153` 改了 checkpoint prompt 存储逻辑
- **解决方案**：cherry-pick `31b8cf153`（strip interleaved compose prefixes）

## 缓存对齐机制核对

### 1. 发送侧字节变化检查

- **检查点**：上游是否修改了 `internal/provider/responses/` 中的请求构建逻辑？
- **结果**：未修改（Responses API 未合并）

### 2. 前缀稳定性检查

- **检查点**：`summaryRequest(prefix,...)` 是否仍复用主请求前缀？
- **结果**：是（压缩/投影核心未合并，逻辑不变）

### 3. 冻结单元完整性检查

- **检查点**：`saveMainRequest`/`savedMainRequest` 是否存在？`summaryRequest` 是否冻结 tools？
- **结果**：是（压缩/投影核心未合并，逻辑不变）

### 4. sidecar 持久化检查

- **检查点**：`last_wire_tools` 是否持久化到 sidecar？`LoadProjectionSidecar` 是否恢复？
- **结果**：是（压缩/投影核心未合并，逻辑不变）

### 5. legacy 回退检查

- **检查点**：缺字段时是否回退 live 值，禁止发送空值？
- **结果**：是（压缩/投影核心未合并，逻辑不变）

## 遥测四方审计

### CompactionTelemetry 结构 ↔ emit 的 detail 键 ↔ recordCompaction/setCompactionInt 解析 ↔ CompactionRecord JSON tag

- **检查点**：任一环缺 = 断链
- **结果**：未涉及（压缩/投影核心未合并，逻辑不变）

## 下一步

1. **desktop 构建验证**：需要运行 `wails build` 验证前端编译
2. **提交到 main-v2**：如果验证通过，可以创建 PR
3. **严格保护区跟进**：等上游稳定后，再评估是否合并 MCP 2026 Apps、Billing、压缩/投影核心、Responses API、agent completion

## 相关文档

- `docs/merge/merge-discipline.md`：合并纪律（13 条）
- `docs/team/20260812-discipline-charter.md`：纪律团审查规范（6 项验收）
- `docs/merge/20260814-v125-merge-risk.md`：v1.25 合并风险评估
- `docs/merge/20260814-v125-unmerged-report.md`：v1.25 未合并报告
