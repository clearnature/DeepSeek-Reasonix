# Agent Team 编排：qwen-code vs Reasonix 架构报告（2026-09-07）

> 范围：架构级分析（状态/通信/任务/生命周期/调用面五个模型），不是逐工具修补清单。
> 明细差距表见 `20260907-qwen-team-gap.md`（登记）与 `20260907-call-surface-audit.md`（调用配套）。
> 本文 = 体系化架构报告 + 目标架构设计建议。

---

## 一、两种编排架构的五个模型

### 1.1 qwen-code（磁盘第一 · 拉模式 · 工具面 · 单例组）

```
持久层：TeamFile + tasks/ + inboxes/（全磁盘，proper-lockfile + atomic 写）→ 进程崩溃全存活
状态模型：磁盘即真相；内存只是热缓存（TeamManager 持有成员/身份/回调，全部可从磁盘重建）
通信模型：磁盘 mailbox（单向消息文件）+ 双向 500ms 轮询（leader poll inbox；teammate flush）
任务模型：拉模式——teammate IDLE 自动认领任务板 pending（scanIdleAgentsForTasks）
生命周期：单 leader 单组（TeamManager 单例）→ spawn 成员（槽预留+rollback）→ 任务 → 
          协作 shutdown（requestShutdown 消息协议）→ allTeammatesTerminated → 组文件保留，显式 team_delete
调用面  ：全部模型工具（agent/task_*/team_create/team_delete/send_message/team_plan_approval）
          ——leader 在对话流内完成整个编排，无 host 命令
进程模型：teammate = 同进程 subagent 运行时（InProcess）或 Tmux/ITerm（跨终端）
```

### 1.2 Reasonix（事件驱动 · 推模式 · host 命令面 · 无组）

```
持久层：fork transcript 落盘（SubagentStore）；mailbox 磁盘层实现但 boot 未接线（ephemeral）；
        状态快照 snapshotPath 仅声明未实现（纯内存注册表）
状态模型：内存即真相（jobs/TeammateStore 状态机在内存，完成事件驱动状态迁移）
通信模型：内存 mailbox + P1 完成信封（结果随 job 完成一次性回 leader）+ team_message 工具
          ——无实时中途注入，无磁盘协议
任务模型：推模式——leader /team-add 显式指派；依赖自动推进（完成事件 → 下个任务）
生命周期：teammate 独立个体（create/add/status/remove/stop）；无组、无协作解散、无事务边界
调用面  ：host 命令（/team-* 用户敲）+ 三个零散模型工具（task/send_message/team_message）
          ——agent（模型）无法自主编排
进程模型：teammate = 后台 job（fork 子代理，事件驱动完成）
```

## 二、架构维度对照

| 维度 | qwen | Reasonix | 差异本质 |
|---|---|---|---|
| 真相载体 | 磁盘（可跨进程/崩溃）| 内存（事件状态机）| qwen 牺牲实时性换存活；我们换事件精确但丢持久 |
| 通信时序 | 500ms 轮询（≤500ms 延迟，磁盘吞吐）| 事件即时（内存，P1 信封一次性）| 我们快但不持续（无中途流）|
| 任务分发 | 拉（teammate 自助）| 推（leader 指派）| 拉=负载自适应+teammate 主动性；推=精确控制 |
| 生命周期粒度 | 组（单例）+ 成员 | 成员（无组）| **组是我们结构性缺口**（事务边界/批量释放无载体）|
| 调用面范式 | 模型工具（模型自主编排）| host 命令（人类编排）| **我们缺"模型可编排"层**——最大调用配套差 |
| 依赖推进 | blockedBy 图 + 拉认领 | 完成事件 → 自动推进 | 同效，机制不同（我们事件驱动更即时）|
| 计划审批 | 工具协议 + 审批期收缩 | request 工具有、应答缺 | 半套 |
| 结果回投 | 实时注入 + 完成 | 完成信封 | 我们少实时层 |

## 三、差异根源（为什么不是补丁能补）

