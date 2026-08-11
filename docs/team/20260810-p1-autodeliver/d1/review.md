# 纪律审查报告：P1 后台结果自动投递（T2 接口缺口裁决）

> 事务：`docs/team/20260810-p1-autodeliver/` · 分支：team · 审查角色：纪律 Discipline
> 审查对象：T1（jobs.go 结构化完成记录 + 只读快照）、T3/T4（新增单测与剥离回归）
> 裁决问题：T2 blocked 接口缺口 G1（无按 session 限 N 条部分 drain）、G2（无待投递完成项 id/status 列表，ResultSnapshotForSession 悬空）、G3（16KB 丢最旧 + overflow 聚合未实现）；方案 A vs 方案 B
> 依据：plan.md（定稿 §一/§二/§三/§四）、plan-3.md（被采纳方案 §二方案 A/§四对抗自检/§五检查点）、jobs.go/jobs_test.go/preview_test.go/input.go 实际代码

---

## 一、审查方法与证据链（幻觉检测前置）

纪律团无法在无 shell 的工作区重跑测试，故交叉验证采用「源码逐行核对 + 测试断言与实现一致性 + 引用父代理已真实验证的命令结果」三明治方式：

- 父代理已真实验证：`go build` ✅、`go test ./internal/jobs/` ✅、`go test ./internal/control/` ✅、`agent -run Preview` ✅、`tool/builtin Bg/Wait` ✅。
- 纪律团静态核对：`internal/jobs/jobs.go`（T1 实现）、`internal/jobs/jobs_test.go`（T3 新增）、`internal/agent/preview_test.go`（T4 新增）、`internal/control/input.go`（T2 现状）逐行读过，断言与实现一致，无编造证据。

---

## 二、审查项逐条

### [PASS] fable5 合规（执行队是否跳步）——**T1/T4 无跳步，T2 阻塞、T3 部分、T5/T6 未启动**

对照定稿 plan.md §四任务表：

| 任务 | 定稿要求 | 现状 | 判定 |
|------|----------|------|------|
| T1 | jobs 结构化完成记录 + 只读快照函数 | **部分完成**：`completion` 扩展 `result` bounded 快照（L193-197）；`ResultSnapshotForSession`（L1317）+ `renderResultEnvelope`（L1294）已落地；`recordCompletion` 在 j.mu 内拷贝 bounded 快照、m.mu append，锁序合规（L844-863） | ⚠️ 未全量 |
| T2 | input 注入接线（容器内信封渲染） | **未实施（blocked）**：input.go L191-195 仍调 `DrainCompletedNoteForSession` 纯文本摘要，未接线信封 | ❌ blocked |
| T3 | bounded 单测（截断/条数/总量/丢弃/XML 转义） | **部分**：截断（`TestBoundedResultRuneSafeTruncation`）、XML 转义（`TestResultEnvelopeEscapesForgeCloseTags`）、只读（`TestResultSnapshotForSessionIsReadOnly`）、单条 4096（`TestResultSnapshotEnvelopeBodyBounded`）已加；**条数上限（8 条/次）、块上限（16KB）、丢最旧、overflow 聚合测试全部缺失** | ⚠️ 未全量 |
| T4 | 剥离回归（preview/strip 零改动）+ wait/bash_output 兼容 | **完成**：preview_test.go 新增 `TestPreviewStripsBackgroundJobResultEnvelope`、`TestPreviewStripsEscapedForgeAttempt`；preview.go `TransientUserBlockTags` 与 history/strip.go `reComposeBlock` 零改动（容器标签不变） | ✅ |
| T5 | 端到端 | 未实施（依赖 T2） | ❌ |
| T6 | 文档 | `execution.md` 不存在于 docs/team/20260810-p1-autodeliver/ | ❌ |

执行队未在已实施部分跳步：T1 的锁序、快照时机（recordCompletion 在状态翻转前调用，L535-541 注释与实现一致）、只读语义均有对应代码与测试，无「只改注释」蒙混。

### [PASS] 幻觉检测（证据真实性）

- `completion` 结构、`boundedResult`、`renderResultEnvelope`、`ResultSnapshotForSession` 的源码真实存在，与测试断言逐一对应（转义测试断言 `&lt;/background-job-result&gt;` 等 6 种逃逸形态、只读测试断言 `readOffset==0`、bounded 测试断言 body ≤4096 且含 `[truncated…]`）。
- jobs_test.go L326 `strings.Contains(note, string(Done))`、L582/L588 `Contains(note, id)` 等既有断言与新信封形态兼容性经核对成立（信封 `status="done"` 含 "done"、`task_id="<id>"` 含 id）——父代理跑绿与源码状态自洽。
- 父代理验证结果与现状吻合：当前代码处于「jobs 侧机制已落地、input 未接线」状态，jobs/control/agent 各包测试通过是可信的。

