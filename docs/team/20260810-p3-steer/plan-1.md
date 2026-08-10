# 计划：P3 steer 通道——向运行中的后台 job/子代理注入消息（plan-1.md）

> 生成：2026-08-10 · 分支：team · 规划：Planner（仅计划，交由执行小队实施 + 纪律团审查）
> 关联：`docs/team-plan.md:33`（P3 定义：job 加输入队列，`send_message(task_id)` 子代理轮次边界消费）、`docs/MULTIAGENT_QWEN_COMPARISON.md:23-27`（Qwen sendMessage 背压 + waitForMessages 轮次边界）、`docs/team/20260810-p1-autodeliver/plan.md`（P1 定稿：锁序/bounded/信封模式参照）
> 目标：社区 #7962 第二大痛点——后台子代理运行期间用户/父代理无法中途指挥（转向输入排队/丢弃）。P3 建立 `jobs` 层消息通道 + 子代理 agent loop 轮次边界注入，复用主 agent 既有 steer 链（`MidTurnSteerPrefix`/`event.Steer`/`SteerText` 剥离），**消息只进子代理会话，父前缀零变化**。

---

## 一、拓扑扫描

### 1.1 现状锚点（grep/read 已验证）

| 位置 | 现状 | 与 P3 的关系 |
|------|------|-------------|
| `internal/agent/agent.go:873-1029` | **主 agent steer 全链**：`MidTurnSteerPrefix`（873）、`midTurnSteerMessage`（875）、`SteerText` 解析（892）、`Agent.Steer` 入队（933）、`consumeSteer`（951）、`flushSteerQueue`（982）、`RecordUnappliedSteer`（1006） | **P3 的语义模板**：消息=“运行中注入的指导，非新任务”。`SteerText`/`trimLeadingSteerWrapper` 已处理 `withTurnPreferences` 包装（语言块 + DeliveryRuntimeMarker）→ 子代理会话回放/继续时前缀可解析 |
| `internal/agent/run_loop.go:273-361` `runToolLoop` | 每 stream 轮顶部（277 for 循环内）：`consumeSteer` → `a.session.Add(user, a.withTurnPreferences(midTurnSteerMessage(text)))` + `a.sink.Emit(event.Steer)`（282-285），随后才 `a.tools.Schemas()`/`capturePrefixShape`/采样 | **P3 注入点**：在 consumeSteer 块后、采样准备前插入 job 消息消费，语义与 Qwen `waitForMessages`（轮次边界投递）一致；父/子代理共用此循环，父代理侧必须 no-op（见 R1） |
| `internal/agent/task.go:924-958` | 后台 task 的 job 闭包：`jm.StartForSession(..., func(jobCtx, _){...})` → `runSession(jobCtx, ...)`（950）→ `runSubSession`（1610）→ `RunSubAgentWithSession`（1792）→ `sub := New(...)`（1830）→ `sub.Run(ctx, prompt)`（1832） | **jobCtx 一路传到子代理 agent loop**，`jobCtxKey{}` 不丢失 → agent 层可从 ctx 消费 job 消息 |
| `internal/jobs/jobs.go:121-150` `Job` 结构 | `mu sync.Mutex` 保护流缓冲 + 终态字段（`tail/status/result/done/cancel` 等） | **新增 `pending []pendingMessage` 的宿主**（同锁保护） |
| `internal/jobs/jobs.go:460` | `StartForSession` 内 `ctx = context.WithValue(ctx, jobCtxKey{}, j)` —— **job 句柄注入 job 的 run goroutine ctx** | **P3 消费端的数据来源**：`jobs.ConsumeMessage(ctx)` 仿 `PublishEvidence`（2102-2110，同样从 ctx 读 `jobCtxKey{}` 的 `*Job`） |
| `internal/jobs/jobs.go:838-888` `recordCompletion` | P1 锁序模式：`m.get`（m.mu 段）取句柄 → 释放 → `j.mu` snapshot → 释放 → `m.mu` append → 释放；destroying 检查在 m.mu 段（858-861） | **P3 `JobMessage` 的锁序模板**：m.mu 段（destroying + find）→ j.mu 段（status + enqueue），**不嵌套** |
| `internal/jobs/jobs.go:548-557` | job goroutine 终态发布后、`close(j.done)` 前，`j.mu` 临界区写 `j.status` | **P3 清空 pending 的落点**（终态后、close 前，与 `done` 观察序一致） |
| `internal/control/controller.go:2339-2383` | `TrySteer`/`Steer`/`SteerConsumed`：主 agent steer 的 controller 入口（`exec.Steer(text)`） | **P3 用户入口的平行扩展**：`Controller.SteerTask(jobID, text)` |
| `internal/control/controller.go:2036` | `c.Jobs()` 返回会话 jobs manager | `SteerTask` 经它取 `*jobs.Manager` → `JobMessage` |
| `internal/event/event.go:88-92` | `event.Steer`：mid-turn steer 被消费并注入时触发，Text 带原始内容，前端据此显示“已投递” | P3 复用（子代理注入 = 指导已投递）；父 UI 经子代理嵌套 sink 可见 |
| `internal/control/input.go:191-195` | P1 父会话 `<background-jobs>` 信封注入 | **P3 不触碰**（父会话零改动，缓存红线） |
| `docs/team/20260810-p1-autodeliver/plan.md` | P1 定稿：信封/bounded（4096B×8 条×64 队）/只读 snapshot/锁序不嵌套 | P3 的 bounded/背压/锁序设计参照 |

