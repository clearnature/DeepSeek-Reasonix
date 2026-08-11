# Plan 3（独立视角）：P4 前台→后台动态降级（backgroundize）

> 事务：docs/team/20260810-p4-backgroundize/ · 本文件为 3 个 team-planner 独立产出之一
> 目标：前台 task 同步阻塞运行中，可在迭代边界平滑转交为后台 job 续跑（用户 /background 手动 + 120s 自动阈值），
>       后台化后 P1 信封投递结果、P3 steer 可指挥，续跑前缀稳定。
> 参照：CCB sync agent 运行中后台化（前台迭代 return() 清理 → isAsync:true 重启同 runAgent 后台跑完、
>       getAutoBackgroundMs / CLAUDE_AUTO_BACKGROUND_TASKS）；P3 plan-3 的 /task-message 落点（controller.go:1440）。
> 基线事实（逐行验证）：internal/agent/task.go:758-992（RunProfileSpec 前台/后台双路径）、
>       internal/agent/run_loop.go:275-368（runToolLoop 迭代边界）、internal/jobs/jobs.go:452-570（StartForSession + jobCtxKey）、
>       internal/agent/task.go:1792-1867（RunSubAgentWithSession，Agent 在函数内构造）、internal/agent/agent.go:1042（Options）、
>       internal/agent/agent.go:146-161（withAgentContext 仅覆盖 jobs/memory/planmode）、internal/control/controller.go:1398-1453（slash 表）。

---

## 一、拓扑扫描

### 文件依赖图

```
internal/agent/task.go:758-992   RunProfileSpec 前台路径（同步阻塞）→ P4 交接点
internal/agent/task.go:1610-1632 runSubSession/runReadOnlySubSession → RunSubAgentWithSession
internal/agent/task.go:1792-1867 RunSubAgentWithSession（内部构造 Agent；err wrap "sub-agent:"）
internal/agent/run_loop.go:275-368 runToolLoop 迭代边界 → P4 后台化检查点（consumeSteer/P3 drain 之后）
internal/agent/run_loop.go:138-271 beginRunTurn（无条件 session.Add 任务 prompt → 续跑需跳过）
internal/agent/agent.go:1042      Options 结构 → 加 ResumeFromSession
internal/agent/agent.go:1374-1437 Agent.Run（runStartedAt 计时点）
internal/jobs/jobs.go:452-570     StartForSession（jobCtxKey 注入；P1/P3 自动生效）
internal/control/controller.go:1398-1453 slash 表（/task-message 参照 → /background）
internal/control/controller.go:5475-5509 Controller.Jobs/KillJob/SendTaskMessage（后台化后直接可用）
internal/ablation/ablation.go:13-26 Module 列表 → 加 AutoBackground
```

### 级联风险

| 风险 | 等级 | 说明 |
|------|------|------|
| **续跑重复 add 任务 prompt** | HIGH | runSubSession 每次调用都会把 prompt 作为新 user turn add 进 sess（run_loop.go:248-251）；续跑若照旧 add，同一任务消息出现两次 → 前缀破坏。必须加 ResumeFromSession 跳过 |
| **前台 goroutine 与后台 job 双写 sess** | HIGH | 交接若不在串行点完成，两个 goroutine 同时 add 会话 → 前缀竞态。必须保证前台 run loop 完全 return 后才 StartForSession |
| **turn 忙碌时用户 /background 的到达路径** | MED-HIGH | 前台 task 阻塞父 turn；/background 是否即时分发取决于 controller 输入路径（未知，需 T0 探查） |
| **run.Release 语义** | MED | 前台 defer run.Release() 与 job defer run.Release() 双调用（Release 幂等，但需确认 release 不销毁 Session 对象） |
| **自动阈值触发过早** | MED | 第一轮采样即超 120s 时，回到迭代边界立即后台化，可能太激进（增强项：至少 2 轮或采样后计时） |
| **event.Kind iota 漂移** | LOW | 若新增 Kind 必须 iota 尾部追加（P3 纪律） |
| **subagentStartContext 重复 prepend** | LOW | 续跑时 isFreshSubagentSession(sess)=false 已阻止（task.go:1818），无需新逻辑 |
| **CompareShape 误报** | LOW | 新 Agent haveLastPrefixShape=false → prevPrefixShape=prefixShape（run_loop.go:296-298 既有防御） |

