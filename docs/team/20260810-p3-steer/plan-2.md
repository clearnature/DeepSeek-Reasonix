# 计划（独立视角）：P3 steer 通道——向运行中的后台 job/子代理注入消息

> 事务：docs/team/20260810-p3-steer/ · 本文件：plan-2.md（team-planner 独立产出，供父代理汇总一致性裁决）
> 背景：社区 #7962——后台子代理运行期间无法中途指挥。参考：Qwen send_message(task_id) 子代理工具轮次边界投递、CCB queuePendingMessage。
> 前置事实（已拓扑验证，非假设）：**agent 层 steer 基础设施已在主分支完整存在**（`Agent.Steer`/`consumeSteer`/`flushSteerQueue`/`closeSteerIntakeIfIdle`/`MidTurnSteerPrefix`/`SteerText`/`event.Steer`/`NoticeCodeUnappliedSteer`），P3 是**增量接线**而非从零建设。

---

## 一、拓扑扫描

### 1.1 已存在的事实（grep/read 实证）

| 组件 | 位置 | 现状 |
|------|------|------|
| steer 注入消费点 | `internal/agent/run_loop.go:275-285` `runToolLoop` 每轮迭代开头 `consumeSteer()` → `session.Add(RoleUser, withTurnPreferences(midTurnSteerMessage(text)))` + `event.Steer` | 已有：每轮边界注入、追加在 session 尾部、逐条消费（每条一次 cache miss，前缀稳定） |
| steer 队列/边界 | `internal/agent/agent.go:446-458`（`steerMu`/`steerQueue`/`steerRunActive`）、`:933-1029`（`Steer`/`consumeSteer`/`closeSteerIntakeIfIdle`/`flushSteerQueue`/`RecordUnappliedSteer`）、`:1398-1401`（Run 开始置 active） | 已有：run 活跃才接受、退出 flush 为 local-only + warn notice |
| 后台子代理链路 | `internal/agent/task.go:924` `StartForSession(session,"task",…)` → `:863 runSession` → `:1610 runSubSession` → `:1792 RunSubAgentWithSession` → `:1830 sub := New(...)` → `:1832 sub.Run` → run_loop.go runToolLoop | 已有：job 闭包 ctx 带 `jobCtxKey{}`（`internal/jobs/jobs.go:460`）与 Manager（`task.go:878` 实证 `jobs.FromContext(ctx)` 可用） |
| jobs 锁序基准 | `internal/jobs/jobs.go:121-150` Job（`j.mu`）、`:153-182` Manager（`m.mu`）、`:838-888` `recordCompletion`（`j.mu` 只读快照 → 释放 → `m.mu` append，不嵌套）、`:924-943` `get`（`m.mu` → `findJobLocked`，即 m.mu→j.mu） | 已有：P1 定稿锁序 |
| 后台作业工具范式 | `internal/tool/builtin/bgjobs.go` `bash_output`/`kill_shell`/`wait`（`jobs.FromContext(ctx)` + `jobs.SessionFromContext(ctx)` + `ProviderVisible` 门控） | 已有：send_message 工具直接复刻此范式 |
| fleet 后台 job | `internal/agent/fleet.go:247` `StartForSession(session,"fleet",…)` → `runFleet` → 子代理 | 已有：多 child 共享一个 job 队列，**本事务明确不支持 fleet steer**（见 1.2 级联风险 R4） |
| Options 依赖方向 | `internal/agent/agent.go:430,1114` `Jobs *jobs.Manager` | 已有：agent 包已依赖 jobs 类型（单向 jobs←agent，jobs 不依赖 agent，注入必须经函数字段而非 jobs 引用 agent） |

### 1.2 缺口与级联风险