### 1.2 级联风险清单

1. **R1 注入点作用域误伤（最高危）**：`runToolLoop` 是父/子共用。若消费点对父 agent 产生副作用（非 no-op），父会话字节漂移 → 破坏缓存红线。**对策**：`jobs.ConsumeMessage(ctx)` 只在 ctx 带 `jobCtxKey{}` 时消费；父 agent ctx 无该键（只有 `StartForSession` 闭包 ctx 有）→ 恒 no-op；T3/T5 用测试断言父 agent 消费点零副作用。
2. **R2 锁序死锁**：`JobMessage` 若持 m.mu 时碰 j.mu（或反向）会与既有 `recordCompletion`/`resolve`/`results` 路径形成死锁面。**对策**：严格 P1 模式——m.mu 段与 j.mu 段分离不嵌套；`ConsumeMessage` 只有 j.mu（job 指针来自 ctx，无 m.mu）→ 零死锁面；`-race` 验证。
3. **R3 job 生命周期竞态**：消息入队与完成/销毁竞态——job 刚 done、正在 Killed 收尾、session 正在 destroying 时入队。**对策**：`JobMessage` 做三层检查（m.mu destroying 检查 → j.mu status==Running 检查 → 限长入队），终态后拒绝；job goroutine 结束清空 pending。
4. **R4 背压失控**：无上限队列 → 内存淹没（Qwen `MAX_PENDING_MESSAGES` L570-577 教训：超限**抛错拒绝**而非静默丢弃——指令不可丢）。**对策**：每 job 限长 8 条 + 单条 ≤4KB，满则返回明确错误让调用方感知。
5. **R5 注入消息的会话污染/回放**：消息进入子代理会话历史，子代理被 `continue_from` 复用或子代理会话回放时，前缀必须可解析、不得泄漏为“用户原话”。**对策**：复用 `midTurnSteerMessage` 包装 → `SteerText`/`trimLeadingSteerWrapper` 全兼容；子代理会话 title/preview 派生已走 `SteerText` 剥离（preview.go:286-295）。
6. **R6 事件/UI 区分**：子代理注入复用 `event.Steer` 会让 UI 显示为“用户 steer 已投递”而非“父/用户发给子代理的消息”。**对策**：语义近似可接受（都是“运行中注入的指导”）；来源区分（user vs parent）标注为后续事务，不阻塞本事务。
7. **R7 入口缺失**：`JobMessage` 若无真实调用方则通道闲置。**对策**：T4 提供最小用户入口（`Controller.SteerTask` + 命令接线）；父代理 `send_message` 工具列可选（缓存代价 🟡，见第六节）。
8. **R8 缓存红线**：需求 4 要求“消息只进子代理会话，父前缀零变化”。P3 只改 `jobs.go` + `run_loop.go` 注入点 + `controller.go` 入口，**不触碰 `input.go`/父会话**；父 agent 消费点 no-op。唯一缓存注意项是可选 `send_message` 工具（父 tool schema 一次性变化）。

