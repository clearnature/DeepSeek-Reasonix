# 纪律审查报告：P6 完成事件驱动 — 第二轮 D2 修复复审

> 复审对象：commit bbc444955 + 8f76af02f（team 分支，工作区 /home/yanli/work/DeepSeek-Reasonix）。
> 背景：第一轮驳回项 D2（幽灵事件）已修复；本轮验证
> `internal/jobs/jobs.go`（fireJobDoneObservers 的 root.Done 守卫、suppress 路径 destroying 守卫）、
> `internal/jobs/jobs_done_observer_test.go`（新增 2 测试）、`internal/boot/boot.go`（注释清理）。
> 父代理已真实验证：`go test ./internal/jobs/` ✅、`go test -race` ✅。

---

## 审查项逐条

### 1) [PASS] fable5 合规（执行队是否跳步）

本轮执行队针对 D2 修复的推进路径完整可追溯：

- **任务分解**：`fireJobDoneObservers` root.Done 守卫（jobs.go:1065-1070）+ suppress 路径 destroying 守卫（jobs.go:1008-1016）+ 2 个新测试（Close Killed 场景 / destroy silent 变体）+ boot.go 注释清理（boot.go:1898-1906，已无「has not landed yet」过时文案）——分解与 review-1 必须修复项 1/3 一一对应。
- **拓扑扫描**：grep 确认 `fireJobDoneObservers` 全库仅两处调用（jobs.go:1015 suppress 路径、1042 正常路径），两处均处于守卫之下，无第三挂点遗漏。
- **多路径推演 + 自我验证**：父代理附 `go test ./internal/jobs/` + `-race` 真实验证结果（本审查环境无 shell 执行工具，交叉验证见审查项 2）。
- **持久记忆**：修复引用 review-1 的 D2/J7/J8 语义，测试注释写明「discipline review-1 D2」「destroy window swallows silent completion」。

**遗留项（非本轮范围，单列）**：review-1 必须修复项 2 中 J2（FiresOnFailed）/ J3（FiresOnKilled）/ J4（startInvalid）/ J1 参数一致性（err/kind/label/session）/ J12（同步固化）/ SetJobDoneObserver 替换语义等测试仍未补——`jobs_done_observer_test.go` 现共 8 个测试，无上述矩阵项。本轮聚焦 D2 不构成跳步，但 review-1 清单未全部闭环，须在后续轮补齐。

### 2) [PASS] 幻觉检测（证据真实性）

- **代码真实性**：所有守卫、挂点、调用点、行号均在本工作区逐行核对（见下文验证点），无编造行号/编造实现。
- **测试真实性**：两个新测试真实存在（jobs_done_observer_test.go:175-191、196-212），断言与注释自洽。
- **证据重跑**：本审查环境无 shell 工具，无法独立重跑 `go test`；父代理提供 ✅ 结果。静态推演（见审查项 3/4）支持「测试可通过且能捕获修复前缺陷」的结论；`-race` 结论在逻辑上成立——守卫处无新增共享状态竞态（root.Done 读是 channel 操作，destroying 读在 m.mu 下），capture 有独立 mutex。
- **agent 层交叉验证**：`teammate_store.go:120 jm.SetJobDoneObserver(ts.HandleJobDone)`、`HandleJobDone`（teammate_store.go:476）接线完好，本轮改动未触碰 handler 逻辑。

### 3) [PASS] 缓存红线（发送侧零变化）

- 本轮改动全部在 jobs 层进程内回调守卫 + 测试 + boot.go 注释文案（1901-1903 仅注释），未触碰 provider 输入、schema、transcript 发送侧。
- 自动投递/steer：`recordCompletion` 的 `completed` append 逻辑（jobs.go:1029-1034）与 envelope 渲染（renderResultEnvelope）未改动，前缀字节稳定；steer 仅 append turn 尾部，未触碰。
- boot.go:1898-1906 注释更新为「completion observer is registered inside NewTeammateStore」，与代码一致，无误导性声明。

### 4) [PASS] 回归（命令 + 结果）

- 父代理证据：`go test ./internal/jobs/` ✅、`go test -race` ✅（本环境无法独立重跑，静态推演如下）。
- **RecordDone 仍触发**：正常路径 jobs.go:1039-1041 无条件触发；suppress 路径 1005-1007 在 destroying 检查**之前**触发——与 review-1「保持 RecordDone 现有行为不变」一致，taskRecorder 生命周期 hook 未被守卫误伤。
- **正常路径不受影响**：
  - destroying 检查带 `parentSession != ""` 空会话守卫（jobs.go:1012、1021），无会话 job 永不进入 destroy 吞事件分支；
  - root.Done() 守卫（jobs.go:1066-1069）只拦截 Close 后的触发，Close 前正常完成仍触发（TestJobDoneObserverFiresOnCompletion / MultipleObserversAllFire / Panic 等既有 6 测断言之基础路径未变）；
  - `fireJobDoneObservers` 仅两处调用，无其他调用方被守卫波及。
