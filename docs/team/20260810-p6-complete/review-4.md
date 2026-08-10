# 纪律审查报告：P6 完成事件驱动 boot 接线（commit 8f76af02f）

> 审查对象：`internal/boot/boot.go` L1898-1906（NewTeammateStore 拆变量 + SetSink）+ L1942
> （ctrlOpts.Teammates）接线；对照 `internal/agent/teammate_store.go`（构造自动注册 L111、
> SetSink L182、HandleJobDone L430、PostMail L635、notifyMail L669、countInbox L683、
> notifyMailBacklog L709、flushMailbox L742）、`internal/jobs/jobs.go`（SetJobDoneObserver
> L336 替换语义、NewManager L354）、`internal/control/controller.go`（Teammates 字段 L496）。
> 依据：`arbitration.md`（仲裁 4/缓存红线）、`mailbox-design.md`（§4.5 M5 接线 / §一.7
> 降级语义 / §4.2 文案）、`review-3.md`（e2e 证据基线）。
> 审查方式：静态交叉验证（本环境无 shell，无法重跑 `go test`；逐条比对 boot 接线与
> store/jobs 生产代码，验证「可重跑即真实」的证据链）。

---

## 一、审查项逐条

- **[PASS（条件）] fable5 合规（执行队是否跳步）**
  本次审查对象为 boot 接线产物（拆变量 + SetSink），无 execution.md 可核对执行过程，按
  产物反推：`teammates := agent.NewTeammateStore(taskTool, jm)`（L1905）+ `teammates.SetSink(sink)`
  （L1906）+ `Teammates: teammates`（L1942）与 mailbox-design §4.5 M5 的指定形态逐字一致；
  构造在 control.New 之前、SetSink 紧随构造之后；`jm` 与 `taskTool` 均取自本函数既有变量
  （jm=L519，taskTool 惰性赋值 L1085-1106），未新建第二份实例。未发现跳步迹象。**但发现
  boot 注释与实现脱节（见发现 F2），执行队未同步更新注释，属文档卫生问题。**

- **[PASS] 幻觉检测（证据真实性——逐条重跑验证逻辑）**
  - 「sink 与 controller/jobs 同源」：L249 `sink := event.Sync(opts.Sink)` → L258-266 二次
    包装（stats.Recorder → `control.NewGoalUsageTee(event.Coalesce(...))`）→ L519
    `jm := jobs.NewManager(sink, ...)`、L1906 `teammates.SetSink(sink)`、L1911
    `ctrlOpts.Sink = sink` 均为**同一变量（最终包装态）**。同源成立，真实。✓
  - 「notifyMail 此前是生产静默 no-op」（L1900 注释声明）：SetSink 之前 `ts.sink == nil`，
    notifyMail（teammate_store.go L669-677）L673 nil 检查后静默返回——声明与代码一致。✓
  - 「构造内自动注册 HandleJobDone」（teammate_store.go L111）：`jm.SetJobDoneObserver(ts.HandleJobDone)`
    在 NewTeammateStore 内、boot 无显式 SetJobDoneObserver——boot 与构造器**无重复注册**。✓
  - **「mailbox-wakeup notices reach the leader」（L1898-1900 隐含完整声明）不完整**：
    SetSink 只使 PostMail **即时**通知（notifyMail）生效；**完成事件积压唤醒
    （notifyMailBacklog）因 boot 未传 inboxRoot 而不会触发**（HandleJobDone L466
    `flipped != "" && ts.inboxRoot != ""` 短路；countInbox L684 `inboxRoot == "" → 0`）。
    生产路径 `NewTeammateStore(taskTool, jm)` 仅 2 参，inboxRoot 恒空。**此声明在「积压
    唤醒」意义上不成立且未明示边界** → 发现 F1。
  - 无法重跑（无 shell）；以上为静态证据链交叉验证，运行确认留执行环境。

- **[PASS] 缓存红线（前缀字节稳定）**
  本次改动为进程内事件路由（SetSink 接线），Notice 走 `event.Sink`（UI 层），不进 provider
  输入、不动 schema/system prompt/transcript；SetSink 接的是 Coalesce 包装 sink，而 Coalesce
  只合并 Text/Reasoning 流（mailbox-design §2.1 核实），Notice 独立送达，无前缀影响；自动
  Assign prompt 为 recordTask 登记时原文快照（字节稳定，既有实现，本次未动）。与
  arbitration.md「缓存红线（全队一致）」一致。golden/boot_test 中的 `team_message` 为 P6.2
  既有状态（ProviderVisible gate：仅 teammate ctx 可见），非本次 commit 引入。✓

