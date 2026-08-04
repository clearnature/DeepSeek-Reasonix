# 深挖报告 C：计费/完成语义家族根因链（#7184/#7168/MiMo 截断）

## 核心结论
#7184 主体已修复（70097ea08 失败路径 usage 对齐），**残留 2 缺口**：
1. agent.go:2862-2864 StreamInterrupted 分支不对齐（无 best-effort/request count）
2. prompt tokens 不可估算（中断时计费大头丢失——三 provider 都丢已收 usage）

## usage 上报路径（三 provider）

| provider | 时机 | 中断行为 |
|---|---|---|
| openai（Chat） | 流中收到 usage 即发（993-1001） | ctx.Err → 丢失（1084-1086） |
| anthropic | 流结束后统一发（677-690） | 即使 haveUsage 也丢弃（665-667） |
| responses | 仅 terminal 事件时发（703-738） | terminal 后取消 → 丢失（760-762） |

## agent 侧（agent.go 2702-2904）

| 路径 | 行为 |
|---|---|
| ctx.Done 中断 | ✅ bestEffortStreamUsage（2768-2772）|
| StreamInterrupted chunk | ❌ 直接返回原始 usage（2862-2864）——**缺口 1** |
| best-effort 估算 | 只估 completion/reasoning，**不估 prompt**（2877-2904）——**缺口 2** |
| 失败路径 emitTurnUsage | ✅ run_loop.go:291 先行调用 |

## 完成语义与计费分离——架构成立、实现不完整

```
✅ 分离成立：FinishReason provider 合成（responses.go:711-724）
              zero-usage 抑制只控制 ChunkUsage 发射（734-738）
⚠️ 不完整：全零+stop 抑制 → agent usage==nil → reasoningOnlyFinishHonoured
          （agent.go:2656）返回 false → reasoning-only 完成语义失效
⚠️ 中断破坏分离：terminal 后、ChunkUsage 前取消 → 双双丢失
```

## MiMo 截断（da3aadd34）——已修复
max_output_tokens 65536 表驱动 + finishReasonMessage 截断警告闭环

## 引入提交时间线
```
7de6a2474 (05-29) include_usage 初始
faed6fc7d (#3377, 06-07) StreamInterruptedError
d3cfa5c26 (#6618, 07-18) reasoningOnlyFinishHonoured
26b256b5d (#7010, 07-28) run_loop 重构
70097ea08 (08-02) emitTurnUsage + bestEffortStreamUsage（#7184 主体修复）
874f5ae52/f76b81d96 (08-01/02) zero-usage 抑制 + stop 合成
da3aadd34 (08-03) MiMo 65536
```
