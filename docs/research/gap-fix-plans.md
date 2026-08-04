# 未闭环缺口修复方案分析（P0-P2，2026-08-04）

> 基于三家族深挖（deepdive-family{A,B,C}），六缺口逐一给出：
> 现状（代码位置）→ 修复方案 → 影响面 → 测试 → 风险。

## P0：#7451 配置层 fail-closed 拦截未知 OpenAI 端点

**现状**：
- `EffortCapabilityForEntry` default（effort.go:117-118）→ `EffortCapability{}`（不支持）
- `NormalizeEffort` default（effort.go:246-247）→ `effortNotConfigurableError`
- 协议层 generic（openai.go:206-224）→ 接受 low/medium/high（fail-open）
- **两层不对称**：TokenHub 等未知 OpenAI 兼容端点配置层拒配、协议层可用

**修复方案**（与协议层对齐）：
```
EffortCapabilityForEntry default 分支：
  if e != nil && e.Kind == "openai" && isGenericOpenAI(e) {
      return EffortCapability{Supported: true, Levels: ["auto","low","medium","high"], Default: "auto"}
  }
  return EffortCapability{}  // 其余保持 fail-closed（anthropic 已单独处理）

NormalizeEffort default 分支：同一判定——kind=openai 未知端点接受
  auto|low|medium|high（max 钳 high 由协议层处理）
```

**影响面**：未知 openai 端点（TokenHub/自托管 vLLM/第三方代理）effort 可配置；
已知 vendor（miniMax/zhipu 等 host 匹配）走各自分支不变；anthropic 不变。

**测试**：
- `TestEffortCapabilityForGenericOpenAIFallback`（未知 openai 端点 → 支持 low/medium/high）
- `TestNormalizeEffortGenericOpenAI`（auto/low/medium/high 通过；max 报错或钳制）
- `#7451 复现路径`（TokenHub 配置 + subagent profile effort → 不再报 not configurable）

**风险**：低——与协议层既有行为对齐（协议层早已 fail-open），配置层只是同步。

## P1a：StreamInterrupted 分支计费缺口（agent.go:2862-2864）

**现状**：ChunkError + IsStreamInterrupted → 直接返回原始 usage（无 best-effort/
request count）——与 ctx.Done 分支（2768-2772）不一致；usage 通常在流尾未到 →
emitTurnUsage(nil) 跳过 → **该次请求计费完全丢失**。

**修复方案**：
```
case provider.ChunkError:
    if provider.IsStreamInterrupted(chunk.Err) {
        stored, _ := finishReasoning()
        usage = bestEffortStreamUsage(usage, text.Len(), reasoning.Len(), "interrupted")
        usage = provider.UsageWithRequestAttemptCount(ctx, usage)
        return ..., usage, true, ..., chunk.Err
    }
    // 非 StreamInterrupted：清理重复调用（bestEffort + RequestCount 各调一次）
```

**影响面**：三 provider 的 StreamInterruptedError（openai.go:669、responses.go:767/771）
中断时计费不再丢失（best-effort + request count）；`true`（interrupted 标志）保留。

**测试**：`TestStreamInterruptedEmitsBestEffortUsage`（模拟 ChunkError+StreamInterrupted
→ 断言 usage 非 nil、Estimated=true、RequestCount>=1）

**风险**：低——与 ctx.Done 分支对齐；顺带清理重复调用残留（幂等无害）。

## P1b：best-effort 不估 prompt tokens（agent.go:2877-2904）

**现状**：只估 completion/reasoning（字节/4）；prompt tokens（计费大头 ~100k）
中断时无法估算 → 计费仍缺 prompt 部分。

**修复方案**（三选一）：
1. **provider 留存已收 usage**（推荐）：anthropic.go:665-667/responses.go:760-762
   中断前把已收 usage 随错误返回（haveUsage 时发 ChunkUsage）——最准确
2. agent 侧按 input 字节估算 prompt（粗略，Estimated 标记）——兜底
3. 组合：provider 留存为主 + agent 估算兜底

**影响面**：计费准确性（中断场景 prompt 部分从 0 → 真实/估算）
**测试**：中断场景断言 usage.PromptTokens 非零（或 Estimated 标记）
**风险**：中——方案 1 改三 provider 返回路径；方案 2 简单但估值粗糙。建议 1+2。

## P2a：vendor 逃生口补全（openai.go:160-205）

**现状**：minimax/zhipu/longcat/ollamaCloud 分支硬校验**无 hasExplicitSupportedEfforts
逃生口**（deepseek/generic 有）——#4099/#3561 残余。

