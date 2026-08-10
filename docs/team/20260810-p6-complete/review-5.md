# 纪律审查报告：P6 完成事件驱动 — 缓存红线专项（commit 8f76af02f）

> 角色：**纪律 Discipline**（独立复核，只读，零代码变更）
> 事务：`docs/team/20260810-p6-complete/` · 审查对象：commit `8f76af02f`
> （`team: drive teammate lifecycle from job completion events`，父提交 `8eaf12a`）
> 审查重点（任务指定 5 项）：发送侧字节零变化逐文件核查 / 自动 Assign prompt 登记原文 /
> mailbox 唤醒 Notice 走 event.Sink / 完成事件不注入 leader input、不写 teammate transcript /
> P1 信封次序与偷读面。
> 必读依据：`arbitration.md`（缓存红线节）、`interaction-design.md`（§2 信封次序、C1-C8 护栏）、
> `risk-review.md`（§2.7 逐发送字节审查、G7 零新发送）、`REASONIX.md`（缓存第一纪律）、
> 同 commit 前序审查 `review-1.md`（jobs 层）/`review-2.md`（teammate 侧）/`review-3.md`（e2e）/
> `review-4.md`（boot 接线）。
>
> **审查方法披露（诚实性约束）**：本环境无 shell 工具，**无法执行 `git show 8f76af02f`** 与
> `go test`。diff 影响面通过三条证据链交叉还原：(a) `.git/logs/HEAD` 确认 commit 存在、
> 父提交为 `8eaf12a`、其后仅 1 个 docs 提交（`3fe0532`），工作区 HEAD = `3fe0532` ⇒
> **工作区代码即 8f76af02f 的实现状态**（零代码漂移）；(b) 实现前基线行号锚点
> （interaction-design / risk-review / implementation-plan 的 L 锚点）与当前代码逐行比对，
> 识别 8f76af02f 的新增物；(c) 新增物逐一核查是否触碰 provider 请求序列化 / system prompt /
> tool schema / transcript。以下所有行号均为工作区当前实测行号。

---

## 一、审查项逐条

- **[FAIL] fable5 合规（执行队是否跳步）**
  缓存红线相关的执行轨迹（任务分解：T1 jobs observer / T2 handler / T3 boot 接线；拓扑扫描：
  两处无锁挂点 / destroy 窗口 / startInvalid；多路径推演：prompt 快照回放、sink 路径、信封
  次序）均有产物且与设计文档一致，**未发现缓存红线维度跳步**。但**整体 fable5 存在既定跳步
  且无修复 commit**（见结论）：`test-strategy.md` D2 硬约束（Close/destroy 后 observer 不触发，
  含 suppress 分支）未落实（review-1 驳回项）；killed 完成事件触发后继自动推进与
  lifecycle-design 裁决矛盾且执行队未上报（review-2 驳回项，防虚假完成红线）；均无后续
  代码修复。8f76af02f 之后工作区仅 1 个 docs 提交 ⇒ **驳回状态持续**。

- **[PASS（静态）] 幻觉检测（证据真实性）**
  全部新增物可核对到实测行号（jobs.go L222-227 字段 / L325-346 注册 API / L1050-1071
  fireJobDoneObservers / L1008、L1034 两处挂点；teammate_store.go L62/L292 SessionID /
  L430-474 HandleJobDone / L517-555 autoAssign / L709-721 notifyMailBacklog；
  boot.go L1905-1906、L1942）。测试文件真实存在且断言与生产时序自洽（review-3 已逐条核对）。
  **运行级证据缺口**：工作区无 execution.md / 测试输出文件，「go test / -race 全绿」为
  **未证实声明**（本环境无法重跑）——沿用 review-1 判定，不视为已证。

- **[PASS] 缓存红线（前缀字节稳定）**
  见下「二、五审查重点」：发送侧零变化成立、prompt 字节稳定、sink 路径正确、无 input/
  transcript 新增写入、信封次序正确、observer 路径零偷读。与 arbitration.md「缓存红线
  （全队一致）」一致。**唯一例外**：F1（生产 inboxRoot 未接线）使「完成事件积压唤醒在
  生产生效」不成立——但该 Notice 走 sink 不进 provider，**不影响前缀字节**，属功能交付
  完整性缺陷而非缓存红线破坏。

