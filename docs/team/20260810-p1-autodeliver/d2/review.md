# 纪律审查报告：P1 后台结果自动投递（执行现状核查 + T2 接口缺口独立裁决）

> 审查角色：Discipline（纪律/防幻觉把关） · 独立视角，不参考他人
> 审查基准：docs/team/20260810-p1-autodeliver/plan.md（一致性定稿）
> 证据方式：静态交叉验证（read_file/grep 实测文件内容）；本环境无 shell 执行能力，**测试通过性无法独立重跑**，依赖父代理验证证据（详见「幻觉检测」与 unresolved）

---

## 一、执行现状核查（幻觉检测 · 逐任务）

| 任务 | 状态 | 可复核证据（文件:行） |
|------|------|----------------------|
| T1 结构化完成记录 + 只读快照 | ✅ 已实现 | `jobs.go:196` completion.result；`jobs.go:835-881` recordCompletion（j.mu 内快照 → m.mu append，**不嵌套**）；`jobs.go:813-829` boundedResult（4096 rune 安全 + marker 计入预算）；`jobs.go:1294-1309` renderResultEnvelope（5 字符 XML 转义）；`jobs.go:1317-1326` ResultSnapshotForSession |
| T1 单测 | ✅ 存在 | `jobs_test.go:823/869/947/977`（截断/防伪造/只读性/body bounded） |
| T2 input 注入接线（容器内信封渲染） | ❌ **未完成** | `input.go:192` 仍调旧 `DrainCompletedNoteForSession`（一行文本摘要），信封未注入 |
| T3 bounded 单测（条数/总量/丢弃/overflow） | ⚠️ 部分 | 单条 4096 + 转义测试齐；**8 条/轮、16KB 块、丢最旧、`<result-overflow>` 测试全部缺失** |
| T4 剥离回归（preview/strip 零改动） | ✅ 已实现 | `preview_test.go:196/225` 两个信封剥离回归；`preview.go:19-29` TransientUserBlockTags 含 `background-jobs`，**零改动** |
| T5 端到端 | ❌ 未发现 | grep `background-jobs` 全仓无集成/e2e 测试文件 |
| T6 文档 | ❌ 未完成 | `docs/team/20260810-p1-autodeliver/` 仅 plan.md + plan-1/2/3.md，无状态文档 |

**防虚假完成结论**：本事务**尚未完成**。任何人声称「P1 全部完成」均不成立——T2（信封接线 + G1/G2/G3 三缺口）、T3 补测、T5 e2e、T6 文档均为剩余项。T1/T4 的代码与测试真实存在（已逐行核读），非编造。

---

## 二、T2 接口缺口裁决

### 待裁决缺口
- G1 部分 drain（每轮最多 8 条，余留下轮）
- G2 待投递列表（剩余可见性）
- G3 16KB 块级聚合（丢最旧 + `<result-overflow count="N"/>`）

### 裁决：**方案 A（jobs 侧升级 `DrainCompletedNoteForSession` 信封渲染，input 只消费）——采纳**

**A1. 队列所有权与原子性（裁决点 2 的核心）**
`m.completed` 及其 `m.mu` 全部归属 jobs 包（写入：`recordCompletion` jobs.go:856、`recordStalled` :895、`adoptUnscopedJobsLocked` :1438；消费：`DrainCompletedNoteForSession` :1252、`BeginDestroySession` :1765）。部分 drain = 原子「按 session 取前 8 条、其余留队」，**只能在 m.mu 单一临界区内完成**。放 control 无法原子实现——「先全取再回写」会被并发 recordCompletion 插队，造成丢条目或重复投递。→ 部分 drain 语义**必须放 jobs 侧**（方案 A 即此），control 只做消费。

**A2. 单一 drain 语义（否决方案 B 的首要理由）**
方案 B「jobs 新增独立接口」会形成新旧两个 drain 接口并存、**互吃同一队列**——这正是 plan-1 攻击点 8 已识别的风险：维护者后续误用旧全清接口，部分 drain 的留队语义被瞬间清空。方案 A 将语义统一升级，单真源；`DrainCompletedNote()`（空 session legacy wrapper，jobs.go:1245）保留，不改签名。

**A3. 锁序最简（裁决点 2 的锁序面）**
信封渲染应消费 **completion 快照字段**（T1 已在 j.mu 内预存 bounded result），drain 只持 m.mu、零 j.mu → 无需任何 m.mu→j.mu 嵌套，锁序风险最低。若改为 drain 时按 id 二次查 job 渲染，需要 m.mu→j.mu 嵌套（本可接受，BeginDestroySession :1783 已有先例）但引入状态漂移（见 B 一节），故不取。