### [PASS] 缓存红线（当前零变化）

- T2 未接线 → **发送侧当前零变化**：input.go 仍是原 `<background-jobs>` 纯文本块，`renderResultEnvelope`/`ResultSnapshotForSession` 未进入任何生产注入路径。
- 信封代码只存在于 jobs 包内部与测试，未触碰 system/tool 前缀、未改 canonical 历史、`ComposeSynthetic` 路径（input.go L293-300 之外）未受影响。
- 纪律红线：jobs 侧新增代码无 `LeaseEvidence`/`CommitEvidence`/`collectBackgroundEvidence` 调用（grep 确认，evidence lease 仍由 wait/bash_output 独占）——自动投递不越权。
- ⚠️ 前瞻：T2 接线时必须保证信封只进新 user turn 的 `<background-jobs>` 容器、容器标签与位置不动（定稿 §三），否则红线告警。

### [PASS] 回归（既有断言与新增代码兼容）

- 父代理验证：`go build`、`go test ./internal/jobs/`、`go test ./internal/control/`、`agent -run Preview`、`tool/builtin Bg/Wait` 全绿。
- 静态核对：既有 drain 断言（`jobs_test.go` L579-590/L653-654/L707/L749 `Contains(note, id)`、`jobs_extra_test.go` `TestDrainMultiple` 非空断言、L326 `Contains(string(Done))`）在新信封形态下仍成立；`input_test.go` L1386-1389 strip 用例（旧文本）与 preview.go 正则对新形态同样剥离。
- R7（改动 drain 返回值破坏既有断言）风险已消解。

### [FAIL] 对抗自检（发现的缺陷——本次审查核心产出）

1. **G1 缺口确认（部分 drain 未实现）**：`DrainCompletedNoteForSession`（L1252-1277）仍是全量 drain + 纯文本拼接，无 8 条/次上限、无余留下轮。grep 全 jobs 包无 `maxResultsPerDrain`。→ 20 条完成会一次注入 20 条，R4（上下文膨胀）未闭合。
2. **G2 缺口确认（待投递 id/status 列表缺失，ResultSnapshotForSession 悬空）**：`completion` 结构仅 `{sessionID, text, result}`，无 id/status/artifact/label 结构化字段；jobs 无「待投递完成项 id/status」查询接口；`ResultSnapshotForSession`/`renderResultEnvelope` 目前**零生产调用方**（input.go 只调 `DrainCompletedNoteForSession`），是测试可触达的孤儿代码。drain 与 snapshot 两条路径未汇聚。
3. **G3 缺口确认（16KB 块上限/丢最旧/overflow 未实现）**：无 `maxBlockBytes`、无 `<result-overflow count>` 输出、无丢最旧逻辑。preview_test.go L201 的 `<result-overflow count="2"/>` 是测试对未来形态的预设（不依赖实现，测试本身可过），但生产侧无对应产出。
4. **bounded 截断方向与 plan-3 §四.3 冲突**：`boundedResult` 取**头部优先**（保留开头，`HasPrefix` 被测试锁定），而 plan-3 对抗自检明确「默认取尾部（离结论最近）」。定稿 plan.md 未强制方向。→ 纪律团裁决见下。
5. **recordCompletion 签名与定稿字面偏差**：定稿 T1 要求「改收快照参数」，实现保持原签名、函数内自行 j.mu 拷贝（L844-849）——功能等价、锁序合规，属可接受的等价实现，但按定稿字面为偏差，已记录。
6. **status 渲染用 `string(st)` 而非 plan.md 提到的 `st.String()`**：Status 为 `type Status string`（"done"/"failed"/"killed"），`string(st)` 输出正确且保 jobs_test.go L326 `Contains("done")` 兼容；与 plan-3 信封 `status="done"` 规格一致。非缺陷，记录澄清。

---

## 三、方案 A/B 裁决（对照定稿）

### 裁决：**采纳方案 A**

