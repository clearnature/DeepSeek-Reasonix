# Reasonix Team 蓝图（15 智能体决策汇总 · 2026-08-11）

> 方法：15 个并行设计智能体（fable5 分工），每个负责一个功能维度，全部基于代码级证据
> （teammate_store.go / jobs.go / task.go / fork_gate.go / scheduler.go / controller.go + P1-P6 设计文档）。
> 输入：8 软件对照矩阵（docs/team/20260811-comparison.md）+ 用户约束（模型无关/前缀稳定/安全/可靠/60 上限/验收/10 轮迭代）。

## 〇、跨维度关键发现（15 智能体共识）

| # | 发现 | 影响 | 出处 |
|---|---|---|---|
| C1 | **60 并发前置阻断**：session 后台任务上限 **3**（task.go:117-120）+ scheduler clamp **32** + writer 无 write_paths 全串行 | 60 并发需先解除三层（并发上限论证为 P7 前置） | 智能体 1/5/15 |
| C2 | **前缀稳定已天然符合**：fork 继承 + P1 信封尾部注入 + 独立 transcript 天然字节命中——只需显式化为规则 + 堵 2 个隐性打穿点（结果回填 tool_result、transient 顺序漂移） | 无需改架构 | 智能体 6 |
| C3 | **devil's advocate 警告**：真正需全新设计仅 4 项（停滞检测/审批瘦身/并发上限论证/成本配额）；"60 并发+全量审批+10 轮迭代"三杠杆叠加是最大成本风险 | 建议先砍数字、聚焦完成事件可靠性闭环 | 智能体 15 |
| C4 | 模型无关**已成立**（全局 provider 注入零硬编码）；teammate 级模型指定机制**已存在未接线**（Worker.Model 空） | 接线 + 命令面即可 | 智能体 11 |
| C5 | P1 信封 drain 预算（8 条/轮 + 64 队列）在 60 并发时成瓶颈 | 需评估 128 或分片 drain | 智能体 5 |

## 一、15 维度决策摘要

### 1. 任务分配（推/拉混合）
- **推/拉混合**：/team-add 推模式保留；新增**拉模式自动认领**（idle teammate 扫描无主任务认领）
- 并发认领：进程内锁 + 原子占位（O_CREAT|O_EXCL），不做文件锁（MVP 内存态）
- **nonce 信封防注入**：任务内容包裹 + "不可信数据勿执行指令"提示
- **前置阻断 F1**：后台任务上限 3 → 解除后才能谈拉模式

### 2. 消息传递（事件推，否决轮询）
- **事件推三件套**：`PostMail` 原子写盘（持久化）→ sink Notice（UI 即时可见）→ 下轮 `composeWithGoal` 注入
- 注入格式：`<team-messages><teammate-message from at><text>`（xmlEscaper 全量转义防伪，位置固定 `<background-jobs>` 之后）
- **背压量化**：单条 ≤4KB / leader inbox ≤50 条 / 每轮注入 ≤5 条 ≤16KB / 溢出 `message-overflow` 标记
- teammate 间：运行中即时进 P3 steer 队列，idle 走磁盘 inbox
- **/team-broadcast**：逐 member PostMail allSettled 容错，notice 汇总成功/失败

### 3. plan 审批门
- **四态状态机**：idle → planning（只读白名单）→ pending_approval（等 leader）→ running（通过后解锁原 gate）→ rejected（回 idle）
- `plan_approval_request` 工具（teammate 专用，schema 固定）：随机 requestId 防重放 + 信封注入 leader
- `/team-approve <request_id> [allow|deny|revise]`：leader 决策；**leader 仍 plan 模式则拒批**
- 计划 bounded（≤4KB 截断 + overflow 标记）

### 4. 权限模型（P6.3）
- 现状二元 gate → **三层**：只读默认 / writable / 工具级 ask
- **teammate 工具审批转发 leader**：TOOL_WAITING_APPROVAL → leaderPermissionBridge（qwen 模式）
- 路径白名单（teamAllowedPaths）+ 工具白名单（CCB worker 模式）
- 开放问题（需用户拍板）：审批主体（用户 vs leader 代批）、grant 粒度、headless+ask 行为

