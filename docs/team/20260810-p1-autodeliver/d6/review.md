# 纪律审查报告：P1 后台结果自动投递（执行现状核查 + T2 补齐签名级建议 + 独立裁决 d6）

> 审查人：纪律 Discipline（独立视角，不参考 d1/d2/d5 结论，仅对照工作区实际代码与 plan.md 定稿）
> 审查基线：`docs/team/20260810-p1-autodeliver/plan.md`（一致性定稿 T1–T6）+ `plan-3.md`（被采纳方案，方案 A/B/C 与对抗自检）
> 必读核对：`plan.md`、`internal/jobs/jobs.go`、`internal/jobs/jobs_test.go`、`internal/agent/preview_test.go`、`internal/control/input.go`、`internal/history/strip.go`、`internal/control/input_test.go`、`internal/jobs/jobs_extra_test.go`
> 证据方式：静态交叉验证（read_file/grep 实测文件内容）。本环境无 shell 执行能力，**测试通过性以父代理已真实验证的「全部测试通过」为准**；本报告做代码级交叉验证与独立裁决（详见「幻觉检测」与「回归」）。

---

## 一、执行现状核查（幻觉检测 · 逐任务，全部可复核）

| 任务 | 定稿要求 | 现状 | 判定 | 可复核证据 |
|------|----------|------|------|-----------|
| T1 | jobs 结构化完成记录 + 只读快照 | **部分完成**：`completion` 扩展了 `result`（bounded 快照）字段但**缺 id/status/kind/label/artifact 结构化字段与 envelope**；`boundedResult`（jobs.go:813-829，4096 rune 安全 + marker 计入预算）、`renderResultEnvelope`（1294-1309，5 字符 XML 转义）、`ResultSnapshotForSession`（1317-1326）已落地；`recordCompletion`（835-881）j.mu 内拷贝 → m.mu append，锁序合规 | ⚠️ 未全量 | jobs.go:193-197 / 813-829 / 835-881 / 1294-1326 |
| T2 | input 注入接线（容器内信封渲染） | **未实施（blocked）**：input.go:191-195 仍 `DrainCompletedNoteForSession` 纯文本一行摘要；`DrainCompletedNoteForSession`（jobs.go:1252-1277）仍全量 drain + `strings.Join(c, "; ")` 拼接 | ❌ blocked | input.go:192-193；jobs.go:1252-1277 |
| T3 | bounded 单测（截断/条数/总量/丢弃/XML 转义） | **部分**：rune 截断、伪造关闭标签转义、只读快照、单条 body bounded 已加；**8 条/次、16KB/块、丢最旧、`<result-overflow>` 测试全部缺失** | ⚠️ 部分 | jobs_test.go:823-1009（4 个测试） |
| T4 | 剥离回归（preview/strip 零改动） | **完成**：`TestPreviewStripsBackgroundJobResultEnvelope` + `TestPreviewStripsEscapedForgeAttempt`；preview.go:19-29 `TransientUserBlockTags` 含 `background-jobs`、history/strip.go:8 `reComposeBlock` 含 `background-jobs`，**二者零改动** | ✅ | preview_test.go:196-237；preview.go:19-29；strip.go:8 |
| T5 | 端到端 | 未实施（依赖 T2） | ❌ | 全仓 grep `background-job-result` 仅 jobs_test.go/preview_test.go 有 |
| T6 | 文档 | `execution.md` 不存在；`docs/team/20260810-p1-autodeliver/` 仅 plan-1/2/3/plan.md + d1/d2/d5 review | ❌ | glob 实测目录内容 |

**防虚假完成结论**：事务**尚未完成**，处于 blocked/进行中。任何「P1 全部完成」声明均为功能级虚假完成——测试全绿（T3/T4 均为防御性/单元测试）不等于产品功能落地（信封零生产调用方、注入路径未接线）。T1/T3/T4 的代码与测试**真实存在且逻辑自洽**（已逐行核读，非编造），这点须与「未完成」分开陈述。

---

## 二、审查项逐条

