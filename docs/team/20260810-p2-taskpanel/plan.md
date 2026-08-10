# 定稿：P2 任务管理面板（规划小组一致性裁决）

> 事务：docs/team/20260810-p2-taskpanel/ · 规划小组：plan-1/2/3.md（3 个 team-planner 独立产出）
> 一致性裁决：父代理汇总 → 共识采纳 + 分歧裁决

## 一、共识（3/3）

1. **jobs 只读快照 API**（核心增量）：新增 `JobSnapshotsForSession` 类型（id/kind/label/session/status/tail/stalled/interrupted + activity），**绝不消费 readOffset/resultRead/evidence lease**（HIGH 红线：面板若用 OutputForSession 会偷走模型输出，bash_output/wait 回归）
2. **stop**：走 `KillForSession`（session 过滤）——现状 `Controller.KillJob` 未按 session 过滤是缺口，`CancelJob` 收紧
3. **UI 载体**：增强现有 `TaskMonitorPanel`（复用列表/stop/5s 轮询/事件流），不新建面板
4. **进度**：复用 #7372 有界事件流（不新增 reasoning 通道、不转发 raw reasoning）
5. **/status**：`jobs <tag>` 行后追加任务明细段（向下兼容）
6. **stalled**：仅修饰 running（terminal 优先），复用 #4562 stalled notice 信号
7. **缓存红线**：零发送侧变化（纯读取/UI，不触 input.go/compose/system prompt/tools）

## 二、分歧裁决

| 分歧 | 方案 | 裁决 |
|------|------|------|
| 快照类型 | plan-1: View 扩展 / plan-2: 独立 Snapshot / plan-3: View 扩展+OutputSnapshot | 采纳 **独立 Snapshot 类型**（plan-2：避免 View 契约污染，Jobs() 状态栏契约零 diff 回归） |
| 双源合并 | plan-1/2: jobs > store 合并 / plan-3: 独立 jobs 面板 | 采纳 **增强 TaskMonitorPanel + jobs 优先合并**（jobs 命中以 jobs 为准，缺口回退 store） |
| ProgressID 映射 | plan-3: T0 前置探查（call id vs job id 是否同号） | 采纳 **T0 执行前探查**（影响进度关联） |
| tail 尺寸 | 4KiB（plan-1）/ 64B 行内（plan-2）/ 64KB bounded（plan-3） | 采纳 **快照 tail 4KiB rune-safe bounded + 前端截断渲染**（完整输出走 bash_output/wait） |

## 三、任务分解（执行小队）

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| T0 | 前置探查：ProgressID（call id）与 job.ID 是否同号 | task.go 探查 | 探查结论记录 |
| T1 | jobs 只读快照 API：`JobSnapshotsForSession`（含 tail 4KiB bounded、stalled/interrupted、非消费）+ 单测锁定「快照后 Output 全量」 | `internal/jobs/jobs.go` | `go test ./internal/jobs/ -race` |
| T2 | control：`JobSnapshots()` 包装 + `CancelJob` 收紧 `KillForSession` + 跨 session 拒绝测试 | `internal/control/controller.go` | `go test ./internal/control/` |
| T3 | desktop bridge：`JobPanelJobsForTab`/`JobOutputForTab`（仿 taskMonitorTargetForTab session 路由） | `desktop/app.go` | `go build ./...` + desktop 测试 |
| T4 | 前端：TaskMonitorPanel 增 kind badge/label/tail 行/stalled 高亮 + types.ts/bridge.ts/useController | `desktop/frontend/src/` | tsc + vitest |
| T5 | CLI /status 明细段（jobs 行后追加，tail ≤64B 单行） | `internal/cli/chat_tui.go` | `go test ./internal/cli/ -run Status` |
| T6 | e2e + 缓存门：grep 断言面板路径零 Output 调用、零 input.go 改动、零 event.Kind 新增 | 测试 + 纪律 | 定向 e2e + repolint |

## 四、缓存/纪律检查点

- 零发送侧变化（读取/UI）；不触 input.go/compose/system prompt/tools ✅
- 快照绝不消费 readOffset（T1 单测锁定）✅
- 不引入调度器（#3799 维护者拒绝）；queued 仅展示枚举 ✅
- 不转发 raw reasoning（#7372 原样保留）✅
- 锁序 m.mu→j.mu 不嵌套，-race 验证 ✅
