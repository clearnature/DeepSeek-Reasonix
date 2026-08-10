# 纪律审查报告：P1 后台结果自动投递（独立视角，d5）

> 审查人：纪律 Discipline（独立视角，未参与规划/执行）
> 审查基线：`plan.md` 定稿（T1–T6 六任务分解） vs 工作区实际代码
> 必读核对：`docs/team/20260810-p1-autodeliver/plan.md`、`internal/jobs/jobs.go`、`internal/jobs/jobs_test.go`、`internal/agent/preview_test.go`、`internal/control/input.go`
> 注：本审查无 shell 执行能力；测试通过性以父代理真实验证为准，本报告做代码级交叉验证与功能完整性裁决。

## 一、核心事实（代码级证据）

1. **注入路径未升级**。`internal/control/input.go:191-195` 仍是
   `note := c.jobs.DrainCompletedNoteForSession(...)` → `"<background-jobs>\n" + note + "\n</background-jobs>"`。
   `DrainCompletedNoteForSession`（jobs.go:1252-1277）返回的仍是**旧单行摘要**：
   `"Background job updates since your last message: id (label) — done; ... . Read their output with bash_output or wait if you still need it."`
2. **结构化信封零生产调用方**。`ResultSnapshotForSession`（jobs.go:1317）与 `renderResultEnvelope`（jobs.go:1294）
   全仓库 grep 仅 `internal/jobs/jobs_test.go` 有调用；`internal/control`、`internal/agent`、`internal/history` 均无引用。
3. **死数据**。`completion` struct（jobs.go:193-197）新增 `result` 字段，`recordCompletion`（jobs.go:847,859）填充了 bounded 快照，
   但 `DrainCompletedNoteForSession` 只 join `item.text`，`completion.result` **从未被读取**。
4. **三层 bounded 只实现第一层**。4096B/条（`boundedResult`）已实现并有单测；
   **8 条/次、16KB/块、`<result-overflow count="N">` 全部未实现**——`DrainCompletedNoteForSession` 对 `m.completed` 全量 drain 无上限。
5. **T5/T6 缺失**。全仓库无 autodeliver 相关 e2e 测试；`docs/team/20260810-p1-autodeliver/` 下无执行队 d 系列文档（仅 plan-1/2/3/plan.md）。
6. **T3/T4 测试真实存在且逻辑自洽**。jobs_test.go 新增 4 个 T3 测试（rune 安全截断 / XML 伪造关闭标签 / 快照只读 / body bounded），
   preview_test.go 新增 2 个 T4 测试（信封整体剥离 / 转义伪造剥离）。父代理已验证通过。

## 二、审查项逐条

- **[FAIL] fable5 合规（执行队跳步）**
  T1（jobs 结构化完成记录+只读快照）大部分完成；**T2（input 注入接线，定稿 §三「替换 input.go:191-195 摘要块」）未执行**——
  input.go 原样，生产路径仍产出旧摘要。T3 只覆盖第一层 bounded（8 条/16KB/overflow 无单测）。**T5（e2e）与 T6（文档）缺失**。
  任务分解表 T1–T6 中至少 T2/T5/T6 未完成，存在明确跳步。
- **[FAIL] 幻觉检测（证据真实性）**
  执行队若声明「P1 完成」即为功能级虚假完成：测试全绿（T3/T4 均为防御性/单元测试）≠ 产品功能落地。
  可复核事实：`ResultSnapshotForSession` 生产零调用（grep 证据）、`completion.result` 无消费者（代码证据）、input.go 未变（代码证据）。
  唯一的「完成」证据是单元测试文件存在且通过，而这些测试测的是**未被接线的独立 API**。测试通过不可外推为 #7962 解决。