- **[FAIL] 回归（测试命令 + 结果）**
  本环境无 shell，无法执行 `go test -race ./internal/jobs/ ./internal/agent/ ./internal/boot/`。
  静态编译一致性：新增符号（SetJobDoneObserver / HandleJobDone / autoAssign / assignContext /
  notifyMailBacklog / countInbox）调用点全部可解析；boot 接线类型匹配（*agent.TeammateStore ↔
  control.Options.Teammates）。**「全绿」无证据 ⇒ 不通过**（无证据的通过 = 未通过）。
  执行队须补：`go test ./internal/jobs/ ./internal/agent/ ./internal/boot/ -count=1` 与
  `go test -race ./internal/jobs/ ./internal/agent/` 的真实输出（或 CI 记录）。

- **[FAIL] 对抗自检（devil's advocate 攻击发现的缺陷）**
  见「四、发现」：F1 为阻塞（生产积压唤醒空转 + ephemeral mail 只通知不投递）；F2 注释
  误导（照注释执行会编译失败）；另确认前序驳回项（D2 幽灵事件、killed 推进矛盾、autoAssign/
  Remove 双条目）均未修复。

---

## 二、五审查重点逐项（任务指定）

### 重点 1：发送侧字节零变化 —— 逐文件核查 [PASS]

| 文件 | 8f76af02f 新增物 | 发送侧触碰核查 | 结论 |
|---|---|---|---|
| `internal/jobs/jobs.go` | `jobDoneObservers` 切片（L222-227，内存字段）、`WithJobDoneObserver`（L328）、`SetJobDoneObserver`（L338，构造后单槽接线）、`fireJobDoneObservers`（L1050-1071，m.mu 内快照、锁外逐个调用、per-observer recover）、`recordCompletion` 两处调用（suppress 分支 L1008 / 正常分支 L1034） | j.mu 快照+信封预渲染（L978-992）、m.mu append `m.completed`（L1021-1026）、closing Notice（L1036-1047）**逐字节未变**；信封渲染函数（jobResultTextLocked/boundedResult/renderResultEnvelope）未动；jobs 不参与 provider 请求序列化（序列化在 provider 包 / control.Compose）、不写 transcript | **零变化** |
| `internal/agent/teammate_store.go` | `TeamTask.SessionID`（L62/L292，内存字段）、`HandleJobDone`（L430-474）、`recordPendingTaskLocked`（L305）、`depsTerminalLocked`（L480）、`assignContext`（L496，ctx 每次重建不缓存）、`autoAssign`（L517）、`countInbox`（L683）、`notifyMailBacklog`（L709）、`Tasks()` 改纯快照（L350，去 jm.Output 派生）、`pendingDependenciesLocked` 去 `jm.Output`（L324-344）、`Remove` 清 tasks（L617-621）、构造内注册 observer（L111） | Assign 的 ProfileExecSpec 装配（L234-247，fork/continue 选择）**未改**（与 interaction-design 基线一致）；system prompt 用既有 `ts.task.sysPrompt`；无新增 tool / schema；flushMailbox→SendMessageForSession 为既有 P3 路径（Assign 成功后调用，非完成事件新增） | **零变化** |
| `internal/boot/boot.go` | 拆变量 + `teammates.SetSink(sink)`（L1905-1906）+ `Teammates: teammates`（L1942） | 3 行接线，sink 为既有变量最终包装态（L249 Sync → 包装链 → L519 jm / L1911 controller 同源）；SetSink 只存指针，不发任何字节 | **零变化** |

- **system prompt / tool schema / transcript 维度**：三文件均无改动。profile 名 / 动态值
  （SessionID / pending 占位 id `pending:<owner>:<ns>` / jobID）只存在于内存 TeamTask /
  tasks 表，**不进任何 provider 请求、不进 schema**（risk-review §2.7 的「jobID/时间戳/动态
  计数」危险清单无一落点）。
- **G7 零新发送核查**：完成事件唯一对外出口 = ①内存态推进（无字节）②`notifyMailBacklog`
  的 event.Notice（sink，UI 层）③autoAssign 启动新 job（与手动 /team-add 逐字节同路径）。
  无新增 leader input 注入、无新增 transcript 写入、无新增 closing Notice。