---

## 二、多路径推演

### 方案 A（推荐）：jobs 层 `pendingMessages` 队列 + `jobCtxKey` 消费 + runToolLoop 复用 steer 注入链

- **jobs**：`Job` 加 `pending []pendingMessage`（j.mu 保护）；`JobMessage(parentSession, id, text) error`（限长 8、单条 4KB、背压拒绝、终态/销毁拒绝）；`ConsumeMessage(ctx) (string, bool)`（从 ctx `jobCtxKey{}` 拿 `*Job`，j.mu pop FIFO）；job goroutine 终态发布后清空 pending。
- **agent**：`run_loop.go:282` consumeSteer 块之后插入 `jobs.ConsumeMessage(ctx)` → 命中则 `session.Add(user, withTurnPreferences(midTurnSteerMessage(text)))` + `emit(event.Steer)`——**与 steer 注入逐字节同构**。
- **入口**：`Controller.SteerTask(jobID, text)` 平行 `TrySteer`，经 `c.Jobs()` → `JobMessage`；命令接线为最小用户入口。
- **复杂度**：低-中（2 个核心文件 + controller 入口 + 测试）；**性能**：O(1) 队列入队/出队，无轮询；**可维护性**：复用 steer 全链（`SteerText`/preview/事件/`withTurnPreferences`），无新增剥离面；**风险**：低（R1-R4 均有明确对策，`-race` 覆盖）。
- **关键取舍**：注入格式**复用 `MidTurnSteerPrefix`**——子代理语境下“父/用户在运行中注入的指导”与“steer”语义同构，`SteerText` 回放解析、preview 剥离、事件渲染全部免费继承，且不必为 P3 新增任何剥离代码。

### 方案 B：独立前缀（如 `[Message from parent agent: …]`）+ 独立解析/剥离

- 语义更精确（区分“用户 steer”与“父代理消息”）、UI 可区分来源。
- **否决理由**：需新增前缀常量、`SteerText` 式解析、preview/剥离链接入（`<autoresearch-runtime>` 泄漏教训：新增包装必须进剥离链，遗漏即 UI 泄漏）；子代理会话 title 派生需单独处理；改动面大、收益仅限 UI 文案区分。来源区分可后续在**事件负载**上做，不必动前缀。

### 方案 C：jobs 层自注入（不改 agent loop）

- **否决**：jobs 包没有任何 agent loop 回调/钩子（`TaskRecorder` 只管 Start/Done 生命周期，无轮次边界事件）；注入必须发生在“工具轮次边界”，只有 `run_loop.go` 具备该位置。jobs 层自注入只能走 500ms 轮询（Qwen 模式，哲学红线：我方禁止 mid-turn 主动注入）——直接违反缓存纪律。

### 选型结论

**选 A**。唯一同时满足 ① 轮次边界注入 ② 复用 steer 链零新增剥离 ③ 背压/锁序与 P1 一致 ④ 父前缀零变化 ⑤ 改动面最小。B 死于新增剥离面，C 无注入点。

---

## 三、设计规格（A 方案细化）

### 3.1 jobs 层消息通道

