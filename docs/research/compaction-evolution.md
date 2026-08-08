# 上下文压缩体系：技术原理与演变史

> 日期：2026-08-08
> 状态：当前实现 + 完整演变记录
> 核心约束（mental-seal）：**前缀字节稳定 = 成本优势之源**——canonical transcript 永不改写，发送侧前缀只允许在「低频压缩」这一个点变化，压缩后必须稳定保持。

## 一、设计哲学（一切工作的锚点）

> 「多数 agent 每回合都为同一段不断增长的 prompt 付全价。Reasonix 让前缀逐字节稳定，让 DeepSeek 的缓存替你扛下来——其余一切由此而来。」（用户 2026-08-04）

- **append-only 历史**：canonical transcript（Session.Messages）是永久事实源，普通压缩/cold resume/tool prune 永不删除或改写它。
- **发送侧 = 投影视图**：压缩生成 model-visible context projection（sidecar `<session>.context.json`），canonical 保持完整（支持恢复/回退/分支/审计）。
- **前缀唯一改写点**：压缩本身（projection 重建）是唯一允许改变发送前缀的位置，且必须「一次性改写 + 改写后稳定保持」。
- **低成本恢复**：24h 缓存窗口内 resume 原样重放（命中 ¥0.32 vs 全价 ¥11.90），是便宜的知识恢复，不是赌博。

## 一.5 数学视角：投影不变量（Sovereign 证明库映射）

> 参考 `/data/work/discrete-mathematics/src/Sovereign`（`Projection/Binary.agda`、`Projection.agda`、`Geometry/ProjectiveInvariants.agda`）。context projection 的设计与四组形式化定理同构：

### 1. 有损投影定理 → canonical 是唯一事实源

```agda
-- Sovereign: 投影有损 — T₀ ≠ T₂ 但投影到相同 Bit（信息折叠不可逆）
projectionIsLossy : ¬ (T.T₀ ≡ T.T₂) × (projectTritToBit T.T₀ ≡ projectTritToBit T.T₂)
```

- 工程对应：canonical → 摘要投影是**有损且不可逆**的（不同消息可折叠进同一摘要）。
- 推论：canonical 永不改写 + `archive/` 归档 = 「有损投影必须保留源」的工程化；摘要仅存在于投影视图。

### 2. 上下文恢复定理 → 投影使用必须携带上下文（fail closed）

```agda
record Context : Set where
  field currentPhase : Fin 144
        wuxingMask   : Fin 5
restoreTritWithContext : Bit → Context → T.Trit    -- 恢复必须携带上下文
restoreT1Perfect : ∀ ctx → restore (project T₁) ctx ≡ T₁   -- 无损部分可完美恢复
```

- 工程对应：sidecar 元数据 `CoveredCount / CoveredPrefixHash / PromptCacheKey / SummaryHash` 即 Context。
- `projectionValid` **fail-closed**（缺 CoveredPrefixHash 或 lineage key 不匹配 → 丢弃重建）——数学上即「无上下文不投影」。
- `restoreT1Perfect`（无损恢复）↔ verbatim 保留（pinned/kept/tail）不依赖摘要质量；`restoreT0CorrectInNonEarthRegions`（有损部分依赖上下文）↔ SPEC §3.6「oversized message 里的事实依赖 summarizer 捕获」= best-effort。

### 3. 投影链条定理 → 增量折叠是链条的原子步

```agda
data ProjectionChain : Category → Category → Set where
  nil : ∀ {A} → ProjectionChain A A                  -- 恒等链 = 不动
  _∷_ : Step B C → ProjectionChain A B → ProjectionChain A C
chainAssoc : f ∘ (g ∘ h) ≡ (f ∘ g) ∘ h              -- 复合结合律
```

- 工程对应：每次增量折叠推进 `covered`（只增不减）= 链条的一个原子步；`nil` = 投影有效时 covered 内不重压。
- **链条不变量（covered 单调）**：canonical append-only + `coveredPrefixHash` 校验 ⇒ covered 严格单调，链条任意前缀良定义；折叠顺序（增量 vs 全量）不影响最终 covered 状态（结合律）。

### 4. 射影不变量 → 前缀字节是投影下的保持量

```agda
t-is-6624 : t ≡ 6624     -- 投影下保持的量（refl 验证）
```

- 工程对应：投影下必须保持的量 = 已覆盖前缀的字节。`coveredPrefixHash` 校验即「前缀不变量」的运行时证明——投影合法 ⟺ 前缀哈希匹配（与 `t-is-6624` 的 refl 验证同构）。

### 设计结论

实现完全符合 Sovereign 投影框架四原则（有损保留源、上下文校验、链条稳定、前缀不变量）——增量折叠正是「投影链条」的工程实现。

## 二、演变史（按阶段）

### 阶段 0：基线（32K 输出预算，append-only + 低频折叠）

```
canonical（只增不减）
  └─ 发送 = canonical 全量（stateless，前缀逐字节稳定）
  └─ 压缩触发：soft 0.5 报告 / snip 0.6 剪枝 / high 0.8 折叠 / force 0.9 强制
  └─ 压缩 = 唯一前缀改写点，改写后稳定
```

