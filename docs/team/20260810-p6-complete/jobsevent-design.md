# 设计：jobs 层「Job 完成事件」多订阅者机制（jobsevent）

> 角色：**规划 Planner**（独立设计，只读，零实现代码）
> 事务：`docs/team/20260810-p6-complete/` · 基线：`internal/jobs/jobs.go`（`recordCompletion` L937 / `WithJobStartObserver` L296 / `WithTaskRecorder` L307 / `SetTaskRecorder` L316）、`internal/jobs/jobs_recorder_test.go`、`jobs_silent_test.go`、`internal/control/controller.go:695`（taskmonitor 占用 TaskRecorder 单槽）、`internal/agent/teammate_store.go`（消费方）
> 对象：给 `internal/jobs` 新增**多订阅者**「Job 完成事件」接口，供 TeammateStore 等消费；**不得抢 TaskRecorder 单槽**（controller.go:695 taskmonitor 已占）、不得抢 `WithJobStartObserver`（boot.go:517 workspaceLease 已占）
> 实现与验证由执行小队承担、纪律团复核（三职能硬性分离；本文件只产出设计）

---

## 0. 结论摘要（决策 → 结论 → 代码锚点）

| # | 决策点 | 结论 | 代码锚点 |
|---|--------|------|---------|
| D1 | 多订阅者存储 | **切片 + append**；`Manager` 新字段 `jobDoneObservers []JobCompletionObserver`，`m.mu` 保护读写 | jobs.go:205-221（m.mu 现役）、NewManager L324-350 |
| D2 | API 形态 | **struct 式回调** `JobCompletionObserver func(JobCompletion)`；字段 `{SessionID, ID, Kind, Label, Status, Err}` 与 implementation-plan 6 参**集合完全一致**（演进，见 §8 偏差表） | 新增导出类型；既有 `completion`（L253-258）是内部 drain summary，不冲突 |
| D3 | 三个入口 | `WithJobDoneObserver(...)`（构造期 Option，可多次调用**追加**）+ `AddJobDoneObserver(fn)`（运行时追加）+ `SetJobDoneObserver(fn)`（清空后设单一，nil=清空；对称 `SetTaskRecorder` L316） | 与 D1（test-strategy）「Set 清空后设单一」一致；支撑 J11 |
| D4 | 触发点 | 私有 `notifyTerminal(id, kind, label, st, err)`：**先 recorder 后 observer**；挂接 `recordCompletion` **两处无锁点**（suppressEnvelope 分支 L975-977 旁、正常分支 L1000-1002 旁），替换现有两处 `taskRecorder.RecordDone` 调用为统一入口 | jobs.go:975-977 / 1000-1002；两处均在 m.mu 释放后（锁序纪律见 §3.2） |
| D5 | 调用上下文 | **同步投递**（回调跑在 job 收尾 goroutine：run goroutine L645，或 `startInvalid` 的调用者 L479）；**不引入分发 goroutine**。与 `TaskRecorder.RecordDone` 同 goroutine 同序（test-strategy D4/J12 固化语义） | jobs.go:585-657（run goroutine 时序） |
| D6 | 与 RecordDone 次序 | **recorder 先、observer 后**（结构化于 `notifyTerminal` 内，不靠调用点顺序保证） | jobs.go:1000-1002 |
| D7 | 防 panic | 遍历分发时 **per-observer recover 隔离**：一个订阅者 panic 不影响其余订阅者、不影响 job 收尾、不影响 manager 后续使用 | 新增私有 `callDoneObservers` |
| D8 | 生命周期 / Close | **Close 后仍触发**（对齐 `TaskRecorder.RecordDone` 既有语义：Close→cancel→run 以 Killed 收尾→recordCompletion）；与 test-strategy J7「Close 后不通知」**偏差**，显式标注交纪律团仲裁（§8） | jobs.go:2138-2143（Close） |
| D9 | 抑制面 | 与 RecordDone 完全对齐：silent/foreground 作业**仍触发**；`destroying` 窗口提前 return **不触发**；`recordStalled` 不触发；loaded tombstone 恢复路径不触发 | jobs.go:963-978 / 982-985 / 1018-1050；recordCompletion 全仓仅两处调用（L479、L645，已 grep 核实） |
| D10 | 权威性契约 | 回调**必须信任 `JobCompletion.Status` 参数**，禁反查 `jm.Output/Status`（触发时 `j.status` 仍 Running、`j.done` 未关） | jobs.go:645（recordCompletion）→ :647-655（status 发布）→ :656（close(j.done)） |

