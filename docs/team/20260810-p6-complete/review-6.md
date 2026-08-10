# 纪律审查报告：P6 完成事件驱动（devil's advocate 独立复审，commit 8f76af02f）

> 角色：**纪律 Discipline**（独立对抗视角，与执行小队、规划、前几轮纪律隔离）
> 审查对象：commit 8f76af02f 工作区现状 —— `internal/agent/teammate_store.go`、
> `internal/jobs/jobs.go`、`internal/boot/boot.go`、`internal/control/controller.go`、
> 测试（`teammate_done_e2e_test.go` / `teammate_store_test.go` / `jobs_done_observer_test.go`）
> 权威裁决：`arbitration.md`（仲裁 1-7）+ `lifecycle-design.md`（表项 3 / 主题 3，killed 语义权威）
> 方法：静态交叉验证 + 锁序逐交错推演 + 测试证据逐条复核（本环境无 shell，无法重跑
> `go test`；证据真实性按「代码行 ↔ 测试断言 ↔ 文档裁决」三方对账判定）。
> 前轮：review-1/2/3/4 已审查；本报告独立重推全部 7 个攻击面，并核对前轮阻塞项是否闭环。

---

## 〇、总览（对抗结论先行）

| # | 攻击面 | 风险 | 一句话结论 |
|---|--------|------|-----------|
| 1 | run goroutine 同步执行 + 锁序 | **高**（时序/设计） | 锁序正确、**无死锁**（m.mu 未被持有已重证）；但同步 Assign 阻塞 job 收尾，**违反 datamodel 三禁 / lifecycle 表项 4 的 worker 化裁决** |
| 2 | 依赖环死等 | **中** | 环安全（不死循环）但**无环检测、无报警、/team-status 不显示 Tasks()** → 用户零感知死等 |
| 3 | mailbox 唤醒重复/风暴 | **低-中**（生产**阻塞**） | 聚合逻辑正确、绑定翻转防重复（有测试）；但 boot 未传 inboxRoot → **生产积压唤醒 + mail 投递空转**（review-4 F1 未闭环） |
| 4 | autoAssign 失败状态残留 | **高** | blocked **无重试、无手动重推、无 UI 可见**；瞬时失败（slot 满 / owner 并发占用）永久化 → 依赖链静默断 |
| 5 | 完成事件 vs Remove/TeamStop | **中**（语义**高**） | Remove 路径安全（先删后 kill）；TeamStop→Killed 事件**触发后继自动推进**，违反 lifecycle 表项 3「只有 done 才推进」 |
| 6 | 新字段/新常量耦合 | **低** | taskPending/taskBlocked 只在 store 内使用，jobs 层无穷举 switch；但类型上是 jobs.Status 的「越界值」，未来有踩坑面 |
| 7 | 未测试边界 | **中** | 环 / 多级链 / Failed·Killed 语义 / blocked 恢复 / slot 满 全部无测试；**killed 语义未固化 = 语义漂移无护栏** |

**结论：驳回**。主体质量高（锁序、幂等、G8 过滤、聚合唤醒均有实证），但存在
**两个 HIGH 语义/设计偏离、一个 HIGH 状态残留、一个生产接线阻塞项**，且前轮
review-2/4 的阻塞发现全部未闭环、执行队未在交付物中上报 → 防虚假完成红线未守。

---

## 一、审查项逐条

- **[FAIL] fable5 合规（执行队是否跳步）**
  - 实现落点齐全（仲裁 1-6 均有代码），锁序/时序推演（R-a~R-e）有文档覆盖。
  - **但跳步三处**：①lifecycle 表项 3「只有 done 才自动推进」→ 实现全终态推进（见攻击面 5）；
    ②lifecycle 表项 4「自动推进必须 worker 化（OnJobDone 内禁同步 Assign）」→ 实现同步 Assign（见攻击面 1）；
    ③risk-review G4（环检测/链深上限/失败策略显式）与 G5（backoff 重试）**整体未落地**且未申报范围裁剪。
  - p6-complete 目录**无 execution.md**，测试命令与输出无记录（review-2/4 同注）。

