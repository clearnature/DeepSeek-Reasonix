# 纪律审查报告：P1 后台结果自动投递（独立视角，d3）

> 审查人：纪律 Discipline（独立视角，未参考他人结论，全部证据为本轮 read_file/grep 实测）
> 审查基线：`docs/team/20260810-p1-autodeliver/plan.md` 定稿（T1–T6 六任务） vs 工作区实际代码
> 必读核对：`plan.md`、`internal/jobs/jobs.go`、`internal/jobs/jobs_test.go`、`internal/agent/preview_test.go`、`internal/control/input.go`
> 注：本审查环境无 shell 执行工具，无法亲自重跑测试；测试通过性以父代理已真实验证为准，本报告做代码级交叉验证与独立裁决。

## 一、执行现状（逐任务独立核实，含行号证据）

| 任务 | 定稿要求 | 现状 | 独立证据 |
|------|----------|------|----------|
| T1 | jobs 结构化完成记录 + 只读快照 | ⚠️ **机制已实现，但结构化不完整 + 悬空** | `jobs.go:813-829` boundedResult；`:835-881` recordCompletion；`:1294-1309` renderResultEnvelope；`:1317-1326` ResultSnapshotForSession；`completion`（`:193-197`）仅 `{sessionID,text,result}`，**缺 id/status/kind/label/artifact**（G2） |
| T2 | input 注入接线（容器内信封渲染） | ❌ **未实施** | `input.go:191-195` 仍调 `DrainCompletedNoteForSession`，产物仍是旧单行摘要（`jobs.go:1252-1277` join `item.text`），`<background-job-result>` 信封未进入任何生产注入路径 |
| T3 | bounded 单测（截断/条数/总量/丢弃/XML 转义） | ⚠️ 部分 | `jobs_test.go:823/869/947/977` 四个测试真实存在且与实现逐一对证（见 §二）；**8 条/次、16KB/块、丢最旧、overflow 聚合测试全部缺失** |
| T4 | 剥离回归（preview/strip 零改动） | ✅ 完成 | `preview_test.go:196/225` 两个测试（信封整体剥离、转义伪造剥离）；`preview.go` `TransientUserBlockTags`、`history/strip.go` `reComposeBlock` 零改动 |
| T5 | 端到端 | ❌ 缺失 | 全仓库 grep `autodeliver`/`background-job-result` 无任何集成/e2e 测试文件 |
| T6 | 文档更新 | ❌ 缺失 | `docs/team/20260810-p1-autodeliver/` 下仅有 plan.md、plan-1/2/3.md 与审查文档；**无执行队执行记录/事务状态文档** |

**防虚假完成总判**：本事务**未完成**。T1/T3/T4 的代码与测试真实存在（已逐行核读，非编造）；但 T2（核心接线）、T5（e2e）、T6（文档）未做，T3 只覆盖三层 bounded 的第一层，T1 的 `completion.result` 是**零消费者死数据**、`ResultSnapshotForSession` 是**零生产调用方的孤儿 API**。任何人声称「P1 完成」即为功能级虚假完成。

## 二、审查项逐条

### [PASS] T1 bounded 截断实现正确性（独立数值核算）

- `resultSnapshotMaxBytes = 4096`，`truncatedMarker = "[truncated…]"`。逐字节核算：`[`(1)+`truncated`(9)+`…`(U+2026, UTF-8 3 字节)+`]`(1) = **14 字节**；`keep = 4096-14 = 4082`。
- `boundedResult`（jobs.go:813-829）：`len(s)<=4096` 原样返回；超限时逐 rune 累加且 `b.Len()+RuneLen(r) > keep` 即 break → body ≤4082，+marker ≤4096，**上限不破**；`for range` 按 rune 迭代 + `WriteRune`，**不会切分多字节 rune**；`keep<=0` 防御分支成立（当前 marker 14B 永不触发）。
- 测试 `TestBoundedResultRuneSafeTruncation`（jobs_test.go:823-867）断言与实现一致：at-cap 返回 verbatim、超限 ≤4096 + `HasSuffix(marker)` + `utf8.ValidString`、中文 3 字节 rune 不被切分、body 是原串 rune 前缀。逐项成立。
- **注意**：`boundedResult` 在 XML 转义**之前**按 4096 原始字节截断；转义后 `<`→`&lt;`（4×）、`&`→`&amp;`（5×），最坏全 `&` 输出时**单信封 body 实际发送字节可达 4096×5 ≈ 20KB**，字面违背「4096 字节/条」的发送侧预算。16KB/块（G3）本可兜底但未实现 → **当前无任何兜底**（见对抗自检 a）。

### [PASS] 锁序（j.mu 拷贝 → m.mu append 不嵌套）与 recordCompletion 时序