- 隐含假设：**canonical 永远小于窗口**（fold 从 canonical 全量中间取）。
- 会话不长时完美工作：缓存命中率高、压缩极少发生。
- 缺陷种子：fold 无界（无 covered_count 门控）——但 32K 时代死锁点远（~992K），几乎不触发。

### 阶段 1：PR 7839 — 默认输出预算 32K→128K（连锁反应起点）

- 改动：`max_output_tokens` 默认 128K（长任务能力提升，方向对）。
- **缺陷**：无配套共享窗口约束。DeepSeek 1M 窗口下 `input + max_output_tokens ≤ 1M`，128K 固定预算让 input 上限从 992K 提前到 917K。
- **连锁反应①**：prompt > 917K 时请求 HTTP 400 死锁；且压缩请求（summarize）也用 128K 预算 → 压缩本身也 400 → 压缩失败 → 会话继续增长 → 更死。

### 阶段 2：PR 7913（206d3372d）— 动态输出预算 + budget-aware resume gate

- `effectiveOutputBudget`：共享窗口 vendor（DeepSeek）下 `max_output_tokens = min(默认, window - est - reserve)`，输出随输入缩小。
- `MaybeCompactOnResume`：resume 超窗会话先压缩再发送。
- `forceThreshold = min(0.9×window, window - budget - 8192)`：force 阈值感知输出预算。
- 修复了正常请求路径的 400；**但漏了 compaction 路径**（summarize 仍用 128K）→ 死锁仍在（973K+131K=400）。

### 阶段 3：36a06c341 — summarize 输出预算裁剪

- `sharedWindowClip`：压缩请求（summarize）同样裁剪输出预算 → **修死锁根因**。
- 至此正常请求 + 压缩请求都不会 400。

### 阶段 4：4480758f8 — 有界拒绝 + 熔断

- `ErrCompactionInputTooLarge`：fold 估算 ≥ `window - minOutputBudget` 时**显式拒绝**（不发必败请求）。
- `compactStuck` 熔断：拒绝后暂停自动压缩（不每 turn 重试同一个必败请求）。
- 把「400 死锁」变成「明确拒绝」——正确保护，但暴露下一层：**fold 无界**（拒绝高频触发 = 压缩永久不可用）。

### 阶段 5：#7930（c8d7b09f5）— 重试累计口径修复

- `maybeCompact` 用 `Usage.PromptTokens`（重试累计的计费总量）判断阈值 → 两次真实 40% 的请求累计成 80% → **提前压缩**。
- 改用 `LatestPromptTokens()`（最近一次请求的真实形状，回退单次 legacy）。
- 修复「重试虚高提前压缩」，但**不覆盖 fold 超窗**（用户遇到的 `compaction input exceeds the provider context window` 是另一层）。

### 阶段 6：C1/A1/B2 — 缓存感知压缩（01528449a / fde7d8635）

- **C1 重放门控**（`maybeColdResumePrune`）：resume 按缓存 TTL 分路——窗口内原样重放（便宜），窗口外只记不压（deferred policy）。
- **A1 摘要滚动合并**：摘要链不无界累积（新摘要保留、旧摘要 fold 进下次 digest）。
- **B2 位置固定小 turns 窗口**：保留按「消息位置固定」而非「最新 N 条」——压缩后已保留部分字节不变。
- 三个机制都正确解决了「压缩后发送前缀字节稳定」；**共同缺陷：covered_count 只写不读**——投影记录覆盖点，但 `planCompaction` 从不使用 → fold 永远取 canonical 全量中间 → 会话长到 canonical 超窗（实测 src=2666458：canonical 270 万 vs 投影 29.7 万）→ 压缩永久不可用。

### 阶段 7：d912be5ca — 增量折叠根治（fold 有界）

- **增量折叠**：`compactToProjection` 先走 `tryIncrementalFold`→`incrementalFoldTarget`：
  - 条件：有效投影 + `0 < covered < len` + 投影 < 50% 窗口 + 非 manual/overflow + base 投影有呼吸空间。
  - fold = `canonical[covered : len-tail]`（只压上次覆盖后新增）→ **fold 有界**（每轮增长量）。
  - 新投影 = 旧投影.Messages + [摘要] + kept + tail → **旧投影字节不变 → 前缀命中**。
  - manual/overflow/超预算/边界 tool → 降级全量折叠（A1 摘要合并，低频前缀改写）。
- **resume gate 真实口径**：`MaybeCompactOnResume` 用 `estimateMessagesTokens`（不带 ×2 safety factor）——×2 让真实 ~49% 窗口的 warm 会话被误压（破坏缓存）。
- **cancel 显示修复**：`Controller.lastContextTokens` 记录最后非零足迹，`ContextSnapshot` 在 `LastUsage` nil 时回退；`bindExecutorProjection` 重置。
- **preflight 恒真放行移除**：`projEst < est` 是 tautology（投影永远小于 canonical），废除 force 防线 → 只留 `projEst < high`。

