# 梦境蒸馏（Dream Consolidation）设计方案

> 日期: 2026-08-08 · 分支: dev/clearnature
> 对照: claude-code-best `/dream`（autoDream/consolidationPrompt.ts 四阶段）
> 现状: Reasonix 有 remember 工具（模型主动写）+ 知识缓存（SaveKnowledge/EventChain）
>       但缺"主动整理/蒸馏"反向流程（跨会话沉淀 + 索引修剪）

## 一、差距（现状审计）

| 能力 | claude-code-best /dream | Reasonix 现状 | 差距 |
|---|---|---|---|
| 跨会话记忆整理（四阶段反思） | ✅ | ❌ 无 | **P1 新建 /dream** |
| 从 transcripts 蒸馏到持久记忆 | ✅ | ⚠️ remember 工具（模型主动） | **P2 蒸馏桥** |
| 知识缓存（语义检索跨会话） | ❌（无此层） | ✅ SaveKnowledge/EventChain | 已超集 |
| 索引维护（≤25KB/一行一钩子） | ✅ | ⚠️ MEMORY.md 有索引无修剪 | P4 补指引 |
| 矛盾消解 | ✅ Phase 4 | ❌ | P4 补指引 |
| 后台 auto-dream | ✅ | ❌ | 可选 P5 |

## 二、方案（P1+P2 组合，P4 指引，P5 可选）

### P1: `/dream` 命令（手动触发，最轻）

新增 slash 命令 `/dream`——注入四阶段 consolidation prompt：

```
## Phase 1 — Orient
- ls 记忆目录（project memory + global memory）
- 读 MEMORY.md 索引，浏览现有记忆文件

## Phase 2 — Gather
- 回顾最近会话摘要（本项目 sessions/ 最新文件）
- 检索知识缓存（EventChain 近期条目）
- 识别"值得沉淀的新事实"

## Phase 3 — Consolidate
- 用 remember 工具写入/更新记忆文件（复用现有格式）
- 合并近义条目（不建重复）
- 相对日期→绝对日期
- 修正被证伪的旧记忆（forget 工具）

## Phase 4 — Prune and index
- MEMORY.md 索引瘦身（每行 ≤150 字符一行一钩子）
- 移除过期/被取代的索引行
- 消解矛盾（两文件冲突时修正错的）
```

**实现**：新增 `internal/cli/slash_dream.go`（或复用 slash 命令注册机制）+ 记忆目录/会话目录路径注入 prompt。

### P2: A1 蒸馏桥（compaction 后自动沉淀）

compact 生成滚动摘要（A1）时，**同时把摘要写入知识缓存**：

```
compactToProjection 完成
  → 滚动摘要（A1 产物）已生成
  → 调 responses.SaveKnowledge(&KnowledgeEntry{
        Query:  摘要主题（首个 user turn 前 50 字）,
        Summary: 摘要正文,
        Tier:   "compaction-digest",
        ...})
  → EventChain 关联（与既有条目建边）
```

- 复用现有知识缓存（`internal/provider/responses/knowledge.go`）
- 收益：**跨会话语义恢复**——未来任何会话 L2 检索命中压缩摘要，即使 canonical 历史被压缩
- 风险：噪声（不是所有压缩都值得沉淀）→ 加门槛：仅当摘要包含"新事实"（与现有条目相似度 <0.6）才写入

### P4: 记忆索引指引（写入 /dream prompt + REASONIX.md）

- MEMORY.md 是索引不是 dump：每行 `- [Title](file.md) — 一行钩子`，≤150 字符
- 记忆文件 ≤25KB；超限拆分成主题文件
- 矛盾消解：新事实推翻旧记忆 → forget 旧的

### P5: 后台 auto-dream（可选）

- 会话 idle > 24h 恢复时（C1 窗口外路径已有）→ 提示"是否运行 /dream"
- 或每次 compaction 后自动跑轻量蒸馏（P2 已覆盖大部分）

## 三、缓存哲学校验（mental-seal）

| 改动 | 是否改变发送前缀 | 判定 |
|---|---|---|
| P1 /dream prompt | 仅手动调用时进入对话（非每轮） | ✅ |
| P2 蒸馏桥 | 写知识缓存（本地落盘），不进发送 | ✅ 零影响 |
| P4 指引 | 文档 | ✅ |

## 四、实施顺序

1. **P1 /dream 命令**（独立，cli 层）→ 验证: slash 命令注册 + prompt 注入
2. **P2 蒸馏桥**（agent/compact 完成后 hook）→ 验证: compact 后知识缓存出现 compaction-digest 条目
3. **P4 指引**（REASONIX.md + prompt）→ 验证: 文档
4. **P5 auto-dream**（可选）→ 验证: idle 恢复提示

## 五、与现有系统的关系

- **A1 滚动摘要**：会话内凝聚（已有）
- **知识缓存**：跨会话语义检索（已有，SaveKnowledge/EventChain）
- **记忆系统**：模型主动写（已有，remember/forget）
- **梦境蒸馏**：把三者串起来——A1 摘要 → 知识缓存沉淀 → /dream 手动整理 → 记忆文件（remember）→ MEMORY.md 索引