- **缺口 G1**：jobs 包无任何 pendingMessage 队列（grep `pendingMessage|queuePending|SendMessage` 仅命中 bot 包，无关）→ 需要新增 `Job.pending []string` + 队列 API + 收尾 flush。
- **缺口 G2**：Agent 无"外部消息源"钩子——`consumeSteer` 只读本地 `steerQueue` → 需要新增 `Options.SteerSource func() []string` + 每轮拉取接线。
- **缺口 G3**：无 `send_message` 工具。
- **级联风险 R1（锁序）**：新队列若在 `j.mu` 临界区内调 `Agent.Steer`（steerMu）会形成 j.mu→steerMu 嵌套；只要不存在反向 steerMu→j.mu 获取即无死锁，但纪律上**强制"临界区内只拷贝、临界区外投递"**（与 P1 recordCompletion 同款纪律）。
- **级联风险 R2（消费滞后/诚实性）**：队列接受 ≠ 子代理实际应用。子代理已近完成时 drain 出的消息会落入 agent 本地队列，经既有 `flushSteerQueue→RecordUnappliedSteer` 变成 local-only + warn notice（诚实）；jobs 侧残留（从未被 drain）需在 job 收尾独立 flush + notice。
- **级联风险 R3（缓存红线）**：注入必须走既有 `withTurnPreferences(midTurnSteerMessage(text))` 尾部追加路径，严禁插历史/重写前缀；每条 steer 一次 cache miss 是不可避免且已被接受的既有语义。
- **级联风险 R4（fleet 多 child 抢食）**：fleet job 的多个 child 若共享同一 `Job.pending`，消息会被任一 child 抢走——本事务**限定 `j.Kind == "task"` 才注入 steerSource**，fleet 明确拒绝（send_message 工具对 kind≠task 报错）。
- **级联风险 R5（job 生命周期竞态）**：`runReturned` 置位（jobs.go:489-491）早于 `status` 终态（jobs.go:550），Queue 判定必须用 `status==Running && !runReturned` 双条件才能线性化"job 已收尾即拒绝"。
- **级联风险 R6（崩溃丢失）**：`pending` 是内存态，进程崩溃/重启即丢——steer 是时序性指令，接受（任务恢复由既有 recovery 机制负责），标注 unresolved。

---

## 二、多路径推演

### 方案 A（采纳）：jobs 持久队列 + `Agent.steerSource` 回调 + `send_message` 工具

- **链路**：`send_message(task_id,message)`（父代理 turn）→ `Manager.QueuePendingMessageForSession(session,task_id,text)` → `Job.pending`（`j.mu`，限长 50 超限拒绝，`status==Running && !runReturned` 才接受）→ 后台子代理 `runToolLoop` 每轮迭代开头 `pullSteerSource()` 经 `Job.DrainPendingMessages()` 一次拿走全部 → 入本地 `steerQueue` → 既有 `consumeSteer` 逐条注入为尾部 user 消息。
- **复杂度**：中。增量三处：jobs 队列（~80 行 + 测试）、agent 钩子（~15 行）、task.go 注入（~10 行）+ 工具（~60 行）。
- **性能**：每轮一次 drain（无锁争用热点，`j.mu` 短临界区）；每注入一条 steer 一次 cache miss（既有语义）。
- **可维护性**：复用全部既有 steer 语义（flush/notice/replay/前缀），jobs 不依赖 agent（回调 `func() []string` 方向正确），与 bash_output/wait 工具范式同构。
- **风险**：低；唯一新概念是"每轮 drain 全部入本地队列再逐条消费"。

### 方案 B（否决）：jobs 持有 per-job `steer func(string) bool` 回调，send_message 直调 `Agent.Steer`，无队列

- **复杂度**：低（~40 行）。但**致命缺陷**：`Agent.Steer` 仅在 `steerRunActive` 时接受（agent.go:936），子代理处于 job 排队等待 slot、两次 Run 之间、或 Run 已退出时投递即丢；无背压/限长（Agent.Steer 无上限，无法满足"防单 agent 淹没"）；jobs 包需持有 agent 侧函数引用，仍要回调注册（收益消失）而丢失"轮次边界投递"的排队缓冲语义。

### 方案 C（否决）：job goroutine 内轮询 goroutine 调 `Agent.Steer`

- **复杂度**：中。但子代理 agent 创建于 `RunSubAgentWithSession:1830` 内部，job 闭包拿不到引用；需回调注册（退化为 B 的耦合），且轮询有延迟/空转、与 `flushSteerQueue` 竞态窗口更难闭合。

### 裁决

采纳 **A**。理由：唯一同时满足（1）消息 API 形态可被父代理工具与用户（CLI/UI 复用 jobs 接口）双触发、（2）50 条限长背压、（3）复用既有轮次边界注入与 flush 诚实性、（4）锁序零新增嵌套、（5）jobs←agent 单向依赖的方案。

### 关键设计决策（执行队必须遵守）