### [PASS/有条件] fable5 合规——执行队未跳步，但任务未完成
- T1/T4 已实施部分：锁序（recordCompletion 先 `m.get` 释放 m.mu → 再 j.mu → 再 m.mu，无嵌套）、快照时机（终态发布前调用，jobs.go:535-541 注释与实现一致）、只读语义均有对应代码与测试，**无「只改注释」蒙混**。
- T2 blocked、T3 部分、T5/T6 未启动，执行队**如实停在 blocked**（见第三节证据），未在接口缺口下硬改 input.go 注入层 = **未越权**。
- 越权面核查：T1 改动仅 `internal/jobs/jobs.go` + `jobs_test.go`（授权文件）；T4 仅新增 `preview_test.go`（授权），`preview.go`/`strip.go` **零改动**（符合 T4「零改动验证」定义）。未发现改动 plan 未授权文件（tool/builtin、task.go、agent.go 等均未被 T1-T4 触碰）。

### [PASS] 幻觉检测——证据真实，但「完成」声明不成立
- T1/T3/T4 全部代码与测试经逐行核读**真实存在**：`boundedResult`、`renderResultEnvelope`、`ResultSnapshotForSession`、`recordCompletion` 的 j.mu→m.mu 两段锁、jobs_test.go 4 个 T3 测试、preview_test.go 2 个 T4 测试，断言与实现一一对应，无编造证据。
- 父代理验证「全部测试通过」与源码状态自洽：jobs/control/agent 包测试通过的前提是 T1/T4 代码真实存在，这与「T2 未接线」不矛盾（信封 API 有测试但无生产调用方）。
- **可复核的反向事实**：`completion.result` 零消费者（grep 实测仅 recordCompletion 写入、drain 只用 `item.text`）；`ResultSnapshotForEngine`/`renderResultEnvelope` 生产零调用方（仅 jobs_test.go 引用）；input.go 原样。→ 当前产品行为与事务开始前**完全一致**。

### [PASS] 缓存红线（当前零变化；补齐方案亦满足）
- 注入点固定：`input.go:191-195` 前缀容器，位置不动（补齐方案不动此点）。
- 发送侧字节：T2 未接线 → 当前零变化；补齐后仅**新 user turn** 的 `<background-jobs>` 容器内容变化（一行摘要 → 信封块），system prompt / tool schema / 已发送消息**零字节变化** → provider 前缀缓存零失效。
- `ComposeSynthetic`（input.go:293-300）不调 jobs 注入 → 不投递 ✓。
- profile 名/动态值进 schema：信封全部在 user 消息正文，不进 schema/system ✓。
- 剥离路径：preview.go `TransientUserBlockTags` 与 strip.go `reComposeBlock` 均以**容器标签**整块剥离；嵌套信封经 `xmlEscaper` 转义（`<`→`&lt;`），不会产生字面 `</background-jobs>` 泄漏——T4 两测试已锁。`input_test.go:1387` 容器剥离用例（旧文本 `1 completed`）与正则仍匹配 ✓。
- evidence lease 红线：jobs 侧新增代码无 `LeaseEvidence`/`CommitEvidence`/`collectBackgroundEvidence` 调用（grep 确认）——自动投递不越权。

### [PASS/有条件] 回归（父代理验证 + 静态兼容核对）
- 父代理已真实验证：jobs/control/agent 相关测试全部通过。
- 静态核对补齐对**既有 drain 断言**的兼容性（§四方案 A 下逐条对证）：
  - `jobs_test.go:326` `Contains(string(Done))` → 信封 `status="done"`（小写）包含 "done" ✓
  - `jobs_test.go:349/350/582/588`、`artifacts_test.go:908` `Contains(j.ID)` → 信封 `task_id="bash-N"` 包含 id ✓
  - `jobs_test.go:352/462` 二次 drain 为空 → 单条场景 ≤8 条，首轮取走 ✓
  - `jobs_test.go:653/707/749` destroy 抑制后 drain 为空 → 与部分 drain 正交（recordCompletion 仍查 `m.destroying`）✓
  - `jobs_extra_test.go:123-138` `TestDrainMultiple` 2 条 → note 非空（2≤8）✓
  - `concurrency_stress_test.go:36` 并发 `DrainCompletedNote()` → 升级后 drain 全在 m.mu 临界区，切片操作原子 ✓