```go
// Job 结构新增（j.mu 保护）
pending []pendingMessage

type pendingMessage struct {
    text string
    at   int64 // nowMs() 入队时间戳（调试/审计）
}

// 常量（集中定义，仿 P1 bounded 常量区）
const (
    maxPendingJobMessages    = 8    // 每 job 队列上限（Qwen MAX_PENDING_MESSAGES 同型）
    maxPendingMessageBytes   = 4096 // 单条消息上限（超限拒绝，防单条淹没）
)

// sentinel 错误（jobs 包需新增 import "errors"）
var (
    ErrJobUnknown            = errors.New("jobs: job not found")
    ErrJobNotRunning         = errors.New("jobs: job is not running")
    ErrJobMessageQueueFull   = errors.New("jobs: message queue is full")
    ErrJobMessageTooLong     = errors.New("jobs: message exceeds size limit")
)
```

**`func (m *Manager) JobMessage(parentSession, id, text string) error`**（锁序仿 `recordCompletion` 三/二段式）：
1. `m.mu` 段：`parentSession` destroying 检查 + `findJobLocked` 取 `*Job` → **释放 m.mu**（若 job 未知返回 `ErrJobUnknown`）；
2. `j.mu` 段：`j.status == Running` 检查（否则 `ErrJobNotRunning`）→ 单条长度检查（`ErrJobMessageTooLong`）→ `len(j.pending) < maxPendingJobMessages` 检查（否则 `ErrJobMessageQueueFull`）→ append `pendingMessage{text, nowMs()}` → **释放 j.mu**。
- **全程不嵌套 m.mu/j.mu**；destroying 检查语义与 P1（jobs.go:858-861）一致。
- 排队等 slot 的后台 job：`StartForSession` 创建时 status 即 `Running`（task.go:924 闭包在 goroutine 内 `acquireSlot` 排队），**排队期消息照常接受**（“指挥即将开始的任务”正确语义）。

**`func ConsumeMessage(ctx context.Context) (string, bool)`**（仿 `PublishEvidence` jobs.go:2102 的 ctx 取句柄模式）：
```go
j, _ := ctx.Value(jobCtxKey{}).(*Job)
if j == nil { return "", false }          // 父 agent/前台子代理 → no-op（R1 防线）
j.mu.Lock()
defer j.mu.Unlock()
if len(j.pending) == 0 { return "", false }
t := j.pending[0].text
j.pending = j.pending[1:]                 // FIFO
return t, true
```
- 只持 `j.mu`，无 `m.mu` 参与 → 零死锁面。
- 消费不改变 job 终态字段/`readOffset`/evidence——与 P1“只读 snapshot”哲学一致。

**job 结束清理**：job goroutine 终态发布后、`close(j.done)` 前（jobs.go:548-557 区间内），`j.mu` 段清空 `j.pending = nil`。语义：job 已完成，未消费消息无处可去，丢弃正确；调用方通过 `JobMessage` 返回值（入队成功与否）已获知投递结果，不额外通知。

### 3.2 agent 层注入点（需求 2）

`internal/agent/run_loop.go` `runToolLoop` 每轮顶部 consumeSteer 块之后（282-285 后）：

```go
// P3: 后台 job 消息通道 —— 子代理从 job 队列消费父侧注入的指导。
// 父 agent / 前台子代理的 ctx 无 jobCtxKey → ConsumeMessage 恒 false，零副作用。
if text, ok := jobs.ConsumeMessage(ctx); ok {
    a.session.Add(provider.Message{
        Role:    provider.RoleUser,
        Content: a.withTurnPreferences(midTurnSteerMessage(text)),
    })
    a.sink.Emit(event.Event{Kind: event.Steer, Text: text})
}
```

- **注入格式复用 `midTurnSteerMessage`**：模型看到“指导而非新任务”（与 Qwen/CCB 语义一致）；`SteerText`（含 `trimLeadingSteerWrapper`）对 `withTurnPreferences` 包装后的内容可完整解析 → 子代理会话回放/`continue_from`/title 派生零泄漏（R5 对策）。
- **时序**：消费点位于 `a.tools.Schemas()`（286）与采样准备之前 → 注入消息参与**本轮**模型请求（“下一轮”= 当前工具轮次之后紧邻的那次采样，Qwen `waitForMessages` 同义）。每注入一次子代理 cache miss 不可避免（run_loop.go:280-281 既有注释同款）。
- **作用域矩阵**：
  | ctx 来源 | `jobCtxKey` | 消费 |
  |---|---|---|
  | 父 agent（controller 提交） | 无 | no-op ✓ |
  | 前台子代理 / fleet / planner / guardian / review | 无 | no-op ✓ |
  | 后台 job 子代理（`StartForSession` 闭包，jobs.go:460 注入） | 有 | **消费注入** ✓ |