**一条总纲**：完成事件 =「终态参数直通 + 同步轻回调 + recorder 先行 + 幂等消费者」。越过此边界的做法（回调内 Wait 自己、回调内反查 Output 依赖终态、回调内做重活、异步分发）均为设计否决项或显式风险项。

---

## 1. 拓扑扫描

### 1.1 文件依赖图

```
internal/jobs/jobs.go
 ├─ Manager 字段 :192-221（m.mu 保护 jobs/order/completed/…；taskRecorder 单槽 :220）
 ├─ Status :49-56（running/done/failed/killed/interrupted）
 ├─ Option :260-261、WithJobStartObserver :296（单例）、WithTaskRecorder :307、SetTaskRecorder :316（单槽，无锁写）
 ├─ recordCompletion :937-1016（终态唯一出口；两处调用 L479 / L645）
 │    ├─ j.mu 非消费快照 + 信封预渲染 :951-961
 │    ├─ suppressEnvelope 分支 :963-978 → RecordDone :975-977 → return
 │    ├─ destroying 窗口 :982-985 提前 return（不 append、不 RecordDone）
 │    ├─ m.mu 内 append m.completed :990-995
 │    └─ 正常分支 RecordDone :1000-1002 → closing Notice :1013-1015
 ├─ run goroutine 时序 :585-657（recordCompletion :645 → status 发布 :647-655 → close(j.done) :656）
 ├─ startInvalid :454-481（close(j.done) :478 → recordCompletion :479，同步于调用者）
 ├─ recordStalled :1018-1050（非终态，不触发）
 ├─ Close/CloseWithGrace :2138-2176（cancel → wg.Wait）
 └─ 待新增：JobCompletion / JobCompletionObserver / WithJobDoneObserver / AddJobDoneObserver / SetJobDoneObserver / notifyTerminal / callDoneObservers

internal/jobs/jobs_recorder_test.go（recordingRecorder 范式 :13-38；nil no-op :118-128；Set 后装配 :101-116）
internal/jobs/jobs_silent_test.go（silent 仍触发 recorder :71-90 —— 完成事件同构测试范式）
internal/control/controller.go:695（SetTaskRecorder(taskmonitor…) 占用单槽 —— 新机制独立字段，不抢）
internal/agent/teammate_store.go（消费方：syncStateLocked :131 懒同步；Complete :302 无调用者；DestroyAll :460）
internal/event/fanout.go:7-25（既有多 sink 扇出先例：nil 跳过、保序 —— 订阅者遍历同构）
docs/team/20260810-p6-complete/{implementation-plan.md,test-strategy.md,risk-review.md,mailbox-design.md}（纪律基线：D1 多订阅者 / D4 同步语义 / G2 m.mu 外回调）
```

### 1.2 级联风险（设计必须覆盖）

