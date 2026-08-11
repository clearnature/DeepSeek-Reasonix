# 计划：P4 前台→后台动态降级（team-planner 独立视角 plan-2）

> 事务：docs/team/20260810-p4-backgroundize/ · 产出：本文件（独立方案，供父代理汇总裁决）
> 立场声明：本方案基于对 `internal/agent/task.go`、`internal/agent/run_loop.go`、`internal/jobs/jobs.go`、`internal/tool/builtin/bgjobs.go`、`internal/control/{controller,turn_orchestrator,slash}.go`、`internal/acp/service.go`、`internal/ablation/ablation.go` 的实读。凡未实读处一律标注 `[待验证]`，不臆断。

---

## 0. 核心认知（一句话）

**我们与 CCB 的结构差异不是"要不要后台化"，而是"切换点在哪"**：CCB 的 signal-race 发生在顶层 runAgent（前台迭代 return() → isAsync:true 重启同一 runAgent）；我们的前台 task 是嵌套在父 agent `executeBatch` 内的子 agent（`task.go:973-991` 前台路径），同步阻塞、无可打断点。因此**最稳的方案不是"打断重启"，而是让前台 task 从第一轮就是一个可放手的 job——切换只是父侧停止等待**，job goroutine 与子 agent Session 全程同一实例，零中断、零重启、零现场搬迁。

---

## 1. 拓扑扫描

### 1.1 相关文件与依赖

| 文件 | 关键符号（实读行号） | 与 P4 的关系 |
|------|----------------------|--------------|
| `internal/agent/task.go` | `TaskTool.Execute`(647)、`RunProfileSpec`(758)、`runSession` 闭包(863)、后台分支(877-971)、前台同步执行(973-991)、`acquireSlot`(994) | **改造主战场**：前台路径改为"可后台化 job + 同步等待" |
| `internal/agent/execute_batch.go` | `executeBatch`(30)：writer 工具**串行**执行 | 约束：同刻至多一个前台 writer task → session 级单前台 job 成立 |
| `internal/jobs/jobs.go` | `Job`(122)、`StartForSession`(452)、`jobCtxKey` 注入(484)、`recordCompletion` 调用点(570)、`close(done)`(581)、`SendMessageForSession`(984，`Kind!="task"` 拒绝 at 999)、`jobCtxKey`/`DrainPendingMessages`(2130-2197)、`TryLeaseEvidenceForSession`(2214) | job 归属、P3 指挥、P1 信封、证据领取的全部底座 |
| `internal/agent/run_loop.go` | P3 drain 注入点(286-292)、`runToolLoop`(275) | 后台化 job 的 P3 指挥注入点已存在，零改动 |
| `internal/tool/builtin/bgjobs.go` | `collectBackgroundEvidence`(201-233)：`TryLeaseEvidenceForSession`+`NoteBackgroundLease`+`MergeChild` 模式 | 前台 job 完成时"领取证据"直接复用此模式 |
| `internal/control/controller.go` | `jobs *jobs.Manager`(158)、`TrySteer`(2353)、`KillForSession`(5497)、`SendMessageForSession`(5508)、`Jobs()`(5481) | `/background` 命令落点：Controller 已有 manager 访问权 |
| `internal/acp/service.go` | `sessionSteer`(1185-1202)：运行中输入经 `TrySteer` | 运行中命令通道：新增 `session_backgroundize` RPC 或在此拦截 `/background` 前缀 |
| `internal/control/slash.go` | slash 解析/处理(53-88, 424-535) | `/background` 注册为管理命令；但**运行中** slash 需独立拦截（见 4.4） |
| `internal/ablation/ablation.go` | `Module`(11-22)、`Set.Off`(72) | 新增 `Backgroundize` 开关模块 |
| `internal/config/config.go` | `BashTimeoutSeconds` 模式(default 120) | 新增 `agent.foreground_backgroundize_seconds` 键 |
| `docs/team/20260810-p1-autodeliver/plan.md` | P1 定稿：`<background-jobs>` 信封注入点 `input.go:191-195` | P1 协同（4.5） |
| `docs/team/20260810-p3-steer/plan.md` | P3 定稿：16 条/8KB 队列、每轮 1 条、kind=="task" | P3 协同（4.5） |