### 3.3 锁序与 bounded（需求 3，P1 一致）

| 约束 | 设计 |
|------|------|
| `j.mu`/`m.mu` 不嵌套 | `JobMessage` 两段式（m.mu 段释放后再进 j.mu 段）；`ConsumeMessage` 仅 j.mu |
| 背压上限防内存淹没 | 每 job 8 条 × 单条 4KB → 单 job 峰值 ≤32KB；job 结束清空 |
| 队列语义 | **拒绝最新**（`ErrJobMessageQueueFull`）而非丢最旧——指令不可静默丢弃，调用方明确感知后可重试（Qwen L570-577 同款） |
| 竞态防线 | destroying（m.mu）→ status==Running（j.mu）→ 限长（j.mu）三层检查；`done` 后消费端取不到（job 已清空） |

### 3.4 入口（最小用户接线，需求 1 的调用方）

`internal/control/controller.go` 平行 `TrySteer`（2339）新增：

```go
// SteerTask 向运行中的后台 job 注入一条指导消息；返回错误（未知/已结束/队列满）。
func (c *Controller) SteerTask(jobID, text string) error {
    // c.mu 取 c.jobs + c.parentSessionID → 释放；再调 jm.JobMessage(parentSession, jobID, text)
}
```

- 命令接线（slash，如 `/steer_task <job_id> <text>`）：注册位置由执行小队在 `internal/control/slash.go`/命令表定位（本计划不预写，避免行号漂移）。
- 父代理 `send_message(task_id, text)` 工具：**可选 T**——tool schema 新增会使父会话前缀一次性失效（🟡 缓存代价明确），且父模型自主发消息非 #7962 首要痛点；本事务先交付用户入口，工具留作后续（见第六节）。

---

## 四、任务分解（子任务 + 验证点）

- [ ] **T1 jobs 层消息通道**
      `Job` 加 `pending []pendingMessage`；常量 `maxPendingJobMessages`/`maxPendingMessageBytes`；sentinel 错误；`JobMessage`（两段式锁序 + 三层检查）；`ConsumeMessage`（ctx 取句柄 + j.mu pop）；`import "errors"`。
      → 验证：`go test ./internal/jobs/ -race`；新单测 `TestJobMessageFIFO`（入队顺序 = 消费顺序）、`TestJobMessageRejectsFinished`（done/非 Running 拒绝）、`TestJobMessageQueueFull`（9 条 → 第 9 条 `ErrJobMessageQueueFull`）、`TestJobMessageTooLong`、`TestJobMessageClearedOnFinish`（job 结束后 pending 清空）。
- [ ] **T2 job 生命周期清理**
      job goroutine 终态发布后、`close(j.done)` 前清空 pending（jobs.go:548-557 区间）；`JobMessage` 的 destroying 检查与 P1 语义一致。
      → 验证：`TestJobMessageDuringDestroy`（`BeginDestroySession` 后拒绝）；既有 `TestDrainMultiple -race` 仍绿（P1 时序不变量不受扰动）。
- [ ] **T3 agent 层注入点**
      `run_loop.go` consumeSteer 块后插入 `jobs.ConsumeMessage(ctx)` 注入（复用 `midTurnSteerMessage` + `event.Steer`）；确认父 agent / 前台子代理 ctx 无 `jobCtxKey` → no-op。
      → 验证：`go test ./internal/agent/ -race`；新单测 `TestBackgroundAgentConsumesJobMessage`（构造带 `jobCtxKey` 的 ctx + `Agent`，驱动 `runToolLoop` 一轮，断言 session 出现 `MidTurnSteerPrefix` user 消息、`SteerText` 可还原原文）；`TestParentAgentIgnoresJobMessages`（无 `jobCtxKey` 的 ctx → session 零变化）；既有 `TestSteerText`/`TestMidTurnSteerMessageRoundTrip`/`TestRunFlushesUnconsumedSteersOnCancel` 全绿。
