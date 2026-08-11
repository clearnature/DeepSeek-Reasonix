# 纪律审查报告：P1 后台结果自动投递（独立视角，d4）

> 审查人：纪律 Discipline（独立视角，未参与规划/执行，不共享 d1/d2/d5 结论，独立取证后交叉印证）
> 审查基线：`docs/team/20260810-p1-autodeliver/plan.md` 定稿（T1–T6）vs 工作区 team 分支实际代码
> 必读核对：`plan.md`、`internal/jobs/jobs.go`、`internal/jobs/jobs_test.go`、`internal/agent/preview_test.go`、`internal/control/input.go`
> 工具限制声明：本审查无 shell 执行能力（无 git/test 命令）。测试通过性以父代理真实验证为准；「零改动」以 grep+全文交叉验证为据，严格 `git diff` 复核列入 unresolved。

## 一、审查项逐条

- **[FAIL] fable5 合规（执行队跳步）**
  任务分解 T1–T6 中：**T1 大部分完成**（`completion.result` bounded 快照、`boundedResult`、`renderResultEnvelope`、`ResultSnapshotForSession` 已落地，锁序合规）；**T2 未实施**（`input.go:191-195` 仍是旧单行摘要 + `DrainCompletedNoteForSession` 全量 drain，信封零接线）；**T3 部分**（4 个 T3 单测存在，但 8 条/次、16KB/块、丢最旧、overflow 聚合测试全部缺失，且对应实现也不存在）；**T4 测试存在**（preview_test.go 2 个剥离回归测试）；**T5 缺失**（全仓库无 autodeliver e2e，`TestComposeInjectsBoundedResultEnvelope` 不存在）；**T6 部分**（无 `execution.md` 执行队文档，仅 d1/d2/d5 审查报告）。T2/T5 明确未完成，属跳步。

- **[FAIL] 幻觉检测（证据真实性——重跑验证结果）**
  父代理已真实验证全部测试通过（T3/T4 单测真实存在、逻辑自洽、与源码断言逐一对应）。但**测试全绿 ≠ 功能落地**，可复核事实：
  1. `ResultSnapshotForSession`（jobs.go:1317）与 `renderResultEnvelope`（jobs.go:1294）全仓库 grep 仅 `jobs_test.go` 引用，**生产零调用**（input.go 只调 `DrainCompletedNoteForSession`）。
  2. `completion.result`（jobs.go:197）被 `recordCompletion` 填充，但 `DrainCompletedNoteForSession`（1252-1277）只 join `item.text`，**result 字段零消费者（死数据）**。
  3. 8 条/16KB/`<result-overflow>`：grep 生产代码**无任何对应常量/实现/单测**（`maxResultsPerDrain`/`maxBlockBytes` 仅存在于 plan-*.md）。
  4. 若任何报告将「单元测试通过」外推为「#7962 已解决」，即功能级虚假完成。当前无 execution.md 声称完成，但事务现状=半成品。

- **[PASS] 缓存红线（前缀字节稳定）**
  发送侧仅新 user turn 的 `<background-jobs>` 容器内容可能变化（当前甚至未变：仍是旧摘要）；稳定 system/tool 前缀、canonical 历史、`ComposeSynthetic` 均零变化。注意：当前「不违规」的代价是「功能未升级」——红线未破坏，但以未实现换取。`history/strip.go:8` 与 `preview.go:23` 均按容器标签整体剥离（`(?s)` 跨换行），容器内嵌套信封不会穿透剥离层，T4 防御成立。

- **[PASS] 回归（测试命令 + 结果）**
  本审查无 shell，无法亲自重跑；父代理已真实验证全部测试通过。代码级交叉验证无回归面：
  - `recordCompletion`（jobs.go:541）在 j.status 终态发布（543-546）**之前**调用，注释说明 Wait 竞态理由（538-540），与 plan.md「置终态前调用」一致 ✅
  - `string(Done)`="done"（jobs_test.go:326）与 `recordCompletion` text 的 `%s` 格式输出小写 "done" 兼容 ✅
  - 既有 `DrainCompletedNoteForSession` 行为未变（input_test.go:1387 旧摘要容器测试仍在）✅
  - 锁序 j.mu（快照）→ 释放 → m.mu（append），无嵌套；`-race` 以父代理验证为准 ✅