- **[FAIL] 幻觉检测（证据真实性）**
  - 真实部分：测试文件均存在、断言与实现逐条对应、无 t.Skip/空断言；`teammateRef`、
    `teammateJobID`、`releaseBlockingProvider` 等符号均可解析；observer 时序（recordCompletion →
    fireJobDoneObservers → status flip → close(j.done)，jobs.go L675/1034/679/686）与测试
    注释一致 —— **代码行、测试、文档三方对账成立**。
  - **虚假/失真部分**：①boot.go L1901-1904 注释声称「E2 not landed / OnJobDone 未合入」与
    现状（HandleJobDone 已落地、构造内 L111 已自动注册）**直接矛盾**，照注释执行会编译失败；
    ②teammate_store.go L323 注释称 pendingDependenciesLocked「consults the manager as a
    fallback」，函数体**从不 consult**（未登记 dep 一律 st=="" 永不放行）；
    ③teammate_store.go L425-429 注释称「synchronous Assign inside the callback is safe per
    arbitration 3」—— 锁序上确实安全，但把「安全」偷换为「应当」，掩盖了与 worker 化裁决的冲突。

- **[PASS] 缓存红线（前缀字节稳定）**
  - 自动 Assign prompt = `recordPendingTaskLocked` 登记原文快照（teammate_store.go L61/L310，
    autoAssign L524 原样回放）→ 字节稳定。
  - fork_first 分支 `captureForkPrefix`（task.go L1280）只读 leader、零发送；continue 分支
    `ContinueFrom: ref` 前缀稳定；完成 Notice 走 `event.Sink`（UI 层）不进 provider 输入。
  - **残余**：自动推进的 job 完成 → P1 信封 → leader `<background-jobs>` 注入 committed
    history（既有机制，但完成事件使增长更频繁）；**链深上限（risk-review G4-e=32）未实现**，
    committed 增长速率无显式约束。低-中风险，非本次字节回归。

- **[PASS（静态）] 回归（测试命令 + 结果）**
  - 本环境无 shell 无法重跑；静态回归：`SetJobDoneObserver` 全仓唯一调用点 =
    `NewTeammateStore` 构造（grep 核实），替换语义不覆盖既有 observer；`Complete`/`TeamStop`/
    `List`/`Remove` 旧 API 签名未动；jobs 层 `recordCompletion` 入队/Emit 段字节未改。
  - 必须由执行队在提交环境补跑并附输出：
    `go test ./internal/jobs/ ./internal/agent/ ./internal/boot/ -count=1`
    `go test -race ./internal/jobs/ ./internal/agent/`（重点：teammate 全部测试 + TestDrainMultiple + TestDestroySession*）
  - 注意：**现有测试全绿并不能证明本报告的攻击面 1/2/4/5 安全** —— 那些场景无测试覆盖（见攻击面 7）。

- **[FAIL] 对抗自检（devil's advocate 攻击发现的缺陷）**
  见攻击面 1-7。核心三连：killed/failed 也自动推进（违裁决）、同步 Assign 阻塞收尾（违
  worker 化）、blocked 无重试无可见（违 G5）+ 生产 inboxRoot 空转（F1 未闭环）。

---

## 二、攻击面逐项（风险等级 + 证据 + 建议）

### 攻击面 1：HandleJobDone 在 run goroutine 同步执行 —— 锁序与阻塞

**锁序（用户要求重新验证 m.mu 是否持有）—— 结论：正确，无死锁。**
- 证据链：`fireJobDoneObservers`（jobs.go L1057-1071）在 m.mu 内**只快照** observer 切片
  （L1058-1061）→ `m.mu.Unlock()`（L1061）→ **锁外**循环调用、per-observer recover
  （L1066-1069）。⇒ **HandleJobDone 执行时 m.mu 未被持有**。
