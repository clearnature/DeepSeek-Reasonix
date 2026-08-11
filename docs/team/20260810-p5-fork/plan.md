# 定稿：P5 fork 机制（规划小组一致性裁决）

> 事务：docs/team/20260810-p5-fork/ · 规划小组：plan-1/2/3.md（3 个 team-planner 独立产出）
> 一致性裁决：父代理汇总 → 共识采纳 + 分歧裁决

## 一、共识（3/3）

1. **前缀构造 = 截断继承**（方案 A，MVP）：fork 子代理前缀 = 父 system（RoleSystem 原样）+ 父历史 `Snapshot()[:len-1]`（**截掉当前未完成的 assistant 轮次**——父已发送字节，天然 byte-identical；尾部未配对剔除是缓存命中硬前提，对应 CCB 占位技巧的真实用途）
2. **结果处理**：fire-and-forget 默认后台 job（复用 `StartForSession` kind="task" → P1 信封/P3 steer/wait 零改动生效），fork 信封默认静默（类比 foregroundClaimPending）
3. **缓存收益**：fork 首请求前缀 = 父前缀 → 命中父已建缓存；**收益条件 = 父 model/effort 与子代理一致**（覆盖 = 显式放弃命中）
4. **安全双轨**：fork 子代理 schema 全量保留（前缀要求）+ 执行层只读 gate/递归 guard（安全要求）——两目标不同层承担
5. **红线**：fork 捕获零发送、父 Session 零改动（Snapshot 深拷贝）；`ablation.Fork` 开关

## 二、分歧裁决

| 分歧 | 方案 | 裁决 |
|------|------|------|
| 工具形态 | plan-3: 独立 ForkTool / plan-1/2: task 参数 `fork: true` | 采纳 **task 参数 `fork: bool`**（复用 RunProfileSpec 全链路，schema 增量最小） |
| 前缀精确度 | plan-2: 发送视图变换链复制 / plan-3: 截断继承 | 采纳 **截断继承（MVP）**——满足 byte-identical 且规避占位漂移；发送视图复制列增强 |
| 递归 | plan-1: fork 内禁止 / plan-3: 深度+大小双 guard | 采纳 **深度 guard（≤maxSubagentDepth）+ 大小 guard（父历史 ≤窗口 80%）**，fork-of-fork 允许到深度上限 |
| 首轮 compact | plan-3: 首轮豁免标志 / plan-2: 不触发 | **T0 探查决定**（若 Prepare 会折叠则加首轮豁免标志） |

## 三、T0 探查（执行前必做，3 项）

- **T0-A 父 Session 可达性**：TaskTool 无父 Agent 引用 → 需 ctx 通道（`WithForkSource(ctx, a)`，仿 evidence.WithSessionMessages 模式）；验证主 agent Run 的 ctx 流经 executeOne 工具 ctx
- **T0-B 首轮 compact**：fork 子代理大预填 session 是否在首轮采样前被 Prepare 折叠（破前缀）→ 决定首轮豁免标志
- **T0-C tools 前缀敏感性**：DeepSeek 服务端缓存前缀是否含 tools（影响收益量级）——cachehit_e2e_test 基座扩展验证

## 四、任务分解（执行小队）

| 任务 | 内容 | 文件 | 验证 |
|------|------|------|------|
| T0 | 三项前置探查（A/B/C） | task.go/sampling_request/cachehit_e2e | 探查结论记录 |
| T1 | 前缀构造：`captureForkPrefix`（父 system + 历史截断 + 尾部剔除，Snapshot 深拷贝零发送） | `internal/agent/subagent_fork.go`（新文件，避免与 fork.go 命名冲突） | 单测：字节断言 |
| T2 | task schema 加 `fork` 参数 + 互斥校验 + RunProfileSpec fork 分支（PrepareParentFork → 后台 job） | `internal/agent/task.go` | schema golden 更新 |
| T3 | 只读 gate + 递归/大小 guard（fork 子代理执行层拦截） | `internal/agent/` gate | gate 单测 |
| T4 | 静默信封（fork job 完成不自动投递 P1 信封）+ wait/steer 协同 | `internal/jobs/jobs.go` | 单测 |
| T5 | byte-identical 断言（首请求消息 = 预填字节）+ cachehit e2e（usage.cache_hit_tokens>0）+ 父零改动断言 | `cachehit_e2e_test.go` | e2e 通过 |
| T6 | 文档（SPEC §3.13 fork 例外 + ablation.Fork） | docs | 文件存在 |

## 五、缓存/纪律检查点

- fork 捕获零发送、父零改动（Snapshot 深拷贝、无 Add/Rewrite/Prepare）✅
- 子代理首请求前缀 byte-identical（T5 硬性断言）✅
- fork 参数静态 bool 不进动态 schema；model/effort 覆盖文档警告丢缓存 ✅
- `isFreshSubagentSession` 对 fork session 自动 false → 不 prepend subagentStartContext（T5 锁定）✅
- P1 信封对 fork 静默（不自动投递结果——fire-and-forget 语义）✅
