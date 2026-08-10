# 纪律审查报告：P6 完成事件驱动 e2e 测试（commit 8f76af02f）

> 审查对象：`internal/agent/teammate_done_e2e_test.go`（4 个 e2e）、
> `internal/agent/teammate_store_test.go`（父代理测试修复）、
> `internal/agent/coordinator_test.go`（mockProvider 加锁）、
> `internal/jobs/jobs.go` + `internal/jobs/jobs_done_observer_test.go`（时序依赖交叉验证）。
> 证据基线：test-strategy.md（§6 组合测试 / §9 D1-D5）、arbitration.md（仲裁 1-7）。
> 审查方式：静态交叉验证（本环境无 shell 工具，无法重跑 `go test`；逐条比对
> 断言与生产代码时序，验证「可重跑即真实」的证据链成立）。

---

## 一、时序依赖是否成立（审查重点 4 的根基）

e2e 全部依赖「`WaitForSession` 返回 ⇒ HandleJobDone 已执行完」这一确定性等待点。

**正常路径（startForSession 的 run goroutine，jobs.go L675→686）**：
`m.recordCompletion(...)` →（内部 L1034）`m.fireJobDoneObservers(...)` 同步执行
（L1057-1071：m.mu 内快照、锁外逐个调用、per-observer recover）→ 返回 →
L686 `close(j.done)`。`WaitForSession`（L1300-1321）阻塞 `<-j.done`。
⇒ **observer 在 close(j.done) 之前完成，WaitForSession 返回时 HandleJobDone 必已执行完。成立。**

**唯一例外**：`startInvalid`（L508-509）是 `close(j.done)` 在 `recordCompletion` 之前，
observer 会在 Wait 返回后才触发。但 e2e 的 job 全部经 `TaskTool.RunProfileSpec` 产生合法
id（`task-N`），不经过 startInvalid。**对 4 个 e2e 的时序断言零影响。**

**e2e 3（mailbox）的附加同步性**：`HandleJobDone` 内 `notifyMailBacklog`（teammate_store.go
L466-470）在锁外同步 `sink.Emit`；测试 sink 为 `FuncSink` 直接 append。WaitForSession 返回
⇒ notice 已入切片。**无轮询、无假阳性。成立。**

**e2e 2（依赖链）的附加同步性**：`autoAssign` 是 HandleJobDone 的同步调用，其内部 `Assign`
经 `RunProfileSpec` 同步注册 jobB（go 起的 run goroutine 是异步，但 job 注册是同步的）。
⇒ WaitForSession(jobA) 返回时 jobB 已存在、LastJobID 已写。`waitForTeammateJob` 的轮询是
防御性写法（注释 L237-238 明言兼容两种时序），实际第一次即命中。**成立。**

---

## 二、审查项逐条

- **[PASS] fable5 合规（执行队是否跳步）**
  本次审查对象是测试产物而非执行过程；无 execution.md 可核对执行证据链。按产物反推：
  测试严格遵循 test-strategy 的矩阵划分——C1-C4 真实异步全链（不直接调 HandleJobDone，
  全部经真实 job 完成事件，验证 NewTeammateStore 自动注册接线 teammate_store.go L111）、
  确定性测试留在 teammate_store_test.go、-race 骨架在并发用例。D1-D5 约束在实现中逐条可查
  （D1 多订阅者 WithJobDoneObserver 追加 / D2 destroy 检查 / D3 per-observer recover /
  D4 同步 / D5 锁外 IO）。未发现跳步迹象。

- **[PASS] 幻觉检测（证据真实性）**
  逐条核对断言与生产代码：
  - e2e 1「Status 不调 List/Tasks 也 Idle」：`Status()`（L170-178）直接返回副本、不触
    `syncStateLocked` ⇒ Idle 只能是事件驱动。**断言有区分度，非懒同步假阳性。**
  - e2e 1 第二次 Assign 放行：WaitForSession(first) 返回 ⇒ handler 已 flip Idle ⇒
    running 守卫不拦。真实。
  - e2e 2 beta 被 gate 拒绝（L96-99）：`pendingDependenciesLocked` 查 tasks 快照
    jobA.Status==Running ⇒ 拒绝文案 "blocked by unfinished dependency"。真实。
  - e2e 3 聚合 N=2：PostMail×2 落盘（inboxRoot）→ countInbox 读磁盘=2 → 通知文案
    "has 2 unread mail"。断言过滤 `"alpha" && "2"`——PostMail 即时通知文案
    "received mail"（L674-675）不含数字，**不会误命中**。真实。
  - e2e 4 终态 Idle：每个 teammate 末轮 WaitForSession 已返回 ⇒ handler 已 flip。真实。
  - 注释「两处无锁点同步触发」（L8-9）：recordCompletion 的 suppress 分支 L1008 与正常分支
    L1034 均调用 fireJobDoneObservers，且 callback 在 m.mu 锁外。**注释与代码一致。**
  - 无法重跑（无 shell）；以上为静态证据链交叉验证，运行确认留执行环境。

- **[PASS] 缓存红线（前缀字节稳定）**
  全部改动为测试代码 + mockProvider 加锁 + jobs observer 机制（进程内回调）。
  无发送侧前缀字节变化、无 schema/system prompt/transcript 改动；自动 Assign 的 prompt
  为登记时原文（recordTask 快照 L62/L292）；mailbox notice 走 event.Sink（UI 层）。
  与 arbitration.md「缓存红线（全队一致）」一致。