- **[FAIL] 对抗自检（devil's advocate 攻击发现的缺陷）**
  (a) **T1/T2 接口缺口**：`completion` 仅 `{sessionID, text, result}`，**无 id/status/kind/label/artifact 结构化字段**——注入层即便接线也无法从 completion 记录渲染 `task_id`/`status`/`label`/`artifact` 信封属性，T1 需返工（与 d1/d2/d5 独立印证）。
  (b) **recordStalled 无信封形态**：stalled 条目无 result、无终态 status，若容器内混入纯文本行会破坏 `<background-jobs>` 内部 XML 结构一致性；信封形态未定义。
  (c) **status 参数可背离**：`ResultSnapshotForSession` 的 `st` 由调用方传入，与最终发布状态理论上可背离（并发 Kill）；当前时序（kill 同步置 Killed → run 侧 st 亦为 Killed）基本安全但无测试锁定。
  (d) **信封属性不计入预算**：label/artifact/task_id 不计入 4096B/条，恶意长 label 可膨胀单信封（16KB 块上限兜底但未实现）。
  (e) **边界形态未定稿**：Killed 走 `<output>` 分支语义类失败却用 output；Interrupted（恢复 tombstone）不进 `m.completed`、根本不投递；会话 reload 后已完成 job 不重新投递。
  (f) **两套剥离标签列表漂移风险**：`preview.go` 的 `TransientUserBlockTags` 与 `strip.go` 的 `reComposeBlock` 是 agent/history 两包独立列表；当前裁决容器内升级恰好规避（无需新增标签），若未来回归 plan-1/2 的独立顶层块方案，两列表须同步修改。

## 二、重点 1：wait/bash_output 兼容红线——ResultSnapshotForSession 只读性【PASS】

- `ResultSnapshotForSession`（jobs.go:1317-1326）实现路径：`m.get` → `j.mu.Lock()` → `renderResultEnvelope(j, st, boundedResult(jobResultTextLocked(j)))` → `j.mu.Unlock()`。
- `jobResultTextLocked`（793-808）只读 `j.result` / `readArtifactAllLocked` / `j.tail`，**不消费 `readOffset`/`resultRead`**（消费逻辑在 `OutputForSession` 948-1004，快照路径不调用它）、**不触碰 evidence lease**（lease 入口 `LeaseEvidenceForSession`/`TryLeaseEvidenceForSession` 2087-2123 不被调用）、不触发 `collectBackgroundEvidence`。
- 测试 `TestResultSnapshotForSessionIsReadOnly`（jobs_test.go:947-975）锁定：快照后 `readOffset==0`；其后 `OutputForSession` 仍返回全量输出；二次消费为空（消费语义保留）。父代理已验证通过。
- **关键限制**：该接口生产零调用，红线当前只在测试层面被锁定，尚未进入真实投递路径——「未破坏」成立，但「真正启用时仍安全」未经生产路径验证。方案 A 接线时**不得**在 drain 的 `m.mu` 内回头调它（j.mu→m.mu 重入违例），正确做法是 `recordCompletion` 的 j.mu 临界区内渲染并缓存信封。

## 三、重点 2：`<background-jobs>` 容器位置不变、preview/strip 零改动【PASS，附 git diff 复核限制】

- `input.go:191-195` 仍是原容器位置与注入方式（`"<background-jobs>\n" + note + "\n</background-jobs>"`），**容器位置未动**。
- `preview.go` 全文审查：`TransientUserBlockTags`（L19-29）含 `background-jobs`，`reTransientUserBlock`（L37-40）由标签自动构建、`(?s)` 跨换行整体剥离——容器内任意内容（含嵌套信封、转义伪造标签）都被整块剥除；文件内无 `background-job-result`/`result-overflow`/`ResultSnapshot`/`renderResultEnvelope` 任何符号。
- `strip.go` 全文审查：`reComposeBlock`（L8）仅 `memory-update|background-jobs|active-goal|hook-context` 四标签，无信封符号。
- T4 测试（preview_test.go:196-237）锁定信封与转义伪造形态不泄漏进 preview/title，父代理已验证通过。
- **限制**：本审查无 shell，无法运行 `git diff` 严格证明「零改动」（无法排除改动但未引入信封符号的情况）。以 grep+全文交叉验证推断成立；**请父代理以 `git diff -- internal/agent/preview.go internal/history/strip.go` 复核**（列入 unresolved）。

## 四、重点 3：T2 裁决——部分 drain 与聚合逻辑归属 jobs（方案 A）【采纳 A，独立论据】

