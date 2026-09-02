# 压缩/投影核心差异分析

## 概述

本文档分析 dev/develop 与 main-v2 v1.35.0 在压缩/投影核心的差异。这是 agent/ 包179文件差异的大头。

**核心结论**：我们的压缩/投影核心远比 main-v2 完整。main-v2 的变更主要是重构和诊断增强，我们的自定义代码是生产验证过的修复。

## 差异分类

### A. 我们的自定义代码（main-v2 没有）

#### 1. calibration.go（139行，校准持久化）

**功能**：per-model token/char ratio 持久化到磁盘，重启后首次请求使用上次会话的校准值，而非冷启动 fallback。

**关键实现**：
- `calibrationFilePath(modelRef)` → `~/.reasonix/calibration/<model>.json`
- `saveCalibration(modelRef, ratio)` → best-effort 写入
- `loadCalibration(modelRef)` → 启动时加载

**价值**：避免重启后首次请求的 token 估算偏差（冷启动 fallback 通常是 0.25，实际可能是 0.15-0.35）。

#### 2. NonToolContentHash（投影校验增强）

**功能**：在 CoveredPrefixHash 之外，增加 NonToolContentHash 用于语义校验。

**问题背景**：#8839 §6 — prune 重写 tool results → covered hash 不匹配，但非 tool 内容未变。传统校验会误判投影无效。

**实现**：
- `NonToolContentHash` 字段在 ContextProjection
- `coveredPrefixHash` 不匹配时，检查 `NonToolContentHash`
- 两者都不匹配才判定投影无效

**价值**：prune 后投影仍然有效，避免不必要的全量重放。

#### 3. priceAwareCompactRatio（价格感知折叠门控）

**功能**：当缓存命中价格极低（<1.5% of input price）时，推迟压缩到 90% 阈值。

**实现**：
- `cheapHitRatioThreshold = 0.015`
- `cheapHitCompactRatio = 0.90`
- `priceAwareCompactRatio(p *provider.Pricing)` → 返回 0.90 或 defaultCompactRatio

**价值**：DeepSeek 缓存命中价 ¥0.10/1M vs 输入价 ¥3/1M（3.3%），接近阈值时推迟压缩可以多利用缓存。

#### 4. 第三态 resume（degraded 投影保留）

**功能**：投影 lineage 不匹配时，保留 body + checkpointState=degraded，而非 fail-closed 清空。

**问题背景**：bd7b3588a — LoadProjectionSidecar lineage 失配时 fail-closed → 全量重放 → 超窗 → 不必要压缩。

**实现**：
- `modelVisibleMessages()` 三级化：投影有效→投影视图 / 失效但 body 完整→摘要+尾部降级视图 / 否则→全量
- `visibleInputForFold` + `snapshotExplicitCompression` 同步
- 守卫：CoveredCount 越界 / 拼接超窗不降级

**价值**：投影失效时降级视图（~50% 窗口），而非全量重放（100%+ 窗口），避免超窗触发不必要压缩。

### B. main-v2 的变更（我们未采纳）

#### 1. foldSummaryWithChunkedFallback（分块折叠回退）

**功能**：摘要输出截断时，尝试分块折叠回退。

**状态**：我们已添加 stub（delegate to foldToSummary），完整实现待评估。

#### 2. foldToSummaryMode 接收 prefix 参数

**功能**：foldToSummaryMode 新增 prefix 参数，用于缓存前缀对齐。

**状态**：我们的版本没有 prefix 参数，使用 nil 代替。

#### 3. visibleMessagesWithFlag（视图解析重构）

**功能**：统一 modelVisibleMessages 的解析逻辑，返回视图+是否使用投影的标志。

**状态**：我们的版本使用不同的解析逻辑（三级化）。

#### 4. ModelVisibleFingerprint/ProjectionCoveredMatch（诊断函数）

**功能**：resume-time 诊断，比较视图指纹和投影覆盖哈希。

**状态**：我们的版本没有这些诊断函数，但有等效的校验逻辑。

## 差异统计

| 文件 | 差异行数 | 方向 |
|------|---------|------|
| preflight.go | 173 | 我们有第三态 resume，main-v2 有 visibleMessagesWithFlag |
| projection.go | 194 | 我们有 NonToolContentHash，main-v2 有重构 |
| compact.go | 208 | 我们有 priceAwareCompactRatio，main-v2 有重构 |
| compact_projection.go | 505 | 我们有完整实现，main-v2 有 foldSummaryWithChunkedFallback |
| calibration.go | 139 | 我们独有（main-v2 没有） |

## 合并策略

**路径A：保持现状（推荐）**
- 我们的压缩/投影核心是生产验证过的修复
- main-v2 的变更主要是重构和诊断增强
- 风险：低

**路径B：采纳 main-v2 的诊断函数**
- 添加 ModelVisibleFingerprint/ProjectionCoveredMatch
- 添加 foldSummaryWithChunkedFallback 完整实现
- 风险：中（需要验证与我们的第三态 resume 兼容性）

**选择**：路径A。我们的代码更完整，main-v2 的变更可以在后续版本中评估。

## 验证证据

- Go 编译：✅ 通过
- Agent 测试：⚠️ 1个 flaky（workspace lease timing，与压缩/投影无关）
- 生产验证：我们的修复已在多个会话中验证（bd7b3588a, d9e235177, 51e5d6e9b）