### 重点 2：自动 Assign 的 prompt=登记原文（字节稳定）[PASS]

- 登记：`recordTask`（L284-297）存 `Prompt: prompt`（Assign 参数原文；Assign 只对 name 做
  TrimSpace（L194），**prompt 无任何加工**）。
- 回放：`autoAssign`（L524）`owner, prompt, sessionID := cur.Owner, cur.Prompt, cur.SessionID`
  → `ts.Assign(ctx, owner, prompt, deps...)`（L536）→ `TaskSpec.Objective: prompt`（L235）。
  与手动 `/team-add` 走同一 Assign 装配，逐字节一致（interaction-design §6.1「对新 job 的
  steer 语义逐字节一致」同型论证）。
- **测试缺口**：teammate_done_e2e_test.go 的依赖链用例（TestTeammateDoneDependencyChainAutoAdvance）
  覆盖了自动推进链路，但**无「自动推进 prompt == 登记原文」的字节级显式断言**。机制上成立
  （纯内存字符串拷贝），动态证据缺失。

### 重点 3：mailbox 唤醒 Notice 走 event.Sink（不进 provider 输入）[PASS（机制）/ FAIL（生产生效）]

- **机制**：`notifyMailBacklog`（L709-721）锁内取 `ts.sink`、锁外 `sink.Emit(event.Event{
  Kind: event.Notice, ...})`；`notifyMail`（L669-677）同型。event.Notice 是 UI 事件层
  （event.go Event struct L525；render.go 只控前端转发），**不进 provider 输入、不写
  transcript**（risk-review §2.7「sink Notice 不进 transcript 则安全」）。✓
- **防风暴**：仅当 `flipped != "" && ts.inboxRoot != ""`（L466）且 `countInbox > 0`（L467）
  才通知，聚合计数 N（一次/已完成 teammate）——符合仲裁 4「mailbox 积压唤醒是唯一例外」。
- **生产接线 FAIL（F1，review-4 驳回项，未修复）**：boot `NewTeammateStore(taskTool, jm)`
  （L1905）仅 2 参 ⇒ 生产 `inboxRoot == ""` ⇒ HandleJobDone L466 短路、countInbox 恒 0、
  flushMailbox（L743）短路。**完成事件积压唤醒在生产永不触发**；生产 PostMail 走 ephemeral
  分支（L648-653）只发即时 Notice、mail 文本不投递（L764 SendMessageForSession 因 inboxRoot
  短路不可达）。「mailbox-wakeup notices reach the leader」这一交付声明**不完整**。
  （注：本缺陷不影响前缀字节——sink 路径本就安全——但使重点 3 的「生效性」不成立，且 boot
  注释未明示该边界。）

### 重点 4：完成事件不注入 leader input、不写 teammate transcript [PASS]

- HandleJobDone（L430-474）全部动作：tasks 快照写终态（内存）、teammate Running→Idle
  （内存）、countInbox（磁盘读）、notifyMailBacklog（sink）、autoAssign（新 job）。**无任何
  leader input 写入路径**。
- leader input 唯一注入点 = input.go L191-193 `<background-jobs>` ←
  `DrainCompletedNoteForSession`（P1 信封，既有），完成事件不触碰。
- teammate transcript 唯一写入方 = job 自身 run_loop / task.go；autoAssign 启动的新 job 的
  transcript（fork/continue 前缀）与手动 Assign **逐字节同源**（P-a/P-c 前缀路径未动）；
  flushMailbox 的 steer 注入是既有 P3 语义（Assign 后调用，非完成事件新增字节）。
- 结论：P-b（leader input）、P-c（teammate transcript）均无完成事件引入的新发送。

### 重点 5：与 P1 信封的次序 + 偷读面 [PASS]

- **次序（与 interaction-design §2.1 一致）**：run goroutine `recordCompletion`
  → j.mu 快照+预渲染信封（L978-992）→ m.mu 内 append `m.completed`（L1021-1026）→
  m.mu.Unlock（L1029）→ RecordDone（L1031-1033）→ `fireJobDoneObservers`（L1034）→
  closing Notice（L1045-1047）。**信封先入队、observer 后触发**；leader 读到信封在下一轮
  turn（input.go L191-193）——三段时间线不重叠，无竞争。