- `recordCompletion`（jobs.go:835-881）锁序实测：`m.get`（m.mu 内查指针→释放）→ `j.mu.Lock()` 拷贝 bounded 快照 → `j.mu.Unlock()` → `m.mu.Lock()` append → `m.mu.Unlock()`。**两段临界区先后、互不嵌套**，无 m.mu→j.mu 或 j.mu→m.mu 反向嵌套。✅
- `writeJobMetaLocked`（jobs.go:614-636）只做文件写（`writeMeta`），内部零 m.mu → j.mu 内调用无反向嵌套风险。✅
- 时序兼容（jobs.go:481-552）：run 返回 → 计算 st（`ctx.Err()!=nil`→Killed，`err!=nil`→Failed，否则 Done，L488-499）→ writeJobMetaLocked → **recordCompletion（L541）→ 发布终态 status（L543-546）→ close(j.done)（L552）**。注释（L535-540）所述「先记账后发布，避免 Wait 观察终态后 drain 抢跑」与代码一致。Kill 路径：BeginDestroySession 置 Killed + cancel → ctx.Err() 非空 → st=Killed，与最终发布一致。
- `jobResultTextLocked`（jobs.go:793-808）只读 `j.result`/`j.artifactPath`/`j.tail`/`j.artifactErr`，**不消费 readOffset/resultRead、不碰 evidence lease**；`ResultSnapshotForSession` 测试（jobs_test.go:947-975）断言 `readOffset==0` 且其后 `OutputForSession` 仍取到全量输出。✅

### [PASS] 缓存红线（当前零变化——但以「功能未落地」为代价）

- 注入位置仍在 `input.go:191-195` 前缀容器内，system/tool 前缀、canonical 历史、`ComposeSynthetic` 均零改动；jobs 侧新增代码 grep 无 `LeaseEvidence`/`CommitEvidence`/`collectBackgroundEvidence` 调用，evidence lease 仍由 wait/bash_output 独占（自动投递不越权）。✅
- **代价声明**：当前「零违规」是因为 T2 未接线——发送侧仍是旧一行摘要，未产生任何信封字节。红线未破坏，但红线内应有的功能也未落地。
- 剥离层 `preview.go`/`history/strip.go` 以容器标签整体剥离，嵌套信封（含转义伪造形态）不会穿透——T4 防御性成立。

### [PASS] 回归（父代理验证 + 本报告代码级交叉验证）

- 父代理已真实验证：全部测试通过（本环境无 shell，无法独立重跑，已如实标注）。
- 我的静态交叉验证与此自洽：T2 未接线 → `internal/control` 旧测试（`input_test.go:1386-1389` 旧文本 strip 用例）全绿是**必然**；T3/T4 新测试测的是**独立 API/纯函数**，与其真实实现逐条对应。既有 jobs 断言兼容性：`jobs_test.go:326` `Contains(string(Done))`（"done" 小写）与信封 `status="done"` 兼容、`Contains(note, id)` 与 `task_id="<id>"` 兼容——但这仅在 T2 接线后才有实际意义。
- 结论：**「测试全绿」与「T2 未接线」不矛盾**，测试通过不可外推为产品功能完成。

### [FAIL] 对抗自检（devil's advocate：独立攻击交付物）

1. **XML 转义膨胀（本轮独立发现，最实质）**：bounded 在转义前截断，全 `&`/`<` 恶意输出转义后可膨胀至 20KB/条；G3（16KB/块）未实现 → 单轮注入无字节闸门。T2 一旦接线，这个缺口立即上线。
2. **信封属性不计入预算**：`task_id/status/label/artifact`（jobs.go:1296-1297）直接进属性，超长 label/artifact path 可撑大单信封，同样无 16KB 兜底。
3. **快照路径全量读文件**：`jobResultTextLocked`→`readArtifactAllLocked`（jobs.go:1013-1025）对 artifact 全量 `os.ReadFile` 后 `boundedResult` 才截断；GB 级 artifact 会在 j.mu 临界区内产生全量 IO+内存分配。T1 快照路径放大了这一既有模式。
4. **stalled 条目混合渲染形态未定义**：`m.completed` 同时容纳 `recordStalled`（无 result、无 id，jobs.go:895）与 `recordCompletion`（有 result）两类 completion；方案 A 升级 drain 渲染信封时必须区分，否则 stalled 条目无法渲染或产生畸形信封。
5. **单条信封 >16KB 块上限的裁决空白**：定稿定义「超 16KB 丢最旧 + overflow 计数」，但若**单条**即超 16KB（转义膨胀场景）应丢弃还是截断？定稿未答。
6. **st 参数可背离**：`ResultSnapshotForSession` 的 st 由调用方传入（jobs.go:1317），与最终发布状态理论可背离（并发 Kill）；当前时序基本安全但无测试锁定。
7. **G2 接口死结**：即便现在接线，`completion` 无 id/status/label/artifact 字段，注入层无法渲染信封属性——**T2 当前接口形态下无法直接接线，T1 需返工**。

## 三、重点裁决