- **[PASS] 缓存红线（前缀字节稳定）**
  注入位置保持在 `input.go:191-195` 前缀容器内，历史 canonical 零改动、稳定 system/tool 前缀零改动；`ComposeSynthetic` 不受影响。
  注意：当前「未违规」的代价是「功能未升级」（发送侧仍是旧摘要，未产生任何新信封字节）。红线未破坏，但以未实现换取。
  history 剥离层 `internal/history/strip.go:8` 与 `internal/agent/preview.go:23` 均以容器标签整体剥离，嵌套信封（经 xmlEscaper 转义）不会穿透剥离层——T4 的防御性成立。
- **[PASS] 回归（基于父代理验证 + 代码审查）**
  本审查无 shell，无法亲自重跑；父代理已真实验证全部测试通过。代码级交叉验证：
  `recordCompletion`（L541）在终态发布（L543-546）前调用、快照时机与 `plan-3.md` 描述一致；
  `jobResultTextLocked` 不消费 `readOffset`/`resultRead`、不碰 evidence lease（只读 `j.mu`）；
  `TestStalledWarning...`（jobs_test.go:326）`Contains(string(Done))` 兼容（`string(st)` 仍为小写 "done"）。
  以上与既有语义均不冲突，回归面无破坏迹象。
- **[FAIL] 对抗自检（devil's advocate 发现的缺陷）**
  (a) **T1/T2 接口缺口**：`completion` 记录仅有 `sessionID/text/result`，**没有 id/status/kind/label/artifact 结构化字段**。
      注入层若想渲染信封，无法从 completion 记录取得 `task_id`/`status`/`label`/`artifact` 属性——
      必须从 text 字符串反解（脆弱反模式）或按 id 回查 Manager（而 id 也不在记录里）。**T2 当前接口形态下无法直接接线，T1 需返工。**
  (b) Failed 信封 body 语义未定稿：`renderResultEnvelope` 对 `st==Failed` 用 `<error>`，但 body 来自 `jobResultTextLocked`，
      混入流式 stdout + `"job artifact incomplete: ..."`，模型无法区分「输出」与「错误」。
  (c) Killed/Interrupted 形态未定义：Killed 走 `<output>` 分支；Interrupted（恢复的 tombstone job）不进 `m.completed`、**根本不投递**。
  (d) stalled 通知无信封形态：`recordStalled` 的 completion 无 result，若注入升级，stalled 条目在信封容器中如何渲染未定义。
  (e) 信封属性（label/artifact/task_id）不计入 4096B/条 预算，恶意长 label 可膨胀单信封（16KB/块 兜底，属小缺口）。
  (f) `ResultSnapshotForSession` 的 `st` 由调用方传入，与最终发布状态理论上可背离（并发 Kill），实际时序（kill 同步置 Killed → run 侧 st 亦为 Killed）基本安全，但无测试锁定。

## 三、重点裁决

### 1) 产品角度：升级后的信封是否解决 #7962「结果不自动投递」——**否，当前未解决**
模型下一轮收到的仍是 `"... — done. Read their output with bash_output or wait if you still need it."` 单行摘要，
**结果正文没有被投递**。`<background-job-result>` 信封、bounded 快照、XML 转义均已就绪但**停在 jobs 包内，未被注入路径消费**。
从用户/模型视角，行为与事务开始前完全一致。测试通过无法掩盖「功能未接线」这一事实。

### 2) bounded 数值合理性——数值本身合理，落地不完整
- **4096B/条**：合理。中文约 1.4K token、英文约 1K token，足够模型判断是否需要 `wait`/`bash_output` 取全文；rune 截断 + `[truncated…]` 计入预算实现正确（jobs_test 覆盖到位）。
- **8 条/次**：合理，配合「余留下轮」语义。保守但可防一夜间批量 job 一次性涌入。
- **16KB/块**：与 8×4096=32KB 存在重叠，最坏 4 条满负载即触顶；但这是硬上限（含信封开销），保证单轮 user turn 增量有界，自洽。
- 三者共同构成「先按条、再按字节」的双重闸门，设计成立；**但当前仅第一层实现**，8 条/16KB/overflow 均为空头承诺。

