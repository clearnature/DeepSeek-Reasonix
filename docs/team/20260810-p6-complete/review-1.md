# 纪律审查报告：P6 完成事件驱动（commit 8f76af02f）

> 审查对象：jobs 层完成事件 observer（`WithJobDoneObserver` / `SetJobDoneObserver` /
> `fireJobDoneObservers` / `recordCompletion` 挂点）+ agent 层
> `TeammateStore.HandleJobDone` / `autoAssign` / `assignContext` + boot 生产接线。
> 依据：`docs/team/20260810-p6-complete/arbitration.md`（父代理裁决）、
> `implementation-plan.md`（T1 任务清单）、`test-strategy.md`（D1-D5 硬约束 + J 矩阵）。

---

## 审查项逐条

### 1) [PASS] 仲裁 1/2 落实（3 参回调、同步+recover、m.mu 外挂点）

- **3 参回调** ✓：`WithJobDoneObserver(func(id string, st Status, err error))`
  （jobs.go:328）、`SetJobDoneObserver`（jobs.go:338）签名与仲裁 1 一致；
  挂点只传 `(id, st, err)`，handler 不反查 jm（teammate_store.go:437 经 `ts.tasks`
  判断归属、`pendingDependenciesLocked` 注释明言「Never consult jm.Output」）——
  符合仲裁 1「禁止反查 jm（时序陷阱）」。
- **同步 + recover** ✓：`fireJobDoneObservers`（jobs.go:1057-1071）锁内快照切片、
  锁外同步遍历调用，无 goroutine 分发；per-observer `defer recover`（L1066-1069），
  订阅者 panic 不破坏 job 收尾管线（`close(j.done)` 在 L686，位于挂点之后）。
- **m.mu 外挂点** ✓：suppress 路径 L1008、正常路径 L1034 均在无锁区
  （正常路径 L1029 `m.mu.Unlock()` 之后）。
- **与 RecordDone 并列** ✓：suppress 路径 L1006/L1008 并列、正常路径 L1032/L1034 并列。
- **快照而非活引用** ✓：回调内修改 observer 列表不影响本轮遍历（L1058-1061 拷贝）。

### 2) [FAIL] destroy 窗口语义：正常路径对齐，suppress 路径未对齐（D2 违反）

- 正常路径 ✓：recordCompletion L1013 `m.destroying[parentSession]` 检查 → return，
  observer（L1034）与 RecordDone（L1031）同被吞——与仲裁 6「destroy 窗口对齐
  RecordDone（吞事件）」一致。
- **suppress 路径 ✗**：L1008 `fireJobDoneObservers` 在 destroying 检查**之前**且
  **无 destroying 检查**——silent（`StartSilentForSession`）/foreground job 在
  destroy 窗口内仍触发 observer（以及 L1006 的 RecordDone）。`test-strategy.md` §9
  D2 硬约束明确要求「notifyJobDone 检查 destroying（含 **suppress 分支**——
  注意 recordCompletion L975-977 现无 destroying 检查，须在 notifyJobDone 统一补齐，
  …两种路径一致）」，`implementation-plan.md` L74-75（J8 含 silent 变体）同要求。
  **实现未落实**。
- 后果：`BeginDestroySession` 后正在运行的 silent/foreground job 以 Killed 完成 →
  HandleJobDone 仍被触发 → set Killed + idle flip + mailbox 计数 + `autoAssign`
  （用 `assignContext` 在销毁中的 session 上重建 ctx 派活）——销毁期幽灵活动，
  正是 D2 想消除的。

### 3) [FAIL] Close 后不触发未实现（D2 违反 + J7 测试名不副实）

- `fireJobDoneObservers` **无 `m.root.Done()` 检查**。`test-strategy.md` J7/D2 要求
  「阻塞 job → `Close()`（cancel）→ job 以 Killed 完成 → 订阅者**未收到**
  （notifyJobDone 检查 `m.root.Done()`）」。
- 实际时序：Close 只 cancel root + `wg.Wait()`；job goroutine 因 cancel 以 Killed
  完成 → `recordCompletion`（L675）在 `wg.Done()` 之前执行 → **observer 收到 Killed**。
  对合作 job 该触发发生在 Close 返回前（wg 同步等待，故「Close 返回后无新触发」在
  时间意义上成立）；对超过 teardownGrace 的非合作 job，**Close 返回后仍可能触发**。
  且 Killed 事件已进入 HandleJobDone → 置 idle + autoAssign 幽灵派活（在已关闭的
  jm 上再 Assign，新 job 立即 Killed）——设计意图（J7/D2）明确禁止，实现未满足。