- HandleJobDone（teammate_store.go L430-474）自身：ts.mu 短临界（L434-464）→ 锁外
  countInbox / notifyMailBacklog / autoAssign；autoAssign（L517-555）→ Assign（L194-278）→
  ts.mu 短临界 → 锁外 RunProfileSpec → StartForSession（jobs.go L564-593）m.mu.Lock 为
  **独立临界区**。ts.mu 与 m.mu 全程无嵌套。→ 仲裁 3「startForSession 重入是独立临界区」
  **成立**。destroy 窗口（recordCompletion L1012-1029 m.mu 内 Emit）与 Assign 路径无锁序相交。
- 证据：jobs.go L1057-1071 / L564-593；teammate_store.go L434-464 / L517-555。

**同步重活阻塞 job 收尾 —— 结论：HIGH，且违反 design 裁决。**
- 证据：`recordCompletion` 在 run goroutine 内（jobs.go L675），其内部 `fireJobDoneObservers`
  → HandleJobDone **同步**执行于 `j.status=st`（L679）与 `close(j.done)`（L686）**之前**。
- HandleJobDone 内同步段含：`countInbox` = `os.ReadDir` 磁盘 IO（L467/L688）+ autoAssign →
  Assign → RunProfileSpec 同步段 = `prepareTranscriptForkWithPrompt`（task.go L1254-1303：
  `captureForkPrefix` + **`PrepareParentFork` 磁盘写**）或 continue 分支、`ReserveStartForSession`
  （jobs.go L953，m.mu 临界）、`MarkRunning`（L963-969 磁盘）、`StartForSession`（L564-599，
  m.mu 临界 + `openArtifactLocked` 磁盘）。
- 后果：`j.done` 关闭延迟 → `Wait`/`ClaimForegroundResult`（阻塞 `<-j.done`）/onJobStart 的
  done 消费者（delivery workspace 写租约释放）全部延迟；fan-out N 个后继时 run goroutine
  收尾被串行拉长 N 倍。
- **设计违规证据**：lifecycle-design.md 表项 4「**自动推进必须 worker 化**（datamodel 三禁：
  OnJobDone 内禁同步 Assign）」+ L80「本 handler 全部内存级 + 可选 ReadDir」+ L87「推进由
  worker 入队执行」。实现把 `for _, t := range ready { ts.autoAssign(t) }`（L471-473）直接
  放进回调。arbitration 2 的理由「handler 全为内存级操作」**与实现不符**。
- 建议：
  1. 推进移出回调：HandleJobDone 只做「终态落定 + 置 idle + mailbox 计数」，把 ready 任务
     append 进 store 自有有界队列（128），独立 worker 串行消费（DestroyAll 时排空）；回调
     文档照抄 jobs.go L295 的「must return quickly」。
  2. 若 MVP 坚持同步（当前实现），必须在事务文档显式声明「同步 Assign 阻塞收尾为已接受
     权衡」并补 risk-review 验证点 2 的测试（回调 sleep 2s → 断言 j.done 关闭 <500ms——
     当前实现**必然失败**，故不能宣称满足 G1）。

### 攻击面 2：依赖环死等

- 证据：A 依赖 B、B 依赖 A → 两次 Assign 均被 `pendingDependenciesLocked`（L324-344）拒绝
  → 双双登记 pending（L221/L305-317）→ 无 job 启动 → 无完成事件 → **永不推进，死等**。
  `depsTerminalLocked`（L480-488）对未知 dep 返回 false → 不自动推进（安全，无死循环，仲裁 3
  缓解 2 成立）。
- **缺口**：①**无环检测**——teammate_store 无任何 cycle/DFS 逻辑（fleet_graph.go L69-94 有
  现成 Kahn 算法可复用，未接）；②`Tasks()` 对环中任务只显示 `pending`（L314），无
  「waiting_on / 环」原因；③**/team-status 根本不显示 Tasks()**（controller.go L6380-6393 仅
  roster：name/state/role/job）→ 环死等对用户**零可见**、零报警。
- 风险：**中**（正确性安全，可用性/可诊断性不满足；risk-review G4-a 环检测、G4-e waiting_on
  原因均未落地）。
- 建议：登记 dependsOn 时做闭包 DFS 环检测（复用 fleet 思路，错误含 `cycle: A → B → A`）；
  至少把 `/team-tasks`（渲染 Tasks() 含 pending/blocked/waiting_on）接入 controller，否则
  环/洞/blocked 全部黑盒。