| # | 决策 | 理由 |
|---|------|------|
| D1 | 队列挂 `Job.pending`（`j.mu`），不挂 Manager | 随 job 生命周期，无需手动清理 map；`m.mu` 零改动 |
| D2 | 上限 `maxPendingMessages = 50`，超限**拒绝**（返回原因，不覆盖不截断） | 任务要求防单 agent 淹没 |
| D3 | Queue 判定 `status == Running && !runReturned` | 收尾窗口（jobs.go:491-557）内 runReturned 已置位，双条件线性化拒绝 |
| D4 | `DrainPendingMessages` 在 `j.mu` 临界区**拷贝+清空**，返回后释放锁再入 `steerQueue`（steerMu） | 两个临界区分离，零嵌套，`-race` 可证 |
| D5 | 每轮消费 1 条 steer（既有 `consumeSteer` 语义不变），50 条最多拉长 50 轮 | 保持事件/注入语义一致；父代理可再补发 |
| D6 | `steerSource` 注入点放 `RunSubAgentWithSession`（task.go:1792），条件 `JobFromContext(ctx)` 命中且 `j.Kind == "task"` | 一处接线，task/fleet 统一受益点；kind 限定排除 fleet 多 child 抢食 |
| D7 | send_message 工具层校验 kind∈{task}，fleet/bash 拒绝 | jobs 队列保持通用（future-proof），工具层表达产品语义 |
| D8 | job 收尾 flush 残留：`j.mu` 内拷贝清空 → 释放 → `NoticeCodeUnappliedSteer` warn notice（文本含数量） | 诚实性：父代理看到"已排队"但任务已结束，必须被告知未应用 |
| D9 | 队列内存态，不持久化 | steer 是时序指令；崩溃恢复走既有 recovery |

---

## 三、任务分解

- [ ] **T1 jobs 队列（jobs 包）** → 验证: `go test ./internal/jobs/ -race`
  - `Job` 增 `pending []string`（`j.mu` 保护，jobs.go:121-150 字段区）；常量 `maxPendingMessages = 50`（`const` 块 jobs.go:186-195 旁）。
  - `func (m *Manager) QueuePendingMessageForSession(parentSession, id, text string) (bool, string)`：`m.get(parentSession,id)` → `j.mu` 下校验 `status==Running && !runReturned`、`len(pending)<50` → append；返回 `(false, 原因)` 覆盖：job 不存在 / job 已结束 / 队列满(50)。文案：`no background job %q` / `background job %q is no longer running` / `message queue for %q is full (50 pending)`。
  - `func (j *Job) DrainPendingMessages() []string`：`j.mu` 临界区内 `msgs := j.pending; j.pending = nil`，返回（临界区外无操作）。
  - job 收尾 flush（jobs.go:489-491 `runReturned=true` 之后、`recordCompletion` 前后均可）：`j.mu` 拷贝清空残留 → 释放 → 若非空 emit `event.Notice{Level:Warn, Code:event.NoticeCodeUnappliedSteer, Text:"background %s %s ended with %d queued steering message(s) that were not applied"}`。
  - 导出 `func JobFromContext(ctx context.Context) (*Job, bool)`（jobs.go:2062 ctxKey 区域，读 `jobCtxKey{}`）。
  - 新增单测（jobs_test.go 或新 steer_queue_test.go）：入队/满拒/已结束拒/未知 id 拒/并发 Queue+Drain（-race）/收尾 flush emit/锁序（-race 全绿）。
  - 回归：`DrainCompletedNote`、`recordCompletion`、`Output`/`Wait` 既有测试全绿（证明 pending 零侵入 P1 envelope）。

- [ ] **T2 agent steerSource 钩子（agent 包）** → 验证: `go test ./internal/agent/ -race -run 'Steer'`
  - `Options` 增 `SteerSource func() []string`（agent.go:1113 附近，注释：nil 表示无外部来源）。
  - Agent 字段区（agent.go:446-458 旁）增 `steerSource func() []string`，`New` 中 `opts.SteerSource` 透传。
  - 新增 `func (a *Agent) pullSteerSource()`：`if a.steerSource == nil { return }` → `msgs := a.steerSource()`（无锁，jobs 侧自加锁）→ 空则返回 → `steerMu.Lock` append 全部 + `steerConsumed=false` → Unlock。**严禁在 steerMu 内调 steerSource**。
  - run_loop.go:282 `consumeSteer()` 之前一行调用 `a.pullSteerSource()`（每轮迭代开头 = 轮次边界）。
  - 单测：注入假 source（返回 N 条）→ 断言逐轮注入 1 条、session 尾部出现 `MidTurnSteerPrefix` user 消息、`event.Steer` 逐条 emit、Run 退出后未消费残留走 `RecordUnappliedSteer`（复用 steer_flush_test 模式）。

