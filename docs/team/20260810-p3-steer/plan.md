# 定稿：P3 steer 通道（规划小组一致性裁决）

> 事务：docs/team/20260810-p3-steer/ · 规划小组：plan-1/2/3.md（3 个 team-planner 独立产出）
> 一致性裁决：父代理汇总 → 共识采纳 + 分歧裁决

## 一、共识（3/3 一致）

1. **agent 层 steer 基础设施已存在**：`run_loop.go:282` 工具轮次边界消费点、`Agent.Steer`/`consumeSteer`/`MidTurnSteerPrefix`/`event.Steer`——P3 是**增量接线**，非从零建设
2. **注入点**：`run_loop.go` consumeSteer 块后，经 `jobCtxKey` 取 Job，`DrainPendingMessages` 拿消息 → 复用 consumeSteer 注入子代理 user 消息尾部（每轮 1 条，前缀缓存稳定）
3. **锁序**：`m.get`（m.mu）释放后 `j.mu` 短临界区，两锁不嵌套（复用 P1 recordCompletion 模式）
4. **关键防线**：只有后台 job 闭包 ctx 携带 `jobCtxKey` → 父 agent/前台子代理/planner 消费点恒 no-op
5. **范围限定**：`Kind=='task'`（fleet 不在 P3），消息只进子代理会话、父前缀零变化

## 二、分歧裁决

| 分歧 | 方案 | 裁决 |
|------|------|------|
| 队列上限 | plan-1: 8×4KB / plan-2: 50 / plan-3: 16×8KB | 采纳 **16 条/8KB**（居中；50 过大有内存风险、8 太小限制连续指挥；背压超限**拒绝**不静默丢弃，Qwen MAX_PENDING_MESSAGES 同款） |
| 触发路径 | plan-1: 仅 slash / plan-3: 工具+slash | 采纳 **双通道**：`send_message` 工具（父代理模型，bgjobs.go 注册模式，子代理隐藏）+ `/task-message <job_id> <text>` slash（用户） |
| 每轮注入条数 | plan-2 W1: 1 条保守 | 采纳 **每轮 1 条**（与 Agent.Steer 语义零偏差；消化多消息归增强项） |
| 未消费消息 | plan-3: Notice 计数 + 丢弃 | 采纳（持久化归 P6，本事务已知局限） |

## 三、任务分解（执行小队）

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| T1 | jobs 消息队列：`Job.pendingMessages`（16 条/8KB，j.mu）+ `Manager.SendMessageForSession`（status==Running 才收，超限拒绝）+ `DrainPendingMessages` | `internal/jobs/jobs.go` | `go test ./internal/jobs/ -race` |
| T2 | run-loop 注入：`run_loop.go` consumeSteer 后经 `jobCtxKey` 消费 → 注入 user 尾部（Kind=='task'，fleet 排除，父/前台 no-op） | `internal/agent/run_loop.go` | `go test ./internal/agent/ -run 'Steer|SubAgent|Task'` |
| T3 | `send_message` 工具注册（bgjobs.go 模式，父可用/子代理隐藏） | `internal/tool/builtin/` | `go test ./internal/tool/builtin/` |
| T4 | `/task-message` slash 命令 | `internal/control/` | `go test ./internal/control/` |
| T5 | e2e + 缓存回归 | 测试 | 定向 e2e + `go build ./...` |
| T6 | 文档 | `docs/team/…` | 文件存在 |

## 四、缓存/纪律检查点

- 消息只进子代理会话（user 尾部追加），父会话/父前缀零改动 ✅
- 每轮 1 条，注入位置固定（consumeSteer 既有路径），子代理前缀稳定 ✅
- 锁序 j.mu 单锁短临界区、不嵌套，`-race` 验证 ✅
- 背压拒绝不静默丢弃（防内存淹没）✅