- **[PASS（静态）] 回归（测试命令 + 结果）**
  本环境无 shell，无法执行 `go test ./internal/boot/ ./internal/agent/ ./internal/jobs/`。
  静态回归分析：
  - 类型匹配：`*agent.TeammateStore` ↔ `control.Options.Teammates`（controller.go L496）；
    `teammates.SetSink(sink)` 参数 `event.Sink` ↔ SetSink 签名（teammate_store.go L182）。
    编译无碍。
  - 改动仅 3 行（拆变量 + SetSink + Options 引用），不触及 boot 其它分支；`jm`/`taskTool`
    仍为原变量，controller 的 Jobs/Sink 引用不变 → 既有 boot 行为零变化。
  - `SetJobDoneObserver` 为**替换语义**（jobs.go L338-346），boot 仅经构造器注册一次、
    无 WithJobDoneObserver 冲突（workspaceLease 占的是 WithJobStartObserver，L517，异字段）。
  - 运行确认（`go test ./internal/boot/ ./internal/agent/ ./internal/jobs/`）留执行环境。

- **[FAIL] 对抗自检（devil's advocate 攻击发现的缺陷）**
  见下「三、发现」：F1（生产积压唤醒未接线）为**阻塞性**、F2（误导性注释）为**必改**，
  其余轻微。核心攻击结论：SetSink 接线本身正确，但「生产路径完整性」不满足——P6 完成
  事件驱动的 mailbox 积压唤醒在生产是空转，且 boot 注释未明示该边界。

---

## 二、审查重点逐项核对

1. **SetSink 接线正确性（同源）** — **PASS**。L1906 与 L1911（controller Sink）、L519
   （jm）、L249（源头 Sync 包装）同一 sink 变量最终态；SetSink 接 Sync 包装（并发安全，
   满足 mailbox-design R6）；teammate_store.go 的 notifyMail/notifyMailBacklog 均锁内取
   sink 指针、锁外 Emit（L670-672/L715-717），与 SetSink 写互斥 → 无数据竞争。
2. **完成观察者接线完整（重复/遗漏）** — **PASS（无重复）**。boot 无显式
   SetJobDoneObserver，注册唯一发生在 NewTeammateStore 构造内（L111）；生产只构造一个
   store，替换语义无覆盖方。**「遗漏」维度**：注册无遗漏（构造内已闭环），但**积压唤醒
   的落盘前提（inboxRoot）在生产遗漏** → 见 F1。
3. **生产路径完整性（mailbox 唤醒 Notice 真实生效）** — **部分 FAIL**。即时通知（PostMail
   → notifyMail）随 SetSink 生效；**完成事件积压唤醒（notifyMailBacklog）不生效**：boot
   未传 inboxRoot → inbox 无落盘 → 无「积压」概念 → HandleJobDone L466 短路。且生产
   PostMail 走 ephemeral 分支（L648-653）只发 Notice、**mail 文本不投递**（无
   SendMessageForSession；flushMailbox L743 因 inboxRoot 空短路）。「leader 收到『有 N 封
   未读』的完成提醒」这一 P6 核心交付在生产不可达。属 mailbox-design §一.7 已仲裁的降级
   路径，但**接线与文档/注释未同步**（设计文档明示了边界，boot 注释未明示）。
4. **无副作用（SetSink 时机）** — **PASS**。L1905 构造 → L1906 SetSink 同 goroutine 串行，
   两语句间无调度点（无函数调用/channel/IO）；teammates 在 L1942 才交给 controller，
   SetSink 先于任何并发访问完成。理论窗口（构造后 jm observer 已注册而 sink 尚 nil，若有
   job 恰在此间完成则即时通知静默）在 boot 早期无 session/job 可完成，实际不可达。

---

## 三、发现（对抗自检结果）

