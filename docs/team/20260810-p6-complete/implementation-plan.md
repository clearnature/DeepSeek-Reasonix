# 计划：「完成事件驱动」实现计划（交付 executor 队的执行依据）

> 事务：docs/team/20260810-p6-complete/ · 视角：集成者（独立评审，只读规划，不写代码）
> 主题：把 TeammateStore 从**懒同步**（List/Tasks 时查 `jm.Output`）升级为**完成事件驱动**（job 终态 → jobs 层回调 → TeammateStore handler 主动置 idle + 依赖推进 + mailbox 唤醒 + tasks 状态补存）
> 章程：planner 出计划 → executor 执行 → discipline 审查（三职能分离，本文件只产出计划）

---

## 一、拓扑扫描

### 1.1 涉及文件与现状事实（已核实）

| 文件 | 现状关键点 | 本次改动 |
|------|-----------|---------|
| `internal/jobs/jobs.go` | `recordCompletion`（L937）是**终态唯一出口**（两处调用：`startInvalid` L479、run goroutine L645）；`destroying[parentSession]` 时提前 return（L982）；`taskRecorder.RecordDone` 已在**两处无锁点**调用（suppressEnvelope 分支 L975-977、正常分支 L1000-1002，均在 m.mu 释放后）；`WithJobStartObserver`（L296）是单例 Option；`SetTaskRecorder`（L316）提供构造后装配模式 | 新增 `WithJobDoneObserver` + `SetJobDoneObserver`，在 `recordCompletion` 两处无锁点并列接线 |
| `internal/agent/teammate_store.go` | `syncStateLocked`（L131）懒同步置 idle（注释明言"no completion callback needed for the MVP"）；`recordTask`（L234）只存 ID/Owner/DependsOn，Status 恒空串；`Tasks()`（L265）Status 纯派生自 `jm.Output`，jm purge 后状态丢失；`notifyMail`（L402）依赖 sink；`Complete`（L302）无调用者（遗留 API）；`pendingDependenciesLocked`（L243）Assign 时懒查 | 新增 `OnJobDone` handler（置 idle + tasks 补存 + mailbox 唤醒）；`recordTask` 补存 Status 初值；Tasks() 兜底逻辑复用补存值 |
| `internal/boot/boot.go` | L1930 `agent.NewTeammateStore(taskTool, jm)` 直接内联进 Options；**未调用 `SetSink`** → 生产路径 mailbox 唤醒 notice 实际不生效（P6.2 接线缺口） | 拆出变量 → `ts.SetSink(sink)` + `jm.SetJobDoneObserver(ts.OnJobDone)` |
| `internal/control/controller.go` | `applyTeamCommand`（L6345）无改动需求；Options.Sink（L495）可作兜底接线点 | 可选小改（见 T3b） |
| `internal/jobs/jobs_test.go` | L84 `TestJobStartObserverSeesLifetimeUntilTerminal` 是 start-observer 测试范式 | 新增 done-observer 测试参考此范式 |
| `internal/jobs/jobs_recorder_test.go` | `TaskRecorder` 测试范式（构造后装配、nil no-op、静默 job 仍触发） | done-observer 测试同构 |

### 1.2 关键事实确认（防止执行踩坑）

1. **终态出口唯一**：`Kill`/`KillForSession`（L1211）只 cancel + 同步置 Killed，终态记录仍在 run goroutine 的 `recordCompletion` 完成 → observer 只需接在 `recordCompletion`。
2. **锁序纪律**：`m.mu → j.mu` 从不嵌套（jobs.go 多处注释明示）。observer 回调**只能在两处已释放全部 job 锁的位置**调用（与 `taskRecorder.RecordDone` 并列）。
3. **时序窗口**：run goroutine 是 `recordCompletion` → `close(j.done)`（L645→L656）；`startInvalid` 是 `close(j.done)` → `recordCompletion`（L478→L479）。**handler 不得依赖 j.done 状态**，只依赖 st 参数。
4. **destroy 边界**：`destroying[parentSession]` 时 `recordCompletion` 提前 return → observer 不触发（与 taskRecorder 既有语义一致，非回归）；TeammateStore 有 `DestroyAll` 兜底。
5. **生产 sink 缺口**：全仓 `SetSink` 只出现在两个测试文件（teammate_store_test.go、team_real_test.go）→ 生产 mailbox 唤醒从未接线，本次一并修复。
6. **test-strategy 文档**：仓内无 `test-strategy` 文档（已 grep 确认）→ 测试清单按分层约定（单元/集成/e2e/手动）映射，沿用事务既有「四验证」（gofmt/vet/repolint/test）。