- [ ] **T4 controller 用户入口**
      `Controller.SteerTask(jobID, text) error`（平行 `TrySteer`，经 `c.Jobs()` → `JobMessage`）；slash/命令接线（执行小队定位注册点）。
      → 验证：`go test ./internal/control/`；新单测 `TestSteerTaskAdmission`（未知 job / 已结束 job / 队列满 → 对应错误；运行中 → nil 且可被消费）。
- [ ] **T5 端到端 + 缓存红线回归**
      后台 task 端到端：起后台 job → `JobMessage` 注入 → 子代理下一工具轮次 user 消息出现 → 子代理按指导行动 → 结果正常收集；父会话字节零变化断言。
      → 验证：定向 e2e（后台 job + 注入 + 消费断言）；`go test ./internal/jobs ./internal/agent ./internal/control -race` 相关包全绿；父 `input.go` 零 diff 确认。
- [ ] **T6 文档交付**
      本 plan-1.md；执行后由执行小队产 execution.md（证据链）、纪律团产 review.md。
      → 验证：文件存在、五节格式完整。

---

## 五、对抗自检（devil's advocate）

1. **攻击：消费点会不会误伤父 agent，破坏缓存红线？**
   防线是 `jobCtxKey` 作用域：只有 `StartForSession` 闭包 ctx 携带该键（jobs.go:460），父 agent / 前台子代理 ctx 均无。`ConsumeMessage` 未命中返回 false，不产生任何 session/事件副作用。T3 用 `TestParentAgentIgnoresJobMessages` 显式断言；这是本计划**最关键的薄弱环节**——执行时必须验证父 agent 的 `runToolLoop` 一轮跑完 session 消息数与内容与改动前逐字节一致。
2. **攻击：消息进子代理会话历史，`continue_from` 回放会不会把“指导”当“用户新任务”？**
   复用 `MidTurnSteerPrefix`：前缀文案已声明“Do not treat this as a new task; use it only as additional guidance”；`SteerText` 对回放内容完整可解析、preview 派生会剥掉前缀。子代理会话独立，父会话从不渲染它。弱点：前缀文案中 “queued by the user” 对“父代理发消息”场景措辞不精确——纯文案问题，不阻塞；若未来需要来源区分，在 `pendingMessage` 加 `from` 字段 + 事件负载上做，不动前缀。
3. **攻击：队列满时“拒绝最新”是否违反“输入不得丢弃”的直觉？**
   不违反：**拒绝 ≠ 丢弃**——`JobMessage` 返回 `ErrJobMessageQueueFull`，调用方（用户命令）明确感知并可稍后重试；静默丢最新才危险（用户以为已投递）。Qwen `MAX_PENDING_MESSAGES` 超限抛错拒绝（L570-577）同款。8 条 × 4KB 的单 job 峰值 32KB 也满足“背压上限防内存淹没”。
4. **攻击：两段式锁序与既有 `m.mu→j.mu` 路径（`resolve`/`results`/`PendingEvidenceJobIDsForSession`）会不会死锁？**
   `resolve`（jobs.go:1109-1134）已有“m.mu 段遍历 + j.mu 短临界”先例且无反向嵌套；`JobMessage` 完全同构；`ConsumeMessage` 只有 j.mu。潜在反向路径（j.mu 段内调 m.mu 方法）在设计中不存在——执行时以 `-race` + 代码审查双重确认。