### 攻击面 3：mailbox 唤醒重复/风暴

- 证据：`notifyMailBacklog`（L709-722）绑定 `Running→Idle` 翻转（L443-451）且仅在
  `countInbox>0` 时发一条（L466-470）；重复事件不重发（单测
  `TestTeammateHandleJobDoneMailboxWakeup` L618-626 断言恰好 1 条）→ **防重复正确**。
- 多 job 完成连发：每完成一个 teammate 至多 1 条聚合 Notice（含 N），非每 mail 一条 →
  风暴面窄。
- **但生产空转（阻塞）**：boot.go L1905 `NewTeammateStore(taskTool, jm)` 仅 2 参，
  `inboxRoot==""` → HandleJobDone L466 `flipped != "" && ts.inboxRoot != ""` 短路、
  countInbox L684 恒 0、flushMailbox L743 短路。⇒ **仲裁 4 的核心交付「完成 → 积压聚合
  唤醒」在生产永不触发**；且生产 PostMail 走 ephemeral 分支（L648-653）只发即时 Notice、
  **mail 文本永不投递**（无 SendMessageForSession），teammate 间 mail 是「只通知不投递」。
  （= review-4 F1，**未闭环**。）
- 风险：**低-中**（实现逻辑正确，生产接线不完整）。
- 建议：二选一——(a) boot 接 inboxRoot（config 目录 + 落盘接线，另立跟踪项）；(b) 在
  boot 注释与事务文档**显式明示**「生产 mail 落盘/积压唤醒未接线，当前仅即时通知」，并禁止
  宣称「生产 mailbox 唤醒完整生效」。boot.go L1901-1904 过时注释（F2）同步必改。

### 攻击面 4：autoAssign 失败状态残留 —— 谁重试？

- 证据：失败写 `taskBlocked + BlockedReason`（L527-529 owner 删除；L542-547 Assign 失败）。
- **谁重试？无人。** ①ready 扫描只认 `Status == taskPending`（L456）→ blocked **永不重新
  尝试**；②无 backoff 重试（risk-review G5「3 次 × 1s/5s/30s」未实现）；③无 /team-retry
  手动重推；④/team-status 不显示 Tasks() → blocked **对用户不可见**。
- **瞬时失败永久化（HIGH 核心）**：autoAssign 的 Assign 会因这些**瞬时条件**失败——
  a) owner 被并发 `/team-add` 占用 → running gate（L202-204）拒绝；
  b) 并发 job 占满 slot → `ReserveStartForSession` 拒绝（task.go L953-957
  "N background tasks are already running…limit M"）→ Assign err → blocked；
  c) leader 未捕获（ForkSource 缺失）→ fork_first 失败（task.go L1271-1273）。
  三者失败后**等条件自然恢复也不会重试** → 依赖链静默断裂。
- 反例对照：手动 Assign 的 gate 拒绝**不登记**（L202-204 直接返回，teammate 保持 Idle 可
  手动重试）—— 手动有出口，**自动推进无出口**。risk-review 2.3 场景 A/B 的缓解（G5 两步
  显式 + 重试）未落地。
- 风险：**高**（真实多 teammate 并行场景极易触发 slot 满 → 永久 blocked → 用户无感知）。
- 建议：
  1. blocked 入重试队列（有界 backoff 3 次）+ 最终失败保持 blocked 并记日志；
  2. Assign 的 slot 满错误对 autoAssign 特殊处理（本次跳过、下次完成事件再评估，而非
     blocked）；
  3. controller 增 `/team-tasks` 展示 pending/blocked + reason，并允许 `/team-retry <taskID>`
     手动重推（G4-e 原方案）。

### 攻击面 5：完成事件 vs Remove/TeamStop 并发竞态

- **Remove 路径：安全。** Remove（L605-629）先删 teammates + 该 owner 全部 tasks → 锁外
  Kill；Killed 事件到达时 tasks 无该 id → HandleJobDone L435-438 no-op（单测
  `TestTeammateRemoveClearsTasks` L631-655 + `TestTeammateHandleJobDoneOwnerRemovedKeepsTask`
  覆盖）。✅
