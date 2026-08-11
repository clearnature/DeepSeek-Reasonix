# 纪律审查报告：P6 完成事件驱动 第二轮修复复审（PASS）

> 角色：**纪律 Discipline**（独立复审，与执行小队/规划隔离）
> 审查对象：commit bbc444955 + 8f76af02f 工作区现状 —— `internal/agent/teammate_store.go`、
> `internal/agent/teammate_store_test.go`、`internal/agent/teammate_done_e2e_test.go`、
> `internal/jobs/jobs.go`（交叉验证 observer 时序）、`internal/boot/boot.go`（生产接线）。
> 裁决依据：`arbitration.md` + `lifecycle-design.md` 表项 3/4（killed 语义 / worker 化权威）。
> 前轮：review-2（第一轮驳回）、review-6（独立复审驳回）。
> 复审范围：killed/failed/interrupted 不自动推进、worker 化、blocked 恢复、Close 生命周期、队列满降级。
> 方法：静态交叉验证 + 逐交错推演 + 测试断言逐条对账。本环境无 shell，无法重跑 `go test`；
> 父代理已声明真实验证（`go test ./internal/agent/`、`-race`、goleak clean），本报告以
> 「代码行 ↔ 测试断言 ↔ 文档裁决」三方对账核验其证据一致性。

---

## 一、审查项逐条

### [PASS] fable5 合规（执行队是否跳步）
- 上一轮三项阻塞（review-2 发现 1 / review-6 攻 1、攻 4）已落实且**有测试钉住**，未再跳步：
  ①killed 短路（`HandleJobDone` L512 `st != jobs.Done` 跳过 ready 收集）+ `TestTeammateKilledDepDoesNotAutoAdvance`；
  ②worker 化（`HandleJobDone` 只 `enqueueAuto`，`autoWorker` 串行 `autoAssignByID`）；
  ③blocked 恢复（ready 扫描认 `taskPending`+`taskBlocked`）+ `TestTeammateBlockedAutoAdvanceRetries`。
- 六项前轮问题闭环核对见 §三；未闭环项（发现 2/3、H5/H9、环检测）均属 LOW/建议级且**已在本报告显式留痕**，
  未再隐瞒（防虚假完成红线：本报告如实列出残余，不宣称"全量清零"）。
- 留痕缺口：p6-complete 目录仍无 execution.md（前两轮同注），回归命令与输出无持久记录；
  父代理口头声明已验证。此项为流程留痕缺陷（非功能缺陷），不再作阻塞。

### [PASS] 幻觉检测（证据真实性——三方对账）
- **killed 短路为真实实现**：`HandleJobDone` L504-519 ready 收集在 L512 `if st != jobs.Done { continue }`
  整段短路；终态落定 L486-488 不受影响（依赖门照常放行）——与 lifecycle 表项 3「只有 done 推进、
  其余只落定终态 + 解锁门」逐字吻合。
- **worker 化为真实实现**：回调锁外段（L522-529）仅 `countInbox`（`os.ReadDir`，仅 inboxRoot 非空时）+
  `notifyMailBacklog`（recover 包裹）+ `enqueueAuto`（非阻塞）。**回调内已无同步 Assign**——
  这是与 review-6 攻 1 所述旧代码（L471-473 直接 `autoAssign`）的本质差异，非文字层面的伪造。
- **blocked 恢复为真实实现**：L509 认 `taskBlocked`；`TestTeammateBlockedAutoAdvanceRetries`
  构造真实场景（beta busy → Assign 拒绝 → blocked → jobB 的 Done 事件 → 重扫 → 恢复成功），
  断言与实现逐条对应，测试用轮询（5s/10ms）兼容 worker 异步时序，无 `t.Skip`、无空断言。
- **测试证据真实性**：teammate_store_test.go 15 个测试 + e2e 4 个测试全部走真实 job 全链或直接
  注入 `HandleJobDone`（确定性分支），符号（`sequencingProvider`/`releaseBlockingProvider`/
  `teamAssignCtx`/`newTaskToolWith`）均可解析；所有 `NewTeammateStore` 调用均 `defer ts.Close()`
  （goleak 前置条件成立）。observer 时序（jobs.go L967 recordCompletion → L1015/L1042
  fireJobDoneObservers → status flip → close(j.done)）与本轮实现不冲突——`HandleJobDone` 不再
  查 `jm.Output`，时序陷阱（回调内读到 Running）已物理消除。