- **R-a 时序陷阱（最高）**：observer 触发时 `j.status` 仍 Running、`j.done` 未关（recordCompletion :645 早于 status 发布 :647-655 与 close :656）。任何"回调内反查 Output/Wait"都会读到错误状态或死锁 → D10 权威性契约 + 测试固化。
- **R-b 锁序**：`m.mu → j.mu` 从不嵌套（jobs.go 多处注释）。notifyTerminal 只能挂 m.mu 已释放的无锁点；observer 遍历不得持任何 manager/job 锁 → D4 触发点 + G2。
- **R-c 单例覆盖**：若沿用单槽 Set，第二个消费者会覆盖 TeammateStore → 完成事件静默失效（interaction-design R-c）。多订阅者切片根治 → D1/D3。
- **R-d 死锁重入**：observer 同步执行在 run goroutine，回调内对**同一 job** 调 `m.Wait` → 等 `<-j.done` 而 close 在回调返回后才执行 → 死锁。文档三禁 + J12 语义固化。
- **R-e 阻塞收尾**：observer 阻塞延迟 `close(j.done)`（:656）→ 拖死 Wait / ClaimForegroundResult / onJobStart 的 workspace 租约释放（interaction-design R-e）。同步方案下以契约约束（快速返回）+ recover 兜底 panic；重活消费者自备 goroutine 适配。
- **R-f 并发注册竞态**：`SetTaskRecorder` 是**无锁写**（L316），分发时无锁读（既有数据竞争面，不在本设计修复范围，但新字段绝不能再裸读写）→ `jobDoneObservers` 读写一律走 `m.mu`（R2 并发测试覆盖）。
- **R-g Close 语义分歧**：test-strategy J7 设计为「Close 后不通知」（检查 `m.root.Done()`）；本设计裁决「Close 后仍通知」对齐 RecordDone → 偏差交纪律团仲裁（§8），两套测试断言互斥，执行前必须裁决定稿。

---

## 2. 多路径推演

### 方案 A（采纳）：多订阅者同步回调 + recover，切片存储
- **API**：`JobCompletion` struct + `JobCompletionObserver func(JobCompletion)`；`WithJobDoneObserver`（追加）/ `AddJobDoneObserver`（追加）/ `SetJobDoneObserver`（清空+设单一）；私有 `notifyTerminal`（先 recorder 后 observer）+ `callDoneObservers`（per-observer recover）；`recordCompletion` 两处无锁点挂接。
- 复杂度：低-中（jobs 层约 +35 行、两处调用点替换）。
- 性能：O(N) 同步（N=订阅者数，个位数）；零 goroutine、零队列、零背压问题。
- 可维护性：与 `WithJobStartObserver`/`WithTaskRecorder` 对称；struct 化回调便于未来加字段；多订阅者根治单例覆盖（R-c）。
- 风险：回调必须快速返回（文档三禁 + J12 固化）；重活消费者需自备异步适配。

### 方案 B（否决为主选，记录代价）：内部 channel + 单分发 goroutine（保序）
- **API**：`notifyTerminal` 改为非阻塞 enqueue（如 `completionCh chan JobCompletion` 容量 64），单 worker 消费并遍历订阅者。
- 复杂度：中-高。需要解决三件难事：
  1. **背压**：channel 满时阻塞 enqueue 违背初衷、丢弃则破坏 TeammateStore 状态一致性 → 必须无限缓冲或溢出协议，均不可接受；
  2. **Close drain 语义**：Close 时队列残留事件 drain 还是丢弃？drain 拖慢 Close、丢弃破坏订阅者状态 → 语义模糊（本设计 D8 明确后即为多余复杂度）；
  3. **次序**：recorder 同步、observer 异步 → 「先 recorder 后 observer」跨 goroutine 无法严格保证，D6 失效。
- 性能：异步延迟 + 每事件一次 channel 往返；测试需轮询等待，时序不稳。
- 风险：R-g 放大（Close 后队列处置）、测试 flake。**否决为主选**；若未来出现重量级消费者，由消费者在回调内自起 goroutine 适配（jobs 层保持可预测同步语义），或届时重新评估。

