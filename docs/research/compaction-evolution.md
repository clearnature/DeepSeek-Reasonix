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

### 阶段 9：prune 延迟死循环修复（maybeCompact 高/force 之间不折叠）

- **症状**（用户实测 2026-08-09）：prompt 802k→837k（high=800k 与 force=864k 之间），**142 条请求、0 次压缩**持续数分钟；session 有 19 次 "pruned" notice 但从未 summarize。
- **根因**：`maybeCompact` 在 `!force` 时装完 prune 投影后**无条件 `return`**（"延迟到 force"）。当**大 user 内容主导**（tool 结果占比不足以把投影压到 high 以下）时，prune 投影 `projEst` 仍 ≥ high，但代码已返回——下一轮 preflight 见投影 `projEst ≥ high` 不放行、继续 prune 装投影，又 return → **死循环**。只有到 force（864k）才强制折叠，中间 64k 的死窗口。
- **修复**：prune 装投影后，用 `estimateMessagesTokens`（未校准、保守）估算 pruned 视图——**只有投影真的降到 high 以下才 return 延迟**；否则继续 `compactToProjection` 立即折叠。
- **回归测试**：`TestMaybeCompactFoldsWhenPruneCannotRelievePressure`（790k user + 20 条 stale tool → prune 后投影仍 ≥ high → 必须折叠）。
- **关联**：这是 `foldEconomics` 之外的第二个"折叠被跳过"路径——之前是 fold 太小不划算（400 token 门槛），现在是 prune 延迟错误判断"投影已降"。估算口径原则见 §四。

### 阶段 9b：fitFoldToWindow 条件反写导致 manual /compact 假成功

- **症状**（用户实测 2026-08-09）：`/compact` 显示 "compacted" 但 0 次 summarizer 调用、prompt 不降；stats 无 `mode=summarized` 记录。真实会话：canonical 估算 2.18M tokens（**assistant reasoning 818K 是估算大头**）、fold 区域 2812 条消息。
- **根因**：`fitFoldToWindow` 第二个边界循环条件写反——`foldTokens-remaining <= maxCompactFoldTokens`（已移走量 ≤ cap）在进入时必为 false（`folded=1.3M > 600K` 才进来），循环一路走到 `cut=0` **把 fold 全部移进 kept**；`planFold` 见 `len(fold)==0` → `CompactionNoop, nil`；`CompactNow` 忽略 outcome 返回 nil → "compacted" 假成功。
- **修复**：条件改为 `remaining <= maxCompactFoldTokens`（保留的 fold ≤ cap 即停）——fold 至少保留 cap 内的量，其余 defer 到后续轮次。
- **回归测试**：`TestFitFoldToWindowKeepsFoldBounded`（reasoning 45K × 40，minFold=1.14M > cap → fold 必须存活）。
- **数据链证据**：官方 #8024 CompactionBench 显示 fold 输入超窗时提供者拒绝；本 bug 是估算超窗时我们自己的裁剪逻辑先把它裁没了。

### 阶段 10：上游 #8019-#8031（08-09 同步进 dev）

上游 esengine 在同一时间线独立实现了折叠架构（与阶段 7-9b 平行），08-09 同步进 dev：

| PR | 内容 | 与本地实现的关系 |
|---|---|---|
| #8019 | fold partition 不变量（每组恰落一组）| 补充 |
| #8021 | **有界折叠**：`splitIntoSummarySpans` 输入截断/分片（fold 太大 → 分多片逐片摘要）| **替代**本地 fitFoldToWindow（机制不同：分片 vs 收缩） |
| #8024 | CompactionBench（cost/fidelity 双模式基准）| 新工具 |
| #8030 | 增量折叠（benchmark arm，**默认关闭**）：fold 上一投影而非 canonical 全量 | 与本地 d912be5ca 平行；上游做成可消融 arm |
| #8031 | **carried digests verbatim**：已有 digest 原样携带，summarizer 只看新工作；分区增 `carried` 组 | 修复 #8030 的 18/71 探针丢失；**前缀从"重写"变"append"**（缓存纪律加分） |

**#8031 关键指标**：71 探针 53→70 存活；fold 恒 1 call；投影前缀字节稳定（append 而非重写）。

### 阶段 11：#8057 吸收 #8006 核心（08-09 合入上游）