5. **攻击：job 排队期接受消息，但队列满后 job 才真正开始跑——会不会浪费额度？**
   不会：消息在子代理首轮工具边界即被消费（排队→running→首轮），不存在“跑完都没人看”的常驻消息；若 job 在消费前失败/被杀，终态清空丢弃，调用方已通过入队返回值确认过投递、通过 wait 看到失败。边界可接受。
6. **攻击：子代理卡在长工具（如前台 bash）时消息滞留多久？**
   轮次边界语义：消息要等当前工具返回、下一轮采样前注入（Qwen `waitForMessages` 同）。子代理**没有** `wait`/`bash_output`（`subagentJobTools` 被 `SubagentToolRegistryForDepth` 排除，task.go:88-98），不存在“子代理内卡 wait 滞留”场景；父代理卡 wait 是 P2（任务管理）范畴，本事务不承诺。
7. **攻击：P1 的信封/剥离链会不会被 P3 弄坏？**
   P3 不触碰 `completion` 队列、`DrainCompletedNoteForSession`、`input.go`、preview 剥离链——只新增 `Job.pending` 字段与两个新方法 + run_loop 消费点。P1 的 `recordCompletion` 时序不变量（入队→发布 status→close done）不受扰动；T2 回归 `TestDrainMultiple -race`。
8. **攻击：入口只有 controller 方法，模型/UI 层没接线，通道闲置？**
   本事务交付**通道 + 注入点 + controller API**（T4 含最小命令接线）；`send_message` 工具与 ACP/Desktop UI 接线列后续（缓存代价 🟡 明确标注）。通道先通、入口逐步扩展，避免一次事务同时触碰 tool schema 与 UI。

---

## 六、缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：**消息只进子代理会话**（需求 4 红线）——`JobMessage` 入子代理 job 队列、`runToolLoop` 注入 `sub.Session`；父会话（`input.go`/`compose`/system prompt/tools）**零改动**。父 agent 的 `runToolLoop` 消费点恒 no-op → 父请求字节与改动前一致。
- **前缀稳定**：子代理请求尾部新增注入 user 消息（带 `MidTurnSteerPrefix`）→ 子代理侧 cache miss（每次注入不可避免，run_loop.go:280-281 既有注释同款）；**父前缀零变化** ✓。
- **不插入历史**：注入仅发生在子代理当前 run 的工具轮次边界；不改写父历史；注入消息持久化在子代理会话但由 `SteerText` 在展示/回放层识别为指导而非用户新任务。
- **防虚假完成**：`JobMessage` 以 sentinel error 如实上报“未入队/已拒绝/队列满”，绝不静默吞掉用户指令；所有“已投递”声明须附测试证据（纪律团按 team-org.md 第六节交叉验证）。
- **bounded 审计**：`maxPendingJobMessages`/`maxPendingMessageBytes` 集中定义在 jobs 常量区，单测覆盖满队列/超长/终态拒绝边界。
- **⚠️ 唯一缓存注意项（可选 T）**：父代理 `send_message` 工具若落地，父 tool schema 新增 → 父会话**一次性**前缀失效（此后稳定）；本事务不执行该 T，仅标注，避免在缓存红线事务里混入 schema 变更。

---

## 七、遗留/假设

- 命令接线（slash `/steer_task`）的精确注册位置未在本计划预写（执行小队在 `internal/control/slash.go`/命令表定位后接线，T4 内完成）。
- 父代理 `send_message(task_id, text)` 工具、ACP/Desktop UI 入口：列后续事务（缓存代价 🟡 已在第六节标注）。
- 消息来源区分（user vs parent，Qwen `<teammate_message from="…">`）：本事务统一按“运行中注入的指导”处理；如需区分，在 `pendingMessage.from` + 事件负载上做，不动前缀（方案 B 否决理由）。
- 子代理完成前未消费的消息直接丢弃（终态清空）；若社区期望“丢弃前通知父侧”，另立事务。
- 后台 job 的 `status==Running` 判断覆盖“排队等 slot”场景（`StartForSession` 创建即 Running）；若未来引入独立 Queued 状态（P2 任务管理），`JobMessage` 的接受条件需同步扩展。