### 方案 C（否决）：扩展现有 `TaskRecorder` 接口或 `WithJobStartObserver`
- 复用 TaskRecorder 单槽 / 给 RecordDone 加参：职责混淆（监控 vs 业务回调），强制 taskmonitor 同步改动，且**直接违反任务红线**（controller.go:695 taskmonitor 已占）；复用 WithJobStartObserver：单例 + done-channel 拿不到 st/err + `startInvalid` done 提前关闭有错过窗口。否决。

**裁决**：方案 A。与 test-strategy D1（多订阅者）/D4（同步语义固化）一致，与 TaskRecorder 先例同构，最小侵入。

---

## 3. 精确 API 与触发点

### 3.1 新增导出类型与函数签名（实现草案，交由执行小队落码）

```go
// JobCompletion carries the terminal transition of one background job.
// The fields are authoritative: consumers must trust Status/Err and must not
// re-query the manager for the job's state (the job's in-memory status is
// still Running and its done channel is not yet closed while this callback
// runs — see recordCompletion call order in startForSession).
type JobCompletion struct {
    SessionID string // parentSession; "" for unscoped jobs
    ID        string
    Kind      string
    Label     string
    Status    Status
    Err       error
}

// JobCompletionObserver receives every terminal job transition. Observers run
// synchronously on the job's finishing goroutine (the run goroutine, or the
// caller's goroutine for a startInvalid job), in registration order, after
// the TaskRecorder. Implementations must:
//   - return quickly (a slow observer delays close(j.done) and blocks Wait),
//   - never call Wait on the finishing job (deadlock: done closes after the
//     callbacks return),
//   - never re-query Output/Status to derive the terminal state (Status in
//     JobCompletion is authoritative; the job still reads Running here),
//   - be safe for concurrent use and idempotent (best-effort, like
//     TaskRecorder). A panicking observer is isolated per observer and never
//     fails the job pipeline.
type JobCompletionObserver func(JobCompletion)

// WithJobDoneObserver appends one completion observer. Call multiple times to
// register several; all receive every terminal transition in registration
// order. Independent of WithTaskRecorder — both may be used together.
func WithJobDoneObserver(observer JobCompletionObserver) Option

// AddJobDoneObserver appends one completion observer after construction.
// Observers registered after a job's completion never see that job.
func (m *Manager) AddJobDoneObserver(observer JobCompletionObserver)

// SetJobDoneObserver replaces the observer set with a single observer
// (nil clears all). Symmetric with SetTaskRecorder.
func (m *Manager) SetJobDoneObserver(observer JobCompletionObserver)
```

### 3.2 存储与锁

- `Manager` 新增字段（`:220 taskRecorder` 旁）：`jobDoneObservers []JobCompletionObserver`，**读写一律在 `m.mu` 下**（`Add`/`Set` 上锁 append/替换；分发前上锁做**快照拷贝**、解锁后遍历）。
- 分发遍历**不得持有 m.mu**：快照在锁内、调用在锁外 → 订阅者回调内可安全 `m.Start` / `m.AddJobDoneObserver` / `m.SetTaskRecorder`（无嵌套锁死锁）。
- nil 元素跳过（与 `event.FanOut` L18-20 同构）。

### 3.3 触发点：`notifyTerminal` 与 recordCompletion 改造

```go
// notifyTerminal: recorder first, then completion observers. Called only at
// the two lock-free points of recordCompletion (both after m.mu release).
func (m *Manager) notifyTerminal(id, kind, label string, st Status, err error) {
    if !nilutil.IsNil(m.taskRecorder) {
        m.taskRecorder.RecordDone(id, st, err)   // ① recorder 先（既有行为，不动语义）
    }
    m.mu.Lock()
    obs := append([]JobCompletionObserver(nil), m.jobDoneObservers...) // ② 快照
    m.mu.Unlock()
    m.callDoneObservers(obs, JobCompletion{...}) // ③ observer 后，m.mu 外
}
```

