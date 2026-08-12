# P13 R8：停滞检测阈值调优分析（2026-08-12）

> R8（功能补强：停滞检测阈值调优）——核查 jobs 层 stalled warning 与 teammate 层 abort 的阈值合理性。

## 现状（两级停滞检测，均配置化）

| 层 | 机制 | 默认值 | 配置 | 测试 |
|---|---|---|---|---|
| jobs 层 | `monitorStalled`（jobs.go:890）：job idle 超阈值 → 置 Stalled + warning | **900s**（15 分钟无 activity） | `stalled_warning_seconds`（上限 24h，0 禁用） | TestBackgroundJobStalledWarningSeconds{Default,ExplicitZero,ParsesExplicitZero,Bounds} |
| teammate 层 | `checkStalled`（team_reliability.go:37）：running 且 job Stalled → kill + 置 idle 供重派 | **0 = 禁用 abort**（warning only） | `team_stall_abort_seconds` | TestTeamStallAbortKillsStalled |

## 阈值分析

- **900s warning**：长任务（多轮工具循环、编译、真实 API 慢响应）正常可超 5-10 分钟；15 分钟无任何 activity 才判 stalled——误报率低、捕获真实挂起（provider 卡死/死锁）及时
- **abort 默认禁用（0）**：安全优先——**绝不自动杀掉可能仍在合法长跑的任务**；warning 始终可见，leader 可人工干预。此默认符合"安全第一"（用户 2026-08-12 拍板 GA/team 评估成本与安全优先）
- 两级组合：warning（信息）→ abort（可选执行）——职责分离正确

## 结论

**阈值与配置机制已完备，无调优必要**：默认 900s warning + 可配 abort（默认禁）是安全保守的正确默认。R8 完成（核查型，无代码改动）。

## 可选后续（不实施）

- 若未来需要默认 abort：可设 `team_stall_abort_seconds = 1800`（30 分钟，长任务安全 + 停滞可恢复）——留作 leader 决策
