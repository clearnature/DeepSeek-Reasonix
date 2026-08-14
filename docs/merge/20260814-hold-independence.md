# hold 域独立性审计报告（2026-08-14）

> 目的：验证"hold 13 独立于已吸收功能（非阻塞）"声明——不依赖既有报告
> 背书，以 diff/符号引用为证据。

## 审计方法（T0-T7）

| 项 | 证据 |
|---|---|
| T0 基线 | HEAD=e9c101957（dev/develop）；hold 13 均不在 develop（git branch --contains 抽查 4 条 ✓） |
| T7a 符号 | hold 提交的改动文件（history_steer.go 等）本地不存在 |
| T7b 重叠文件 | Transcript.tsx：HEAD vs main-v2 **仅 6 行差异**——本地未含 `hasTranscriptScrollableRange`（#8727/#8760/#8813 语义） |
| T7c hydrate 专项 | hydrateHistoryApply.ts：**本地 73 行 vs 上游 123 行——API 分叉**（本地旧 API + isStaleResidentProjection vs 上游 #8727 三态 API）——本地不依赖 #8727 |
| T7d 后端恢复域 | recovery/gate.go（6 行差）、recovery_gate.go（39 行差）、preview.go（55 行差）——各有演进——本地未含 hold 的 scrollableRange 语义 |

## 结论（P1/P2/P3 + 限定）

- **P1 自洽**：验证链全绿（评估报告 74b7a986e——T2-T5 复跑）
- **P2 独立**：语义独立成立——hold 13 的独特符号/语义（hasTranscriptScrollableRange、history_steer、hydratedHistoryApplyMode 三态）**本地均未含**
- **P3 非阻塞**：成立——hold 域独立于已吸收功能，不影响当前发布

**L1 限定（诚实标注）**：
- **文件面重叠是事实**（Transcript.tsx/useController.ts/hydrateHistoryApply.ts 等 4-6 个前端文件被两边改动）——独立 = **语义/依赖独立**（非文件独立）
- **hydrate API 分叉**（本地 73 行旧 API vs 上游 123 行三态）——未来合入 #8727 时**需 API 适配**（本地迁移或保留本地变体）——记录为合入前置工作
- **hold 13 = 解锁条件未满足（#8739 open）非永不合并**——#8739 修复后按序列重放（R4）

## #8806 评估（2026-08-14——吸收尝试后回退）

- **尝试**：cherry-pick 4206bb5ad（runtime projection——切换文件夹保持运行中会话）
- **回退原因**：冲突复杂度超出"独立功能"评估——ProjectTree.tsx/App.tsx 重构与本地
  team 面板 + 已吸收 #8599（recovery lineages fold）深度交错——6 块冲突逐块解决
  引入重复声明/接口不匹配（tsc 14+ 错误）——**本地前端定制与上游重构的集成面大**
- **结论**：加入 hold（与 hold 13 同类）——#8806 依赖 useProjectCreation.tsx（上游
  新文件）且改 ProjectTree 核心——**等 #8739 解锁后随会话树域整体处理**（届时需要
  专门的"本地前端定制（team 面板）与上游 ProjectTree 重构"集成计划）

## 未来合入前置清单（记录）

1. #8727 合入需 hydrate API 适配（本地旧 API vs 上游三态）
2. baseline.json 需重校准（#8727 与已吸收双方改动）
3. 按上游提交序列重放（#8727→#8728→#8731→#8758…——#8731 已吸收——序列部分跳过）