**A4. 缓存红线（裁决点 3）**
方案 A 下 `input.go:192-194` 调用点与 `<background-jobs>` 容器位置**完全不动**，仅 `DrainCompletedNoteForSession` 返回内容从一行摘要变为信封块。发送侧仅新 user turn 的该动态块内容变化；system prompt / tool schema / 已发送消息字节零变化 → provider 前缀缓存零失效。方案 B 若同样原位注入也合规，但无此收益。

**A5. 证据链闭合（消除悬空字段）**
T1 的 `completion.result`（jobs.go:196）当前**零消费者**（grep 实测仅 recordCompletion 写入、drain 只用 `item.text`）。方案 A 恰好让该快照字段获得消费者（信封正文源），悬空字段就地消解。方案 B 会让它继续悬空或再造一份渲染逻辑。

### 方案 B 否决理由（汇总）
1. 双 drain 接口互吃队列（已识别攻击点，持续风险）。
2. `remaining/dropped` 显式返回无收益——剩余条目留队自愈，下一轮续投；control 为此处理 remaining 纯属新增逻辑。
3. 双渲染路径 = 双真源，T3 测试面翻倍。
4. completion.result 继续悬空。
5. 唯一优点（旧接口/旧测试零波及）方案 A 已通过「保留 wrapper + 兼容断言」获得（见下）。

### 兼容性核查（方案 A 落地前提，已逐条对证）
- `jobs_test.go:326` `string(Done)`="done"（小写）→ 信封 `status="done"` 匹配 ✅
- `jobs_test.go:349/350/582/588`、`artifacts_test.go:908` 断言含 `j.ID`（`bash-N`）→ 信封 `task_id="bash-N"` 匹配 ✅
- `jobs_test.go:352/462` 二次 drain 为空 → 单 job 场景 ≤8 条，首轮清空 ✅
- `jobs_test.go:653/707/749` destroy 抑制后 drain 为空 → 与部分 drain 正交 ✅
- `input_test.go:1387` 容器剥离 → 容器标签不变，regex 照常匹配 ✅
- `preview_test.go:196/225` 信封剥离 → 已就绪 ✅
- ⚠️ `jobs_test.go:456` stalled 断言含 "may be stalled" → **stalled 条目信封化形态未定，见对抗自检 A1**

---

## 三、ResultSnapshotForSession 悬空裁决（重点问题 1）

**现状**：`ResultSnapshotForSession(parentSession, id, st)`（jobs.go:1317）仅测试调用（jobs_test.go:891/960/989），生产代码零调用方。

**裁决：不得接入自动投递主流程；保留为「受测试守护的公开只读原语」，并在其 doc 注释中声明定位。**

理由（推演各候选调用方）：
1. **drain 内调用 → 死锁/重入**：drain 持 m.mu 遍历队列时若调它，其内部 `m.get`（jobs.go:908）再次 `m.mu.Lock()` → 自死锁；若在 m.mu 外按 id 调用，则二次查 job 引入 m.mu→j.mu，且**状态漂移**：drain 时点 job 可能已被 `purgeSessionLocked` 删除（ok=false → 条目投递丢失），或终态发布后 `j.result/j.tail` 已被清空（jobs.go:547-550），与 completion 快照（完成时点、同源）不一致。
2. **recordCompletion 内调用 → 锁序违约**：recordCompletion 在 j.mu 临界区（jobs.go:846-848）内若调它，其内部 m.get 再次锁 m.mu → **j.mu→m.mu 反向嵌套，违反 plan.md 共识第 4 点**。
3. **正确归属**：它是「按 id 渲染信封」的只读读取原语，服务对象是测试与外部按需读取；其「零生产调用方」是**设计使然**而非缺陷——但**必须**补注释：`// 自动投递主流程消费 completion 快照（recordCompletion 预渲染），不调用本函数；本函数仅供只读按需读取。` 防止执行队为「消除悬空」而错误接线（会引入上述死锁/漂移缺陷）。

配套要求：信封渲染的**单一事实源**统一为 `renderResultEnvelope`（jobs.go:1294）——recordCompletion 与 ResultSnapshotForSession 共用，杜绝两套信封形状。

---

## 四、部分 drain 队列语义位置裁决（重点问题 2）

**裁决：jobs 侧（方案 A 即此），不落 control。**

- 队列所有权：`m.completed`/`m.mu` 全在 jobs；部分 drain 的原子「选 8 留 N」只能是 jobs 内部操作。
- 锁序：jobs 侧 drain 仅持 m.mu（渲染走 completion 快照），无跨包锁序问题；control 侧无法在不暴露队列的前提下原子完成。
- 一致性：G1「余留下轮」由队列自然承载，**无需对调用方暴露**——G2 的可见性诉求用只读 `PendingCompletionCountForSession(parentSession) int`（不消费、不 drain、m.mu 读）满足即可，非必须项。

---

## 五、缓存红线复核（重点问题 3）