- **observer 是否偷信封**：`fireJobDoneObservers` 只传 `(id, st, err)`（L1057-1069）；
  HandleJobDone 只用回调参数 + `ts.tasks` 内存表（L435-441），**不调 jm.Output、不读
  m.completed、不读 j**。✓
- **`pendingDependenciesLocked` 已去 Output**（L324-344，tasks 快照为主源，注释明言
  「Never consult jm.Output」）——修复了「门永堵 + 偷读 result」双重风险（时序陷阱 R-a：
  observer 触发时 j.status 仍 Running，反查必误判且顺带消费）。✓
- **还有别处消费吗（全仓 jm 调用点排查）**：
  - `syncStateLocked`（L164）`ts.jm.Output(tm.LastJobID)`——**唯一剩余消费点**（Output 推进
    readOffset/resultRead）。但：(a) 它是 List() 惰性兜底（L146-156 注释「Lazy fallback
    only」），主路径 HandleJobDone 已置 Idle ⇒ State!=Running 短路（L161）；(b) **属既有
    基线行为**（risk-review 基线 L131-138），8f76af02f 未引入，且本次已把 Tasks() 迁为纯
    快照、pendingDependenciesLocked 去 Output，消费面**收窄**而非扩大；(c) interaction-design
    C7 建议的「非消费 status-only 查询迁移」对 syncStateLocked 未落实——非阻塞，建议跟踪。
  - `flushMailbox` L764（SendMessageForSession，写 steer 队列，非读）、`TeamStop` L596
    （KillForSession）、`Remove` L625（Kill）——均非信封消费。
  - 结论：observer 路径**零偷读**；信封（m.completed）唯一消费方仍是
    `DrainCompletedNoteForSession`（leader 下轮），未被任何完成事件路径触碰。

---

## 三、逐文件结论

| 文件 | 结论 | 备注 |
|---|---|---|
| `internal/jobs/jobs.go` | **PASS（发送侧零变化）** | observer 机制纯内存回调；信封/Notice 输出未改；挂点均在 m.mu 外。遗留：review-1 的 D2 缺陷（suppress 路径无 destroying 检查、fireJobDoneObservers 无 root.Done() 检查 → Close/destroy 窗口幽灵事件）未修复——非发送侧问题 |
| `internal/agent/teammate_store.go` | **PASS（发送侧零变化）** | tasks 快照化减少 jm 消费面；prompt 原样回放；SessionID 为内存字段。遗留：review-2 的 killed 推进矛盾与 autoAssign/Remove 双条目未修复——非发送侧问题 |
| `internal/boot/boot.go` | **PASS（发送侧零变化）** / 接线完整性 **FAIL** | SetSink 3 行接线正确同源；但 F1（inboxRoot 未接线 → 积压唤醒空转）与 F2（L1901-1904 注释过时，照注释执行 `jm.SetJobDoneObserver(ts.OnJobDone)` 会编译失败——实为构造内自动注册 L111）必改 |

---

## 四、发现（对抗自检结果）

1. **【阻塞 · 必改 · F1，review-4 驳回项，未修复】** 生产 `inboxRoot` 未接线 ⇒ 仲裁 4 的
   「完成事件积压唤醒（聚合 N）」在生产**永不触发** + 生产 PostMail 只通知不投递（ephemeral
   分支无 SendMessageForSession）。处置二选一：(a) boot 补传 inboxRoot；(b) 显式文档化降级
   + 更新 boot 注释 + 立独立跟踪项。**不得宣称「生产 mailbox 唤醒已完整生效」**。
2. **【必改 · F2，review-4 驳回项，未修复】** boot.go L1901-1904 注释声称 observer 未接线
   （"OnJobDone has not landed yet (E2)"）——E2 已落地且构造内已注册（teammate_store.go L111）；
   照注释执行会编译失败（方法名实为 HandleJobDone）。改为「构造器已自动注册，boot 无需接线」。