---

## 二、多路径推演

### 方案 A：run loop 迭代边界 sentinel 交接 + 同一内存 Session 续跑（推荐）

runToolLoop 循环顶部（consumeSteer 之后）检查后台化信号 → 返回 `errBackgroundizeRequested` sentinel →
RunSubAgentWithSession 识别后原样上抛（不 wrap）→ RunProfileSpec 前台路径在同一 goroutine 串行点：
释放前台执行权 → MarkRunning → StartForSession 启动 job → job goroutine 内以 resume 模式重跑 runSession
（Options.ResumeFromSession=true，复用同一 *Session，不重复 add prompt）→ 返回 "Started background task …"。

- 复杂度：中。改动集中在 task.go 前台路径 + run_loop 检查点 + Agent 续跑标志。
- 性能/缓存：续跑第一轮请求与"若未后台化则下一轮"字节一致 → 前缀稳定、0 额外 cache miss。
- 可维护性：新增一个包内 sentinel + 一个 ctx 信号 + 一个 resume 标志；不触碰 jobs 包、不新造 job 状态。
- 风险：交接串行点论证（B1/B2）；turn-busy 输入路径未知（T0）。

### 方案 B：ctx 注入 drainer/回调，run loop 调用后由外部接管（否决）

把"后台化"做成 run loop 回调 `ctx = WithBackgroundizer(ctx, func()...)`，run loop 调用回调后 return 普通 nil。

- 否决理由：回调需要携带 run/trk/slot 等上下文，导致 agent→task 反向依赖；且 run loop 返回 nil 会被
  RunSubAgentWithSession 当"正常完成"处理（latestAssistantAnswer 可能为空 → 误报"没有最终答案"），
  必须额外引入"已交接"标记绕过，等于再造一个 sentinel，只是藏得更深。方案 A 的显式 sentinel 更直白。

### 方案 C：后台化 = 先 Kill 前台再手动重建 job（否决）

前台 run 被取消，父 agent 收到"任务已停止"，由用户/模型用 continue_from 重新 task 后台启动。

- 否决理由：依赖模型主动续跑，非确定性；Kill 后 evidence/transcript 状态复杂（SaveFailed/MarkRunning 竞态）；
  与"平滑续跑、前缀稳定"目标相悖（重建 session 需从盘加载，前缀字节相同但 compare 成本高）。

### 裁决

**采纳方案 A**。核心不变量：**交接发生在前台 goroutine 的串行点上，前台 run loop 已 return 之后才启动 job；
续跑复用同一内存 Session，不新增任何消息**。这同时满足并发安全（无双跑）与缓存红线（前缀稳定）。

---

## 三、最终设计

### 3.1 状态机（需求 1）

```
foreground-running ──(迭代边界检查点：/background 或自动阈值)──▶ backgroundizing ──▶ background-running ──▶ done
       │                                                            (同一 goroutine 串行完成：           │
       │                                                             前台 run loop 已 return、            └─ failed / killed
       │                                                             StartForSession 已注册)                (P1 recordCompletion)
       └────(run loop 自然结束 / 失败 / 取消)──────────────▶ done/failed
```

- `foreground-running`：TaskTool 前台路径，trk.running()，尚未注册 job。
- `backgroundizing`：瞬时交接事务窗口，由 Notice/事件表达（Job.Status 不新增值——交接前尚非 job，交接后即 Running）。
- `background-running`：Manager 中 Running 的 Kind=="task" job（P1/P3/wait/bash_output 全部生效）。
- `done`：P1 recordCompletion 信封投递（自动，无需新逻辑）。

### 3.2 并发安全（需求 2）——交接串行点 + 幂等信号