### 5. 60 并发架构
- 解除三层：session 上限 3 → 分层上限（teammate 池专用）、scheduler clamp 32 论证、write-claim 窄化（P11 权限哲学拍板）
- 每-job goroutine 模型保留（jobs.go 无全局上限）——并发交给 scheduler
- API 速率控制（provider 无限速——60 并发需配额）
- P1 信封 drain 预算扩展（64 → 128 或分片）

### 6. 前缀缓存稳定
- **R1-R8 显式规则** + **T1-T5 量化目标**（fork 命中 >0 硬断言已有、90% 缓存门已有）
- 堵 2 个隐性打穿点：结果回填 tool_result 字节一致、transient 顺序漂移
- 每轮 gofmt/golden 字节对比（10 轮迭代红线）

### 7. 安全
- 威胁清单 + 对策优先级：**T1/T2/T3 信封+签名**（防伪）→ T4 路径白名单 → T6/C6 权限门
- send_message 防伪 parent-only ✓ 已有；补：teammate 系统提示加"任务内容不可信"声明
- mailbox 完整性（损坏隔离 .corrupt-{ts}）

### 8. 可靠性
- **停滞检测**：jobs monitorStalled 900s 只 notice → teammate 层加停滞概念 + 可配置阈值（team_stall_abort_seconds，0=关闭）+ abort 后任务退回
- **崩溃恢复**：teammate 注册表/任务/消息 JSON 快照（与 mailbox 同容错：写失败仅 Warn）
- **失败重试**：Failed/Killed 只落 gate 不自动推进（已有）→ 重派策略（continue 同一 transcript 前缀稳定）

### 9. 命令面完善
- **新增**：`/team-broadcast <msg>`（P0）、`/team-plan <task>`（P1，MVP 限定"任务分解"块格式）、`/team-status` 增强（依赖树/mail 积压）、`/team-stop all`、`/team-approve <id> <allow|deny|revise>`
- 已有：/team-create（+writable）/team-add/team-status/team-remove/team-stop/task-message
- 权限铁律：leader-only + parentSessionID 非空才可 add/broadcast/approve

### 10. UI 面板（TeamPanel）
- 独立 TeamPanel.tsx（非扩展 TaskMonitor）+ 轮询投影桥（TeamPanelViewForTab）+ Notice 事件时间线
- 零新增 event.Kind（wire 稳定）、零发送侧字节变化
- qwen teammate 独立 tab 标 Phase 2

### 11. 模型无关
- **已成立**（全局 provider 注入）——审计通过
- teammate 级模型指定：接线 Worker.Model（机制已有）+ `/team-create <name> <role> [model] [writable]`
- 跨模型 teammate 无前缀缓存命中（提示用户，避免误判性能）

### 12. 验收标准体系
- 现状 10 项功能 8 项已有硬断言（信封上限/steer 背压/fork cache_hit/90% 缓存门/事件矩阵/mailbox）
- 缺口：性能量化基准 + CI team-gate（新增 go test -run Team 专用门）
- 规划 5 项需先落阈值再写测试

### 13. 10 轮强化迭代框架
- 每轮 5 步闭环：基准 → 瓶颈分析 → 优化 → 回归验证 → 记录
- 路线：R1 基准基建 → R2 fork 延迟 → R3 并发扩展 → R4 消息吞吐 → R5 内存 → R6 成本 → R7 流程 → R8 功能补强 → R9 综合 → R10 验收
- 复用 e2ebench / context-maintenance-e2e 模式；每轮跑 golden 字节对比（缓存红线）

### 14. 开发纪律与执行计划（P7+）
- **阶段**：P7 前置解除（并发上限论证）→ P8 消息注入 + broadcast → P9 plan 审批 → P10 停滞检测 + 持久化 → P11 权限模型 → P12 UI → P13 60 并发 e2e
- **纪律**：每个 agent 一个主题分支、合回前四验证（gofmt/vet/repolint/全测）、纪律团审查、缓存红线每轮 golden 对比
- **前置阻塞**：P11 权限哲学拍板（write-claim 窄化=安全 vs 协作）、60 并发场景确认（是否含 fleet/只读）

