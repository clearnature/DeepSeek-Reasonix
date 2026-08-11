# P5 fork 机制：执行与审查报告

> 事务：docs/team/20260810-p5-fork/ · 分支：team · 状态：✅ 已执行（执行小队 + 父代理补完）

## 一、流程记录（按团队章程）

| 阶段 | 参与者 | 产出 | 结果 |
|------|--------|------|------|
| 规划 | 3 × team-planner（独立） | plan-1/2/3.md（正文在 preview，定稿综合） | 共识：截断继承 + fire-and-forget + 安全双轨 |
| 定稿 | 父代理 | plan.md | 裁决：task 参数 fork、截断继承、深度+大小 guard |
| 执行 | 5 × team-executor（T0-T6） | subagent_fork.go/fork_gate.go/静默信封/测试 | 实现落盘后**超时**（15m stalled） |
| 补执行 | 父代理（用户指示亲自执行） | 验证 + 修 3 个测试断言 + T0 结论 + repolint | 全部通过 |

## 二、父代理补完发现与修复

1. **执行小队超时**：fleet-14 写代码后 stalled（15m 无输出）——kill 释放工作区，父代理接手验证。执行队的**代码质量尚可**（编译通过、Fork 测试通过），但测试断言有 3 处问题：
   - `TestTaskToolSchemaExposesOnlyContinueFromForPersistence`：fork 参数描述含 "fork_from" 字样触发旧断言 → 改匹配 JSON 键形式 + 补 fork 参数存在断言
   - `TestStartSilentUnscopedAndNormalJobStillDelivers`：期望旧文本格式（P1 后是信封）→ 改断言信封含 normal job id + silent job 不泄漏
   - `TestGoldenBaselineNoExtensions`：schema 加 fork 后 golden 过期 → REASONIX_UPDATE_GOLDEN 重新生成

## 三、最终实现

- **T1 前缀构造**（subagent_fork.go）：`captureForkPrefix`（父 system 原样 + 历史截断去未完成轮次 + cloneForkMessages 深拷贝）——byte-identical 硬前提
- **T2 fork 分支**（task.go）：schema 加 `fork` bool（静态）+ 互斥校验（fork × continue_from/fork_from）+ `prepareTranscriptForkWithPrompt`（PrepareParentFork → 后台 job）
- **T3 安全双轨**（fork_gate.go）：`forkReadOnlyGate`（read-only 工具 + 只读 bash 放行，writer 拒绝）+ 递归 guard（depth ≤ maxSubagentDepth）+ 大小 guard（父历史 ≤ 窗口 80%）
- **T4 静默信封**（jobs.go）：`StartSilentForSession`/`silentCompletion`（fire-and-forget 不自动投递 P1 信封，wait/steer 可用）
- **T0-A 父 Session 可达性**：execute_one.go:679 `WithForkSource(cctx, a)`

## 四、验证（全部真实执行）

```
TestForkChildFirstRequestReusesParentCachePrefix  ✅ 首请求复用父前缀
TestCaptureForkPrefixByteIdentical                ✅ 前缀字节一致
TestCaptureForkPrefixParentSessionUntouched       ✅ 父 Session 零改动
TestIsFreshSubagentSessionForkLock                ✅ fork session 不 prepend
TestForkReadOnlyGatePolicy/Execution              ✅ 只读 gate
TestCacheHitPrefixStable                          ✅ 前缀稳定 → 命中上升
go build ./... + 全量 5 包测试                    ✅
go vet + gofmt + repolint                          ✅（功能增长入 baseline）
```

## 五、缓存红线确认

- fork 捕获零发送、父 Session 零改动（Snapshot 深拷贝、无 Add/Rewrite/Prepare）✅
- 子代理首请求前缀 byte-identical（T5-1 断言）→ 命中父已建缓存（T5-2 cache_hit_tokens）✅
- tools 与父一致（schema 全量透传，执行层 gate 只读）→ ToolsHash 不打断前缀 ✅
- fork 参数静态 bool 不进动态 schema ✅
- `isFreshSubagentSession` 对 fork 恒 false（不 prepend subagentStartContext）✅

## 六、遗留

- 发送视图变换链复制（plan-2 增强方案）未做——截断继承已满足 MVP
- fork-of-fork 到深度上限允许（递归 guard 控制）
- 真实 provider 收益需发布后 stats 验证（cache_hit 实测）
- 下一事务：P6（team 组织）或收尾