- **TeamStop 路径：语义违规（HIGH 子集）。** TeamStop（L580-600）置 Idle → KillForSession →
  Killed 事件 → HandleJobDone：settle 写 killed → idle flip no-op（State 已 Idle）→ **ready
  扫描见 killed 终态 → 自动推进后继**。而 lifecycle-design.md 表项 3 / 主题 3（L134-146）
  权威裁决：「killed/failed/interrupted **只落定终态 + 解锁依赖门，绝不自动启动后继**；
  只有 done 才触发推进；自动重启 = 覆盖用户意图 + 无限循环」。`isTerminalStatus`（L398-404）
  含 Killed/Failed/Interrupted 且 ready 扫描（L455-463）不区分 st → **A 被杀/失败 → B 被
  自动 Assign**。**= review-2 发现 1，未闭环**（arbitration.md L29「Killed 依赖放行」只裁决
  放行门，未裁决推进；实现把放行误读为推进授权，正是 lifecycle L146 警告的误读）。
- **autoAssign × Remove 双条目（LOW）**：autoAssign 的 Assign 内 gate 因依赖消失拒绝 →
  `recordPendingTaskLocked` 新 placeholder（L221）→ autoAssign 标旧条目 blocked（L542-547）
  → 同任务 pending + blocked 双条目。= review-2 发现 3，未闭环。窗口极窄（并发 Remove）。
- 风险：**中**（Remove 安全；TeamStop→killed→推进为 HIGH 语义缺陷；双条目 LOW）。
- 建议：`HandleJobDone` 在 ready 扫描前对 `st != jobs.Done` 直接跳过推进段（依赖门照常
  放行），补 `TestOnJobDoneKilledDoesNotAutoAdvance` 钉住；autoAssign 失败路径若识别
  「依赖消失」则删除 Assign 内新注册的孤儿 placeholder（或 Assign 提供「不登记」选项）。

### 攻击面 6：新字段/新常量（taskPending/taskBlocked）耦合

- 证据：`taskPending="pending"` / `taskBlocked="blocked"`（teammate_store.go L70-73）声明为
  `jobs.Status` 类型；grep 全仓**仅 teammate_store.go + 测试引用** → 未泄漏到 jobs 层/control
  层输出；`isTerminalStatus`/`depsTerminalLocked`/`pendingDependenciesLocked` 均把 pending/
  blocked 当非终态正确处理（L336-341 default 分支 / L480-488）。
- **残余**：jobs.Status 是 jobs 包语义域，store 往其中注入「越界值」属类型误用——未来若
  jobs 层新增穷举 switch（如渲染/持久化）会踩未知值；TeamTask 目前纯内存无持久化，风险未
  放大。`BlockedReason` 只在 blocked 时设置，无 UI 消费（见攻击面 4）。
- 风险：**低**。
- 建议：TeamTask.Status 改为独立 teamTaskStatus 类型 + 显式转换，或至少在 jobs 层文档标注
  「jobs.Status 存在包外扩展值」，防未来穷举。

### 攻击面 7：未测试边界（对照 risk-review 验证点）

| 边界 | 状态 | 说明 |
|---|---|---|
| 环 A↔B 死等 | **缺失** | 无测试；且无环检测（攻 2） |
| 多级链 A→B→C | **缺失** | review-2 发现 5 H5，未补 |
| Failed/Killed 依赖语义 | **缺失** | review-2 发现 5 H6，未补；**实现与裁决相反**（攻 5） |
| blocked 恢复 / 重试 | **缺失** | 无实现（攻 4） |
| slot 满 → autoAssign blocked | **缺失** | 真实高频路径无测试（攻 4-b） |
| fan-in 双完成并发推进 | **缺失** | 推演最终一致（幂等 delete 兜底），但瞬时 blocked 误标无测试钉住 |
| 回调重活阻塞收尾 | **缺失** | 验证点 2「回调 sleep → j.done<500ms」当前实现必然失败 |
| Remove×autoAssign 双条目 | **缺失** | review-2 发现 3 建议补并发测试 |
| destroy 窗口吞事件 | 部分 | 幂等测试有（TestTeammateHandleJobDoneIdempotent），destroy 时序未测 |
| G8 silent/foreground 过滤 | **缺失** | 逻辑正确（tasks 过滤），无显式 P5 fork 完成不推进测试 |