### 1.3 级联风险

- **R1（死锁）**：observer 误在锁内调用 → `ts.mu → jm 锁` 逆序。缓解：调用点固定在 `recordCompletion` 两处无锁点；handler 内先释放 ts.mu 再做 IO/通知。
- **R2（阻塞/panic 破坏管线）**：observer 阻塞会延迟 `close(j.done)` 拖死 Wait；panic 会打穿 run goroutine。缓解：jobs 层包一层 recover；handler 只做 O(1) 内存操作，ReadDir 检查放 ts.mu 外。
- **R3（旧 job 完成误置 idle）**：teammate 连续两 job，旧 job 完成事件晚到 → 必须 `LastJobID == id` 严格匹配才置 idle（同 `Complete` L309-311 语义）。
- **R4（双真源漂移）**：tasks.Status 补存值 vs `jm.Output` 派生值。缓解：补存值来自 jm 终态快照且写入后不可变；`Tasks()` 保持「jm 优先、补存兜底」顺序确定。
- **R5（mailbox 重复通知噪音）**：PostMail 已即时 notifyMail，完成事件再查积压可能二次通知。缓解：`flushMailbox` 已清空 inbox，完成事件检查到的积压必然是未 flush 的新 mail，语义不重复。

---

## 二、多路径推演

### 方案 A（采纳）：jobs 层单例完成观察者，同步回调 + recover
- **API**：`WithJobDoneObserver(func(parentSession, id, kind, label string, st Status, err error))` Option + `SetJobDoneObserver(...)` 方法（构造后可装/换，对称 `SetTaskRecorder`）。`recordCompletion` 两处无锁点通过私有 `notifyJobDone(...)` 统一调用（nil 检查 + recover）。
- 复杂度：低-中（jobs +~20 行，teammate +~45 行，boot +3 行）。
- 性能：O(1) 同步、零 goroutine、零队列；handler 为内存操作。
- 可维护性：与 `WithJobStartObserver` 对称；回调直接携带终态参数，消费者无需二次查询。
- 风险：回调必须快速返回（recover + 文档化约束）；单消费者场景够用。

### 方案 B（备选）：done-channel 分发
- **API**：`WithJobDoneObserver(func(done <-chan struct{}))`，observer 自起 goroutine 监听 done 后查 `jm.Output` 拿状态。
- 复杂度：中。与 `WithJobStartObserver` 签名完全一致，但**拿不到 st/err**；`startInvalid` 的 done 已提前关闭存在错过窗口；`recordCompletion`→`close(done)` 的间隔使 Wait 先观察到终态 → 状态查询竞态。
- 性能：每 job 一个 goroutine，异步延迟；测试时序不稳定。
- 风险：竞态 + 错过 + 状态需二次查询 → 明确否决为主方案。

### 方案 C（否决）：扩展现有 TaskRecorder 接口
- 给 `RecordDone` 加参或加方法承载 teammate 逻辑。职责混淆（监控 vs 业务回调），强制 `taskmonitor` 同步改动，接口膨胀。否决。

**裁决**：方案 A。「分发 goroutine 非必需」——handler 全部操作为内存级 + 毫秒级 ReadDir，同步回调 + recover 足够；未来若需多消费者扇出，可在方案 A 之上加注册表/fanout（预留扩展点，不在本次范围）。

---

## 三、任务分解（T1-T5）