- **替换点**：`recordCompletion` 两处现有 `taskRecorder.RecordDone` 调用（L975-977 与 L1000-1002）替换为 `m.notifyTerminal(id, kind, label, st, err)` → 两处共享同一入口，「先 recorder 后 observer」**结构化保证**而非靠调用点顺序。
- **不新增第三个触发面**：destroying 提前 return（L982-985）在 notifyTerminal 之前 → 不触发（与 RecordDone 一致）；`recordStalled`（L1018）非终态不触发；loaded tombstone 恢复路径（L1963-1981）不调 recordCompletion，天然不触发。
- **触发时序全景**（run goroutine，L645-656）：`recordCompletion`（信封入队 + notifyTerminal：RecordDone → observers）→ `j.status` 置终态（L649）→ `close(j.done)`（L656）。observer 收 `Status=Done/Failed/Killed`（参数）时 `j.status` 字段仍 Running → D10 契约的来源。
- **startInvalid**（L454-481）：`close(j.done)`（L478）→ `recordCompletion`（L479，同步于调用者 goroutine）→ observer 收到 `Failed`。与 run 路径的 j.done 状态相反（已关）——**两种路径下回调都不得依赖 j.done**，只依赖参数。

### 3.4 抑制面矩阵（与 `TaskRecorder.RecordDone` 逐格对齐）

| 场景 | RecordDone | 完成事件 | 说明 |
|------|-----------|---------|------|
| 正常 job（Done/Failed/Killed） | ✓ | ✓ | 唯一出口 recordCompletion |
| silent（P5 fork，L963-978 分支） | ✓ | ✓ | StartSilentStillFiresTaskRecorder 同构 |
| foreground（claim 分支） | ✓ | ✓ | 同上 |
| startInvalid（L479） | ✓（miss 时 taskmonitor noop） | ✓ | 消费者容忍未知 ID（幂等） |
| destroying 窗口（L982-985） | ✗ | ✗ | 与既有语义一致，非回归 |
| recordStalled（L1018） | ✗ | ✗ | 非终态 |
| loaded tombstone 恢复 | ✗ | ✗ | 不经过 recordCompletion |
| Close 后收尾的 job | ✓（既有行为） | ✓（本设计，见 §4/§8 仲裁） | 对齐 RecordDone |

---

## 4. 生命周期：Close 语义

- 现状：`Close`（L2138）→ `cancel()` → 正在运行的 run goroutine 因 `ctx.Err()` 计算 `st=Killed`（L594-595）→ 仍走 `recordCompletion`（L645）→ `RecordDone` 收到 Killed（**既有事实**，taskmonitor 在 Close 路径写 Killed）。
- 本设计：完成事件与 RecordDone 同点触发 → **Close 后仍触发（Killed）**。语义自洽：Close 不产生新抑制面，订阅者（如 TeammateStore）对未知/陈旧 ID 幂等 no-op，事件无害且与监控一致。
- **偏差**：test-strategy J7 设计「Close 后不通知（notifyJobDone 检查 `m.root.Done()`）」。本设计不引入第三条抑制规则，**裁决仍触发**，偏差显式交纪律团仲裁（§8）。仲裁通过后 J7 断言改为「Close 后 job 以 Killed 完成 → 订阅者收到 Killed」。
- 对异步方案的附加收益：同步投递下不存在「Close 后队列残留事件」的歧义（方案 B 代价之一消失）。

---

## 5. 防重入 / 防 panic

1. **per-observer recover**：`callDoneObservers` 内对每个订阅者包独立 `defer recover`——一个订阅者 panic 不阻断后续订阅者、不打穿 run goroutine、不影响 `close(j.done)`。
2. **m.mu 外遍历**：回调内可安全重入 manager 公开 API（Start/Add/Set/Kill）；唯二禁区见文档三禁（Wait 自己 / 反查 Output 依赖终态 / 重活阻塞）。
3. **快照而非活引用**：遍历的是锁内拷贝的切片——订阅者回调内调用 `AddJobDoneObserver`/`SetJobDoneObserver` 修改列表不影响本轮遍历（无迭代器失效问题）。