SivanCola 的 `fix/shared-window-output-budget` 明确是 **#8006（clearnature）的 consolidation**（Co-authored-by: clearnature）：

- **采用并适配**：provider 共享窗口能力检测（`SharedWindowInputPolicyProvider`）、普通+summarizer 请求输出预算裁剪（`effectiveOutputBudget(req)`）、CJK cold-start 保守尺寸、同会话 usage 校准（`promptTokenCalibration` + `requestCalibrationShape`）+ 跨会话重置。
- **审查但未采纳**：① 增量折叠改默认（保留 full-fold 默认）；② resume 时 provider 调用；③ pricing/cache 诊断（视为无关）。
- 新增 `internal/agent/output_budget.go`（293 行）+ `internal/provider/*/output_budget.go`。
- **#8004/#7972（HTTP 400）由上游正式关闭**。

### 阶段 12：dev 统一到 #8057 结构 + 本地保留（08-10）

dev 同步上游后，**预算/校准统一用 #8057 权威实现**：

- 删本地旧实现：`budget.go` 的 `sharedWindowClip`/`effectiveOutputBudget(msgs)`/`tokPerChar`/`msgChars`（被 #8057 的 `output_budget.go` 吸收）。
- **保留本地独有**（#8057 明确未采纳、上游没有）：
  - `MaybeCompactOnResume`（resume gate：model-visible 形状估算，真超窗才压，不误伤 warm 缓存）
  - `forceThreshold`（共享窗口 force 高水位钳制）
  - `maybePredictOverflow`（record-only 溢出预判 notice）
  - 静默退出遥测（`status=installed/noop/aborted` + 校准 src/proj）
  - prune deferral（[high, force) 死窗修复）
  - `/compress-fast`（no-AI 工具结果压缩命令族）
- 校准机制：`tokPerChar(lastUsage)` → 上游 `promptTokenCalibration`（promptTokens/requestChars 配对，`calibratedPromptTokens` 校准，跨会话重置）。

### 阶段 13：上游 #8109 架构分叉 + dev/上游 对比（08-10）

上游 `d541409e2 → 623760f43`（#8109 context-maintenance 事务化 + #8114 subagent-spec-split + #8126 context-command）后，**维护架构与 dev 分叉扩大**：

**上游 #8109 新增（dev 没有）**：
- `ContextManager`（context_manager.go）：维护路径从 `applyToolResultMaintenanceView` 迁入，`tryToolMaintenance` 每次请求前执行（est < fold → snip；≥ fold → prune）
- `compactionRunMu` 锁 + `installProjectionIfCurrent` 两阶段事务 + `Generation` 版本 + `LastReceipt` 幂等（`InputHash` 比对）——**投影-idempotent**（同样输入不重复维护，#7935 修复）
- **prune 改投影视图**（不再 `session.Rewrite` canonical）——**#8111 Bug 2 根因从源头消除**（方案不同：我们 `realignProjectionAfterRewrite` 重算哈希 vs 上游改掉 Rewrite）

**#8111 缺陷的解决状态**：
- Bug 2（prune 投影失效连锁）→ ✅ 上游 #8109 从源头修复（dev 的 realign 方案适用面收窄到 `/compress-fast` force 路径，同步上游时评估去留）
- Bug 1（模型切换误压）→ ❌ 上游未修（`projectionValid` 的 cacheKey gate 仍在 `623760f43`）；我们的 `bcbcecda7`（删 cacheKey gate）仍是唯一修复

**dev vs 上游对比（623760f43 基线）**：

| 维度 | 上游 main-v2 | dev/clearnature |
|---|---|---|
| 折叠机制（`splitIntoSummarySpans`/`carryPriorDigests`）| ✅ | ✅（已同步 #8031）|
| 预算校准（`output_budget.go`/`effectiveOutputBudget`）| ✅ | ✅（已同步 #8057）|
| 维护架构 | `ContextManager` 事务化 + `LastReceipt` 幂等 | `applyToolResultMaintenanceView` + `realignProjectionAfterRewrite` |
| resume gate（`MaybeCompactOnResume`）| ❌ | ✅（#8057 未采纳项）|
| 投影有效性 gate | ⚠️ cacheKey 硬条件（Bug 1 未修）| ✅ 只 gate `coveredPrefixHash` |
| 压缩遥测（stats 落盘/status/tpc）| ❌（#8112 在途）| ✅ 全套 |
| /compress-fast + warm 守卫（`cacheColdAfter`）| ❌ | ✅ |
| 请求前维护 warm 门控 | ❌（#8118 未满足）| ⚠️ 部分（/compress-fast 有，自动路径无）|