- ⚠️ 两个**护栏点**（补齐时不得破坏，否则 T3 测试/既有断言回归，详见第五节）。

### [FAIL] 对抗自检（devil's advocate 攻击发现的缺陷/边界）
见第三节裁决与第五节边界清单。

---

## 三、重点裁决 1：T2 blocked 报告的诚实性与完整性

工作区**不存在执行队报告文件**（`execution.md` 缺失，d1/d2/d5 均为纪律审查而非执行队报告），故无法核验报告原文措辞。但可从**代码状态与「blocked 因果」的一致性**独立裁决：

1. **因果真实（blocked 不是借口）**：
   - G1（无按 session 限 N 条部分 drain）：`DrainCompletedNoteForSession` 全量 drain，无 8 条上限、无留队语义——**属实**（jobs.go:1252-1277）。
   - G2（completion 缺结构化字段）：`completion` 仅 `{sessionID, text, result}`，无 id/status/kind/label/artifact——**属实**（jobs.go:193-197）。注入层要渲染信封属性，无法从 completion 取得。
   - G3（16KB/丢最旧/overflow）：全 jobs 包无 `maxBlockBytes`、无 `<result-overflow>` 产出——**属实**（grep 实测）。
   - **锁序硬约束**：`ResultSnapshotForEngine`/`renderResultEnvelope` 若要接入 drain，`ResultSnapshotForSession` 内部 `m.get` 再次持 `m.mu`（jobs.go:1318→908-912）——在 drain 持 m.mu 的临界区内调用即**自死锁**；在 drain 外按 id 二次查 job 则状态漂移（终态发布后 j.result/j.tail 可能已清空，jobs.go:547-550）且需 m.mu→j.mu 嵌套。→ 这证明「接口缺口」是**真实技术约束**，不是执行队偷懒的托词。
2. **未越权（克制证据）**：若执行队在缺口下硬接，必然改动 input.go 注入逻辑或从 text 字符串反解 id/status（脆弱反模式）。实测 **input.go 原样**（191-195 未动）→ 执行队选择如实 blocked 上报而非越权硬上，行为诚实。
3. **完整性缺口**：T5（e2e）、T6（execution.md）客观缺失。无论执行队报告是否提及，事务状态为「进行中」；若报告仅陈述 T2 blocked 而未声明 T5/T6 未完成，则完整性不足——但报告原文不在工作区，此点**无法独立核验，列入 unresolved**。

---

## 四、重点裁决 2：T2 补齐签名级实现建议（供执行队落地，纪律只给裁决）

**裁决：方案 A（jobs 侧升级 `DrainCompletedNoteForSession` 为信封渲染 + 部分 drain + 块聚合；input.go 调用点零改动）——采纳**，与定稿 plan.md §二/§三、plan-3 §二方案 A 一致。

### A1. 常量（新增，jobs 包；不得改名既有常量）
```go
const (
    resultSnapshotMaxBytes = 4096   // 既有，T3 测试直接引用（jobs_test.go:829），不可改名
    maxResultsPerDrain     = 8      // 新增：每轮每 session 最多投递条数
    maxResultBlockBytes    = 16 * 1024 // 新增：单轮信封块字节上限（按转义后 len 计）
)
```

### A2. `completion` 结构（扩展，新增 envelope 字段）
```go
type completion struct {
    sessionID string
    text      string // 兼容旧摘要语义（保留；stalled 文案继续用）
    result    string // bounded body（保留，T1 既有）
    envelope  string // 新增：recordCompletion 在 j.mu 内预渲染的完整 <background-job-result>（转义后字节）
}
```
**为什么不存结构化字段（id/status/kind/label/artifact）而直接预渲染 envelope**：
- drain 只持 m.mu、零 j 查询 → 无锁序风险、无状态漂移；
- 16KB 预算可直接用 `len(envelope)`（**转义后真实字节**，满足 d2 A4「块级按转义后计」）；
- **单真源**：recordCompletion 与 ResultSnapshotForSession 共用 `renderResultEnvelope`，杜绝两套信封形状漂移；若存字段再在 drain 渲染，则两处渲染逻辑必须时刻同步 = 双真源。