### 1.2 级联风险（按严重度排序）

1. **【阻断级】前台 task 的 mutation 证据归属**：现有前台路径子 agent 证据直接 merge 进父 turn ledger（runSession 用父 ctx）；后台路径证据在 job 独立 ledger（`task.go:929` `evidence.WithLedger(jobCtx, backgroundEvidence)`）。若前台改 job 化而不做证据桥接，父 turn 的 delivery 门（`FinalReadinessError`）会看不到前台 task 的写证据 → **前台 task 写入的文件变更不满足 readiness，turn 必挂**。→ 必须实现"前台领取"动作（4.3-B）。
2. **【高危】前台 job 完成时的 P1 信封时序**：`recordCompletion`(570) 在 `close(j.done)`(581) **之前**执行。若"抑制信封"标志在 job.done 之后由 Execute 侧设置，则信封已排队、抑制失败。→ 标志必须在 **Start 时预设**（`foregroundClaimPending`），放手时清除（两种顺序下放手方向都正确，见 4.3-C 论证）。
3. **【高危】turn cancel 语义回归**：现有前台 task 随父 turn cancel（ctx 传播到 runSession）；后台 job ctx 是 session 级（`m.root`，不随 turn 死）。前台 job 化后若不显式 kill，用户 Esc 后任务仍在后台跑 = **行为回归**。→ Execute 等待循环的 `ctx.Done` 分支必须显式 `KillForSession`（4.3-D）。
4. **【中危】mutationObserver writer 注册语义**：后台路径以 `backgroundWriter` 身份 `RegisterWriter("background_subagent")`（`task.go:857-869`）；前台原本不注册。前台 job 化后按后台路径注册 → 前台 task 运行期间占一个 checkpoint writer 席位。**语义差异需 checkpoint 团队确认** `[待验证]`。
5. **【中危】双前台并发**：`executeBatch` 写者串行 → 同刻至多一个前台 writer task；但 read-only 子 agent 可并行。前台 job 注册必须处理"同 session 已有一个前台 job"（幂等拒绝或替换）`[设计决策见 4.3-F]`。
6. **【低危】启动/完成 Notice 噪音**：后台路径发 `startedText`/完成 Notice；前台 job 若照发会误导用户（"Background task started"）。→ foreground 标志同时抑制 started/completed Notice。
7. **【低危】`Job.Kind` 兼容面**：新增 `kind='task-background'` 会破坏 P3（`SendMessageForSession` 的 `Kind!="task"` 拒绝，jobs.go:999）与 P1 信封判定，**否决新 kind**（见 4.1）。
8. **【低危】`planmode.Active` 语义**：`collectBackgroundEvidence`(bgjobs.go:206) 在 plan turn 跳过证据合并。前台领取必须遵守同一规则，否则 plan 阶段的 task 写证据被提前消费。

---

## 2. 多路径推演

### 方案 A（推荐）：可后台化前台 job —— 前台 task 天然是 job，切换=父侧放手

前台 task（`RunProfileSpec` 前台分支）改造为：

```
开关未开 → 走现有同步路径（零回归）
开关已开 →
  1. job := jm.StartForegroundHandoffable(session, "task", label, closure)
     closure 复用后台分支闭包主体（jobCtxKey / evidence ledger / P3 drain / acquireSlot / runSession / PublishEvidence / trk.finish），
     仅 recordCompletion 前检查 foregroundClaimPending 标志（见 4.3）
  2. Execute 侧 select 循环：
       job.done        → ClaimForegroundResult（merge evidence 进父 turn ledger + 抑制信封已由预设标志完成）→ 返回前台答案文本
       switch 信号     → ReleaseForegroundClaim + 返回 "已转入后台 job-id=task-N" 文本
       ctx.Done(父turn) → KillForSession → 返回 ctx.Err()（与现有前台取消一致）
```