- **测试名不副实**：`TestJobDoneObserverNotFiredAfterClose`
  （jobs_done_observer_test.go:149-167）的场景是「job **正常完成后** Close →
  断言只收到 1 次 + Close 后新装 observer 不触发（此时已无 job 运行，恒真）」——
  完全没有覆盖 J7 要求的「Close 时 job 运行中 → Killed 完成不触发」真实窗口。
  该测试若被当作「Close 语义已验证」的证据，属于虚假安全感。

### 4) [FAIL] 测试质量：6 个测试存在矩阵缺项与弱断言

| J 矩阵项（test-strategy） | 实现状态 |
|---|---|
| J1 FiresOnDone（含**参数一致性**：session/kind/label/err） | 部分：只断言 `id\|status`，capture 未记录/未验证 err、kind、label、session（jobs_done_observer_test.go:20-24） |
| J2 FiresOnFailed | **缺** |
| J3 FiresOnKilled | **缺** |
| J4 FiresOnInvalidStart | **缺** |
| J5 MultipleSubscribers | ✓ |
| J6 PanicIsolated（含「manager 可继续 Start」） | 部分：panic 隔离 + sibling 均验证 ✓，但未验证后续 Start 可用 |
| J7 NotCalledAfterClose（真场景：阻塞 job → Close → Killed 不触发） | **缺**（现有同名测试为弱化版，见审查点 3） |
| J8 NotCalledWhileDestroying（含 **silent 变体**） | **缺**——且实现确实不符合（见审查点 2） |
| J9 SilentAndForegroundStillFire | **缺** |
| J10 NilNoop | ✓ |
| J11 SetAfterConstruction | 部分：Set 后触发 ✓；Set 前已完成的 job 不补发有覆盖（经 m2 nil 清除路径间接）；但 Set **替换**（非 nil 覆盖旧 observer）未测 |
| J12 BlockingStallsWait（同步语义固化） | **缺** |

- 另：SetJobDoneObserver 的「替换（非 nil）」语义（jobs.go:345 注释声明
  「replaces any observers」）无测试覆盖。
- agent 层补充（teammate_store_test.go H 系列 6 测 + teammate_done_e2e_test.go
  4 测）覆盖了 handler 业务分支与全链接线，质量良好；但**均不覆盖** jobs 层
  Close/destroy 的 observer 触发语义，无法弥补 jobs 层缺口。

### 5) [PASS] 缓存红线（发送侧零变化）

- 改动均为进程内回调（jobs→agent）+ `event.Notice`（UI 层，不进 provider 输入）。
- 自动 Assign：`autoAssign` 用 `cur.Prompt`（recordTask 登记时原文，
  teammate_store.go:284-297）重放 → 字节稳定（仲裁 3.4）；fork/continue 前缀逻辑
  （ContextRequest{Fork, Silent} / {ContinueFrom}，teammate_store.go:240-247）未改动。
- `assignContext` 重建最小 ctx（WithManager/WithSession/WithParentSession/
  WithForkSource(leader *Agent)，teammate_store.go:496-510），绝不缓存 ctx 对象
  （仲裁 3.1）。recordTask 补存 `SessionID` string ✓（TeamTask.SessionID，L62）。
- mailbox 唤醒 notice 走 event.Sink（UI）；`SetSink` 接线（boot.go:1906）不改发送字节。
- 未发现 schema/system prompt/transcript/task.go 发送侧触碰。

### 6) [PASS] fable5 合规（执行队是否跳步）

- 执行队覆盖了任务分解（T1 实现 + T4 组合测试 + 生产接线）、拓扑扫描（挂点两处
  无锁点、destroy 窗口、startInvalid 路径）、对抗自检（teammate_store_test.go 的
  StaleJobNoop/Idempotent/未知 job 防御）与 -race 组合测试骨架。
- **跳步点**：D2 硬约束（test-strategy §9，含 J7/J8 两矩阵项）未被实现，且没有
  纪律团对 D2 的豁免裁决记录——按「偏离需显式裁决」要求，执行队擅自放弃了矩阵项。

### 7) [PASS] 幻觉检测（证据真实性）