### 3) T2 裁决 A/B 倾向与理由——**plan-3（位置不变、容器内升级）正确，予以维持；并补充独立理由**
- 尾部注入（plan-1/2）需新增尾部剥离逻辑，与 `stripTrailingMemoryRecall`/`stripTrailingDeliveryRuntime`（preview.go:97-98）顺序纠缠，且 `internal/history/strip.go:8` 的 `reComposeBlock` 仅处理前缀——尾部块在历史回放路径无剥离覆盖，UI/标题泄漏风险真实存在。
- 前缀容器是既有、已被剥离层整体覆盖的结构，内容升级零改动成本；模型行为上「后台状态在用户开口前告知」符合工具结果惯例。
- **补充代价提示**：前缀容器内容随 job 结果动态变化，若未来任何层对「user turn 前缀」做字节级缓存假设需重新审视；当前 Reasonix 缓存前缀在 system/tool 侧，不受影响。
- 附带发现：plan-3 被采纳的另一个好处是 recall 仍独占用户消息尾部，job 结果与记忆的相对顺序稳定。

### 4) 定稿未覆盖的边界（执行需补，定稿需修订）
| 边界 | 现状 | 建议 |
|------|------|------|
| job 失败（Failed）信封形态 | `<error>` 实现，但 body=流式输出+artifactErr 混入 | 定稿明确 `<error>` 内容 = 错误/结果正文的语义边界 |
| job 被杀（Killed） | 走 `<output>` 分支，语义类失败却用 output | 定稿裁决 Killed 是否用 `<error>` |
| 中断恢复（Interrupted） | 不进 completed，**不投递** | 决定是否对恢复会话补一次投递（#7962 盲区） |
| stalled 通知 | 无信封形态、无 result 字段 | 定义 stalled 在信封容器中的渲染（或继续旧摘要） |
| 会话 reload 后已完成 job | tombstone 不重新投递 | 明确「仅本生命周期内完成才投递」为预期行为并写入文档 |
| T1/T2 接口 | completion 缺结构化字段，无法渲染信封 | T1 返工：completion 记录 id/status/kind/label/artifact |

## 四、结论

**驳回**

驳回理由：事务处于**半成品**状态，存在功能级虚假完成风险——
1. **T2 注入接线未执行**：`input.go` 仍产旧单行摘要，#7962「结果不自动投递」在产品角度**未解决**（重点 1 证据）。
2. **三层 bounded 仅实现一层**：8 条/次、16KB/块、overflow 计数均为定稿承诺但零实现、零单测。
3. **T5 e2e、T6 文档缺失**；执行队无 d 系列执行记录可交叉核对。
4. **T1/T2 接口缺口**：completion 记录缺结构化字段，即便现在接线也无法渲染信封，T1 需返工。
5. 定稿本身遗漏 Killed/Interrupted/stalled/恢复会话等边界信封形态，需修订后执行。

必须修复项（按序）：
1. T1 返工：`completion` 记录补 `id/status/kind/label/artifact`（或等价结构化承载），使注入层可逐条渲染信封。
2. T2 接线：`input.go` 注入改为渲染 `<background-job-result>` 信封（≤8 条/次、≤16KB/块、丢最旧 + `<result-overflow count>`、余留下轮），
   保持容器前缀位置与剥离层零改动（继承 plan-3 裁决）。
3. T3 补测：8 条上限、16KB 预算、overflow 计数、部分 drain 余留下轮 的单元测试。
4. T5 e2e：后台 job 完成 → 下一轮信封真实注入的端到端断言（模型视角拿到结果正文）。
5. 定稿修订：裁定 Killed/Interrupted/stalled/会话恢复 的边界信封形态与 Failed `<error>` body 语义。
6. T6：执行队输出执行记录 + 本事务状态文档。

（独立裁决完成；父代理可据上述可复核证据逐条核实。）