### 1) T1 bounded 截断正确性——**正确（实现+测试逐项自洽）**
4096 含 marker（14B 核算成立）、rune 安全（迭代/写入均按 rune）、XML 转义五字符（`& < > " '` 全转，jobs.go:1282-1288；测试 `TestResultEnvelopeEscapesForgeCloseTags` 断言 6 种逃逸形态全被转义、`</background-job-result>` 计数为 1、`</background-jobs>` 零泄漏——jobs_test.go:869-945 逐条成立）。**唯一字面缺口**：转义后发送字节可超 4096（对抗点 1）。

### 2) T2 裁决 A/B——**采纳方案 A（jobs 侧升级 drain 为信封渲染，input 只消费）**

| 维度 | 方案 A（jobs 侧升级 `DrainCompletedNoteForSession`） | 方案 B（jobs 新增独立待投递列表 + 限 N 消费接口） |
|------|------------------------------------------------------|----------------------------------------------------|
| 原子性 | 部分 drain（取前 8 条余留下轮）在 m.mu 单一临界区内原子完成 ✅ | control 侧「先全取再回写」被并发 recordCompletion 插队 → 丢/重条目 ❌ |
| 队列所有权 | `m.completed` 写/读方全在 jobs 包，单真源 ✅ | 新旧双 drain 接口**互吃同一队列**（plan-1 已识别攻击点），遗留维护风险 ❌ |
| 锁序 | drain 消费 completion 快照字段，只持 m.mu、零 j.mu ✅ | 若消费时按 id 回查 job 渲染 → m.mu→j.mu 嵌套 + 状态漂移（drain 时 job 可能已销毁）❌ |
| 悬空消解 | `completion.result` 获得消费者（信封正文源）✅ | `completion.result` 继续悬空、再造一份渲染逻辑 ❌ |
| 缓存红线 | input.go 消费点与容器位置零改动，改动面最小 ✅ | 可原位注入但无额外收益 ❌ |

**方案 A 落地前置**（G2 缺口）：`completion` 需补 `id/status/kind/label/artifact` 结构化字段，`recordCompletion` 在既有 j.mu 快照点（L844-849）用同一 `boundedResult`+`renderResultEnvelope` 渲染缓存信封，drain 直接拼接。**绝不在 drain 的 m.mu 内回头取 j**（锁序红线，与 d 系列一致）。

### 3) 防幻觉核查——**T1/T3/T4 声称可复核且属实；「P1 完成」的整体声称不成立**
- 属实：boundedResult/renderResultEnvelope/ResultSnapshotForSession 源码真实存在且与测试断言逐一对应（行号证据见 §一）；preview_test.go 两个 T4 测试真实且只测剥离纯函数。
- 不成立：`ResultSnapshotForSession` 生产零调用（grep 全仓仅 jobs_test.go）、`completion.result` 零消费者（drain 只 join `item.text`）、input.go 未变（代码证据）、T5/T6 无产物（目录/仓库证据）。执行队当前**无执行记录文档可交叉核对**（T6 缺失本身即证据链缺口）。

## 四、结论

**驳回**

驳回理由：事务处于**半成品**状态，存在功能级虚假完成风险——
1. **T2 注入接线未实施**（input.go 仍是旧单行摘要）：#7962「结果不自动投递」在产品角度**未解决**，模型下一轮仍收不到结果正文。
2. **三层 bounded 仅实现第一层**：8 条/次、16KB/块、`<result-overflow>` 均为定稿承诺但零实现、零单测。
3. **T5（e2e）、T6（执行/状态文档）缺失**。
4. **G2 接口缺口**：`completion` 缺结构化字段，当前接口形态下 T2 无法接线，T1 需返工。
5. **对抗自检 7 项缺陷中至少 3 项（转义膨胀/属性不计预算/单条超块上限空白）属定稿未覆盖的发送侧字节闸门缺口**，接线前必须裁决。

必须修复项（按序）：
1. T1 返工：`completion` 补 `id/status/kind/label/artifact` 结构化字段；`recordCompletion` 在既有 j.mu 快照点缓存信封（与 `ResultSnapshotForSession` 同源，消除孤儿代码）。
2. 按**方案 A** 接线：`DrainCompletedNoteForSession` 升级为「部分 drain（≤8 条/次，余留下轮）+ 信封渲染 + 16KB 块上限 + 丢最旧 + `<result-overflow count>`」；`input.go:191-195` 消费点与容器位置不动；确认 `ComposeSynthetic` 不投递。
3. **修字节闸门**：转义后 bounded（保证不切断实体序列且发送侧 ≤4096 实字节），或转义前按压缩预算截断；裁决「单条信封 >16KB 块上限」的处置。
4. T3 补测：8 条上限留队、16KB 块 + 丢最旧、overflow count（对齐 preview_test.go 已预设的 `<result-overflow count="2"/>` 形态）、转义膨胀回归。
5. T5 e2e：后台 job 完成 → 下一轮信封真实注入（模型视角拿到正文）的端到端断言。
6. 定稿修订补边界：Killed/Interrupted/stalled/会话恢复 的信封形态、Failed `<error>` body 语义（区分错误与输出）。
7. T6：执行队执行记录 + 本事务状态文档；收尾 `go test ./internal/jobs/ -race` 复核锁序。