**交接串行点**（避免双跑的核心论证）：
1. 父 agent 工具轮次 goroutine 内同步执行 task.Execute → RunProfileSpec 前台路径 runSession。
2. runToolLoop 迭代边界返回 `errBackgroundizeRequested` → runSession 返回 → RunSubAgentWithSession 上抛
   （原样，不 wrap "sub-agent:"）→ RunProfileSpec 前台 goroutine 捕获。
3. 前台 goroutine 在同一点执行 `backgroundizeForeground`：StartForSession（注册 job，job goroutine 开始
   acquireSlot 排队）→ backgroundHandoff=true → return。
4. job goroutine 的 runSession 只在 jobCtx 内启动，此时前台 run loop 已完全退出（其栈已 unwind），
   sess 不会出现双写。**无双跑窗口**：交接是同一 goroutine 的前后两段，不是两段并发。
5. slot：前台 defer releaseSlot() 在 return 时释放；job goroutine 内 acquireSlot 重新排队——允许其他任务
   抢 slot（正常调度语义，与"后台任务排队"一致）。

**幂等信号**（用户手动 + 自动同源互斥）：

```go
// internal/agent/backgroundize.go（新文件，agent 包内）
// 前台 task 的后台化请求信号。ctx value 注入，Request 幂等（sync.Once 语义），
// Requested 只读检查。锁序：signal.mu 独立，不嵌套 m.mu/j.mu。
type backgroundizeSignal struct {
    mu        sync.Mutex
    requested bool
}
func WithBackgroundizeSignal(ctx context.Context, s *backgroundizeSignal) context.Context
func BackgroundizeSignalFromContext(ctx context.Context) *backgroundizeSignal // nil 安全
func (s *backgroundizeSignal) Request() bool   // 幂等：返回是否首次触发
func (s *backgroundizeSignal) Requested() bool
```

- Controller 持有当前信号句柄（turn 开始创建并注入 turn ctx，turn 结束清理）。
- 前台子代理 ctx 链继承：withAgentContext 只覆盖 jobs/memory/planmode（agent.go:146-161），
  不触碰自定义 key → run loop 可读。✓

### 3.3 run loop 检查点（需求 1 触发点）

runToolLoop 循环顶部，P3 DrainPendingMessages 之后、schemas/采样之前（run_loop.go:292 之后）：

```go
// P4: backgroundize checkpoint — one per tool-round boundary. A request may
// arrive mid-sampling (user /background or the auto threshold); it is only
// honored here, so the current provider round and tool execution finish
// cleanly first (CCB "foreground iteration return()" semantics). Returning
// errBackgroundizeRequested unwinds the foreground chain to RunProfileSpec,
// which performs the serialized handoff to a background job.
if a.backgroundizeRequested(ctx) {
    return errBackgroundizeRequested
}
```

`a.backgroundizeRequested(ctx)` 判定：
- 用户信号：`sig := BackgroundizeSignalFromContext(ctx); sig != nil && sig.Requested()`。
- 自动阈值：`a.ablation.Off(ablation.AutoBackground)` 时不触发自动（手动仍可用）；
  `a.runStartedAt`（Agent.Run 开头 `time.Now()`）距当前 `> AutoBackgroundMsFromContext(ctx)` 时置为已触发并返回 true。
- 计时起点与保底：runStartedAt = Agent.Run 开头；检查点在迭代边界 → 第一轮采样超时也会触发
  （标注为已知激进行为，增强项：至少 2 轮才允许自动）。

### 3.4 续跑模式（需求 5 缓存红线的技术载体）

**Options 增加字段**（agent.go:1042）：

```go
// ResumeFromSession continues an in-memory session already carrying the task
// prompt: Agent.Run skips appending the input as a new user turn, so the
// historical prefix stays byte-identical across the foreground→background
// handoff. Only the backgroundize handoff path sets it.
ResumeFromSession bool
```

**beginRunTurn 改造**（run_loop.go:248-251）：

```go
if !a.resumeExistingSession {
    a.session.Add(provider.Message{Role: provider.RoleUser, Content: input, RawContent: rawContent, Images: ..., CreatedAt: userCreatedAt})
}
```