**修复方案**：四个分支统一加 `hasExplicitSupportedEfforts` 白名单前置判断
（与 deepseek 分支 133-148 同模式）：用户声明 supported_efforts 时尊重声明，
未声明才用内置词汇校验。

**测试**：每 vendor 分支：声明词汇绕过 + 未声明仍校验 + 非法词汇报错
**风险**：低——模式已确立（#7273 修复同款）。

## P2b：#4814 残余（markPersisted 不查 Price，pricing.go:186-188）

**现状**：isStandardDeepSeekProviderTemplate 只查 APIKeyEnv/BalanceURL/ContextWindow，
**不查 Price**——用户"标准模板+只改价格"时 persistedOfficialCurrency 被清空 →
locale 自动刷新把 USD 刷成 CNY。

**修复方案**：isStandardDeepSeekProviderTemplate 增加 `p.Price == nil && len(p.Prices) == 0`
条件（用户改了价格 → 非标准模板 → provenance 保留）。

**测试**：标准模板+自定义 USD 价格 → locale 刷新不覆盖（USD 保留）
**风险**：低——守卫更严，只影响"官方模板+改价"组合（正是 #4814 场景）。

## P2c：全零+stop 完成语义解耦（responses.go:734-738）

**现状**：全零+stop 抑制 ChunkUsage → agent usage==nil → reasoningOnlyFinishHonoured
（agent.go:2656）false → reasoning-only 完成语义失效（与 #7168 宣称不符）。

**修复方案**（完成语义与计费数据彻底分离）：
```
方案 1（推荐）：抑制时仍发一个"完成语义专用"信号——ChunkUsage 不发，
   但 FinishReason 通过独立通道（如 ChunkDone/新字段）传递
方案 2：不抑制全零+stop（改为只抑制全零且无 finish reason）——让 agent
   收到 usage 对象（FinishReason=stop），计费层自行过滤零记录
```
方案 2 更简单：`usage.TotalTokens > 0 || usage.FinishReason != ""`（stop 也发）
——计费层（cost 计算）对全零自然 0 成本，不污染；完成语义恢复。

**测试**：全零+stop → agent 收到 usage（FinishReason=stop）→ reasoning-only
完成 honored；计费不出现虚假成本
**风险**：中——需确认计费层对零记录的过滤（cache-ratio 统计处）

## 实施顺序建议

```
P0（#7451）→ P1a（StreamInterrupted）→ P2a（逃生口）→ P2b（#4814）
→ P2c（完成语义）→ P1b（prompt 估算，改动最大最后做）
```


## 第二轮分析补充（2026-08-04 实施前验证）

### P0 影响面实测（现有测试无冲突）
- effort_test.go 现有 12 个测试全部是 vendor 特定断言（deepseek/miniMax/zhipu/
  longcat/ollamaCloud/glm-gateway）——**无"未知 openai 端点不支持"断言**
- isOllamaCloudEntry 的 localhost=false 是 vendor 判定（不受 default 分支影响）
- → P0 修复不破坏现有测试，只需新增 generic 回退测试

### P1a 下游确认（interrupted 标志独立）
- interrupted=true 消费在 2689-2693（恢复提示文案）——与 usage 修改无耦合
- → 修复 StreamInterrupted 分支的 usage 对齐不影响恢复路径

### P1b 三 provider 具体改法（留存已收 usage）
```
anthropic.go:665  if ctx.Err() != nil { return }  →  return 前：
                 if haveUsage { send(ChunkUsage{usage}) }  // mergeUsage 已收 input tokens
responses.go:760  同样——terminal 已到、usage 已收集时先发再 return
openai.go:1084    收到 usage 即发（993-1001）——中断时若 usage chunk 已到已发；
                  未到才丢（agent best-effort 兜底，可接受）
```

### P2c 计费层验证（方案 2 安全）
- Pricing.Cost（provider.go:715）：全零 usage（hit/miss/prompt/completion 全 0）
  → 返回 0 成本——**计费层天然过滤零记录**
- sessCacheHit/Miss 加 0 不影响 cache-ratio
- → 抑制条件放宽为 `TotalTokens > 0 || FinishReason != ""`（stop 也发）
  零记录不污染任何统计，完成语义恢复——方案 2 无副作用

### 缺口间关联
```
P2c（stop 也发）↔ #7168 zero-usage 语义：DashScope 全零场景会多一条
  ChunkUsage（FinishReason=stop）——agent 收到后 reasoningOnly 完成 honored
  ——这正是 #7168 评审"完成语义保留"的完整实现（原实现只做了一半）
P1a ↔ P1b：同一中断路径的两个面（agent 侧 best-effort + provider 侧留存）
  ——实施时应同批做，测试共享中断场景