### A3. `recordCompletion`（修改：在既有 j.mu 临界区内预渲染信封）
签名不变：`func (m *Manager) recordCompletion(parentSession, id, kind, label string, st Status, err error)`。
关键改动（锁序与现状一致，**无新增锁面**）：
```go
var result, envelope string
if j := m.get(parentSession, id); j != nil {   // m.get 在持 m.mu 之前调用，无重入
    j.mu.Lock()
    result = boundedResult(jobResultTextLocked(j))
    envelope = renderResultEnvelope(j, st, result)   // j.ID/j.Label/j.artifactPath 均在 j 上
    j.mu.Unlock()
}
// …m.mu.Lock() 后 append completion{sessionID, text, result, envelope}…
```
`renderResultEnvelope` 无需改动（Failed→`<error>`、其余→`<output>`、5 字符 xmlEscaper、`string(st)` 保 "done" 兼容）——T3 测试锁定其形状，改它 = 破坏测试。

### A4. `recordStalled`（修改：定义 stalled 信封形态）
d2 A1 裁决二选一，本审查裁定：**`status="running"` 的 `<background-job-result>` 信封（空 artifact），正文为既有 stalled 文案**。理由：stalled 仍入 `m.completed`（不新增队列/不新增 drain 分支），`DrainCompletedNote`/`ForSession` 两入口天然覆盖；兼容 jobs_test.go:456-458（`Contains(note,"may be stalled")` && `Contains(note,j.ID)`——task_id 含 id、正文含文案，均成立）。
```go
// 在 m.mu 临界区外构造或直接在 append 前构造：
envelope := fmt.Sprintf(
    `<background-job-result task_id="%s" status="running" label="%s" artifact=""><output>%s</output></background-job-result>`,
    xmlEscaper.Replace(id), xmlEscaper.Replace(label),
    xmlEscaper.Replace("may be stalled — still running after … with no visible output. …"))
```

### A5. `DrainCompletedNoteForSession`（升级：签名不变，内部改 8 条部分 drain + 16KB 聚合）
签名不变：`func (m *Manager) DrainCompletedNoteForSession(parentSession string) string`（input.go 调用点**零改动**，仅返回值从一行摘要变信封块）。
关键语义（两阶段都在**单一 m.mu 临界区**内完成，原子）：
1. **分流 + 8 条部分 drain**：
   - 无 session（`DrainCompletedNote()` 委托）：`taken = m.completed[:min(len,8)]`，`m.completed = m.completed[n:]`（余量留队）。
   - 有 session：遍历 `m.completed`，**该 session 的条目只取前 8 条**，其余（其他 session 全部 + 该 session 超量部分）全部留队。
   - **8 条之外 = 余留下轮（G1），不计 overflow**——队列自然承载，无需对调用方暴露。
2. **16KB 块预算 + 丢最旧（G3）**：从新到旧累计 `len(envelope)`，超限丢弃**最旧**（队首）并 `overflow++`；`overflow` = **本轮因 16KB 块超限被丢弃的条数**（跨轮重置）。
3. **输出**：
   ```
   <background-job-result task_id="…" status="…" label="…" artifact="…">
   <output>…bounded…</output>
   </background-job-result>
   …
   <result-overflow count="N"/>   // 仅 N>0 时输出
   ```
   `input.go:193` 外层容器 `"<background-jobs>\n" + note + "\n</background-jobs>"` 原样包裹 → 与 preview_test.go:201 预设的 `<result-overflow count="2"/>` 容器内形态一致。
4. **边界（记录，需执行队测试锁定）**：单条信封因 XML 转义放大（全 `&` 极端场景）可能超 16KB——裁决：**单条超块时仍投递（不丢唯一结果）**，overflow 只统计被丢弃的其他条。避免「唯一结果被块预算误杀」。

### A6. `input.go` 消费（零改动 + 注释更新）
调用点 191-195 **原样不动**；仅将注释从「drain 一行摘要」更新为「drain 信封块」。T2 的**全部代码改动落在 jobs 包**——这是方案 A 的本质，也是最小侵入与缓存红线的双重保证。