---

## 6. 测试设计（`internal/jobs/jobs_done_observer_test.go` 新增，范式对齐 jobs_recorder_test.go）

| # | 测试 | 断言 | 覆盖决策 |
|---|------|------|---------|
| J1 | TestJobDoneObserverFiresOnDone | StartForSession → Wait → 回调恰一次，`{SessionID,ID,Kind,Label,Status=Done}` 与 `*Job` 字段一致 | D4/D10 |
| J2 | TestJobDoneObserverFiresOnFailed | run 返回 error → `Status=Failed`、`Err` 非 nil、恰一次 | D4 |
| J3 | TestJobDoneObserverFiresOnKilled | run 阻塞 → Kill → Wait → `Status=Killed` 恰一次（终态出口唯一） | D4 |
| J4 | TestJobDoneObserverFiresOnInvalidStart | parentSession 含 `/` → startInvalid → 回调收到 Failed | D4 |
| J5 | TestJobDoneObserverMultipleSubscribers | 注册 3 个 → 同一事件全部收到、**注册顺序**一致（快照保序） | D1/D3 |
| J6 | TestJobDoneObserverPanicIsolated | 订阅者 A panic、B 正常 → B 收到；job Done；Wait 正常返回；manager 可继续 Start | D7 |
| J7 | TestJobDoneObserverAfterClose | 阻塞 job → Close → job 以 Killed 完成 → **订阅者收到 Killed**（对齐 RecordDone；J7 语义以 §8 仲裁为准） | D8 |
| J8 | TestJobDoneObserverNotCalledWhileDestroying | 阻塞 job → BeginDestroySession → WaitTeardown → 订阅者未收到（silent 变体同） | D9 |
| J9 | TestJobDoneObserverSilentAndForegroundStillFire | silent/foreground job 完成 → 回调触发（同 TestStartSilentStillFiresTaskRecorder 同构） | D9 |
| J10 | TestJobDoneObserverNilNoop | 无订阅者：零开销、不 panic、行为与现状逐字节一致 | — |
| J11 | TestSetJobDoneObserverAfterConstruction | Set 后启动的 job 触发；Set 前已完成的 job 不补发（时序固化）；Set(nil) 清空后不触发 | D3 |
| J12 | TestJobDoneObserverBlockingStallsWait（文档化语义固化） | 订阅者阻塞 `<-release` → Wait 被拖延（**证明同步投递**，防实现擅自改 goroutine 分发） | D5/D4 纪律 |
| R2 | TestJobDoneObserverConcurrentStartDone（-race） | 10 个 job 并发 start/done + 并发 AddJobDoneObserver → 每 job 每订阅者恰一次、无数据竞争 | D1 并发安全 |
| 回归 | `go test ./internal/jobs/` 零失败 | 重点 TestDrainMultiple -race 不 flake（recordCompletion 改造的回归面） | — |

---

## 7. 对抗自检（devil's advocate）