- [ ] **T3 task.go 接线（子代理注入点）** → 验证: `go test ./internal/agent/ -race -run 'Task|Subagent|Steer'`
  - `RunSubAgentWithSession`（task.go:1792）内、`sub := New(...)`（:1830）之前：`if opts.SteerSource == nil { if j, ok := jobs.JobFromContext(ctx); ok && j.Kind == "task" { opts.SteerSource = j.DrainPendingMessages } }`。
  - 前台子代理/无 job ctx 不受影响（JobFromContext 不命中）；fleet 子代理不命中（Kind 限定）。
  - 单测：后台 task job 内 mock provider 子代理多轮，验证第二轮请求消息列表尾部含 `MidTurnSteerPrefix`；fleet job 内子代理不注入。

- [ ] **T4 send_message 工具（builtin 包）** → 验证: `go test ./internal/tool/builtin/ -run 'SendMessage|Bgjob'`
  - 新文件 `internal/tool/builtin/send_message.go`（或并入 bgjobs.go，执行队自决）：`type sendMessageTool struct{}`，`init(){tool.RegisterBuiltin(...)}`，`Name()=="send_message"`。
  - Schema：`{task_id: string 必填, message: string 必填}`；`ReadOnly()==true`（不改文件系统）；`ProviderVisible` 同 bash_output（`jobs.FromContext(ctx)` ok 才可见）。
  - Execute：`jm, ok := jobs.FromContext(ctx)`；`jobs.SessionFromContext(ctx)` 限定本 session；`j, ok := jm.JobForSession(session, task_id)`（复用 get 或新增只读查询）→ kind != "task" 报错 `background job %q is a %s job and cannot receive steering messages` → `QueuePendingMessageForSession` → 成功返回 `Queued message for background task %q. It is delivered as guidance on the task's next step; it is applied only while the task is still running.`，失败返回原因。
  - Description 明确："Send a mid-turn guidance message to a running background task (task_id from task run_in_background). The task model receives it as guidance, not a new task. Rejected when the task already finished or its message queue is full (50)."
  - 单测：成功入队 / 未知 id / bash job 拒绝 / fleet 拒绝 / 队列满拒绝。

- [ ] **T5 端到端（集成）** → 验证: `go test ./internal/agent/ -race -run SteerE2E`
  - 新 `steer_e2e_test.go`（agent 包）：mock provider 驱动父代理 turn 调 `send_message`（父 registry 注册该工具）→ `jm.QueuePendingMessageForSession` → 后台 task job 子代理 mock 的**下一轮 API 请求消息列表**断言含 `MidTurnSteerPrefix` 且位于尾部；随后子代理 mock 返回体现新指导的工具调用/回答。
  - 覆盖：子代理完成后 send_message 被拒；队列满被拒；-race 并发（父 turn 投递 vs 子代理 drain）。

- [ ] **T6 文档** → 验证: `docs/team/20260810-p3-steer/` 下 plan.md 定稿 + execution.md 存在
  - 汇总 3 份 team 方案裁决，记录本事务状态、API 形态定稿、锁序说明。

---

## 四、对抗自检（devil's advocate）

1. **攻击：drain 全部拿走 vs 每轮一条的矛盾——50 条消息是否可能让子代理在 50 轮 API 后才完成，成本爆炸？**
   回应：`maxPendingMessages=50` 是背压上限而非常规负载；D1-D5 允许执行队后续把"每轮注入条数"提为可调常量（如 3-5 条/轮，一次 cache miss 注入多条 user 消息，模型一轮消化）。本方案保持 1 条/轮与既有 Agent.Steer 语义零偏差，属保守正确。**薄弱环节 W1**：每轮 1 条的成本上限未在 P3 内做自适应，标注为后续增强。
2. **攻击：Queue 成功返回"已排队"，但子代理可能立即完成导致未应用——父代理被误导。**
   回应：D8 收尾 flush + `NoticeCodeUnappliedSteer` warn notice 保证诚实；send_message 返回文案已含"applied only while the task is still running"。**薄弱环节 W2**：父代理 turn 结束前 notice 可能晚到，父代理本轮只见"queued"——可接受（与 P1 的 drain 语义一致）。