### T1：jobs 层 `WithJobDoneObserver`
- **改动文件/函数**：
  - `internal/jobs/jobs.go`：`Manager` 增字段 `jobDoneObserver func(parentSession, id, kind, label string, st Status, err error)`；新增 `WithJobDoneObserver`（单例 Option，语义同 `WithJobStartObserver` L296-298）与 `SetJobDoneObserver`（对称 `SetTaskRecorder` L316）；新增私有 `notifyJobDone(...)`（nil 检查 + defer recover，防 panic 泄漏进 job 管线）；`recordCompletion` 两处接线：suppressEnvelope 分支 L975-977 `taskRecorder.RecordDone` 旁、正常分支 L1000-1002 `taskRecorder.RecordDone` 旁（两处均已无锁）。
- **明确边界**：destroying 提前 return（L982）不通知（与 taskRecorder 一致）；`recordStalled`（L1018）不通知（非终态）；loaded tombstone 恢复路径不通知。
- **验证**（`internal/jobs/` 新增 `jobs_done_observer_test.go`）：
  - ✅ TestJobDoneObserverFiresOnDone：正常 job → 回调收到 (parentSession, id, kind, label, Done)
  - ✅ TestJobDoneObserverFiresOnFailedAndKilled：失败/Kill 各触发一次
  - ✅ TestJobDoneObserverFiresOnInvalidStart：startInvalid（含 control 字符的 parentSession）也触发 Failed
  - ✅ TestJobDoneObserverSilentAndForegroundStillFire：静默/前台 job 触发（与 TaskRecorder 同构）
  - ✅ TestJobDoneObserverNotCalledWhileDestroying：BeginDestroySession 窗口内完成的 job 不触发
  - ✅ TestJobDoneObserverPanicDoesNotBreakPipeline：回调 panic → job 仍正常 done、Wait 正常返回
  - ✅ TestSetJobDoneObserverAfterConstruction：Set 后启动的 job 触发
  - ✅ 回归：`go test ./internal/jobs/` 零失败（尤其 TestDrainMultiple -race 不 flake）

### T2：TeammateStore handler（置 idle + 依赖推进 + mailbox 唤醒）+ recordTask 补存
- **改动文件/函数**（`internal/agent/teammate_store.go`）：
  - 新增 `OnJobDone(parentSession, id, kind, label string, st jobs.Status, err error)`，**实现顺序固定**：
    1. `st == jobs.Running` → 直接返回（只处理终态）；
    2. `ts.mu.Lock()`：`tasks[id]` 存在 → `tasks[id].Status = string(st)`（**依赖推进=终态落定**，后续 Assign 的 `pendingDependenciesLocked` 查 `ts.tasks[dep].Status` 即放行，L246-248）；
    3. 同锁内遍历 `ts.teammates`，找 `tm.LastJobID == id && tm.State == TeammateRunning` → 置 `TeammateIdle`（**严格匹配防 R3**，同 `Complete` L309-311）；
    4. `ts.mu.Unlock()`；
    5. 若命中 teammate 且 `ts.inboxRoot != ""` → 新增私有 `hasMailBacklog(name)`（`os.ReadDir(inbox)` 非空，容错）→ 有积压则 `ts.notifyMail(name)`（**mailbox 唤醒**，ts.mu 已释放，安全）；
  - `recordTask`（L234）**数据补存**：`TeamTask{..., Status: "running"}`（记录时快照初值）；完成事件再写终态 → `Tasks()`（L265-283）「jm.Output 优先、`t.Status` 兜底」逻辑**不改**即获得 purge 兜底能力。
  - 更新 `syncStateLocked` 注释：懒同步降级为兜底（事件驱动为主、懒查为兼容）。