- **依赖方**：teammate_store.go:120 构造器接线、boot.go:1908 生产接线均完好；boot.go 过时注释已清理（review-1 必须修复项 3 ✓）。

### 5) [PASS] 对抗自检（devil's advocate）

验证点 1 — **root.Done() 守卫在 Close 后拦截（含 Killed 场景）** ✓

- `fireJobDoneObservers`（jobs.go:1065-1070）入口 select `<-m.root.Done()` → return。`CloseWithGrace`（jobs.go:2225）先 `m.cancel()`（关闭 root.Done）再 `wg.Wait()`；run goroutine 收尾顺序为 `recordCompletion`（L675）→ `close(j.done)`（L686）→ `wg.Done()`（defer），故 Close 返回前 Killed 完成的 recordCompletion 必然已看到 root.Done 关闭 → observer 被吞。非合作 job 超过 teardownGrace 的场景同样被拦截（root.Done 一旦关闭永不复位）——正是 review-1 审查点 3 的目标窗口。
- Close 测试（jobs_done_observer_test.go:175-191）有 `<-started` 显式屏障，确定性成立；断言 `len==0` 在修复前必失败（observer 收到 `id|killed`），能捕获缺陷。✓

验证点 2 — **suppress 路径 destroying 守卫对称** ✓

- suppress 路径（jobs.go:1011-1016）：m.mu 下读 `destroying[parentSession]`，为 true 时跳过 `fireJobDoneObservers`，与正常路径（jobs.go:1020-1024 锁内 return）语义对称——两条路径在 destroy 窗口内均不触发 observer，满足 D2「两种路径一致」。
- Destroy 测试（jobs_done_observer_test.go:196-212）用 `StartSilentForSession`，修复前 suppress 路径无条件 fireJobDoneObservers → 断言 `len==0` 必失败，能捕获缺陷。✓

验证点 3 — **测试真实覆盖** ✓（附 1 项低风险观察）

- 两个新测试均真实存在、断言有效、修复前必红。
- **观察（低风险，不阻塞）**：Destroy 测试无显式同步屏障（Close 测试有 `<-started`，destroy 测试没有）。测试通过依赖 Go 调度行为（新 goroutine 在调用 goroutine 无让出点期间不会抢先运行；`event.Discard` 是空 FuncSink、taskRecorder 为 nil，StartSilentForSession 返回至 BeginDestroySession 之间无阻塞/系统调用，测试 goroutine 先拿到 m.mu 置 destroying）。该推理在常规运行下稳定，但建议后续将 `_ = j2` + 重复两次 `BeginDestroySession`（L205/L207 各一次，幂等无害但属编辑残留）改为显式屏障（如 run 中 `close(started)` 后等待 destroy 先行信号），彻底消除对调度器的隐式依赖。

验证点 4 — **回归** ✓（详见审查项 4）

**对抗发现的附加边界（均不阻塞）**：

1. **suppress 路径 RecordDone 在 destroy 窗口内仍触发**（jobs.go:1005-1007 先于 destroying 检查），而正常路径 destroy 时连 RecordDone 一起吞（L1020-1024）——不对称。但 review-1 明言「保持 RecordDone 现有行为不变」，且 taskRecorder 是监控 hook、不走 HandleJobDone/autoAssign 幽灵派活链，风险低，属已裁决边界。
2. **destroying 检查与 fireJobDoneObservers 之间 TOCTOU**（锁释放后调用前若 BeginDestroySession 并发启动，observer 仍触发）——窗口极小，且 FinishDestroySession 后 HandleJobDone 经 `ts.tasks` 查无此 id 幂等 no-op（review-1 攻击点 4 已论证），可接受。
3. **Close 后正常路径仍 append completed / RecordDone / emit Notice**（jobs.go:1029-1041、1053-1055）——review-1 明确仅要求 observer 级守卫，且无自动派活链，属既有行为。

---

## 结论

**通过**（针对第二轮 D2 修复的本复审任务范围）。

驳回理由（review-1 的 D2 幽灵事件）已消除：`fireJobDoneObservers` 现于入口拦截 `m.root.Done()`（Close 后 Killed 完成不再触发 observer），suppress 路径补上 destroying 守卫（与正常路径对称），两个新测试真实存在、断言有效、修复前必红；boot.go 过时注释已清理；RecordDone 与正常路径行为未回归。缓存红线零触碰。

### 遗留项（不阻塞本轮 PASS，须后续轮闭环 review-1 清单）

1. review-1 必须修复项 2 中 J2（FiresOnFailed）/ J3（FiresOnKilled）/ J4（startInvalid）/ J1 参数一致性（err/kind/label/session）/ J12（同步固化）/ SetJobDoneObserver 替换语义测试仍未补。
2. Destroy 测试建议加显式同步屏障（消除对 Go 调度器的隐式依赖），并清理 `_ = j2` 与重复 `BeginDestroySession` 编辑残留。