### 阶段 8：f90bf0ebc — 对抗审查加固

- **增量边界**：covered 处 tool result 属于 pre-covered turn → 降级全量（不回退折叠，避免重复内容）；head 回退循环 `head >= 0` 防护。
- **增量 partition**：`partitionFoldForProjectionIncremental` 禁用 fixed-early skip，否则新增小 user turns 从投影静默丢失。
- **stuck 投影保护**：`compactStuck` 只在投影有效时放行；无投影则 force guard 拒绝超窗发送。
- **增量后收敛**：`tryIncrementalFold` 增量后若投影仍 ≥ `high - tailBudget`（无呼吸空间）→ 递归全量折叠一次——否则每轮增长必超 high → 连续压缩（守卫测试 `TestCompactionHealthyWindowNeverLoops` 断言 consecutive ≤ 1）。

## 三、当前架构（阶段 8 后）

```
canonical transcript（只增不减，永久事实源）
    |
    +-- ContextProjection（sidecar：Messages + CoveredCount + CoveredPrefixHash）
    |     ├─ 全量折叠（首次/manual/overflow/超预算/边界 tool）
    |     │    fold = canonical[head : len-tail]，head = pinnedPrefixLen
    |     │    → 重建投影（A1 摘要合并）
    |     └─ 增量折叠（投影有效 + covered 中间 + 有呼吸空间）
    |          fold = canonical[covered : len-tail]  ← 有界
    |          → 旧投影.Messages + [摘要] + kept + tail（旧字节不变）
    |
    ├─ 触发：maybeCompact 每轮尾检查 usage ≥ high(0.8×window) 才压
    ├─ 请求前：preflight force 防线（est ≥ force → 压缩/拒绝）
    └─ resume：MaybeCompactOnResume（真实口径，真超窗才压）
```

**发送视图**：`projection.Messages + canonical[CoveredCount:]`（`modelVisibleFromProjection`）。

## 四、估算口径总原则

| 路径 | 估算 | 理由 |
|---|---|---|
| maybeCompact 触发 | `LatestPromptTokens`（#7930）| 真实单请求形状，重试累计会虚高 |
| summarize fold 拒绝 | `estimateMessagesTokens` | 真实含 framing/reasoning/tool-call |
| resume gate | `estimateMessagesTokens`（无 ×2）| 宁可低估不误压 warm 缓存 |
| preflight force | `estimatedPromptTokens`（×2 保守）| 宁可保守不超窗 |

**原则**：每个路径按「宁可低估不误压」或「宁可保守不超窗」分别选择口径。

## 五、守卫测试契约（缓存纪律）

| 测试 | 断言 | 守护的缺陷 |
|---|---|---|
| `TestCacheHitSurvivesTooSmallWindow` | collapse ≤ 2、tail hit ≥ 85%、paused notice | 窗口太小仍逐轮压（打穿缓存）|
| `TestCompactionPausesWhenWindowTooSmall` | total ≤ 2、paused | 单条 tool 输出超阈值仍循环重压 |
| `TestCompactionHealthyWindowNeverLoops` | consecutive ≤ 1、paused=false | 健康窗口压缩后无呼吸空间 |
| `TestPruneKeepsToolHeavySessionBounded` | 会话有界 | 工具密集会话剪枝后仍超限 |

## 六、残余风险与后续

1. **CJK resume gate 低估**（已知）：去 ×2 后 CJK 会话真超窗 resume 可能不提前压——有兜底（preflight force 用 ×2 拒绝/压缩，请求路径 sharedWindowClip 防 400），代价是 resume 后首轮抖动。可接受。
2. **canonical 磁盘增长**：只增不减（历史完整性代价），归档目录（`reasonix/archive/`）可追溯。
3. **上游同步**：main-v2 已含 C1/A1/B2（01528449a）但缺阶段 2-5/7-8 的修复——增量折叠方案可移植（diff 集中在 compact_projection.go/preflight.go/budget.go）。
4. **响应侧缓存验证**（CCB 分析唯一实质差距）：DeepSeek `prompt_cache_hit_tokens` 已在 Usage，只差判定逻辑（>5% 且 ≥2000 tokens 记录不告警）——待真实会话验证 token 语义后实施。

## 七、时间线

| 日期 | 事件 |
|---|---|
| 2026-08-04 | 用户定义核心定位：前缀字节稳定 = 成本优势之源 |
| 2026-08-07 | C1/A1/B2 设计定案（重放门控/摘要合并/位置固定）|
| 2026-08-08 | PR 7839（128K）→ 7913（动态预算）→ 36a06c341（summarize 裁剪）→ 4480758f8（有界拒绝）→ #7930（重试口径）→ d912be5ca（增量折叠）→ f90bf0ebc（对抗加固）|

---

关联：`docs/SPEC.md §3.6`、`docs/research/cache-aware-compaction-design.md`（阶段 6 设计）、`REASONIX.md`（mental-seal）。