- **验证**（`internal/agent/teammate_store_test.go` 新增，复用 `testTaskToolForTeam`）：
  - ✅ TestTeammateOnJobDoneFiresIdle：Assign → Wait 终态（不调 List）→ 直接断言 `ts.Status(name).State == Idle`（证明事件驱动，不再依赖懒同步）
  - ✅ TestTeammateOnJobDoneStaleJobNoop：完成事件带旧 jobID → 当前 LastJobID 的 Running 状态不被误置
  - ✅ TestTeammateOnJobDoneUnknownJobNoop：未 track 的 jobID → 无副作用
  - ✅ TestTeammateTaskStatusPersistedOnDone：任务完成后 `ts.Tasks()` 该条目 Status == 终态（再手动清空 jm 侧验证兜底值仍在，可用纯内存 OnJobDone 调用模拟）
  - ✅ TestTeammateMailboxWakeupOnDone：inbox 有积压 + sink 捕获 → 完成事件后收到 notice；无积压 → 无 notice
  - ✅ 回归：现有 5 个 teammate 测试零失败（`TestTeammateAssignStartsBackgroundJobWithEnvelope` 懒同步断言与事件驱动不冲突）

### T3：生产接线
- **改动文件/函数**：
  - `internal/boot/boot.go` L1930：拆出变量 `teammates := agent.NewTeammateStore(taskTool, jm)` → `teammates.SetSink(sink)` + `jm.SetJobDoneObserver(teammates.OnJobDone)` → `Options{Teammates: teammates, ...}`（**修复 R5 生产 sink 缺口**）。
  - T3b（可选，防御性）：`internal/control/controller.go` `New`（L582）中 `if opts.Teammates != nil && opts.Sink != nil { opts.Teammates.SetSink(opts.Sink) }` —— 覆盖所有经 Options 组装的路径；若做，`team_real_test.go` L69 的手动 SetSink 可保留（幂等）。
- **验证**：
  - ✅ `go build ./...` 通过
  - ✅ `go test ./internal/control/` 零失败（team_real_test 为 `//go:build realapi`，默认不跑，需确认编译仍过）

### T4：测试清单（映射到 test-strategy 分层）
| 层 | 命令/文件 | 覆盖 |
|----|----------|------|
| 单元（jobs） | `go test ./internal/jobs/ -run 'JobDoneObserver|Drain'` | T1 全部 + 回归（重点 TestDrainMultiple） |
| 集成（agent） | `go test ./internal/agent/ -run 'Teammate'` | T2 全部 + 既有 5 个 teammate 测试 |
| 集成（control 组装） | `go test ./internal/control/` | T3 接线零回归 |
| 竞态 | `go test -race ./internal/jobs/ ./internal/agent/` | 锁序/并发（observer 触发时序） |
| e2e（真实 API，发布前手动） | `REASONIX_REAL_API=1 go test ./internal/control/ -run TestTeamChainRealDeepSeek` | 事件驱动不改变真实链路断言（List 懒同步兼容） |
| 全量 | `go test ./...` | 最终门 |

### T5：验证与提交
- **命令序列**（事务四验证）：`gofmt -l internal/jobs internal/agent internal/boot internal/control` → `go vet ./...` → `go run ./tools/repolint`（功能增长入 baseline，参考 p6 execution.md L29）→ `go test ./...`。
- **提交**：单主题 commit。
  - 建议 message：`team: drive teammate lifecycle from job completion events`
  - body：
    ```
    jobs: add WithJobDoneObserver/SetJobDoneObserver, fire at recordCompletion
    (both envelope and suppressed branches) with panic recovery, mirroring
    the TaskRecorder hook; destroy window stays silent.
    agent: TeammateStore.OnJobDone marks the owning teammate idle, persists
    the terminal status into the dependency tree (recordTask snapshot), and
    wakes the leader when a finished teammate has a mail backlog.
    boot: wire the done observer and mailbox sink in production (P6.2).
    ```
  - 提交后按章程：executor 产出 `execution.md`（证据链：测试输出/文件 diff/数字）→ discipline 6 人审查输出 `review.md` → 通过才交付。

### 依赖关系图

```
T1（jobs 观察者 API + 测试）
 └─► T2（TeammateStore.OnJobDone + recordTask 补存 + 测试）
      ├─► T3（boot.go 生产接线 + 可选 T3b control.New 兜底）
      │       └─► T5（四验证 + 提交 + execution/review 记录）
      └─► T4（测试清单贯穿：T1/T2 测试内嵌于各自 T，T4 为汇总门）
```

