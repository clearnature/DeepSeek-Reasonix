# P3 steer 通道：执行与审查报告

> 事务：docs/team/20260810-p3-steer/ · 分支：team · 状态：✅ 已执行并通过纪律审查

## 一、流程记录（按团队章程）

| 阶段 | 参与者 | 产出 | 结果 |
|------|--------|------|------|
| 规划 | 3 × team-planner（独立） | plan-1/2/3.md | 共识：steer 基础设施已存在（run_loop.go:282），P3 是增量接线 |
| 定稿 | 父代理 | plan.md | 裁决：队列 16 条/8KB、双触发（send_message + /task-message）、每轮 1 条 |
| 执行 | 3 × team-executor（T1→T2→T3/T4/T5） | jobs 队列 / run-loop 注入 / 工具+slash+e2e | 实现落盘，诚实报告「无 shell 未验证」+ 4 处契约同步点 |
| 补执行 | 父代理 | 真实验证 + 契约同步 + 遗留补齐 | 全部通过 |

## 二、执行中发现并被确认的真实问题

1. **无 shell 验证缺口**：3 个执行者环境均无 bash，实现落盘但未编译测试——父代理补跑全量验证
2. **契约同步 4 处**（执行者预告准确）：`acceptsDefaultSnip`、boot 工具列表、TOOL_CONTRACT.md、golden 重新生成
3. **Kind=='task' 防线缺口**（T2 遗留）：fleet 透传 jobCtxKey 会误注入——父代理在 `SendMessageForSession` 加 Kind 检查（入队端拒绝非 task）
4. **未消费消息终态清理**（T1 遗留）：`recordCompletion` 清空 pending + Notice 计数（用户知道指引没送达）

## 三、最终实现

- **jobs 层**：`Job.pendingMessages`（16 条/8KB，j.mu）+ `SendMessageForSession`（m.mu find → j.mu 入队，两锁不嵌套；Running + Kind=='task' 才收；超限 ErrPendingQueueFull 拒绝）+ `DrainPendingMessages`（jobCtxKey，一次一条）+ recordCompletion 终态清空计数
- **agent 层**：`run_loop.go` consumeSteer 后经 jobCtxKey 消费 → midTurnSteerMessage 注入 user 尾部（每轮 1 条；父/前台/planner no-op）
- **入口**：`send_message` 工具（父代理可见、子代理隐藏）+ `/task-message <job_id> <text>` slash
- **缓存**：消息只进子代理会话尾部；父前缀零变化；golden 更新（send_message 入 provider 工具面 = deliberate 一次性打穿后稳定）

## 四、验证（全部真实执行）

```
go build ./...                                              ✅
go test ./internal/jobs/ -race -count=1                     ✅（队列/锁序/终态清理）
go test ./internal/agent/ -run 'Steer|Task' -race -count=1  ✅（注入/FIFO/no-op）
go test ./internal/control/ -count=1                        ✅（/task-message + e2e）
go test ./internal/tool/builtin/ -count=1                   ✅（send_message 工具）
go test ./internal/boot/ -count=1                           ✅（契约 + golden）
go vet + gofmt + repolint                                   ✅（essay 清零，baseline 记录功能增长）
```

## 五、缓存红线确认

- 消息注入子代理会话 user 尾部（consumeSteer 既有路径，位置固定），子代理前缀每消息一次 miss（不可避免）
- 父会话/父前缀零字节变化 ✅
- send_message 加入 provider-visible 工具面 → golden 更新，一次性打穿后新工具面稳定 ✅

## 六、遗留（后续阶段）

- 每轮注入 1 条为保守默认（多消息消化速率可调，增强项）
- 消息正文不持久化（归 P6 team 组织）
- 下一事务：P2（任务管理面板）或 P4（前台→后台降级）