- 唯一无法独立复核项：`go test`/`-race`/goleak 的运行输出（本环境无 shell）。父代理声明已验证，
  静态对账未发现与其矛盾的证据。

### [PASS] 缓存红线（前缀字节稳定）
- 自动推进回放 `cur.Prompt`（`recordPendingTaskLocked` 登记原文快照），`TestTeammateHandleJobDoneAutoAdvancesDependent`
  L543-547 断言 `tk.Prompt == "second step"` —— 字节稳定有测试钉住。
- 本轮新增代码（`enqueueAuto`/`autoWorker`/`autoAssignByID`）纯进程内；完成事件只写内存 tasks 快照 +
  `event.Notice`（UI 层），不进 provider 输入。fork/continue 前缀逻辑未动（Ref 回填机制不变）。
- profile 名/动态值未进 schema：SessionID 仍以字符串入 `TeamTask`，ctx 每次重建。

### [PASS] 回归（静态 + 父代理已验证）
- 静态：`SetJobDoneObserver` 全仓唯一调用点仍为 `NewTeammateStore` 构造（L120，grep 核实），
  生产无其它 observer 覆盖；`Complete`/`TeamStop`/`List`/`Remove`/`Assign` 旧 API 签名未动；
  jobs 层 `recordCompletion` 入队/Emit 段字节未改。boot.go L1908 生产接线
  `NewTeammateStore(taskTool, jm, inboxRoot)` + L1909 `SetSink`。
- 执行证据：父代理声明 `go test ./internal/agent/` ✅、`-race` ✅、goleak clean ✅。
  本环境无法独立重跑；静态对账（测试逻辑、defer 顺序、worker 退出路径）与声明无矛盾。

### [PASS] 对抗自检（devil's advocate 攻击发现的缺陷）
见 §二。攻击未发现阻塞级缺陷；发现 4 处 LOW 残余（注释不一致 ×3、Close 不等 worker）与
2 处已知缺口（blocked 恢复窗口、H5/H9 测试）。

---

## 二、对抗自检发现（残余，非阻塞）

### 发现 A（LOW）`HandleJobDone` 注释残留"同步 Assign"表述（teammate_store.go L471-475）
- L473-475 仍写 "The synchronous Assign inside the callback is safe per arbitration 3's
  lock-order analysis (startForSession re-enters jm locks...)"；L467-469 责任描述亦未提 worker/队列。
- **现状**：回调内已无同步 Assign（L527-529 仅 `enqueueAuto`）。注释与实现不符，会误导维护者
  以为回调仍承担重 IO。同属 review-2 发现 4 那类"注释滞后于实现"。
- 建议：改为 "the auto-assign is delegated to a single worker goroutine via enqueueAuto（非阻塞入队）"。

### 发现 B（LOW）`pendingDependenciesLocked` 函数头注释与函数体矛盾（L365-369 vs L370-390）
- 函数头（L368-369）仍称 "the manager is only consulted as a fallback for deps that were
  never registered"；函数体（L370-390）从不 consult `ts.jm`，且 L371-376 新注释明示
  "Never consult jm.Output here"（时序 + 消费性理由正确）。= review-2 发现 2，**未闭环**（LOW）。
- 行为安全（未登记 dep 永不放行，保守不循环），仅注释措辞矛盾需清理。

### 发现 C（LOW）`autoAssignByID` 文档注释重复（L568-572 与 L573-577 几乎逐字重复）
- 编辑残留：两段注释内容相同、函数名不同。纯文档瑕疵。

### 发现 D（LOW）`Close` 不等待 worker 退出（无 WaitGroup/drain）
- `Close`（L127-129）只 `close(doneCh)`；worker 在处理队列剩余任务后、select 到已关闭的
  doneCh 才退出。`goleak clean` 依赖此退出路径成立（autoCh 空时 select 必选 doneCh），
  且测试 defer 顺序（`ts.Close()` 先于 `jm.Close()`）保证 worker 处理任务时 jm 仍存活、
  Assign 失败仅记日志不 panic。**功能上成立，但语义上"Close = 异步停"**——若调用方在 Close 后
  立即销毁依赖（jm），worker 可能仍短暂活动。建议后续补 drain + WaitGroup（非阻塞）。