- **复杂度**：中高（jobs 层 3 个新原语 + task.go 前台分支改造 + controller 命令通道）
- **性能**：job 化有 artifact 文件/表项开销；证据领取一次 TryLease（廉价）
- **可维护性**：高——后台路径机制全部复用，新增代码量最小；无中断/重启状态机
- **风险**：证据桥接（1.2-1）与完成时序（1.2-2）是新增正确性负担，但**都有明确、可测的解法**；无 signal-race

### 方案 B（对照）：CCB 式 —— 前台同步跑，收到信号后 cancel + `continue_from` 重启后台

`sub.Run` 前台执行中收到信号 → cancel 子 agent（保留 Session）→ 以 run_in_background=true + 同 ref 重启 → Execute 返回 job-id。

- **复杂度**：高——需要安全中断点（子 agent 可能正处于流式响应/tool 执行中）、现场搬迁（mutationObserver/recoveryTaskID/evidence/trk 全部要跨实例转移）、且重启 = 新一轮 provider 请求（上下文前缀重放，缓存命中但耗时）
- **风险**：cancel 与重启之间工具竞态；与现有 interrupted 恢复路径（run_loop.go:322 附近）交互复杂
- **结论**：CCB 的 signal-race 是"顶层 runAgent 架构"的产物；我们的嵌套子 agent 场景**没有理由引入竞态**。否决。

### 方案 C（对照）：前台时长封顶，超时返回部分结果

不真正后台化，只是超时强制结束前台 task 并返回"已完成部分"。
- 不是任务要求的"后台化续跑"，仅作语义边界。否决。

### 选优结论

**采纳方案 A**。理由：(a) 续跑语义 = 天然"保留现场续跑"（同一 goroutine、同一 Session 从头跑到尾），比 CCB 的"同 prompt 重启"更稳——无重启竞态、无前缀重放、无现场序列化；(b) 复用 P1/P3/后台 job 全部既有机制，改动面收敛在 jobs 原语 + task 前台分支 + controller 命令；(c) 证据桥接与完成时序两个难点都有确定性解法与单测（见 4.3）。

---

## 3. 设计决策（覆盖任务 6 个必答点）

### 3.1 job 归属：复用 `StartForSession`，**不新增** `kind='task-background'`

- 新 job 仍 `kind="task"`，经 `StartForSession` 启动（沿用 P3 注入的 `jobCtxKey`、artifact、evidence、teardown 全链路）。
- Job 新增私有标志：`foregroundClaimPending bool`（Start 时 true = 前台发起、完成不注入 P1 信封不发完成 Notice）+ 现有 `result`/`evidence` 字段不变。
- 否决新 kind 的实证：`SendMessageForSession` 对 `Kind!="task"` 直接拒绝（jobs.go:999）——新 kind 会让后台化 job **无法被 P3 指挥**，还须改 jobs.go + input.go drain 判定 + View 渲染，破坏面大且零收益。
- 状态栏区分（可选 UI 增强）：`jobs.View` 增 `foreground` bool，不进核心逻辑。

### 3.2 续跑语义：**保留现场续跑**（方案 A 天然实现），明确优于 CCB 重启

- 同一子 agent Session（`prepareTranscriptRunWithPrompt` 产物）由 job goroutine 从头持有到尾；切换只是父侧 select 走 switch 分支。
- 语义对齐点：CCB "前台迭代 return() 清理" 在我们 = Execute 等待循环退出（不 kill、不 cancel）；CCB "isAsync:true 重启" 在我们 = job 继续在后台跑（**本就在跑**，无重启）。
- 切换后 P3 指挥立即可用：job 的 ctx 已带 `jobCtxKey`（StartForSession 注入，jobs.go:484），`SendMessageForSession` 对 kind="task" 放行 → 父 agent 后续轮 `send_message` / 用户 `/task-message` 直接生效，`run_loop.go:289` drain 无需改动。