- 续跑首轮：perTurnState 清零、TurnStarted 事件、delivery 意图重分类照常（classifier 用 opts.ClassifierTaskText，
  结果一致，无副作用）；只是不重复 add 任务 prompt。
- isFreshSubagentSession(sess)=false → 不再 prepend subagentStartContext（task.go:1818 既有检查）。✓
- 新 Agent 的 haveLastPrefixShape=false → 首轮 prevPrefixShape=prefixShape → CompareShape 无 rewrite 告警
  （run_loop.go:296-298 既有防御）。✓

**RunSubAgentWithSession 改造**（task.go:1832-1838）：

```go
if err := sub.Run(ctx, prompt); err != nil {
    if errors.Is(err, errBackgroundizeRequested) {
        return "", err // pass through unwrapped so RunProfileSpec can recognize the handoff
    }
    ... // 现有 wrap "sub-agent:" 逻辑不变
}
```

### 3.5 RunProfileSpec 交接（需求 2/5 落点）

runSession 闭包改为带 resume 标志：

```go
runSessionMode := func(runCtx context.Context, sink event.Sink, writerAlreadyRegistered, resume bool) (string, error) {
    ... // 现有 863-875 逻辑；resume 透传给 runSubSession → subagentOptions(opts.ResumeFromSession = resume)
}
```

前台路径改造（974-991 现有前台分支）：

```go
answer, err := runSessionMode(ctx, trk.wrap(), false, false)
if errors.Is(err, errBackgroundizeRequested) {
    return t.backgroundizeForeground(ctx, spec, trk, run, subReg, modelRef, effortRef, acquireReq, ...)
}
```

`backgroundizeForeground`（复用现有后台分支 877-971 的骨架）：
1. `t.transcripts.MarkRunning(run)`（若 transcripts 非 nil；前台路径原本不调，交接时补上）。
2. `trk.queued()` → `jm.StartForSession(SessionFromContext(ctx), "task", label, func(jobCtx, _ io.Writer) (string, error) { ... jobCtx = WithParentSession(...); evidence.WithLedger(...); defer run.Release(); acquireSlot(jobCtx, slotReq); trk.running(); return runSessionMode(jobCtx, trk.wrap(), writerRegistered, true) ... })`。
3. `backgroundHandoff = true`（外层 defer 不 finish trk；trk 所有权交接给 job goroutine，job defer 里 finish）。
4. return 现有后台启动文本（"Started background task %q …"）。

- MarkRunning 时序：StartForSession 之前（与现有后台路径 900-906 一致）。
- transcript 完成：job goroutine 里 runSessionMode 成功后 SaveCompleted（现有后台逻辑，无需新方法）。✓
- 前台 defer run.Release() 与 job defer run.Release() 双调用安全：Release 幂等（release=nil 保护，
  subagent_store.go:114-119）；需执行队确认 release 只释放 store 锁、不解构 Session（SubagentRun.Session 独立对象）。

### 3.6 自动后台化阈值（需求 3）

- env：`REASONIX_AUTO_BACKGROUND_MS`（沿用 REASONIX_* env 惯例，os.Getenv + strings.TrimSpace），
  默认 `120000`（120s），`0`/负 = 禁用自动后台化（手动 /background 仍可用）。
- 解析位置：TaskTool 装配处（controller boot，与 WithAblation 并列）；存 TaskTool 字段 `autoBackgroundMs`；
  subagentOptions 里 `ctx = WithAutoBackgroundMs(ctx, t.autoBackgroundMs)` 注入（run loop 经 ctx 读取）。
- ablation：`internal/ablation/ablation.go` Module 列表追加 `AutoBackground Module = "auto-background"`；
  `Ablation.Off(AutoBackground)` 时自动触发跳过（benchmark 控制臂），手动通道不受影响。
- reasonix.toml 映射（可选）：`agent.auto_background_ms`，优先级 env > toml > 默认。

### 3.7 用户入口与 UI 信号（需求 4）

**`/background` slash**（controller.go:1398 switch，参照 /task-message 落点）：

```
/background [<label 忽略>]
```