- 代码真实性：全部挂点、回调、handler、ctx 重建、prompt 重放均可在工作区核对到
  具体行号（见上文）；测试文件真实存在、断言与注释自洽。**无编造行号/编造实现**。
- **证据缺口**：工作区未见执行队的测试输出/`-race` 证据文件（无 execution.md
  证据链），且本审查环境无 shell 执行工具，**无法重跑** `go test`/`gofmt`/`vet`/
  `repolint` 独立验证「测试全绿」声明。静态推演：jobs 层 6 测逻辑自洽、API/import
  均已核实存在，大概率可编译通过；但「全量回归 + -race 全绿」属**未证实声明**，
  需执行队补充证据或复跑。

### 8) 对抗自检（devil's advocate）

攻击发现的附加缺陷：
1. **Close 后的幽灵自动派活链**：Close 时运行中 job → Killed 完成 → HandleJobDone
   → idle flip → autoAssign（若存在 pending 任务且 owner 存在）→ 在已关闭 jm 上
   重新 Assign → 新 job 立即 Killed → 再次触发……虽 autoAssign 成功后
   `delete(ts.tasks, t.ID)` 与 recordTask 的新条目无 DependsOn 会终止链条，但
   每次 Close 都产生 1 次无意义的 fork/continue 派活 + 事件往返。
2. **boot.go 过时注释**（boot.go:1901-1904）：注释声称「TeammateStore.OnJobDone
   has not landed yet (E2)，jm.SetJobDoneObserver 延期」——但 E2（HandleJobDone）
   已落地且 `NewTeammateStore` 构造器内已自动注册（teammate_store.go:110-112）。
   注释与代码矛盾（后半句「E2 may also register inside the constructor」自证），
   会误导后续维护者以为生产未接线。**低严重度文档漂移**。
3. **notifyMailBacklog 幂等缺陷窗口**：`countInbox`（锁外 ReadDir）与
   `flushMailbox` 无互斥，完成事件与下一次 Assign 并发时计数可能读旧值——
   已由「lost notice 由下次 Assign 的 flush 兜底」注释声明为接受边界，且 e2e 测试
   的聚合断言在单线程下成立；判定为已接受风险，不阻塞。
4. **IsDestroying 竞态**：FinishDestroySession 删除 destroying 标志后，迟到完成
   事件不再被吞——但届时任务已 purge，HandleJobDone 经 `ts.tasks` 查无此 id 即
   no-op，幂等兜底成立。

---

## 结论

**驳回**（必须修复后复审）。

驳回理由：`test-strategy.md` §9 D2 硬约束（Close/destroy 后不触发，含 suppress 分支
与 silent 变体）**明确未落实**——`fireJobDoneObservers` 无 `m.root.Done()` 检查，
suppress 路径（L1008）无 destroying 检查；对应 J7（真场景）/J8 测试缺失，且现有
`TestJobDoneObserverNotFiredAfterClose` 名不副实、无法捕获缺陷。这使「完成事件驱动」
在 session 销毁/关闭窗口产生幽灵事件与幽灵派活，破坏事件驱动的生命周期边界。

### 必须修复项

1. **jobs.go**：在 `fireJobDoneObservers`（或统一 `notifyJobDone`）入口检查
   `m.root.Done()` 与 `m.destroying[parentSession]`，使 Close 后 Killed 完成与
   destroy 窗口内（**含 suppress 分支**）的完成均不触发 observer；保持 RecordDone
   现有行为不变（D2 明言「与 taskRecorder 现有行为略有出入」是可接受的设计选择）。
2. **补测试（jobs 层）**：
   - J7 真场景：阻塞 job → `Close()` → 断言 observer **未收到** Killed；
   - J8：阻塞 job → `BeginDestroySession` → `WaitTeardown` → 断言未收到，
     **含 `StartSilentForSession` 变体**；
   - J2/J3/J4：Failed / Killed / startInvalid 各触发恰一次；
   - J1 补参数一致性断言（err、kind、label、session）；
   - J12（同步固化：阻塞回调 → Wait 被延迟）与 Set 替换语义。
3. **清理 boot.go:1901-1904 过时注释**（E2 已落地，构造器内已注册）。
4. **补证据链**：执行队须附 `go test ./internal/jobs/ ./internal/agent/` 与
   `go test -race`（相关包）真实输出，供纪律团复核「全绿」声明。