### 3.3 自动后台化阈值与开关（三层）

| 层 | 键/模块 | 默认 | 语义 |
|----|---------|------|------|
| config | `agent.foreground_backgroundize_seconds` | `120` | 前台等待/运行超过该秒数自动放手；`0`=禁用自动（仅手动 `/background`）；`<0`=非法拒绝 `[实现时按 config 校验模式对齐]` |
| ablation | 新增 `Module Backgroundize = "backgroundize"`（ablation.go:13 处加 + `Modules()` 注册） | 开 | off 时**完全禁用**自动+手动后台化，前台 task 回到纯同步路径（benchmark A/B 归因） |
| env | config 系统既有 env 覆盖机制 `[待验证：config 包 env 覆盖具体键格式]` | — | 运维侧调阈值/禁用 |

计时语义：阈值 = **前台 task 从 job 实际 Running 起算的运行时长**，由 Execute 等待循环以 ticker 轮询 job 的 `startedAt/activityAt`（jobs 包暴露只读状态，复用 `OutputForSession` 同款 `j.mu` 读路径）触发 switch；简化实现可退化为"Execute 进入等待起算"（等待时长），语义差异在文档注明。**推荐后者（简单、无跨 goroutine 状态）**，运行时长版列为增强项。

### 3.4 用户入口：`/background` 运行中命令（新控制通道，不走 steer）

- **现状**：运行中输入通道活跃（ACP `sessionSteer` → `TrySteer`，service.go:1198）；但 steer 是"给模型看的文本"，前台 task 阻塞期间父 agent 消费不到，且 `/background` 是主机控制命令而非模型指令 → **必须走独立通道**。
- **Controller 新端口**：`TryBackgroundize() (jobID string, ok bool)`——查找 session 当前前台 job 控制句柄并请求放手；无前台 job 时返回 false（前端可提示"无运行中的前台任务"）。
- **ACP**：新增 RPC `session_backgroundize`（sessionSteer 同款样板，service.go:1185 模式）；或在 `sessionSteer` 内先识别 `/background` 前缀路由到 backgroundize。
- **TUI/CLI**：运行中输入提交路径拦截 `/background` 前缀 → 调 `TryBackgroundize`，不作为 steer 入队；空闲时 `/background` 走常规 slash 处理（提示无前台任务）`[待验证：chat_tui 运行中输入提交具体路径，须与 TrySteer 拦截点并列]`。
- **slash.go**：将 `/background` 加入管理命令列表（slash.go:424 `managementNotice` 模式）用于空闲态兜底。

### 3.5 P1/P3 协同 + 缓存红线

- **P3**：kind="task" 不变 → `SendMessageForSession`/`DrainPendingMessages` 对后台化 job **零改动生效**（见 3.2）。
- **P1**：放手（后台化）的 job 完成 → `recordCompletion` 正常排队 → `<background-jobs>` 信封注入父下一轮（input.go:191-195 容器不变，P1 定稿已固化的注入点）**零改动**。
- **前台领取（未放手完成）**：`foregroundClaimPending` 预设标志使 `recordCompletion` **跳过该 job 的信封排队与完成 Notice**（父 turn 已在 tool card 拿到结果）；同时 `ClaimForegroundResult` 把 job evidence 按 `collectBackgroundEvidence` 模式（TryLease → NoteBackgroundLease → MergeChild，bgjobs.go:201-233，planmode 跳过规则同源）并入父 turn ledger → 父 turn readiness 正常放行。
- **缓存红线**：
  - 切换**不修改子 agent 会话**（同一 Session 从头跑，无重写）；
  - 不修改父会话历史，仅按 executeBatch 既有机制追加 Execute 的 tool result 消息（与前台完成同构的尾部增长，非重写）——**注意禁止**在切换时额外注入 host steer/系统消息（避免无谓前缀变化）；
  - P1/P3 注入路径、位置、每轮 1 条规则全部保持定稿状态。
  - **结论：发送侧前缀稳定 ✅**（详见第 6 节）。