- 语义：请求把"当前前台 task"后台化。无前台任务 → `c.notice("no foreground task is running")`。
- 实现：`c.backgroundizeSignal.Request()`（幂等）→ `c.notice("backgrounding current task (takes effect at the next iteration boundary)…")`。
- 后台化完成反馈：StartForSession 自动 emit startedText Notice（现有行为），交接点可加一条
  "moved to background as task-N"（复用现有 Notice 通道）。

**UI 信号**：
- 状态栏：Controller.Status()（controller.go:347 区域）增加前台任务信息
  `ForegroundTask *ForegroundTaskView{Label, StartedAt, ElapsedMs, AutoBackgroundMs, Backgroundable}`，
  desktop/TUI 据此显示"可后台化"与自动倒计时（P2 面板的底子，本事务只暴露 API）。
- 结构化事件（可选增强）：event.Kind 在 iota 尾部追加 `Backgroundized`（防 Kind 漂移纪律同 P3）。
- **T0 前置探查（阻断项）**：确认 controller 输入路径在父 turn 忙碌时对 /background 是否即时分发。
  若输入排队到下一轮，则须在输入层加"turn-busy 紧急命令"通道（只允许白名单 /background 类命令直通），
  否则前台 task 跑完前用户命令无法生效。此点现状未知，必须先探查再定接线。

### 3.8 P1/P3 协同（需求 5）

- 后台化后 job 就是普通 Kind=="task" 后台 job：P1 recordCompletion → `<background-job-result>` 信封
  注入下一轮 user turn（零改动）；P3 SendMessageForSession + DrainPendingMessages 经 jobCtxKey 直接可用
  （StartForSession 已注入 jobCtxKey，jobs.go:484）；wait/bash_output 只读快照同样生效。✓
- 缓存红线：续跑复用同一内存 Session，交接不 add 任何消息（不在子代理历史里写"后台化"标记，
  只在父侧 Notice）；续跑首轮请求与不后台化的下一轮请求字节一致。✓

---

## 四、任务分解（执行小队）

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| **T0** | 前置探查：父 turn 忙碌时 controller 输入路径对 /background 的分发行为（阻塞项，决定 T5 接线） | `internal/control/input.go`、`internal/control/controller.go` | 探查记录：忙碌时 slash 即时分发 or 排队；据此选直通或紧急通道 |
| **T1** | 信号与检查点：`backgroundize.go`（signal + ctx helper + Request 幂等）+ `errBackgroundizeRequested` 定义 + runToolLoop 检查点 | `internal/agent/backgroundize.go`（新）、`internal/agent/run_loop.go` | `go test ./internal/agent/ -run 'Backgroundize' -race`；单测：Request 幂等、ctx 无信号时 no-op、检查点返回 sentinel |
| **T2** | 续跑模式：Options.ResumeFromSession + Agent 字段 + beginRunTurn 跳过 add + RunSubAgentWithSession sentinel 直传 | `internal/agent/agent.go`、`internal/agent/run_loop.go`、`internal/agent/task.go` | 单测：resume 首轮 sess 无重复 prompt、subagentStartContext 不重复、CompareShape 无告警 |
| **T3** | 交接：runSessionMode(resume) + backgroundizeForeground（MarkRunning + StartForSession + backgroundHandoff + slot 释放） | `internal/agent/task.go` | `go test ./internal/agent/ -run 'Task' -race`；单测：交接后 job Running、无双跑（sess 无并发 add） |
| **T4** | 自动阈值：`REASONIX_AUTO_BACKGROUND_MS` 解析（默认 120000/0 禁用）+ WithAutoBackgroundMs ctx + ablation.AutoBackground | `internal/agent/task.go`、`internal/agent/backgroundize.go`、`internal/ablation/ablation.go` | 单测：默认/0/负/ablation-off 分支；计时触发与手动触发互斥 |
| **T5** | `/background` slash + Controller 前台任务状态暴露 + Notice | `internal/control/controller.go`、`internal/control/controller_test.go` | `go test ./internal/control/ -run 'Background'`；分支：无前台任务 / 幂等重复请求 / 成功 |
| **T6** | e2e：前台 task 运行 → /background → 转 job → wait 收结果；P1 信封下一轮投递；P3 /task-message 指挥后台化 job；缓存断言（sess 历史字节不变） | 集成测试（jobs + agent + control 组合） | 定向 e2e 通过；`go build ./...` |
| **T7** | 事务状态更新 | `docs/team/20260810-p4-backgroundize/` | 文件存在，定稿引用本 plan 编号 |