3. **攻击：steerSource 在 steerMu 外调用，但 `pullSteerSource` 与 `flushSteerQueue` 并发（Run 退出竞态）时，source drain 出的消息在 flush 之后才入队 → 残留。**
   回应：job 收尾的 D8 flush 兜底 jobs 侧残留；agent 本地队列残留走既有 `flushSteerQueue`。两条路径都收敛到"unapplied 通知"，无丢失。**薄弱环节 W3**：存在极窄窗口 jobs 队列被 drain、agent 已 closeSteerIntakeIfIdle、消息入队后不再消费直到 flush——仍需收尾 flush 覆盖，执行队需在 T1 单测覆盖"drain 后立即结束"竞态。
4. **攻击：`JobFromContext` 泄漏 job 对象，runSubSession 若误用在父代理 ctx（无 jobCtxKey）会 nil。**
   回应：签名返回 `(*Job, bool)`，不命中返回 `nil,false`，T3 接线只在 ok 时使用；`New` 的 Agent 不持有 Job 引用（仅回调闭包），无生命周期外泄。
5. **攻击：P3 触碰了 P1 的 envelope/completion 吗？**
   回应：`pending` 是 Job 新增独立字段，`recordCompletion`/`recordStalled`/`DrainCompletedNote`/`ResultSnapshotForSession` 零改动；T1 回归测试显式验证。**零交集**。
6. **攻击：锁定顺序真的无新增嵌套吗？**
   回应：Queue=`m.mu`(get) 短临界 → `j.mu`；Drain=`j.mu` 单独；flush=`j.mu` 单独 + 释放后 emit；pullSteerSource=无锁 source 调用 → `steerMu`。不存在 m.mu→j.mu→steerMu 或反向 steerMu→j.mu→m.mu 的持锁链。-race 全量必跑。**薄弱环节 W4**：若执行队为省事把 emit 放 `j.mu` 内（emit 可能回调 sink → 未知锁），违反纪律——在 T1 验收里显式检查"emit 在临界区外"。
7. **攻击：send_message 工具能否被子代理自身调用（子代理给兄弟子代理发消息）？**
   回应：子代理 ctx 带 Manager 时工具可见，语义上允许（同 session 后台任务互指挥），属能力而非缺陷；但 `ProviderVisible` 与 bash_output 完全一致，行为可预期。**薄弱环节 W5**：工具描述未禁止子代理间互发，若产品上要禁止需在 Execute 检查 `SubagentDepth>0` 拒绝——标注为可选收口，本方案保持宽松。
8. **攻击：50 条上限按 job 计，一个会话多个 job 各 50 条，总内存无界？**
   回应：每条是短字符串、并发后台 task 有既有 `maxConcurrentBackgroundTasks` 上限，总量有界。接受。

---

## 五、缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：子代理下一次 API 请求**尾部**新增一条 user 消息（`withTurnPreferences(midTurnSteerMessage(text))`）。前缀（system + canonical 历史 + 既有消息）**零变化**；每条 steer 一次不可避免的 cache miss，之后缓存恢复——与既有 `Agent.Steer` 完全同语义，无新增缓存破坏面。
- **前缀稳定**：不插入历史中间、不重写 canonical、`ComposeSynthetic`/preview/strip 路径零改动；replay 经既有 `SteerText` 识别（含 turn-preferences 包裹，已有测试覆盖）。
- **锁序**：全部新临界区为 `j.mu` 或 `steerMu` 单锁短临界，无新增嵌套；`-race` 全量验证（T1/T2/T5 必跑）。
- **P1 隔离**：completion/envelope/`resultRead`/evidence lease 零触碰（T1 回归显式断言）。
- **防虚假完成**：每个 T 有独立验证命令；T5 e2e 必须断言"请求消息列表尾部含 steer 前缀 + 子代理行为响应"，仅靠单测通过不算完成；`acceptance_criteria` 逐条附命令证据。
- **执行队越权边界**：T1-T4 文件 disjoint（jobs.go / agent.go+run_loop.go / task.go / builtin）；T5 依赖前序；纪律团对 T1 锁序与 T3 注入点做代码审查。

---

## 附：与参考设计的对应

| 参考 | 本方案落点 |
|------|-----------|
| Qwen send_message(task_id) 子代理工具轮次边界投递 | `send_message` 工具 + `runToolLoop` 轮次边界 `pullSteerSource` 注入（D5：每轮 1 条） |
| CCB queuePendingMessage | `Job.pending` + `QueuePendingMessageForSession`（D1-D3：限长 50、Running+!runReturned 双条件） |
| P1 completion/envelope/锁序（必读 docs/team/20260810-p1-autodeliver/plan.md） | 零交集（§五）；锁序沿用 `j.mu` 快照 → `m.mu` append 纪律 |