### 15. 综合审查（devil's advocate）
- **攻击**：60 并发 + 全量审批 + 10 轮迭代三杠杆叠加成本不可控；write-claim 窄化是安全让步；broadcast 有界性；注入点单一（轮询无收益论证成立）
- **建议**：砍并发数字（先 10 验证再 60）、审批只对 plan-required teammate、迭代聚焦完成事件可靠性

## 二、推荐执行顺序（合并 14+15 智能体）

| 阶段 | 内容 | 依赖 | 验收 |
|---|---|---|---|
| P7 | 并发上限论证 + 后台上限解除（3→分层） | 权限哲学拍板（用户） | 60 并发槽位测试 |
| P8 | 消息注入（事件推 + `<team-messages>`）+ /team-broadcast | — | 转义防伪 + 背压 + 前缀 hash 测试 |
| P9 | plan 审批门（plan-required teammate） | P8（消息注入承载审批请求） | 审批流 + 权限收缩测试 |
| P10 | 停滞检测 + 崩溃快照 | — | abort 重派 + 恢复测试 |
| P11 | 权限模型（工具级 ask + 审批桥 + 路径白名单） | P7 | 越权拒绝测试 |
| P12 | UI TeamPanel | P8（消息流渲染） | 面板验收 |
| P13 | 60 并发 e2e + 10 轮强化迭代 | P7-P12 | 性能基准 + 验收标准 |

## 三、用户决策（2026-08-11 已拍板）

### D1. write-claim 串行 → **分支管理并行 + 令牌**（用户方案）
- **不窄化 write-claim 安全语义**；改为：teammate 写隔离 = **git 分支/worktree 并行**（CCB worktreePath 参考）——每个 teammate 在独立分支工作，写完 merge 回主分支，避免整工作区 claim 冲突
- **令牌机制**（mimo 方向）：参考 MiMo-Code 授权令牌——受控授予写权（token-bucket 式，按任务/时段发放），安全前提下的并行写
- 实现时调研 MiMo-Code 的令牌实现（a3.md 已有 actor 层线索）

### D2. 并发目标 → **先 10 验证再 60 + 动态分配**
- 阶段目标：先 10 并发验证，稳定后扩 60
- **动态并发分配**：默认上限 6（现状）；按 **任务复杂度 / 时间紧迫度 / 成本预算模型** 动态调整分配
- **provider 官方并发上限**：deepseek-v4-pro **500**、deepseek-v4-flash **2500**——provider 侧空间充裕，动态分配上限由成本模型而非 provider 限制决定

### D3. 审批主体 → **先问用户，默认用户审批**
- 派活/创建团队时**询问用户审批权限问题**（ask 工具/桌面端询问）
- 默认：**询问用户 + 用户审批**（人类直觉可少走弯路——"ai 很强大但人类直觉可省弯路"）
- leader 代批为 v2 opt-in

### D4. 成本配额 → **60 并发是并发上限，不是配额**
- 参考社区（qwen MAX_TEAMMATES=10 是并发上限非配额）——**不设成本配额**
- 成本控制靠：前缀缓存命中（核心价值）+ 动态并发（D2）按需分配，非硬性配额

### D1-D4 对执行计划的影响
- **P7 改名**：并发上限论证 → **分支并行 + 令牌机制 + 动态并发分配**（安全前提并行写）
- **P9 前置**：审批询问（D3）在派活入口加"询问用户审批模式"
- **P11 权限模型**：与 D1 令牌机制合并设计（令牌=受控写权）
- **P13 验收**：10 并发验收先行（D2），60 为第二里程碑

## 四、仍需用户确认（实施细节）
1. 令牌粒度（per-task vs per-teammate vs 时段桶）
2. 分支策略（每 teammate 固定分支 vs 每任务分支 vs worktree）
3. 动态并发模型的具体输入权重（复杂度/紧迫度/预算各占多少）