| 红线 | 复核结果 |
|------|----------|
| 发送侧前缀字节稳定 | ✅ 注入点为 composeWithGoal 构造的**新 user turn**（input.go:191-195），位于全部已发送消息之后；信封仅改该动态块内容，稳定历史前缀（system/tool/已发消息）零变化 |
| 自动投递只 append turn 尾部/原位 | ✅ 采纳 plan-3 原位容器升级（plan.md §二裁决）；**严禁**回归 plan-1/2 的尾部注入方案（会新增 stripTrailingMemoryRecall 顺序陷阱风险） |
| profile 名/动态值进 schema | ✅ 信封全部在 user 消息正文，不进 tool schema / system prompt |
| 剥离路径 | ✅ preview.go:23 / strip.go:8 容器标签 `background-jobs` 不变，零改动；preview_test.go:196/225 已锁 |
| ComposeSynthetic 不投递 | ✅ input.go:293-300 无 jobs 注入 |

---

## 六、对抗自检（devil's advocate 攻击发现的缺陷）

- **A1 [必须修复] stalled 条目信封化形态未定**：`recordStalled`（jobs.go:895）生成的 completion **无 result、无终态 status**（job 仍 Running）。若按完成信封渲染，status 属性无从取值；若在容器内混排纯文本行（"…may be stalled…"），破坏 `<background-jobs>` 内部 XML 结构一致性。**裁决**：为 stalled 定义明确信封形态（如 `status="running"` 的空正文 `<background-job-result>`，或独立 `<background-job-stalled>` 标签，二选一），并补测试保证 jobs_test.go:456 的 "may be stalled" 断言继续成立。这是 T2 落地必须同步解决的缺口。
- **A2 [必须守住] ResultSnapshotForSession 误接线风险**：见第三节——任何为消除悬空而把它接进 drain/recordCompletion 的尝试都会引入 m.mu 重入或 j.mu→m.mu 锁序违约。注释声明为强制项。
- **A3 [测试缺口] 8 条与 16KB 的交互语义**：8 条 × 4096 = 32KB > 16KB，两种丢弃语义必须区分并分别测试：**8 条之外 → 留队续投（G1）；8 条之内但块超 16KB → 丢最旧（前部）+ `<result-overflow count="N"/>`（G3）**。overflow 计数定义为**本轮丢弃数**（跨轮重置），需测试锁定。
- **A4 [细节] 16KB 预算按转义后字节计**：XML 转义会使正文膨胀（`&`→`&amp;` 等），块级聚合须在**转义后**逐条累计 `len()`，否则超限。条目边界截断天然 UTF-8 安全（boundedResult 已 rune 安全）。
- **A5 [验证] 既有 drain 测试条数假设**：jobs_extra_test.go:134「DrainCompletedNote with multiple jobs」假设一次全清；升级后若该测试完成条数 >8，首轮只取 8、二次非空会失败。落地时需核对（当前未见该文件内条数，风险低）。
- **A6 [确认] `renderResultEnvelope` 的 artifact 属性**：recordCompletion 时点 artifact 已完成 moveArtifactToDirLocked（jobs.go:526），`j.artifactPath` 是终态路径，可安全进信封属性。

---

## 七、回归把关

- 触发范围（执行队/父代理必须实际执行）：`go test ./internal/jobs/ -race`、`go test ./internal/control/`、`go test ./internal/agent/ -run Preview`、`go vet ./internal/jobs/ ./internal/control/`、`repolint`。
- 本环境无 shell 执行能力，**我无法独立重跑以上命令**；父代理声称的「全部测试通过」未被我亲自复核，已如实列入 unresolved。静态层面，T1/T4 涉及的代码与测试均真实存在且调用图自洽。

---

## 八、结论

**驳回（当前事务状态为「进行中」，不可宣称完成）**

驳回理由与必须修复项：
1. **T2 未实现**：按方案 A 升级 `DrainCompletedNoteForSession` 为信封渲染（部分 drain ≤8 条/轮 + 16KB 块丢最旧 + `<result-overflow count="N"/>`），input.go 消费点与容器位置不动；同时补 A1（stalled 信封形态）与 A3-A5 测试。
2. **ResultSnapshotForSession 维持只读原语定位**，补注释声明主流程不调用；统一 `renderResultEnvelope` 为单真源。
3. **completion 需补充信封渲染所需字段**（id/kind/label/artifactPath/status，或直接预渲染 envelope 存入 completion）——T1 现存的 `result` 字段方有消费者。
4. **T5 e2e、T6 文档缺失**：补端到端（后台 job 完成 → 下一轮信封注入）与事务状态文档。
5. 回归命令（§七）须由有 shell 能力的执行方实际跑通后再报完成。

上述修复完成后，由执行队重跑全量相关测试并留档输出，方可进入下一轮审查。
