# T0 探查结论：P5 fork 三项前置（执行队探查 + 父代理复核）

> 事务：docs/team/20260810-p5-fork/ · 2026-08-10

## T0-A 父 Session 可达性 —— ✅ 已解决

主 agent Run 的 ctx 流经 executeOne 工具 ctx（execute_one.go `executeOne → prepareToolExecution → tool.Execute` 同一 ctx 链）。实现：
- `execute_one.go:678-679`：工具执行前 `cctx = WithForkSource(cctx, a)`（挂父 Agent 引用，仿 evidence.WithSessionMessages 模式）
- `task.go:1264`：fork 分支 `ForkSourceFromContext(ctx)` 取父 Agent → captureForkPrefix
- 结论：**可达性成立，无需额外通道**

## T0-B 首轮 compact 折叠 —— ✅ size guard 兜底

ContextManager.Prepare 会在采样前按 est 阈值折叠。fork 子代理预填大 session 理论上可能首轮前被折叠破前缀。
- 实现兜底：`checkForkSizeGuard`（父历史 ≤ 子上下文窗口 80%，`forkHistoryWindowRatio=0.8`）——超限直接拒绝 fork，从根上排除折叠风险（80% 通常低于 compact 触发线）
- 结论：**无需首轮豁免标志**；size guard 即兜底。若未来调整 compact 阈值需重审该比率

## T0-C tools 前缀敏感性 —— ✅ 收益量级确认

- cache_shape.go `PrefixShape` 含 system+tools+messages（PrefixHash 三者）
- 测试基座 `TestCacheHitPrefixStable` 断言：前缀字节稳定 → cache_hit_tokens 上升（mockDeepSeek 计算 commonPrefix）
- fork 子代理 tools = 父全量透传（forkReadOnlyGate 保 schema 原样、执行层只读）→ **ToolsHash 与父一致**，前缀不因 tools 差异打断
- 结论：**收益成立**——system/tools/messages 三者均与父一致，首请求命中父已建缓存

## 验证锚点（全部真实通过）

```
TestForkChildFirstRequestReusesParentCachePrefix   ✅ 首请求复用父前缀
TestCaptureForkPrefixByteIdentical                 ✅ 前缀字节一致
TestCaptureForkPrefixParentSessionUntouched        ✅ 父 Session 零改动
TestIsFreshSubagentSessionForkLock                 ✅ fork session 不 prepend startContext
TestForkReadOnlyGatePolicy/Execution               ✅ 只读 gate（writer 拒绝）
TestCacheHitPrefixStable                           ✅ 前缀稳定 → 命中上升
```
