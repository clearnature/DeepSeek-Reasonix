# 定稿：P4 前台→后台动态降级（规划小组一致性裁决）

> 事务：docs/team/20260810-p4-backgroundize/ · 规划小组：plan-1/2/3.md（3 个 team-planner 独立产出）
> 一致性裁决：父代理汇总 → 共识采纳 + 分歧裁决

## 一、共识（plan-2/3 高度一致，plan-1 部分一致）

1. **核心机制（方案 A）**：前台 task 从第一轮即注册 `kind="task"` job；后台化 = 前台 run-loop 检查点返回 sentinel → **同一 goroutine 串行点交接**（MarkRunning → StartForSession）→ job 以 resume 模式**续跑同一内存 Session**（现场保留，零 signal-race）
2. **否决**：plan-1 的「优雅停止 + continue_from 重启」（引入 SubagentBackgroundized 新状态 + prompt 快照重放——复杂且重放有前缀风险）；CCB 的 `return() → isAsync:true` 重启（我们前台 task 嵌套在 executeBatch 内，不适用）
3. **job 归属**：复用 `StartForSession` + `kind="task"`，**否决新 kind**（`SendMessageForSession` 对 Kind!="task" 拒绝——P3 防线正好兼容）
4. **P1/P3 协同**：后台化 job 即普通 task job → P1 信封、P3 steer、wait 全部零改动生效
5. **缓存红线**：续跑同一内存 Session + 跳过 beginRunTurn 的 prompt add → 前缀逐字节稳定、0 额外 cache miss

## 二、分歧裁决

| 分歧 | 方案 | 裁决 |
|------|------|------|
| 切换机制 | plan-1: 状态持久化重启 / plan-2: 前台即 job select 三出口 / plan-3: 检查点 sentinel 交接 | 采纳 **plan-3**：runToolLoop 迭代边界检查点返回 `errBackgroundizeRequested` → 同 goroutine 串行点交接（无双跑） |
| 自动阈值 | plan-1: REASONIX_AUTO_BACKGROUND_TASKS / plan-3: REASONIX_AUTO_BACKGROUND_MS | 采纳 **plan-3**：`REASONIX_AUTO_BACKGROUND_MS` 默认 120000，0 禁用 + `ablation.AutoBackground` |
| 入口 | 三份一致：`/background` slash + Controller.Backgroundize | 采纳（ACP session_backgroundize 可选，UI 接线后续） |
| P1 信封抑制 | plan-2: foregroundClaimPending 标志 Start 预设 | 采纳（recordCompletion 先于 close(done)，抑制标志必须预设） |

## 三、任务分解（执行小队）

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| T1 | jobs 原语：`foregroundClaimPending` 标志（Start 预设，抑制 P1 信封）+ `ClaimForegroundResult`（复用 collectBackgroundEvidence 模式桥接证据） | `internal/jobs/jobs.go` | `go test ./internal/jobs/ -race` |
| T2 | runToolLoop 迭代边界检查点 sentinel（`errBackgroundizeRequested`，幂等单次）+ task.go 前台改造：交接串行点（前台完全 return 后 StartForSession）+ 续跑跳过 prompt add | `internal/agent/run_loop.go` `internal/agent/task.go` | `go test ./internal/agent/ -run 'Background|Task' -race` |
| T3 | Controller.Backgroundize + `/background` slash（参照 /task-message P3 落点）+ Notice + ForegroundTask 状态 | `internal/control/controller.go` | `go test ./internal/control/` |
| T4 | config `foreground_backgroundize_seconds` + env + ablation.AutoBackground + e2e（前台转后台 → P1 信封 + P3 steer 生效） | config/ablation + 测试 | `go build ./... && go test` |

## 四、前置探查（执行前必做）

- **T0**：父 turn 忙碌时 `/background` 到达路径（排队 or 直通）——plan-3 已给降级接线（紧急命令通道）
- B1：`SubagentRun.release` 是否解构 Session（影响 defer release 位置）
- B2：续跑首轮各 `session.Add` 落点无额外注入（run_loop.go 多处 add 点核查）

## 五、缓存/纪律检查点

- 续跑同一内存 Session + 跳过 prompt add → 前缀稳定、0 额外 cache miss ✅
- 交接串行点无双跑（前台 run loop 完全 return 后才 StartForSession）✅
- 禁用开关时字节级等价旧路径（ablation 兜底）✅
- 父 cancel 传导不能被信号分支短路（runCtx = WithCancel(ctx)）✅