- **[PASS] 回归（测试命令 + 结果）**
  本环境无 shell，无法执行 `go test -race ./internal/jobs/ ./internal/agent/`。
  静态回归分析：
  - **mockProvider 加锁（coordinator_test.go L19-49）对既有测试零影响**：Stream 加锁后
    串行行为与原来逐字节一致（写 lastReq/requests/读 chunks 均在锁内）；全部读取方
    （L80/L83 等）为串行测试、Stream 返回后读取，无并发 ⇒ 无新增 -race 报告。**审查重点 3 通过。**
  - **父代理修复 1（并发降 2 teammate）合理**：e2e 4 用 2 个 teammate × 2 轮，每 goroutine
    串行 Assign→Wait，同时最多 2 个 running job < `maxConcurrentBackgroundTasks`=3
    （write_claims.go:18 DefaultMaxParallelWriters=3）。若用更多 teammate 会撞 cap 导致
    Assign 间歇失败（非本测试目标），降到 2 是正确收缩。论证已写入注释 L196-198。
  - **父代理修复 2（DependencyGate 断言 3 条目）合理**：teammate_store_test.go L208 断言
    warmup(beta, Done) + first(alpha, Running) + pending(beta, pending) = 3；L245 断言链终后
    warmup + first + second = 3。两处都与 recordTask/recordPendingTaskLocked 的注册语义吻合。
  - 旧测试锚点（test-strategy §10 所列）均不依赖「List 前不 idle」的旧时序；事件驱动使
    「List 后 idle」断言变强而非冲突。

- **[PASS] 对抗自检（devil's advocate）**
  见下「四、发现」。攻击结果：未发现阻塞性缺陷，5 项轻微发现。

---

## 三、4 个 e2e 对用户价值的真实覆盖（审查重点 1）

| 测试 | 用户价值 | 覆盖判定 |
|------|---------|---------|
| 1. IdleAllowsReassign | 完成事件驱动置 idle（非懒同步），第二次 Assign 不再被 running 守卫拒绝 | **真实覆盖**：Status 直读证明事件驱动；第二次 Assign+Wait 闭环 |
| 2. DependencyChainAutoAdvance | 依赖门登记 pending → 完成事件自动推进 → beta 跑完回 Idle | **真实覆盖**：无 warmup 走 fork 路径（ts.leader 捕获），与旧 TestTeammateDependencyGate（warmup 走 continue 路径）**互补覆盖自动推进两条路径**——好设计 |
| 3. MailboxBacklogWakeup | 运行中 PostMail 积压（flushMailbox 不跑）→ 完成时一条聚合 Notice 含 N=2 | **真实覆盖**：恰好一条聚合、含 "unread mail"、N 计数；PostMail 即时通知不误命中 |
| 4. ConcurrentAssignCompletion | 多 teammate 并发 Assign + 真实完成事件交错，-race 护栏 | **真实覆盖**：2×2 并发交错、终态全 Idle；cap 论证成立 |

## 四、发现（对抗自检结果，均轻微、非阻塞）

1. **轻微 · e2e 3 理论 flake 面**：PostMail 文件名 `%d.json`（UnixNano，teammate_store.go
   L659）两次调用若落在同一纳秒会覆盖 → countInbox=1 → 断言失败。概率为 ns 级，可接受；
   建议未来改用原子计数器或加入纳秒+序号。
2. **轻微 · e2e 4 隐式依赖常量值**：cap 论证依赖 `DefaultMaxParallelWriters`=3 的**当前值**。
   若未来降到 2，本测试将间歇红。注释已写明论证，建议在常量处加交叉引用。
3. **矩阵缺口（非 e2e 缺陷）**：jobs_done_observer_test.go 已实现 7 项，但 test-strategy 的
   J4（invalid start）、J8（destroy 窗口）、J9（silent/foreground）、**J12（阻塞订阅者固化
   同步语义）缺失**。J12 是 D4 确定性等待点的唯一**负向**固化——当前同步语义仅靠代码审查
   保证；若实现未来改 goroutine 分发，e2e 的「WaitForSession 返回 ⇒ handler 已执行」将静默
   失效（测试变慢但不一定红）。建议补 J12。
4. **轻微 · e2e 2 的「顺序」断言偏隐式**：注释声称「jobA 在 jobB 之前终态」，实际通过
   waitForTeammateJob 的 LastJobID 出现时机隐含断言（WaitForSession(jobA) 返回 ⇒ jobB 已
   启动）。顺序性事实成立，但无显式时刻记录——可接受。
5. **边界提醒 · SetJobDoneObserver 是替换语义**（jobs.go L336-345，非追加）：若一个 jm 被
   多个 TeammateStore 共享，后建者会覆盖先建者的 observer。测试内每 jm 一 store，不触发；
   生产 boot 若多 store 共 jm 需注意（与 D1 的 With 追加对称，属设计内）。

---

## 结论

**通过**（条件性：静态交叉验证；运行确认留执行环境执行
`go test -race ./internal/jobs/ ./internal/agent/`）。

驳回理由不成立——未发现阻塞项。必须修复项：无。建议项（非阻断）：补 J12 同步语义负向
固化（发现 3）；留意 PostMail 文件名碰撞与 cap 常量值两个理论 flake 面（发现 1/2）。

- 4 个 e2e 全部经真实 job 全链驱动，覆盖 idle 复派、依赖自动推进（fork+continue 双路径）、
  mailbox 聚合 N、并发交错四项用户价值；断言均与生产代码时序逐条核对成立，无虚假断言。
- mockProvider 加锁对既有测试零影响（串行行为逐字节一致）。
- 父代理两处修复（并发降 2、DependencyGate 3 条目）均合理且论证充分。
- 时序依赖（WaitForSession ⇒ handler 已执行）在正常 job 路径上由
  recordCompletion→fireJobDoneObservers→close(j.done) 同 goroutine 串行结构保证，成立。