1. **调用面是架构决策，不是缺工具**：qwen 把编排权给模型（工具面 = 模型在对话里当"指挥者"）；我们从 P6 裁决起把编排权放人类（host 命令）。要让 agent 能自主组队/派活/解散，不是加几个工具，是**承认"模型是指挥者"范式**——需要 leader 侧完整的编排工具族（创建/派/查/批/解散）+ 它们与现有 job/mailbox 的接线。
2. **组是状态模型缺口**：事务级协作（写作小组→任务→解散）需要一个**组实体**承载成员聚合、事务状态、批量生命周期——内存 map + 逐个 remove 无法表达。组 = 持久化聚合根（qwen TeamFile 的位置）。
3. **持久化缺席**：mailbox 磁盘层写了没接线、状态快照只声明——不是漏一行，是**没决定"磁盘在哪一层"**（全量磁盘 vs 事件日志 vs 快照）。要做组 + 跨重启存活必须定。
4. **实时层缺失**：完成信封是"点"，qwen 的注入是"流"——需要 leader 侧消息队列 + 每轮注入协议（改变发送侧前缀结构 → 触碰缓存红线，必须单独设计）。

## 四、目标架构设计建议（蓝图）

```
目标：Reasonix team = 基础编排能力层（可被模型工具面 + host 命令面共同驱动）

┌ 调用面（双层）
│  模型工具族：team_group_create / team_group_add_task / team_group_status /
│              team_group_dissolve + send_message(teammate) / team_plan_approval
│  host 命令：/team-*（人类手动，保留）
├ 组聚合根（新）：TeamGroup{ name, members[], state(active/ dissolving), tasks[], 事务边界 }
│   └ 持久化：原子快照（P10 snapshotPath 落地）+ 构造加载（跨重启存活）
├ 既有：TeammateStore（成员执行/状态）/ jobs（job 生命周期）/ 完成事件（状态迁移触发器）
│   └ 接线：inboxRoot 落盘（mailbox 磁盘层启用）
├ 任务板（可选阶段）：teammate 可 task_list 自助（拉模式补充，保留推）
└ 生命周期语义：事务开始=组 active；全部成员完成（完成事件聚合）→ 
     dissolve（协作收尾 or 自动清组）→ 归档 transcript + 释放 mailbox
```

设计要点：
- **组 = 现有 TeammateStore 之上的聚合视图**（成员复用，不重造执行器）——状态与成员仍由 TeammateStore/jobs 负责，组只加"集合 + 事务状态 + 批量生命周期 + 持久化"
- **模型工具面封装同一 Controller 路径**（applyTeamCommand 同源逻辑，避免双份语义）
- **实时注入**单独立项：需设计"teammate 消息 → leader 下轮前缀"的位置固定注入（碰缓存红线，哲学审核后做）
- 磁盘策略：组快照（状态）+ mailbox 落盘（消息）两处即够，无需 qwen 全磁盘任务板（我们事件驱动更强）

## 五、演化路径（架构级阶段，非补丁）

| 阶段 | 交付 | 架构意义 |
|---|---|---|
| A | 组聚合根（内存）+ /team-group start/status/dissolve（host）| 建立组状态模型与事务边界（可测）|
| B | 组快照持久化 + boot 接线 inboxRoot | 组跨重启存活（磁盘策略定案）|
| C | leader 模型工具族（组管理封装 host 逻辑）| 承认"模型是指挥者"范式（调用面补齐）|
| D | dissolve 语义：事务完成自动释放 vs 显式（用户决策）| 生命周期闭环 |
| E | 实时注入（位置固定协议，单独哲学审核）| "点"到"流" |
| F | 任务板拉模式（teammate 自助）| 可选，看需求 |

每阶段独立可交付、守卫测试、防丢模板（上游无路径文件）。

---

## 附：本报告与其他文档的关系

- 差距登记明细 → `20260907-qwen-team-gap.md`
- 调用配套逐项表 → `20260907-call-surface-audit.md`
- 本文 = 架构范式对比 + 目标蓝图 + 演化阶段（决策输入）
