# 技术路线回归 · 未来展望 · 社区情况 · 边界情况
# Technical Roadmap Retrospective · Outlook · Community Landscape · Boundaries

> 日期 / Date: 2026-08-09
> 状态 / Status: 当前技术版图说明（含 PR #8006 发布背景）
> 双语 / Bilingual: 中文为主，英文对照

---

## 一、技术路线回归 / Technical Roadmap Retrospective

### 核心锚点 / The Anchor

> 多数 agent 每回合为同一段不断增长的 prompt 付全价；Reasonix 让前缀逐字节稳定，让 DeepSeek 的缓存替你扛下来。
> Most agents pay full price every turn for the same ever-growing prompt. Reasonix keeps the prefix byte-stable so DeepSeek's cache carries the cost.

### 演进线 / Evolution Line

| 阶段 / Phase | 里程碑 / Milestone | 内容 / Content |
|---|---|---|
| 1. 地基 / Foundation | #7839（**已上游合并** / merged upstream）| 128K 输出预算 + cache-aware context projection（canonical 永不改写，投影 = model-visible 视图）|
| 2. 修复 / Fixes | #7913 系列（本地 / local）| summarize 预算裁剪、有界折叠（`ErrCompactionInputTooLarge` + 熔断）、resume gate 形状校准 |
| 3. 缓存友好 / Cache-friendly | 增量折叠（local）| 有投影时只折叠新增段 → 旧投影字节不变 → **前缀缓存持续命中** |
| 4. 上游融合 / Upstream fusion | **pr7913-v3** | 融合上游 #8000 SchemaV2：投影保留 user-turn 边界，出站才合并角色；显式压缩工具在增量折叠后的投影上可锚定 |

### 关键验证 / Key Validations

- **社区复现**：issue #8004（macOS）+ #7972（Windows）报的 400 错误，与我们 8/8 修复的根因**完全同款**（`messages + 131072 completion > 1M`）
- **上游走向我们**：#8000 从「canonical 重写」改为「投影安装」——我们 3 天前落地的设计
- **数据实证**：8/8 DeepSeek 用量——修复后 miss 从 164 万/时降到 20.6 万/时（-87%），命中率 96.2% → 98.7%

---

## 二、未来展望 / Future Outlook

| 方向 / Direction | 状态 / Status | 说明 / Note |
|---|---|---|
| **响应侧缓存验证闭环** / Response-side cache verification | 记录侧已实施（`CacheMissDrop`），**UI 展示待接** | 唯一实质差距（CCB 三方向分析结论）；按哲学：纯诊断、零发送侧改动 |
| **余额按需拉取 + UI** / On-demand balance + UI | 后端 `Balance()` 已就绪，**前端接线待做** | 避免 CCB 覆辙「数据就绪 UI 未接线」；计费透明：官方接口（deepseek）查询、黑箱（mimo）只估算 |
| **显式压缩路径有界化** / Bounded explicit compression | 部分（共享 `runCompactionSummary` 裁剪）| 上游 #8000 的 `prepareVisibleCompression` 仍无 `fitFoldToWindow`——大 fold 显式压缩同样有 400 风险，可后续补 |
| **预测式检查** / Predictive check | 已实施（1a，只记不压）| CCB query.ts:856 借鉴，哲学边界：只发 Notice 不触发压缩 |
| **多 provider 窗口自适应** / Multi-provider window adaptation | 设计已备 | headroom 分档（13K/30K/50K）仅限 400K+ 窗口，当前 8192 实测 99.6% 命中，无需求不实施 |

---

## 三、社区情况 / Community Landscape

### 事实 / Facts

- **296 个 open PR**（8/8）：SivanCola（主要维护者）主导合并，绝大多数 PR 长期无人问津
- **#8004 / #7972 至今 0 comment、无 assignee、无修复 PR**——我们 8/8 已修，**PR #8006（draft）已发出，3 天测试窗口（截至 08-12）**
- **上游方向**：Goal 状态机（#7959，open）、显式压缩工具（#8000，已合）——都在走向「投影化、canonical 不动」= 我们已落地的哲学
- **上游合并不可预期**（管理员不处理 PR）→ 本地优先策略：`dev/clearnature` 为真成果，PR 只是对外发布窗口

### 我们的位置 / Our Position

- 差异化：296 个 PR 多为 UI/deps/平台类；**压缩内核（预算/有界/增量）只有我们在做**
- 已验证：上游 #7839 被合并 = 质量路线被认可
- 社区价值：PR #8006 声明「3 天后合入 fork 基线」，给报障者（#8004/#7972）一个真实可用的修复出口

---

## 四、边界情况 / Boundaries

### 硬边界 / Hard Boundaries（不可违反）

1. **前缀字节稳定**：任何发送侧改动先问「会不会改变前缀字节」——会 = 破坏缓存 = 否决或必须有缓存收益补偿
2. **canonical 永不改写**：普通压缩/恢复只写投影，历史是永久事实源（append-only）
3. **估算口径分离**：发送保护路径（preflight/resume）用保守估算（含 safety factor）；resume gate 用 model-visible 真实形状——两类口径不可混用

### 已知边界 / Known Edges

4. **小窗口边界**：窗口 ≤ 8K 时压缩请求必然失败（输入 + 8K floor > 窗口）——`fitFoldToWindow` 对此类窗口跳过，保留失败传播
5. **显式压缩无界风险**：上游 `prepareVisibleCompression` 的 fold 无窗口裁剪——我们共享的预算裁剪兜底输出侧，但 fold 输入侧边界未封
6. **增量折叠边界**：投影超预算 → 低频全量 re-fold（一次改写，哲学已接受）；`Manual/Overflow` 触发恒走全量（用户显式操作可接受前缀重写）
7. **SchemaV2 兼容**：旧 sidecar（无 version / V1）加载时校验兼容，写入统一 V2；V1 时代「投影内合并 user runs」语义已废弃（出站合并替代）

---

### 一句话总结 / One-line Summary

**回归**：我们从「128K 预算」走到「有界增量折叠 + SchemaV2 融合」，核心哲学（前缀稳定、canonical 不动）始终如一且被上游印证；**展望**：补上响应侧验证 UI 与显式压缩有界化，其余按「无需求不实施」；**社区**：296 PR 无人问津的池子里，我们的修复是 #8004/#7972 唯一真实出口（PR #8006，3 天窗口）；**边界**：两条硬纪律（前缀字节、canonical 只读）+ 三处已知边缘（小窗口、显式 fold 无界、估算口径）构成可防御的技术版图。

---

## 关联文档 / Related

- `docs/research/cache-aware-compaction-design.md`（压缩体系设计）
- `docs/research/compaction-evolution.md`（压缩演进）
- 记忆：[[prefix-stability-core-value]] · [[upstream-pr-unreliable-local-first]] · [[ccb-three-direction-analysis]]