1. **数据所有权**：`m.completed` 与保护它的 `m.mu` 完全归属 jobs 包——写侧 `recordCompletion`(835)/`recordStalled`(883)/adoptUnscoped；消费侧 `DrainCompletedNoteForSession`(1252)/`BeginDestroySession`(1765，清理该 session 待投递项)。control 无该数据的任何既有合法访问路径。
2. **原子性**：部分 drain = 「按 session 分流 + 取前 8 条 + 余量留队 + 累计块字节 + 丢最旧」必须在 `m.mu` 单一临界区内原子完成。若放 control，只能组合「全取再回写」，两调用之间被并发 `recordCompletion` 插队 → 丢条或重复投递；`recordCompletion` 注释（538-540）记载的 `TestDrainMultiple -race` flake 正是此并发风险的实证。
3. **锁序**：`ResultSnapshotForSession` 只持 `j.mu`。control 若要逐条渲染信封，须从 completion 记录取 id 再回查 job（j.mu）——在 drain 持 `m.mu` 时构成 `m.mu` 重入 `j.mu`（j.mu→m.mu 顺序被打破）。jobs 内部方案：`recordCompletion` 在既有 j.mu 快照点渲染并缓存信封，drain 只拼接缓存，永不回头取 j。
4. **单一事实源**：`renderResultEnvelope` 已是 jobs 包函数。放 control 会与 jobs 内 `boundedResult`/`renderResultEnvelope` 形成两套渲染逻辑，漂移风险（plan-3 §二方案 B 否决逻辑同构）。
5. **测试边界**：jobs 包可独立构造 Manager + 并发 `recordCompletion` 验证不丢条/不重复，测试边界窄、可确定性；control 测试需装配 Controller + 真实 turn 循环，边界宽、更脆。**结论：部分 drain + 聚合归属 jobs（方案 A），input 只消费 jobs 返回的信封字符串；与 d1/d2/d5 裁决一致。**

## 五、重点 4：T5 端到端未实施的影响评估

- **直接后果**：产品功能未闭环。模型下一轮收到的仍是旧单行摘要 `"... — done. Read their output with bash_output or wait if you still need it."`——**结果正文从未被投递**，用户/模型视角与事务开始前完全一致，#7962 未解决。
- **链路无整链验证**：jobs 渲染、input 容器注入、preview/strip 剥离各有单测，但「jobs 渲染信封 → input 注入容器 → provider 收到 → 模型可读 → strip 不泄漏」整链无 e2e。
- **T2 接线后才会暴露的验证点**：completion 结构化字段（返工后）与 input 消费的一致性；多轮部分 drain「余留下轮」在真实 turn 循环中是否成立（input 每轮仅调一次 drain）；多 job 并发完成 + 用户发消息下的 `-race` 干净度；bounded/XML 转义字节真实到达 provider 的形态。
- **结论**：T5 是验收门禁。在 T2 未实施、T5 缺失的前提下，本事务**不可验收**。

## 六、结论

**驳回**

驳回理由：事务处于**半成品**状态——
1. **T2 注入接线未执行**：`input.go` 仍产旧单行摘要，#7962「结果不自动投递」在产品角度**未解决**（证据：input.go:192 调 `DrainCompletedNoteForSession`；`ResultSnapshotForSession`/`renderResultEnvelope` 生产零调用）。
2. **三层 bounded 仅实现第一层**：8 条/次、16KB/块、丢最旧、`<result-overflow count>` 均为定稿承诺但零实现、零单测（grep 证据）。
3. **T1/T2 接口缺口**：`completion` 记录缺 id/status/kind/label/artifact 结构化字段，即便接线也无法渲染信封属性，T1 需返工。
4. **T5 e2e、T6 execution.md 缺失**；定稿遗漏 Killed/Interrupted/stalled/会话恢复边界信封形态。

通过项：只读快照红线（代码+测试）、容器位置与剥离层零改动（grep 交叉验证）、锁序（j.mu→m.mu 不嵌套）、`Done` 字符串兼容。

必须修复项（按序）：
1. T1 返工：`completion` 补结构化字段（id/status/kind/label/artifact），`recordCompletion` 在 j.mu 快照点渲染并缓存信封（保持 ResultSnapshotForSession 只读定位、单一事实源）。
2. T2 接线（方案 A，jobs 侧）：`DrainCompletedNoteForSession` 升级为部分 drain（≤8 条/次余留下轮 + 16KB 块丢最旧 + `<result-overflow count>`），input.go 容器位置与消费形态不变、preview/strip 零改动。
3. T3 补测：8 条留队、16KB 丢最旧、overflow 计数、多轮余留（对齐 preview_test.go 已预设的 `<result-overflow count="2"/>` 形态）。
4. T5 e2e：后台 job 完成 → 下一轮信封真实注入 + `StripComposePrefixes` 回退用户原文（plan-3.md:112 设计）。
5. 定稿修订：裁定 Killed/Interrupted/stalled/恢复会话边界信封形态与 Failed `<error>` body 语义。
6. T6：执行队输出 execution.md。

（独立裁决完成。请父代理复核：① `git diff -- internal/agent/preview.go internal/history/strip.go` 零改动；② 全量测试 -race 通过；③ 上述 grep 证据可逐一重跑。）
