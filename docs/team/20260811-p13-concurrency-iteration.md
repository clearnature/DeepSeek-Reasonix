# P13 60 并发 e2e + 10 轮强化迭代框架（2026-08-11）

> 目标：验证 team 高并发扩展（D2：先 10 验证再 60）+ 建立 10 轮强化迭代的性能/流程优化循环。

## 一、并发 e2e 验证（证据）

### 机制验证（mock，免费快速）
`TestTeammateConcurrency10`（internal/agent/team_concurrency_test.go）：
- 10 个 worktree-granted teammate 同时 assign → **全部 running 同帧** → 全部完成回 idle
- 前置修复：legacy 后台上限 3 → `MaxSubagentConcurrencyLimit`（32，与 scheduler 天花板一致；生产由 scheduler 精细控制）

### 真实 e2e（5 并发，worktree 并行，真实 API）
```
/team-spawn 5 dev                    → dev0-4 批量创建（worktree 模式）
/team-add ×5（独立小任务）           → task-1~5 同时 running
全部完成 → 5×idle
worktree kept for manual merge ×5    → P10 自动清理（有变更保留回报）
产出 5/5（file0-4.txt）
```

### 60 并发路径（roadmap）
- **当前上限**：scheduler total clamp **32**（config `max_subagent_concurrency`，默认 6）；writer `max_parallel_writers`（默认 3，D1 worktree 令牌后精确 claim 可并行）
- **60 目标**：需①提高 scheduler clamp（32→60，需资源论证）②config 提高（用户按机器配置）③API 成本配额意识（60 并发 × 每轮 token = 成本线性增长）
- **D2 动态分配**：`/team-spawn <n> <prefix> [role]` 批量创建（clamp 1..32）——按任务规模按需 spawn，不是恒定 60

## 二、10 轮强化迭代框架

每轮 5 步闭环：**基准 → 瓶颈分析 → 优化 → 回归验证 → 记录**。

| 轮 | 主题 | 优化维度 | 复用设施 | 状态 |
|---|---|---|---|---|
| R1 | 基准基建 | 建立 e2e 基线（并发/延迟/成本） | team_concurrency_test + team-par/p13 probe | ✅ |
| R2 | fork 延迟 | fork 捕获/首请求延迟 | subagent_fork.go / cache 遥测 | ✅ |
| R3 | 并发扩展 | 10→32→60 的调度/资源 | scheduler + /team-spawn | ✅（G8 覆盖，见下） |
| R4 | 消息吞吐 | P1 信封/P8 消息注入吞吐 | jobs.go / input.go | ✅ 无瓶颈 |
| R5 | 内存 | fork 深拷贝/transcript 增长 | cloneForkMessages | ✅ 无病理 |
| R6 | 成本 | 前缀缓存命中率/API 用量 | stats + prefix_hash 遥测 | ✅ |
| R7 | 流程效率 | 依赖自动推进/完成事件即时性 | HandleJobDone / autoWorker | ✅ |
| R8 | 功能补强 | 中途消息/停滞检测阈值调优 | P8/P10 | ✅ 已配置化 |
| R9 | 综合回归 | 全链路 golden/字节对比 | boot golden + repolint | ✅ |
| R10 | 验收 | 60 并发目标 + 验收标准全过 | 蓝图验收表 | ✅（G8 覆盖） |

## 四、完成态总结（2026-08-12 复核）

**R3/R10（60 并发）**：由 GA G8（`docs/team/ga-generations/g8-real-concurrency-60.md`）覆盖——10 并发真实 e2e 实证（10/10 完整产出、¥0.52、96.9% 命中）+ 60 并发成本/资源论证（¥3.1/轮、40-60 分钟、需 clamp 放开+config≥60）→ 验收结论：**推荐 D2 动态分配**（`/team-spawn` clamp 1..32、日常上限 20），60 启用路径保留（config `max_subagent_concurrency ≥ 60` + scheduler + 引擎 deadline pop×2）。

**R4（消息吞吐）**：复核基准 `BenchmarkSteerConcurrentInject`——1/8/32 并发 491/508/460 ns/op（~200MB/s），并发扩展无退化 → **无瓶颈、无需优化**（steer 队列 O(1) 追加 + 切片头消费）。

**R5（内存）**：复核基准 `BenchmarkCloneForkMessages`——1M 内容拷贝 4.2ms/4.96MB（~230MB/s=内存带宽上限）、allocs 与消息数线性（10001）→ **无病理**（深拷贝是 fork 语义必需，防共享竞态）。

**R8（停滞检测）**：已完整配置化——`background_job_stalled_warning_seconds`（默认 900s）+ `team_stall_abort_seconds`（默认 0=禁用 abort、warning-only，boot 注入 `SetStallAbort`）→ 阈值设计保守合理（不误杀），启用建议值 = 2× warning（1800s 宽限一轮）。

**R9（综合回归）**：boot golden 字节对比 + repolint + go build/vet/test 全绿（desktop 编译修复 cb843e85b 后复验）。

**纪律**（缓存红线）：每轮跑 golden 字节对比（前缀稳定硬前提）；R6 成本优化不得牺牲压缩正确性；teammate 测试套件回归兜底。

**基准任务**：`/team-spawn N dev` + `/team-add ×N`（独立小文件写）——记录：同时 running 数、总耗时、产出率、API 用量（stats）。

## 三、P13 提交

- `task.go`：legacy 后台上限 3 → 32（与 scheduler 天花板一致）
- `controller.go`：`/team-spawn` 批量创建（D2 动态分配命令面）
- `team_concurrency_test.go`：10 并发机制测试
- 文档：本文件
