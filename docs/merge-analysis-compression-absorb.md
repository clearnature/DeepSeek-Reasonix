# 压缩/投影核心语义吸收分析

## 概述

本文档分析 main-v2 v1.35.0 在压缩/投影核心的变更中，哪些语义值得吸收。

**核心原则**：不能全面无条件合并，需要逐个分析语义价值。

## 值得吸收的语义

### 1. foldSummaryWithChunkedFallback（分块折叠回退）

**语义**：摘要输出截断时，尝试分块折叠。

**实现**：
```go
func (a *Agent) foldSummaryWithChunkedFallback(ctx context.Context, trigger string, fold []provider.Message, instructions string, sourceTokens int, inputMode string) (foldSummary, CompactionTelemetry, error) {
    res, tele, err := a.foldSummaryWithTelemetry(ctx, trigger, fold, instructions, sourceTokens, inputMode)
    if err == nil || (!errors.Is(err, errSummaryOutputTruncated) && !errors.Is(err, ErrCompactionRequired)) {
        return res, tele, err
    }
    chunked, chunkedErr := a.chunkedFoldSummary(ctx, fold, instructions, nil)
    // ... 合并 usage/spans ...
    return chunked, tele, nil
}
```

**价值**：
- 长会话摘要截断是常见问题
- 分块折叠是好的降级策略
- 独立函数，不影响现有逻辑

**风险**：低

**建议**：吸收，但需要适配我们的 `foldToSummary` 签名。

### 2. compact.go 的 summaryRequest 重构

**语义**：统一 summary 请求构建逻辑。

**变更**：
- 移除 `summaryRequest` 函数（独立的请求构建器）
- 将逻辑内联到 `summarize` 函数

**价值**：
- 减少重复代码
- 提高可维护性

**风险**：低（重构，不改变行为）

**建议**：吸收，但需要适配我们的实现。

### 3. projection.go 的字段清理

**语义**：简化投影结构，增强校验。

**变更**：
- 移除 `SummaryHash` 字段
- 移除 `SourceTokens` 字段
- 移除 `ProjectionTokens` 字段
- 移除 `NativeContextEditingAccepted` 字段
- 移除 `ContextEditingFallbackLocal` 字段
- 移除 `UpdatedAt` 字段
- 添加 `NonToolContentHash` 字段（我们已有）

**价值**：
- 简化投影结构
- 增强校验（NonToolContentHash）

**风险**：中（需要验证与我们的校验逻辑兼容）

**建议**：部分吸收（保留 NonToolContentHash，移除旧字段）。

### 4. preflight.go 的 modelVisibleMessages 重构

**语义**：统一视图解析逻辑。

**变更**：
- 移除 `visibleMessagesWithFlag` 函数（我们独有）
- 移除 `replayProjectionView` 函数（我们独有）
- 移除 `ModelVisibleFingerprint` 函数（我们独有）
- 移除 `ProjectionCoveredMatch` 函数（我们独有）

**价值**：
- 简化视图解析逻辑

**风险**：高（我们的函数是第三态 resume 的核心）

**建议**：不吸收（我们的实现更完整）。

## 不吸收的语义

### 1. 第三态 resume（我们独有）

**语义**：投影 lineage 不匹配时，保留 body + checkpointState=degraded。

**价值**：避免投影失效时全量重放。

**风险**：高（核心功能，不能删除）

**建议**：保留。

### 2. calibration.go（我们独有）

**语义**：per-model token/char ratio 持久化。

**价值**：避免重启后首次请求的 token 估算偏差。

**风险**：低（独立文件，不影响其他逻辑）

**建议**：保留。

### 3. priceAwareCompactRatio（我们独有）

**语义**：当缓存命中价格极低时，推迟压缩。

**价值**：多利用缓存。

**风险**：低（独立函数，不影响其他逻辑）

**建议**：保留。

### 4. NonToolContentHash（我们独有）

**语义**：在 CoveredPrefixHash 之外，增加 NonToolContentHash 用于语义校验。

**价值**：prune 后投影仍然有效。

**风险**：低（增强校验，不影响其他逻辑）

**建议**：保留。

## 吸收策略

### 路径A：逐个吸收（推荐）

1. 吸收 `foldSummaryWithChunkedFallback`（分块折叠回退）
2. 吸收 `summaryRequest` 重构（统一请求构建）
3. 部分吸收 `projection.go` 字段清理（保留 NonToolContentHash）

**风险**：低
**工作量**：中（需要适配我们的实现）

### 路径B：保持现状

不吸收任何变更，保持我们的实现。

**风险**：低
**工作量**：无

**选择**：路径A。分块折叠回退是好的语义，值得吸收。

## 验证证据

- Go 编译：✅ 通过
- Agent 测试：⚠️ 1个 flaky（workspace lease timing，与压缩/投影无关）
- 生产验证：我们的修复已在多个会话中验证（bd7b3588a, d9e235177, 51e5d6e9b）