### 3.6 任务分解 + 验证点：见第 4 节。

---

## 4. 任务分解（执行小队）

> 每任务标注验证命令；T1-T4 依赖有序，T5 并行于 T3/T4 的接口可先以 stub 推进。

### T1（jobs 层原语）`internal/jobs/jobs.go`

- [ ] T1.1 `Job` 增私有标志 `foregroundClaimPending`（Start 时按变体预设）；`StartForegroundHandoffable(session, kind, label, run) *Job` = `StartForSession` + 设标志 + 抑制 started Notice（`startedText` 调用点 task.go 后台分支同款，jobs.go:499）。
- [ ] T1.2 `recordCompletion` 内 `j.mu` 临界区读标志：`foregroundClaimPending==true` → 跳过 `m.completed` 排队与完成 Notice（状态/artifact/evidence 照常）。
- [ ] T1.3 `ReleaseForegroundClaim(session, id) error`：清标志（后台化放手路径；`j.mu` 短临界区，不嵌套 `m.mu`）。
- [ ] T1.4 `ClaimForegroundResult(session, id) (evidence.ChildEvidenceSummary, bool)`：校验 terminal + `foregroundClaimPending`，返回可领取证据（复用 `TryLeaseEvidenceForSession` 内部路径，不重复实现锁）。
- [ ] T1.5 `RegisterForegroundJob(session, id) error` + `ForegroundJobs(session) []string` + `RequestForegroundSwitch(session) (int, error)`（Controller → 切换信号；session 级注册表，`m.mu` 保护；同 session 已有一个前台 job 时幂等拒绝——对应 execute_batch 写者串行约束）。信号实现：job 级 atomic flag（Execute select 轮询）或 `m` 级 chan（推荐 atomic flag + ticker 轮询，规避 select 泄漏）。
- **验证**：`go test ./internal/jobs/ -race -run 'Foreground|Claim|Handoffable'` + 新增单测覆盖：预设标志抑制信封、Release 后恢复信封、Claim 只对 terminal 生效、Register 幂等拒绝。

### T2（agent 层前台改造）`internal/agent/task.go`

- [ ] T2.1 `TaskTool` 增注入字段 `foregroundSwitch func(session string) (switchRequested bool, err error)` 与 `foregroundEnabled func() bool`（boot 接线，模式同 `recoveryGate`/`profileLookup` 注入）。
- [ ] T2.2 `RunProfileSpec` 前台分支（`!spec.Sched.RunInBackground`，task.go:779 后）按 `foregroundEnabled` 分流：
  - 关 → 现有同步路径（零改动）。
  - 开 → `StartForegroundHandoffable` + 等待循环（见 T2.3）；**job closure 主体复制后台分支闭包**（task.go:924-958：`WithParentSession`/`evidence.WithLedger`/`PublishEvidence`/`trk.finish`/acquireSlot/runSession/transcripts），即前台与后台闭包合流为同一实现，杜绝漂移。
- [ ] T2.3 Execute 侧等待循环（select）：
  - `job.done` → `ClaimForegroundResult` + 按 `collectBackgroundEvidence` 规则（planmode 跳过）merge 证据到父 turn ledger → 返回前台答案文本（`FormatSubagentRunResult`，与现有前台返回一致）。
  - 切换信号（`foregroundSwitch` 回调或 job flag 轮询）→ `ReleaseForegroundClaim` → 返回 `Started background task "task-N"…` 文本（复用后台分支返回文案，含 `FormatSubagentReference`）。
  - `ctx.Done()` → `jm.KillForSession(session, jobID)` → 返回 `ctx.Err()`（保持现有前台取消语义）。
- [ ] T2.4 阈值 ticker：按 config 秒数（默认 120）在等待循环内触发切换信号；`foreground_backgroundize_seconds==0` 时不启动 ticker。
- **验证**：`go test ./internal/agent/ -race -run 'Task|Foreground|Background|Steer'` + 新增：前台 job 完成答案正确且证据入父 ledger、switch 后返回 job-id 且 P1 信封下一轮注入、ctx cancel 杀 job、planmode 下不 merge、`foregroundEnabled=false` 时全路径字节级等价旧行为（回归锚点）。