- 风险：**中-高**（缺口的共性是：**所有「自动推进」的错误分支都无测试**，而自动推进恰是
  本次最大新逻辑面）。

---

## 三、前轮阻塞项闭环核对

| 前轮发现 | 现状 | 闭环？ |
|---|---|---|
| review-2 发现 1（HIGH）：killed 触发自动推进 ≠ lifecycle 表项 3 | `isTerminalStatus`+ready 扫描仍全终态推进 | **未闭环** |
| review-2 发现 2（LOW）：pendingDependenciesLocked 注释与实现不符 | L323 注释仍称 consult jm，函数体不 consult | **未闭环** |
| review-2 发现 3（LOW）：autoAssign×Remove 双条目 | 代码路径仍在 | **未闭环** |
| review-2 发现 5（MEDIUM）：H5/H6/H9 测试缺口 | 未补 | **未闭环** |
| review-4 F1（阻塞）：生产 inboxRoot 未接线 → 积压唤醒+mail 投递空转 | boot L1905 仍 2 参 | **未闭环** |
| review-4 F2（必改）：boot.go L1901-1904 过时注释 | 原文仍在，含错误代码指引 | **未闭环** |

**六个前轮问题零闭环 + 无任何修复/裁决修订记录** —— 这是「宣称完成但未自证」的防虚假完成
红线违反。

---

## 四、结论

**驳回**（附驳回理由与必须修复项）。

**通过的部分（如实记录）**：锁序无死锁（m.mu 未持有已重证）；observer 机制（快照+recover+
幂等）实现正确；G8 过滤（tasks 存在性）正确；mailbox 聚合防重复正确（有测试）；缓存红线
零字节回归；Remove 清理 tasks（仲裁 6）正确；assignContext 重建（存 SessionID 字符串、
不缓存 ctx、leader 存 *Agent）正确。

**必须修复（阻塞）**：
1. **killed/failed/interrupted 不得自动推进后继**（lifecycle 表项 3：只有 done 推进）。
   `HandleJobDone` ready 扫描前按 `st != jobs.Done` 短路推进段；补
   `TestOnJobDoneKilledDoesNotAutoAdvance`。或父代理显式修订裁决并更新文档（当前实现与
   权威裁决矛盾，任选其一但必须闭环）。
2. **推进 worker 化**（lifecycle 表项 4 / risk-review G1）：OnJobDone 内禁同步 Assign，
   有界队列 + 单 worker；若坚持同步则显式申报权衡并补收尾延迟测试（当前必然失败，不得
   宣称满足 G1）。
3. **blocked 可恢复**（risk-review G5）：backoff 重试 / slot 满跳过而非 blocked / 至少
   `/team-tasks` + `/team-retry` 让用户可见可推；当前瞬时失败永久化 + 零可见 = 静默断链。
4. **生产 inboxRoot 接线或显式降级声明**（review-4 F1）+ **boot.go L1901-1904 注释修正**
   （F2，照注释执行编译失败）。
5. **环检测 + 链深上限**（risk-review G4-a/e，fleet 有现成 Kahn 可复用）。

**建议修复**：pendingDependenciesLocked 注释修正（发现 2）；autoAssign×Remove 双条目护栏
（发现 3）；补齐环/多级链/killed 语义/slot 满/fan-in 测试（攻击面 7 表）。

**回归命令（执行队在提交环境补跑并附输出，本次未执行）**：
`go test ./internal/jobs/ ./internal/agent/ ./internal/boot/ -count=1`
`go test -race ./internal/jobs/ ./internal/agent/`

---

*本报告为纪律独立复审；实现与修复由执行小队承担，修复后需复审（重点复核攻击面 1/4/5 与
前轮六项闭环）方可通过。*
