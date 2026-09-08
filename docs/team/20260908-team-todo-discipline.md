# Team 任务管理：确定性机制（2026-09-08）

**原则**：任务管理是**系统机制**，不是模型自觉——任何模型（能力无关）调用 team 工具派发任务，
跟踪/核销/整组完成提示由内核必然执行。模型只需：派发、收到组完成通知后汇总交付。

## 机制链（代码实现，模型无关）

| 环节 | 机制 | 触发 |
|---|---|---|
| 派发 | team add → 后台 job 进入 jobs/teammate 状态机 | 模型调用 add |
| 单任务完成 | job done 事件 → HandleJobDone → 成员 idle + P1 信封结果回 leader | 内核自动 |
| **整组完成** | **checkGroupCompleted**：全成员 idle + 无 pending/blocked → 自动 emit 组完成 Notice 到 leader 下轮 | **内核自动（8d624e957）** |
| 结果交付 | P1 信封每份完整回 leader | 内核自动 |

## 模型（leader）的确定性职责（薄，仅剩这些）

1. 派发：`team` group_create → create N → add ×N（每任务 prompt 明确"用 retrieve_info 深度研究 + 输出报告"）
2. **收到系统"组完成"通知**（内核必然发）→ 此时汇总交付各信封结果
3. 交付后 group_delete 清场

模型**不需要**：轮询 status 盯完成、记住检查任务、自行判断"是否全完成"——系统在最后完成瞬间通知。

## 并发/安全（机制语义）

- 只读研究员（无 grant）：并行、不锁 leader；完成各自信封回投
- 写者：需 grant（路径级）——写者串行（leader 会遇其持锁，属预期）
- 默认 ≤6 只读并行；任务 prompt 建议用 retrieve_info（缓存优先 + 联网兜底）

## 历史

- 前置调研：qwen team 移植 + 并行授权蓝图（docs/team/20260907-*）
- 本机制补齐"整组完成提示"（原只到 per-task 信封；leader 需主动汇总 = 非确定性，已系统化）