### A7. 可选附赠（非必须，不阻塞）
`func (m *Manager) PendingCompletionCountForSession(parentSession string) int`：m.mu 下只读计数，满足 G2「剩余可见性」诉求，不消费、不 drain。d1 中 G2 的可见性诉求可用其满足，但非 T2 必需。

### A8. `ResultSnapshotForSession` 保持现状，不得改为读 completion
它是「按 id 只读渲染」的公开原语，服务测试与外部按需读取；改读 completion 会引入「completion 是否已存在」的时序耦合（job 完成但 drain 前、或 purgeSession 后）。**补注释声明**：`// 自动投递主流程消费 completion.envelope（recordCompletion 预渲染），不调用本函数；本函数仅供只读按需读取。` 防止执行队为「消除悬空」而错误接线（死锁/漂移风险）。

---

## 五、重点裁决 3：补齐是否影响 T3 已写测试——**4 个 bounded 单测全部继续有效**

逐一分析（关键：T3 测试测的是 `boundedResult`/`ResultSnapshotForSession` 两个**独立原语**，而方案 A 的补齐只改 `recordCompletion` 的预渲染与 `DrainCompletedNoteForSession` 的聚合——两者不触碰 T3 测试所依赖的原语）：

| T3 测试 | 依赖对象 | 方案 A 补齐后是否有效 | 理由 |
|---------|----------|----------------------|------|
| `TestBoundedResultRuneSafeTruncation`（823-867） | `boundedResult` + 常量 `resultSnapshotMaxBytes` | ✅ **有效** | 纯函数，不依赖 Manager；补齐不改 `boundedResult` 签名/语义；**常量名必须保留**（829 行直接引用） |
| `TestResultEnvelopeEscapesForgeCloseTags`（869-945） | `ResultSnapshotForSession` + `renderResultEnvelope` | ✅ **有效** | 补齐不改这两者；信封 schema（`task_id/status/label/artifact` + `<output>/</output>` + 结尾 `</background-job-result>` 恰一次）被测试锁定，补齐**必须保持此 schema 不变** |
| `TestResultSnapshotForSessionIsReadOnly`（947-975） | `ResultSnapshotForSession` 只读语义 | ✅ **有效** | 补齐方案 A8 明确 ResultSnapshotForSession 保持从 job 渲染、不消费 readOffset/resultRead；补齐仅改 drain 路径 |
| `TestResultSnapshotEnvelopeBodyBounded`（977-1009） | `ResultSnapshotForSession` + `boundedResult`（4096 + marker） | ✅ **有效** | 同上；body ≤4096 且含 `[truncated…]` 由 boundedResult 保证，补齐不动它 |

**结论**：在「补齐复用现有原语、不更名常量、不改 renderResultEnvelope 信封 schema」三前提下，T3 的 4 个测试**零改动继续通过**。这两个护栏点也是既有 drain 断言（§二回归）不回归的前提，应写入 execution.md 作为执行队的硬约束。

**执行队须补的新测试**（T3 缺口，供参考）：
- 8 条上限 + 余留下轮：9 条完成 → 首轮 drain 恰 8 条、二次 drain 恰 1 条（G1 锁死）；
- 16KB 块：8 条满负载（每条 ~4KB 转义后）→ 丢最旧 + `count=N` 恰为丢弃数，二次 drain 不再见丢弃条（已丢不重投）；
- overflow 跨轮重置：本轮丢弃 2 条 → 下轮新完成 1 条 → overflow 从 0 重新计；
- stalled 信封：`may be stalled` 文案 + `status="running"` 属性断言；
- 跨 session 分流 + 部分 drain 组合：session-a 10 条 + session-b 3 条 → drain(a) 取 8 留 2，drain(b) 取 3 留 0。

---

## 六、重点裁决 4：缓存红线最终确认