**dev 同步 33 commit 时的关键决策**：realign 方案去留（上游 prune 已不重写 canonical）；`MaybeCompactOnResume`/遥测/`/compress-fast` 是否保留本地独有；#8118 建议的 warm 守卫是否实现。

### 阶段 14：DeepSeek 缓存 TTL 实测——持久层证实，"5 分钟热层"证伪（08-10，数据实证 v2）

用官方用量（`/home/yanli/文档/deepseek/analyze_usage.py`）+ 本地请求级 stats 实测缓存行为：

**间隔-命中分桶（5 天合并 9503 对）**——初看支持"5 分钟边界"，但**子集验证证伪**：

| 间隔 | 对数 | 第二次命中率 |
|---|---|---|
| 0-2 分钟 | 9257 | 99.17% |
| 2-5 分钟 | 158 | 93.47% |
| 5-8 分钟 | 33 | 80.32% ← 看似骤降 |
| 8-15 分钟 | 29 | 85.50% |

**证伪判据（全 miss 点分析）**：纯 append 请求应命中旧前缀（hit ≈ 前一 prompt）——8/7 全部 9 个全 miss 点（间隔 230-1586s）增量仅 +1~8K 却 hit 0-15.8% → **前缀中间改变（工具结果重写/插入）→ 内容变化，不是 TTL 过期**。间隔分桶的"5 分钟下降"是长间隔更可能跨重大变化的**伪影**。

**已证实的硬结论**：
- **持久层保留天级**：8/7 05:07:48 重放 118 万前缀（3 天前发送）99.2% 命中
- **常规 append 命中旧前缀**：间隔<5min 99%（尾部新增工具结果命中——append-only 设计有效的证明）
- **写放大惩罚真实**：压缩/维护后首请求 0.3-15.9% miss（08-09 实测）
- **持久层按前缀大小分层**：小<50K 空闲 5-15min 仅 9.6% 命中（重放便宜只对大前缀成立）

**方案 B 落地（3472eb3df）**：stats usage 行加 `prefix_hash`（复用发送侧 CacheDiagnostics，零行为改变）——区分"缓存键变 vs 缓存过期"的唯一字段；`cacheColdAfter` 的 TTL 数值假设（24h/5min）都无直接证据，守卫语义应聚焦"别破坏当前活跃前缀"（写放大窗口）。

## 三、当前架构（阶段 12 后）

```
canonical transcript（只增不减，永久事实源）
    |
    +-- ContextProjection（sidecar：Messages + CoveredCount + CoveredPrefixHash）
    |     ├─ 全量折叠（默认，上游 #8021 有界：splitIntoSummarySpans 分片）
    |     │    fold = canonical[head : len-tail]（超窗 → 分片逐片摘要，永不 400）
    |     └─ 增量折叠（ablation arm，#8030/#8031：carried digests verbatim）
    |          旧 digest 原样携带 → 前缀 append 而非重写（缓存纪律）
    |
    ├─ 输出预算裁剪：#8057 effectiveOutputBudget(req)（普通 + summarizer 请求）
    │    est = calibratedPromptTokens(shape) → 输入 + 输出 ≤ window - reserve
    ├─ 触发：maybeCompact 每轮尾检查 usage ≥ high(0.8×window) 才压
    │    prune 延迟仅在投影真降 < high 时 return（阶段 9/12 修复）
    ├─ 请求前：preflight force 防线（forceThreshold 共享窗口钳制）
    └─ resume：MaybeCompactOnResume（真实口径，真超窗才压；本地保留）
```

**发送视图**：`projection.Messages + canonical[CoveredCount:]`（`modelVisibleFromProjection`）。
**校准**：`requestCalibrationShape`（requestChars/compactChars/cjkRunes）→ `promptTokenCalibration`（promptTokens/requestChars 配对，同会话稳定，跨会话重置）。

## 四、估算口径总原则