依赖顺序：T0 → T1 → T2 → T3（依赖 T1/T2）→ T4（可 T2 后并行）→ T5（依赖 T0/T1）→ T6（依赖 T1-T5）→ T7。

---

## 五、对抗自检（devil's advocate）

- **B1（最高）双跑竞态**：交接串行点论证依赖"前台 run loop 已 return 才启动 job"。攻击点：StartForSession
  返回后，job goroutine 的 acquireSlot 可能阻塞（排队），而前台 goroutine 立即 return——此时两者是否都持有
  run/trk 引用？前台 return 后不再触碰 run/sess/trk（defer 只释放 slot/store 锁）；job 阻塞在 slot 上，未进入
  runSession → sess 无并发访问。薄弱点：**SubagentRun.release 是否解构 Session**——执行队须验证
  SubagentStore 锁释放语义，若 release 销毁会话内容则必须把前台 defer run.Release() 改为交接路径专用
  （job 内统一释放）。
- **B2 prompt 重复 add**：T2 跳过 add 的正确性依赖 beginRunTurn 唯一 add 点（已核实 248-251）。若后续有
  forkRestore/salvage 路径也 add 用户消息，续跑可能重复——执行队须 grep 所有 `session.Add(RoleUser` 落点。
- **B3 turn-busy 输入路径**：T0 阻断项。若现有架构根本不支持 turn 中 slash，则 /background 只能依赖自动阈值
  或 UI 层直通——设计已给出降级接线（紧急命令通道），但这是当前最大的不确定面。
- **B4 自动触发过早**：第一轮采样即超阈值会立即后台化。缓解：增强项"至少 2 轮"；默认接受（CCB 同语义），
  在文档中标注。
- **B5 sentinel 传播遗漏**：runSubSession 与 runReadOnlySubSession 都要透传 sentinel；若某中间层 wrap 掉了
  sentinel（errors.Is 失效），任务会被误判失败。执行队须对两条子代理路径各加透传断言。
- **B6 event.Kind**：若加 Backgroundized 事件，必须 iota 尾部追加（P3 纪律），且单测断言数值稳定。
- **B7 trk 交接**：前台 trk.running() 与 job trk.running() 重复 emit（状态相同，无害）；job defer 唯一 finish。
  若 job 因 slot 排队被 Kill，trk.finish(Killed) 路径要覆盖"已前台 running 过的"交接态。
- **B8 自动阈值与 ablation 并发**：run loop 读 autoBackgroundMs 与 ablation 均为只读，无锁竞争；ctx 值在
  TaskTool 注入点一次性写入，子代理不可变。✓

---

## 六、缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：父侧仅工具结果文本（"Started background task …"）与 Notice；子代理 sess 历史零插入、
  零改写（交接不写任何标记消息）。续跑首轮采样请求与"未后台化则下一轮"字节一致 → 0 额外 cache miss。OK
- **前缀稳定**：续跑复用同一内存 Session + ResumeFromSession 跳过 add + 新 Agent 首轮
  prevPrefixShape=prefixShape（既有防御）→ CompareShape 无 rewrite 告警。OK
- **P1/P3 协同**：后台化 job = 普通 Kind=="task" job，P1 信封 / P3 steer / wait 全部自动生效，零改动。OK
- **锁序**：signal.mu 独立短临界区，不嵌套 m.mu/j.mu；交接串行点无跨 goroutine 锁序问题；-race 验证。OK
- **执行队不越权**：T1/T2 与 T3 职责 disjoint（信号/续跑 vs 交接）；T4 可并行；T0 是 T5 的阻断前置。OK

<!-- 本 plan-3 以 ASCII 渲染（CJK write_file 链路在本子代理工具链中乱码，见 P3 plan-3 先例） -->