| 对照维度 | 方案 A（jobs 侧升级 drain 为信封+部分 drain+聚合） | 方案 B（jobs 新增独立待投递列表+限 N 消费接口） |
|----------|----------------------------------------------------|--------------------------------------------------|
| 定稿 plan.md §二/§三 | ✅ 完全吻合：「替换 input.go:191-195 当前的一行摘要块（DrainCompletedNoteForSession）」「每轮最多 8 条（部分 drain，余留下轮），超 16KB 丢最旧 + overflow 计数」——消费逻辑全部落在 jobs 侧 | ❌ 偏离：把信封组装/聚合逻辑移入 input.go，input 职责从「只组装 `<background-jobs>` 容器」膨胀为「组装信封」，违背定稿最小侵入 |
| plan-3 §二方案 A | ✅ 即其推荐方案：「DrainCompletedNoteForSession（L1185-1210）改为渲染 `<background-jobs>` 信封（含结构转义 + 截断标记）」 | ❌ 是 plan-3 已否决的「结构逻辑移出 jobs」倾向的变体（对照 plan-3 §二方案 B 独立顶层块的否决逻辑：结构分裂 → 两套标签/两处渲染漂移风险） |
| 缓存/剥离 | 容器标签与注入位置不变，preview/history 零改动（T4 已验证） | 无额外收益，但要求 input 知晓信封 schema，schema 演进（如 usage 落地）会穿透到 control 层 |
| 锁序 | G1/G3 实现只需在 drain 的 m.mu 临界区内做条数/字节控制，G2 在 recordCompletion 既有 j.mu 快照点渲染缓存——两段锁区间结构不变 | 需新接口在 m.mu 内遍历待投递列表并渲染，若按 id 查 job 渲染会引入 m.mu→j.mu 嵌套（R3 死锁风险重现） |

**方案 A 的 G1/G2/G3 落点（供执行队，纪律只给裁决不给实现）**：
- G1：`DrainCompletedNoteForSession` 内按 session 分流后**最多取 8 条**，余量留队下轮（沿用现 L1260-1270 分流骨架加上限）。
- G2：`completion` 扩展 `id`（+status/label/artifact 供信封渲染）；`recordCompletion` 在**已有 j.mu 快照点**（L844-849）用同一 `boundedResult`+`renderResultEnvelope` 渲染并缓存信封，drain 直接拼接——`ResultSnapshotForSession` 保留为同源公开只读接口，与缓存信封同构，不再悬空。绝不在 drain 的 m.mu 内回头取 j（锁序红线）。
- G3：drain 累计信封字节，超 16KB 时丢弃最旧条目并输出 `<result-overflow count="N"/>`；**overflow 仅计因 16KB 块超限被丢弃的条数**（8 条余留是自然留队下轮，不计 overflow——定稿 §一.2 语义）。

### 纪律团对 bounded 截断方向的拍板

- 现状：头部优先（`HasPrefix` 被测试锁定），与 plan-3 §四.3「默认取尾部」相左。
- **裁决：维持头部优先**，理由：(1) 定稿 plan.md 未强制方向，现有实现+测试自洽；(2) task 的 result 以「Final answer:」开头，答案结论常在开篇段落，头部优先对 task 更连贯；(3) bash tail 本身是 64KB 最新环形缓冲，头部优先丢的是最旧流尾（正是 tail 语义），与正文价值方向一致；(4) 信封已附 `artifact` 路径 + `[truncated…]` 指示，父代理可 read_file 补全尾部。此裁决**取代** plan-3 §四.3 的「默认取尾部」，执行队无需改方向，但须在 T3 用例中显式锁定头部语义（现有 `HasPrefix` 已锁定，保留即可）。

---

## 四、结论

**有条件通过（T1 机制层）+ 驳回（T2 及 G1/G2/G3）**

理由：T1 的核心机制（bounded 快照、只读信封渲染、锁序、XML 转义、只读快照）实现质量良好且有测试锁定，T4 剥离回归完成，缓存红线当前零变化；但 T2 注入接线未实施、G1/G2/G3 缺口真实存在且未实现、T3 的条数/块/丢弃测试与 T5/T6 缺失，事务处于 blocked，不得宣告完成。

**必须修复项（按序）**：
1. 按**方案 A** 实施：`DrainCompletedNoteForSession` 升级为「部分 drain（≤8 条/次，余留下轮）+ 信封渲染 + 16KB 块上限 + 丢最旧 + `<result-overflow count>` 聚合」。
2. `completion` 扩展结构化字段（id/status/label/artifact）；`recordCompletion` 在既有 j.mu 快照点渲染缓存信封；`ResultSnapshotForSession` 与缓存信封同源收敛，消除孤儿代码。
3. input.go L191-195 保持容器标签与位置不变，仅消费 jobs 侧返回的信封块；确认 `ComposeSynthetic` 不投递。
4. T3 补齐：8 条上限留队测试、16KB 块上限 + 丢最旧 + overflow count 测试（对齐 preview_test.go 已预设的 `<result-overflow count="2"/>` 形态）。
5. T5 端到端、T6 `execution.md`（含 usage 档位 C2 决策记录，plan-3 §三 T7）。
6. 全程保持锁序（drain 只持 m.mu，禁止 m.mu 内回头取 j）与 evidence lease 不触碰红线；`go test ./internal/jobs/ -race` 收尾复核。

**已确认无需修复（记录澄清）**：`string(st)` 等价 `st.String()`；recordCompletion 内部快照拷贝为定稿「收快照参数」的等价实现。