| 路径 | 估算 | 理由 |
|---|---|---|
| maybeCompact 触发 | `LatestPromptTokens`（#7930）| 真实单请求形状，重试累计会虚高 |
| 输出预算裁剪 | `calibratedPromptTokens(requestCalibrationShape)`（#8057）| 同会话 usage 校准，CJK 保守 floor |
| summarize fold 拒绝 | `effectiveOutputBudget(req)` + #8021 分片 | 输入+输出 ≤ 窗口，永不 400 |
| resume gate | `estimateMessagesTokens`（无 ×2，本地保留）| 宁可低估不误压 warm 缓存 |
| preflight force | `forceThreshold`（共享窗口钳制，本地保留）| 宁可保守不超窗 |

**原则**：每个路径按「宁可低估不误压」或「宁可保守不超窗」分别选择口径；校准统一由 `promptTokenCalibration`（#8057）提供，跨会话重置。

## 五、守卫测试契约（缓存纪律）

| 测试 | 断言 | 守护的缺陷 |
|---|---|---|
| `TestCacheHitSurvivesTooSmallWindow` | collapse ≤ 2、tail hit ≥ 85%、paused notice | 窗口太小仍逐轮压（打穿缓存）|
| `TestCompactionPausesWhenWindowTooSmall` | total ≤ 2、paused | 单条 tool 输出超阈值仍循环重压 |
| `TestCompactionHealthyWindowNeverLoops` | consecutive ≤ 1、paused=false | 健康窗口压缩后无呼吸空间 |
| `TestPruneKeepsToolHeavySessionBounded` | 会话有界 | 工具密集会话剪枝后仍超限 |
| `TestMaybeCompactFoldsWhenPruneCannotRelievePressure` | summarize 被调用 | high/force 之间 prune 延迟死循环（阶段 9）|

## 六、残余风险与后续

1. **CJK resume gate 低估**（已知）：resume gate 无 ×2 保守因子，CJK 会话真超窗 resume 可能不提前压——有兜底（preflight force 钳制 + #8057 effectiveOutputBudget 防 400），代价是 resume 后首轮抖动。可接受。
2. **canonical 磁盘增长**：只增不减（历史完整性代价），归档目录（`reasonix/archive/`）可追溯。
3. **本地独有功能未回上游**：#8057 明确未采纳（增量折叠默认/resume 调用/pricing-cache 诊断），本地保留：`MaybeCompactOnResume`/`forceThreshold`/`maybePredictOverflow`/遥测 status/prune deferral//compress-fast。如需贡献上游需另开 PR 单独论证。
4. **响应侧缓存验证**（CCB 分析唯一实质差距）：DeepSeek `prompt_cache_hit_tokens` 已在 Usage，只差判定逻辑（>5% 且 ≥2000 tokens 记录不告警）——待真实会话验证 token 语义后实施。
5. **#8030/#8031 增量 arm 默认关闭**：carried digests 保真（70/71）与前缀 append 是加分项，但上游保持 full-fold 默认——本地如需默认启用需在 ablation 层决策。

## 七、时间线

| 日期 | 事件 |
|---|---|
| 2026-08-04 | 用户定义核心定位：前缀字节稳定 = 成本优势之源 |
| 2026-08-07 | C1/A1/B2 设计定案（重放门控/摘要合并/位置固定）|
| 2026-08-08 | PR 7839（128K）→ 7913（动态预算）→ 36a06c341（summarize 裁剪）→ 4480758f8（有界拒绝）→ #7930（重试口径）→ d912be5ca（增量折叠）→ f90bf0ebc（对抗加固）|
| 2026-08-09 | 上游 #8019-#8031 同步（partition/bounded fold/bench/incremental arm/carried digests）；#8057 合入吸收 #8006 核心（Co-authored-by: clearnature），#8004/#7972 关闭；#8006 关闭 |
| 2026-08-10 | dev 统一到 #8057 结构（output_budget.go 权威实现 + calibration shape），本地保留 resume gate/overflow 预测/遥测 status/prune deferral//compress-fast；#8111 报告两个上游缺陷（模型切换误压 + prune 投影失效连锁），同日修复（bcbcecda7/286e6a886）+ 遥测体系（#8112 在途）；上游 #8109 合入（context maintenance 事务化 + 投影幂等，#7935 修复；prune 改投影视图从源头消除 #8111 Bug 2）|

---

关联：`docs/SPEC.md §3.6`、`docs/research/cache-aware-compaction-design.md`（阶段 6 设计）、`REASONIX.md`（mental-seal）。