### T3（controller 命令通道）`internal/control/controller.go` + `internal/acp/service.go` + `internal/control/slash.go`

- [ ] T3.1 `Controller.TryBackgroundize() (jobID string, ok bool)`：查 `c.jobs` 前台注册表 → `RequestForegroundSwitch`；前台 job 不存在/无前台运行 → `("", false)`。
- [ ] T3.2 ACP `session_backgroundize` RPC（sessionSteer 样板，service.go:1185 模式）；`sessionSteer` 内 `/background` 前缀预拦截路由（防误入 steer 队列）。
- [ ] T3.3 TUI 运行中输入路径拦截 `/background` → `TryBackgroundize`（不做 steer 入队）；slash.go 注册 `/background` 空闲态兜底 `[待验证 chat_tui 提交路径]`。
- **验证**：`go test ./internal/control/ -run 'Background|Steer|Slash'` + `go test ./internal/acp/ -run 'Steer|Background'`；e2e：前台 task 运行中发 `/background` → 收到 job-id 确认。

### T4（配置与开关）`internal/config/config.go` + `internal/ablation/ablation.go`

- [ ] T4.1 config 键 `agent.foreground_backgroundize_seconds`（默认 120，0 禁自动；getter 命名对齐 `BashTimeoutSeconds` 模式）。
- [ ] T4.2 ablation 新增 `Backgroundize` 模块（ablation.go:13 常量 + 25 `Modules()` 注册）；`foregroundEnabled = !ablation.Off(Backgroundize)`，且 off 时 T3 的 TryBackgroundize 返回 false。
- **验证**：`go test ./internal/config/ -run Backgroundize` + `go test ./internal/ablation/`。

### T5（回归与 e2e）`internal/agent/` + `internal/control/`

- [ ] T5.1 后台回归：现有 `task(run_in_background)` 路径零行为变化（前台改造不得触碰后台分支返回文案）。
- [ ] T5.2 缓存前缀回归：切换前后父会话前缀 CompareShape 无重写信号（复用现有 cache-diagnostics 测试锚）。
- [ ] T5.3 e2e：前台 task（sleep/长任务）→ 120s 自动后台化（测试内用短阈值注入）→ P1 信封下一轮注入 → 父 agent `wait`/`send_message` 指挥 → job 完成。
- **验证**：`go test ./internal/agent/ ./internal/control/ -race -run 'Background|Cache|Deliver'` + `go build ./...`。

### T6（文档）

- [ ] 本事务 `execution.md`/定稿汇总（父代理裁决后），更新 `docs/team/…` 状态；plan-2 的 `[待验证]` 项在执行时逐条落定。

---

## 5. 对抗自检（devil's advocate）

1. **攻击：前台 job 化后，子 agent 的运行不再属于父 turn，deliveryScope 的 `CriteriaEstablished`/`readinessRecovered` 语义是否漂移？**
   回答：`ClaimForegroundResult` 把 job evidence merge 进父 turn ledger 的动作在 readiness 检查（`handleFinalResponse`）**之前**完成（Execute 返回后父 run 循环才继续到最终响应），故前台完成的证据链与旧同步路径等价。**薄弱点**：`NoteBackgroundLease` 的 per-turn 幂等与"同 turn 内既 wait 又前台领取"的交互——父 agent 阻塞期间无法调 wait，理论无冲突，但需单测钉死（T5.3）。
2. **攻击：放手瞬间 job 恰好完成（select 竞态），返回 job-id 但任务已 done，父 agent wait 拿到结果——用户看到"已转入后台"后立刻又收到完成，体验割裂？**
   回答：可接受（信息不丢，wait 语义正确）；可在 select 前先非阻塞检查 `job.done` 已关闭则走领取分支。**薄弱点**：此优化需 `Job` 暴露 done 状态读法，列为 T2.3 实现细节。