1. **【阻塞 · 必改】生产 inboxRoot 未接线 → 完成事件积压唤醒不生效 + mail 内容不投递**
   - 事实：boot L1905 `NewTeammateStore(taskTool, jm)` 只传 2 参（测试 e2e3/单测均传
     `t.TempDir()`）；生产 `inboxRoot == ""` → HandleJobDone L466 短路、countInbox 恒 0、
     flushMailbox L743 短路。
   - 后果 A：仲裁 4 的核心交付「job 完成 → 置 idle 后若 inbox 有积压 → 聚合 Notice（含 N）」
     在生产**永不触发**。
   - 后果 B：生产 PostMail 走 ephemeral 分支（L648-653），**只发即时 Notice、mail 文本丢弃**
     ——teammate 间 mail 是「只通知不投递」；L649 注释「the P3 steer queue is the mailbox
     while running」与代码不符（无任何 SendMessageForSession）。
   - 处置：要么给 boot 接 inboxRoot（需 config 目录 + 落盘接线，超出本 commit 范围），
     要么在 boot 注释与事务文档**显式明示**「生产 mail 落盘/积压唤醒未接线、当前仅即时
     通知」，并立独立跟踪项。当前 boot 注释 L1898-1900 声称「mailbox-wakeup notices reach
     the leader」未限定范围，属误导。
2. **【必改】boot.go L1901-1904 注释过时且含错误代码指引**
   - 注释声称「jm.SetJobDoneObserver(ts.OnJobDone) is intentionally deferred:
     TeammateStore.OnJobDone has not landed yet (E2)… add the line after E2 merges」。
   - 事实：E2 已落地（HandleJobDone 在 teammate_store.go L430，构造内 L111 已注册）；
     若有人照注释执行 `jm.SetJobDoneObserver(ts.OnJobDone)`，**方法名不存在（实为
     HandleJobDone）→ 编译失败**；即便改成 HandleJobDone 也是自我覆盖的冗余（替换语义）。
   - 注释应改为：观察者由 NewTeammateStore 构造内自动注册（L111），boot 无需显式接线。
3. **【轻微】control.New 无 SetSink 兜底**（mailbox-design §4.5 的可选项未实现）：非 boot
   组装路径（嵌入/未来调用方）传 Teammates 不 SetSink 则 notifyMail 静默。当前生产 boot
   已接，无实害；建议在 `control.Options.Teammates` 字段注释或 control.New 内加防御性
   SetSink（幂等，重复调用覆盖）。
4. **【轻微】SetJobDoneObserver 替换语义的长期陷阱**（jobs.go L338-346）：若未来出现
   「一 jm 多 TeammateStore」或 boot 重复注册，后建者覆盖先建者。现生产单 store 单注册，
   不触发；与 review-3 发现 5 同性质，建议在 jobs.go 文档标注。
5. **【轻微】ephemeral 语义断裂的用户可见性**：生产 leader 会收到「received mail」Notice
   （即时，已生效），但收信 teammate 永远收不到内容、也不会有积压聚合提醒——用户感知为
   「mail 已投递」而实际未投递（Execute 返回文案「mail delivered…」亦误导）。F1 修复前
   应在 team_message 工具文案/文档中明示 ephemeral 边界。

---

## 结论

**驳回（条件性通过被 F1 阻断）** —— SetSink 接线本身正确（同源、无重复、无副作用、缓存
零影响），但「生产路径完整性」这一审查重点不成立，且 boot 注释存在误导性声明与错误代码
指引。

必须修复项（阻塞）：
1. **F1**：生产 inboxRoot 未接线 → 完成事件积压唤醒（P6 核心交付）与 mail 内容投递在生产
   均不生效。修复选项二选一：(a) boot 补传 inboxRoot（含 config/落盘目录接线）；
   (b) 显式文档化该降级 + 更新 boot 注释 + 立独立跟踪项。**当前不得宣称「生产 mailbox 唤醒
   已完整生效」**。
2. **F2**：boot.go L1901-1904 注释必须更新——E2 已落地、构造内已自动注册
   （teammate_store.go L111）、照旧注释执行会编译失败；改为「boot 无需显式 SetJobDoneObserver」。

建议项（非阻断）：F3 control.New 防御性 SetSink；F4 替换语义文档标注；F5 ephemeral 边界
在 team_message 文案/文档明示。

- 静态证据链结论：4 个审查重点中 3 项（同源/无重复/无副作用）成立；第 4 项（生产路径
  完整性）仅「即时通知」生效，「积压唤醒」空转——与 boot 注释的完整声明不符。
- 运行确认（`go test ./internal/boot/ ./internal/agent/ ./internal/jobs/` 及 -race）留执行
  环境，本报告为静态交叉验证。