1. **攻击：「同步回调是偷懒，重活消费者迟早踩坑」**——回应：jobs 层同步契约可预测且与 RecordDone 同构；重活由消费者自备 goroutine 适配（责任在消费侧，jobs 层不替消费者做生命周期决策）。若未来验证背压确实成问题，再引入显式异步适配器（带 drain 语义），但**不得改动本同步契约**（J12 固化）。
2. **攻击：「notifyTerminal 里先读 taskRecorder 再锁 m.mu 快照，`SetTaskRecorder` 无锁写仍是竞态」**——属实，但那是**既有面**（L316/L1000 现状），本设计不扩大它；新字段全走 m.mu。可选加固（taskRecorder 读纳入 m.mu）列为执行小队低优先级选项，不阻塞本机制。
3. **攻击：「destroying 窗口吞事件会让 TeammateStore 状态滞留 running」**——回应：与 RecordDone 既有语义一致（非回归）；TeammateStore 有 `DestroyAll`（teammate_store.go:460）与懒同步兜底（syncStateLocked :131）。文档明示消费者须幂等。
4. **攻击：「struct 式 JobCompletion 偏离 implementation-plan 的 6 参签名，接线文档全要改」**——回应：参数集合完全一致（SessionID=parentSession），仅形态演进；struct 免去未来加字段（如 Ref）时的签名破坏。已列 §8 偏差表交纪律团确认。
5. **攻击：「J7 测试与 test-strategy 冲突，先写哪套？」**——回应：**执行前必须先完成 §8 仲裁**；设计已给裁决（仍触发）+ 理由（对齐 RecordDone、零新抑制面、幂等无害），纪律团若否，回退为「notifyTerminal 入口检查 `m.root.Done()`」且 J1-J6/J8-J12 不受影响（改动面收敛在 notifyTerminal 一行）。
6. **攻击：「AddJobDoneObserver 与 WithJobDoneObserver 语义重复，且 Option 里 append 多份如何测试？」**——回应：Option 期 append 与运行期 append 走同一切片（构造期无并发故 Option 内可不持锁，直接赋值语义）；J5 覆盖构造期多订阅者，R2 覆盖运行期并发。
7. **薄弱环节标注**：① D8/J7 仲裁未决是唯一阻塞项；② `notifyTerminal` 替换两处 RecordDone 时若漏改一处 → silent/foreground 作业静默丢事件（J9 覆盖）；③ 锁内快照若误用活引用遍历 → 迭代器失效/竞态（R2 覆盖）。

---

## 8. 与既有 p6-complete 文档的偏差表（交纪律团仲裁）

| 文档 | 原设计 | 本设计 | 偏差性质 |
|------|--------|--------|---------|
| implementation-plan T1 | 单例 `WithJobDoneObserver` + `SetJobDoneObserver`（func 6 参） | **多订阅者**切片 + `With`（追加）/`Add`/`Set`（清空+单一）；struct 式 `JobCompletion` | test-strategy D1 已升级多订阅者（本设计采纳 D1）；struct 化为形态演进，参数集合不变 |
| test-strategy J7 | Close 后**不通知**（notifyJobDone 检查 `m.root.Done()`） | Close 后**仍通知**（对齐 RecordDone 既有行为） | **实质冲突**，见 §4 理由；需仲裁 |
| test-strategy D4/J12 | 同步投递固化 | 采纳（本设计 D5 同步 + J12 固化） | 一致 |
| risk-review G1 | 完成事件 + teammate 有界队列（128）+ 单 worker | jobs 层同步；**异步队列责任下沉到消费者**（teammate 侧队列为消费方实现细节，不在本设计范围） | 范围边界说明：本设计只管 jobs 层机制；teammate 侧队列由 T2 自行裁决 |
| mailbox-design §11 | 不新增第二个全局钩子 | 新增独立多订阅者完成事件字段（非单槽钩子，不抢 TaskRecorder/JobStartObserver） | 不冲突：独立字段即「非抢槽」实现 |

---

## 9. 缓存/纪律检查点（Reasonix 领域）

- **发送侧字节变化**：零。本机制是纯 in-process 回调，不新增任何 Notice/envelope/事件注入（`recordCompletion` 的 m.completed 入队与 closing Notice 输出**逐字节不变**）。
- **前缀稳定**：`JobCompletion`/`JobCompletionObserver`/`WithJobDoneObserver`/`AddJobDoneObserver`/`SetJobDoneObserver`/`notifyTerminal`/`callDoneObservers` 均为新增符号，不修改既有导出符号；`recordCompletion` 内部两处调用点替换为 `notifyTerminal`，行为不变（recorder 先、顺序同）。
- **验证**：J10（nil no-op 逐字节一致）+ 全量 `go test ./internal/jobs/` 回归 + `-race`（R2）。无模型可见性影响。