3. **攻击：`foregroundClaimPending` 预设标志意味着"每个前台 job 天生抑制信封"——若 Execute 意外崩溃（panic）未走放手也未领取，job 完成时信封被抑制、证据无人 merge → 任务写入静默丢失审计？**
   回答：**真实薄弱点**。缓解：(a) Execute 的 deferred recover 里若 job 未完成则 `ReleaseForegroundClaim`（回退到后台语义，P1 信封兜底）；(b) job 完成且 `foregroundClaimPending` 仍为 true 且父 turn 失败时，`beginRunTurn` 的 background evidence 重lease（run_loop.go:184-196 既有机制）在下一轮兜底。**需在 T1.2/T2.3 中把此路径写进单测**。
4. **攻击：120s 阈值内用户没有 /background 意图，纯长任务被自动后台化后 P1 信封注入父下一轮，父模型被迫处理？**
   回答：这是 P1 已定义行为（信封只进下一轮容器、bounded、可被父模型自然消化），且默认 120s 足够长、config/ablation 可关。**薄弱点**：默认值合理性需产品侧确认；计划保持 120（与 CCB 对齐）并暴露开关。
5. **攻击：`RegisterForegroundJob` 单例在"一个 turn 内先 read_only_task 并行再前台 writer task"下，前台注册发生在 writer 分支，无冲突；但 read_only_task 的 Execute 是另一入口（ReadOnlySubagent…），它不会注册前台——是否漏掉"并行只读子 agent 也需后台化"？**
   回答：P4 范围限定**前台 writer task**（任务描述即此）；并行只读子 agent 后台化列为后续增强，计划不越界。
6. **攻击：`StartForegroundHandoffable` 抑制 started Notice 后，前台 job 在状态栏/`Jobs()` 里可见性如何？**
   回答：`Jobs()`/`View` 照常列出（status Running→Done），仅 Notice 与 P1 信封抑制；这对用户是**新增能力**（前台 task 现在可被 `kill_shell`/`wait` 观察），副作用需在文档标注。
7. **攻击：mutationObserver writer 席位（1.2-4）会不会让前台 task 与并行后台 writer 冲突（claim 失败）？**
   回答：`backgroundWriter` 注册是**先占位、后运行**，与后台 writer 的冲突语义本就存在且由 scheduler 写声明（`TryClaimWritePaths`）串行化；前台 job 化只是把"前台运行期"也纳入该席位。**薄弱点**：需 checkpoint 团队对 `CloneForSubagent(recoveryTaskID, turn, backgroundWriter=true)` 的审计含义确认 `[待验证]`。

---

## 6. 缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：仅新增 (a) 切换时 Execute 的 tool result 文本（父会话尾部**追加**，与前台完成同构，非重写）；(b) P1/P3 定稿已定义的既有信封/消息路径（本事务**不改**它们的注入位置、条数、bounded 策略）。
- **前缀稳定**：切换不改子 agent Session、不改父会话历史、不注入额外 host steer；`foregroundEnabled=false` 时前台路径与改造前**字节级等价**（T2.4/T5.2 回归锚）。
- **锁序**：所有新增 `j.mu`/`m.mu` 临界区沿用 jobs.go 既有"不嵌套"纪律（T1 单测 + `-race`）。
- **不越权**：本计划只产出方案；实现由执行小队实施、纪律团审查，plan-2 不宣称任何完成。

---

## 7. 遗留确认项（执行时落定，本计划不臆断）

1. `chat_tui` 运行中输入提交路径（T3.3 拦截点）。
2. config 包 env 覆盖键的具体格式（3.3）。
3. checkpoint `MutationObserver` 对 `background_subagent` writer 席位的前台化含义（1.2-4）。
4. `recordCompletion` 内部结构（是否单函数可加"跳过"分支，需执行时读实现确认 `[jobs.go:570 调用点已见，函数体未实读]`）。