- 严格先后：T1 → T2 → T3（T2 依赖 T1 的 observer 签名；T3 依赖 T2 的 `OnJobDone`）。
- 可并行：T1 与 T2 的**测试编写**可先行起草（不阻塞）；T3b 与 T3 主路径互不依赖，可同 T 内完成。
- 阻塞项：T2 的 `TestTeammateOnJobDoneFiresIdle` 在 T1 合入前无法通过（依赖 observer 触发）→ 集成测试必须在 T1 后启用。

---

## 四、对抗自检（devil's advocate）

1. **「同步回调会拖慢 job 完成管线」**——攻击：回调阻塞 → `close(j.done)` 延迟 → Wait 卡死。缓解：handler 全部为内存操作 + 毫秒级 ReadDir（ts.mu 外）；jobs 层 recover 兜底；文档化「observer 必须快速返回」。若实测超标，方案 B（goroutine 分发）为记录在案的退路。
2. **「依赖『推进』名不副实——没有自动启动依赖任务」**——攻击：完成事件只落定状态，不派活。缓解：语义对齐现有 MVP（拒绝式依赖门 + leader 手动 `/team-add`，见 p6 plan.md 裁决）；事件的价值是**状态落定**（jm purge 兜底）+ **唤醒通知**，自动调度不在本次范围（已明示边界，防执行队擅自扩大）。
3. **「双真源漂移：tasks.Status vs jm.Output」**——攻击：purge 后仅补存值，两源可能不一致。缓解：补存值来自 jm 终态快照、写入后不可变；`Tasks()` 优先级顺序确定（jm 优先 + 补存兜底）；写测试固化。
4. **「旧 job 完成事件误置 idle（R3）」**——缓解：严格 `LastJobID == id` 匹配（同 `Complete` 语义），测试 `TestTeammateOnJobDoneStaleJobNoop` 固化。
5. **「destroy 窗口吞掉完成事件 → tm 永久 Running」**——攻击：destroying 时 observer 不触发。缓解：`DestroyAll` 兜底（Remove → Kill + delete）；与 taskRecorder 既有行为一致，非回归；测试固化。
6. **「Set 时序：SetJobDoneObserver 前 job 已完成」**——攻击：漏事件。缓解：boot 期装配早于任何 Assign；handler 幂等（重复/未知 id 无副作用）；测试 `TestSetJobDoneObserverAfterConstruction` 固化。
7. **「mailbox 重复通知噪音（R5）」**——攻击：PostMail 已即时通知。缓解：`flushMailbox` 清空 inbox 后，完成事件查到的积压必为未 flush 新 mail，语义不重复；低风险标注。

---

## 五、缓存/纪律检查点（Reasonix 领域）

- ✅ **发送侧字节零变化**：本次改动是进程内回调（jobs→agent）+ event 层通知（mailbox notice），**不触碰** schema/system prompt/transcript/task.go/subagent_store；teammate fork/continue 三要素与 prefix 完全不受影响。
- ✅ **前缀稳定**：无新发送前缀结构、无历史插入、无 canonical 重写；mailbox 唤醒 notice 走 event.Sink（前端显示），不进 provider 输入。
- ✅ **运行值不进 schema/system prompt**：name/role/任务文本/任务状态（tasks.Status）均为内存态，非发送前缀的一部分。
- ✅ **锁序纪律**：observer 两处调用点均无 m.mu/j.mu 持有；handler 先释 ts.mu 再做 IO/通知。
- ✅ **防虚假完成**：每个 T 的「完成」= 测试输出 + 四验证通过 + execution.md 证据链，discipline 独立审查。

---

## 附：执行队禁止事项（按章程）

- 不得把本计划中「可选/防御性」项（T3b、反向索引优化）擅自扩为必做；自动调度依赖任务不在范围。
- 不得改动 `WithJobStartObserver` 既有签名与行为（workspaceLease 依赖，boot.go L517）。
- 不得在 `recordCompletion` 的 m.mu 临界区内调用 observer。
- 不得声称完成——须附证据（测试输出/文件 diff）。