| 红线 | 裁决 | 证据 |
|------|------|------|
| 发送侧前缀字节稳定 | ✅ **最终确认通过**（补齐后亦通过） | 注入点 input.go:191-195 固定；补齐仅改 `<background-jobs>` 容器**内容**（新 user turn 动态块），system/tool/已发送消息零字节变化 |
| 自动投递只原位/尾部 append | ✅ | 采纳 plan-3 原位容器升级；**严禁**回归 plan-1/2 尾部注入（新增剥离逻辑 + stripTrailingMemoryRecall 顺序陷阱风险） |
| profile 名/动态值进 schema | ✅ 未触碰 | 信封全在 user 消息正文；`ComposeSynthetic`（input.go:293-300）不投递 |
| 剥离路径 | ✅ 零改动 | preview.go:19-29 / strip.go:8 容器标签不变；T4 测试已锁 |
| evidence lease | ✅ 不越权 | 补齐方案（recordCompletion 预渲染 + drain 拼接）无 LeaseEvidence/CommitEvidence/collectBackgroundEvidence 调用 |
| 锁序 | ✅ | recordCompletion 序：m.get（m.mu 进/出）→ j.mu → m.mu，无嵌套；drain 仅持 m.mu；补齐不引入新锁面 |

---

## 七、结论

**驳回（当前事务状态为「进行中/blocked」，不可宣称完成）**

驳回理由（可复核证据）：
1. **T2 注入接线未实施**：#7962「结果不自动投递」在产品角度**未解决**——input.go 仍产旧单行摘要，信封/快照停在 jobs 包内零生产调用方（jobs.go:1294/1317 仅测试引用；completion.result 零消费者）。
2. **三层 bounded 仅实现一层**：4096B/条已实现；8 条/次、16KB/块、丢最旧、`<result-overflow count>` 全部空头（DrainCompletedNoteForSession 全量 drain）。
3. **T5 e2e、T6 文档缺失**；无 execution.md 可交叉核对执行队命令证据。
4. 定稿未覆盖边界仍需裁决后落地（见下「必须修复项」之 5）。

**必须修复项（按序，均给出签名级裁决，执行队按 §四落地）**：
1. **T2 补齐（方案 A）**：§四 A1-A8——新增 `maxResultsPerDrain`(8)/`maxResultBlockBytes`(16KB) 常量；`completion` 加 `envelope` 字段；`recordCompletion` 在既有 j.mu 临界区预渲染 `renderResultEnvelope` 结果；`recordStalled` 产出 `status="running"` stalled 信封；`DrainCompletedNoteForSession` 升级为「8 条部分 drain 留队 + 16KB 丢最旧 + `<result-overflow count>`」，签名不变；`input.go` 调用点零改动仅更新注释；`ResultSnapshotForSession` 保持只读原语并补定位注释。
2. **T3 补测**：§五列出的 5 类新测试（8 条留队、16KB 丢最旧+count、overflow 跨轮重置、stalled 信封、跨 session 分流组合）。
3. **护栏点**：保留常量名 `resultSnapshotMaxBytes`（jobs_test.go:829 引用）；不改变 `renderResultEnvelope` 信封 schema；不改 `boundedResult` 语义——三条破坏任一即 T3 测试回归。
4. **T5 e2e**：后台 job 完成 → 下一轮 user 消息真实携带信封正文（模型视角拿到结果）的端到端断言；同轮 wait/bash_output 仍可取全量输出。
5. **定稿修订/决策记录**：Killed 信封形态（走 `<output>` 还是 `<error>`）、Interrupted 恢复 job 不投递的预期语义、stalled 信封（§四 A4 已裁定）、恢复会话是否补投——写入 execution.md；usage 按 plan-3 C2 档记录决策。
6. **T6 文档**：产出 `docs/team/20260810-p1-autodeliver/execution.md`，含每步命令与 git diff 证据（防虚假完成核查点）。

**已确认无需修复（记录澄清）**：`string(st)` 与 `st.String()` 等价（Status 为 `type Status string`）；recordCompletion 函数内自行 j.mu 拷贝为定稿「收快照参数」的等价实现（功能等价、锁序合规）；boundedResult 头部优先方向维持（d1 裁决，现有 `HasPrefix` 测试已锁定，不改为 plan-3 §四.3 的「取尾部」）。

（独立裁决完成；父代理可据上述行号与路径逐条复核。unresolved 见下方。）
