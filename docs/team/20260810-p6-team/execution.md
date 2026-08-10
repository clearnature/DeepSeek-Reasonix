# P6 team 组织：执行与审查报告

> 事务：docs/team/20260810-p6-team/ · 分支：team · 状态：✅ 已执行（父代理亲自实现，吸取 P5 超时教训）

## 一、流程记录（按团队章程）

| 阶段 | 参与者 | 产出 | 结果 |
|------|--------|------|------|
| 规划 | 3 × team-planner（独立） | plan-1/2/3.md（正文在 preview） | 共识：方案 A（消息驱动轮次），否决 Qwen 常驻循环 |
| 定稿 | 父代理 | plan.md | 裁决：teammate 只读、mailbox 简化（运行中 P3 指挥）、slash 入口 |
| 执行 | **父代理亲自实现**（P5 执行小队超时的教训） | T1 解耦 + T2 TeammateStore + T4 slash | 全部通过 |

## 二、实现

- **T1 fork/silent 解耦**（profile_spec.go ContextRequest.Silent + task.go fork 分支）：fork 默认静默（保 P5 fire-and-forget）；teammate 显式 `Silent: false` → 结果经 P1 信封回 leader
- **T2 TeammateStore**（teammate_store.go，agent 包）：注册（name/role）/列表（惰性状态同步）/派活/移除/destroy；`Assign` 首轮 fork 非静默 + 续轮 continue_from（同一 transcript 前缀稳定）
- **T3 派活管线**：复用 RunProfileSpec 全链路（fork/continue/后台 job/信封/steer 零新通道）——按定稿「复用基座」原则，未新增独立管线
- **T4 slash**：`/team-create` `/team-add` `/team-status` `/team-remove` + Controller 接线（WithForkSource/WithParentSession/WithManager 注入）
- **T5 team_message 工具裁减**：列 P6.1 增强（MVP 用 P1 信封回报足够；避免工具面变化）
- **安全**：teammate 只读（forkReadOnlyGate 复用 P5）；运行中重复派活拒绝（steer 用 send_message）

## 三、验证（全部真实执行）

```
TestTeammateStoreCreateListRemove          ✅ 注册/列表/移除
TestTeammateStoreAssignRejectsUnknownAndRunning ✅ 两个 guard rail
TestTeammateAssignStartsBackgroundJobWithEnvelope ✅ e2e：fork 非静默 + P1 信封回 leader + 惰性 idle
go build ./... + 全量 5 包测试            ✅
go vet + gofmt + repolint                  ✅（功能增长入 baseline）
```

## 四、缓存红线确认

- 首轮 fork 复用 captureForkPrefix（零发送、leader 零改动）→ 首请求命中 leader 缓存 ✅
- 续轮 continue_from 同一 transcript（前缀稳定，validateMeta 三要素强制）✅
- teammate 结果经 P1 信封（append leader 下轮容器，位置固定）✅
- 运行中指挥走 P3 steer（teammate job 即普通 task job）✅
- 无新发送前缀结构、无插入历史 ✅

## 五、遗留（P6.1 增强清单）

- teammate 写权限（forkReadOnlyGate 参数化）
- 磁盘持久化 mailbox（teammate 间/未运行消息）
- 任务依赖树（MiMo 完成门）
- team_message 工具（teammate→leader 运行中汇报）
- teammate↔teammate 直连（当前 hub-and-spoke）
- 持续运行 + task_stop/paused（Qwen）