### 发现 E（LOW/已知边界）blocked 恢复窗口仅限 Done 事件
- ready 扫描被 L512 `st != jobs.Done` 整体短路 ⇒ **failed/killed/interrupted 事件不触发 blocked
  重扫**。此保守选择与"killed 不自动推进"原则一致（重扫 blocked 可能启动新 job，违背用户
  主动终止意图），代价是：若链尾以非 Done 终态收场，此前被 queue-full/owner-busy 标记的
  blocked 任务无自动恢复窗口（`Tasks()` 可见 blocked + reason，非静默丢失，可手动重推）。
- 判定：设计一致、可接受；建议在注释中显式声明该窗口语义。

### 发现 F（LOW）测试缺口残留
- `TestTeammateKilledDepDoesNotAutoAdvance` 的 100ms sleep 后断言（L768）：当前实现 killed
  根本不入队，断言必然成立（无假阳性）；但若未来实现回退为全终态推进，100ms 窗口可能不足以
  让错误的 Assign 写入 LastJobID → 测试可能误报通过（假阴性防护弱）。建议改用「断言 autoCh
  长度/无新 job 登记」的更确定信号。
- H5（两级链 a→b→c）与 H9（关闭 jm 后 Tasks 快照）未补；failed/interrupted 分支无显式测试
  （与 killed 同路径，L512 覆盖）。均 LOW。

---

## 三、前轮阻塞项闭环核对

| 前轮问题 | 现状 | 闭环 |
|---|---|---|
| review-2 发现 1 / review-6 攻 5：killed 触发自动推进 | L512 `st != jobs.Done` 短路 + `TestTeammateKilledDepDoesNotAutoAdvance` | ✅ |
| review-6 攻 1：同步 Assign 阻塞收尾 | `HandleJobDone` 仅 `enqueueAuto`，`autoWorker` 串行 `autoAssignByID` | ✅ |
| review-6 攻 4：blocked 无恢复 | blocked 参与 ready 扫描 + `TestTeammateBlockedAutoAdvanceRetries` | ✅ |
| review-6 F1：生产 inboxRoot 未接线 | boot L1907-1908 `inboxRoot := filepath.Join(sessionDir, "team-inbox")` 已传 | ✅ |
| review-6 F2：boot 注释过时 | boot L1898-1906 已更新为 P6.2 接线说明 | ✅ |
| review-6 攻 2：环检测 / 链深上限 | 未实现（不在本轮三项修复范围） | ⚠️ 建议跟踪（非阻塞） |
| review-2 发现 2：pendingDependenciesLocked 注释不符 | 函数头注释未改（见发现 B） | ⚠️ LOW |
| review-2 发现 3：autoAssign×Remove 双条目 | 路径仍在（L608 旧条目标 blocked，Assign 内新注册 pending） | ⚠️ LOW，窗口极窄 |
| review-2 发现 5：H5/H6/H9 测试缺口 | H6 已补；H5/H9 未补（见发现 F） | ⚠️ LOW |

---

## 四、结论

**通过。**

三项阻塞修复全部真实落地且有测试钉住（非文字声称）：killed/failed/interrupted 事件只落定终态 +
解锁依赖门、绝不自动推进后继（L512 短路 + 测试）；回调不再承载同步 Assign（enqueueAuto 非阻塞 +
单 worker 串行 autoAssignByID，回调剩余 IO 仅为轻量 ReadDir + 非阻塞入队）；blocked 任务在下一次
Done 完成事件重扫恢复（有真实场景测试）。队列满降级（enqueueAuto select-default → blocked + reason）
与 Close 生命周期（workerOnce + doneCh、全测试 defer Close）静态验证成立，父代理已声明
`go test`/`-race`/goleak 全绿。生产接线（inboxRoot + SetSink）与 boot 注释也已闭环。

**必须修复项：无。**

**建议修复项（不阻塞，随下一事务处理）**：发现 A/B/C 三处注释与实现不符/重复（含 HandleJobDone
"同步 Assign"残留——本轮 worker 化核心改动的注释必须同步，建议尽快）；发现 D Close 补 drain+WaitGroup；
发现 F 测试信号强化与 H5/H9 补齐；发现 E 的 blocked 恢复窗口在注释中显式声明；环检测（review-6 攻 2）
另立跟踪项。

**遗留流程问题（如实记录）**：本报告为静态交叉验证，未独立重跑 `go test`（本环境无 shell）；
回归证据依赖父代理声明，建议在可执行环境补跑 `go test ./internal/jobs/ ./internal/agent/ ./internal/boot/ -count=1`
与 `go test -race ./internal/jobs/ ./internal/agent/` 留档（execution.md 仍缺失）。