3. **【阻塞 · review-1 驳回项，未修复】** D2 硬约束：`fireJobDoneObservers` 无 `m.root.Done()`
   检查、suppress 路径（L1008）无 destroying 检查 ⇒ Close/destroy 窗口内 Killed 完成仍触发
   observer → 幽灵 idle flip + autoAssign 幽灵派活。J7（真场景）/J8 测试缺失，
   `TestJobDoneObserverNotFiredAfterClose` 名不副实。
4. **【阻塞 · review-2 驳回项，未修复】** HandleJobDone ready 扫描不区分 st，Killed 也触发
   后继自动 Assign，与 lifecycle-design 表项 3「自动推进不触发」矛盾（仲裁文本内部矛盾，
   实现取其一未上报——防虚假完成红线）；autoAssign 与 Remove 竞态产生 pending+blocked 双条目。
5. **【中 · 同步回调重活偏离仲裁 2 假设】** 仲裁 2 裁决理由为「handler 全为内存级操作」，
   但实现中 HandleJobDone 同步执行 `countInbox`（os.ReadDir 磁盘 IO）+ `autoAssign`（fork
   prefill = transcript 磁盘写）——均在 run goroutine 内、close(j.done) 之前（review-3 已确认
   时序依赖「WaitForSession 返回 ⇒ handler 已执行」），会延迟 job 收尾（risk-review 2.1 高）。
   无死锁（仲裁 3 缓解 5 锁序分析成立），属**已知仲裁范围**但值得记录；G1 的 worker 化护栏
   未采纳。
6. **【轻微】SetJobDoneObserver 替换语义陷阱**（jobs.go L338-346，C8）：未来若第二消费者用
   WithJobDoneObserver 追加，会被 SetJobDoneObserver 覆盖。当前全仓唯一注册点 = NewTeammateStore
   构造（grep 核实），不触发。
7. **【轻微】缓存红线动态证据缺失**（G10）：无「自动推进 N 级链后 leader committed transcript
   字节 == 基线」或「teammate 续轮前缀 byte-identical」的运行时断言；缓存零变化目前仅靠静态
   论证，无 cachehit_e2e 基建加持。

---

## 结论

**驳回**（缓存红线机制层面通过，commit 整体仍处驳回状态，必须修复后复审）。

- **缓存红线专项判定**：五审查重点中，发送侧零变化（逐文件）、prompt 登记原文、sink 路径、
  无 input/transcript 写入、P1 信封次序与零偷读 **五项机制全部 PASS**——与本 commit 的
  arbitration.md 缓存红线裁决一致，未发现任何发送侧前缀字节变化，也未引入 jobID/时间戳/动态
  计数进 provider 输入或 schema。**「完成事件驱动不破坏 DeepSeek 前缀缓存」成立（静态证据）。**
- **驳回理由**（必须修复项，按优先级）：
  1. **F1**：生产 inboxRoot 未接线 ⇒ 仲裁 4 积压唤醒在生产空转（review-4 阻塞项，未修复）。
  2. **review-1 驳回项**：D2 幽灵事件（Close/destroy 窗口 observer 仍触发 + 测试名不副实）。
  3. **review-2 驳回项**：killed 自动推进语义矛盾（需父代理二选一裁决）+ autoAssign/Remove
     双条目竞态。
  4. **F2**：boot.go L1901-1904 误导性注释。
  5. **回归证据**：执行队补 `go test ./internal/jobs/ ./internal/agent/ ./internal/boot/ -count=1`
     与 `go test -race ./internal/jobs/ ./internal/agent/` 真实输出；并建议补一条缓存红线动态
     断言（自动推进链后前缀字节 == 基线）。

- 方法局限声明：本报告为静态交叉验证（无 shell，未执行 git show/go test）；运行级确认
  （测试全绿、-race、前缀缓存实测）留执行环境，其声明在获得真实输出前不视为已证实。
- 与前序 review 关系：缓存红线维度结论与 review-1/2/4 的发送侧判定一致（零变化）；
  驳回状态亦一致（三者分别因 D2、killed/竞态、F1/F2 驳回，均未修复，8f76af02f 后无代码
  commit）。review-3（e2e 测试质量）通过，与本报告不冲突。

---

*本文档为纪律复核交付物；修复与验证由执行小队承担，复审后放行。*
