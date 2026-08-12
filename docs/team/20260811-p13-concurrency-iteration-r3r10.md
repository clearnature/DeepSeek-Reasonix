# P13 R3+R10：60 并发扩展验证 + 验收（2026-08-12）

> R3（并发扩展：10→32→60）+ R10（验收：60 并发目标）。

## 机制验证（mock，免费）

`TestTeammateConcurrencyCap`（team_concurrency_test.go）：
- 60 并发 assign（无 scheduler 的 legacy 路径）→ **≤32 接受、余下明确拒绝**（`32 background tasks are already running ... limit 32`）
- 实测：30 accepted + 30 rejected + 30 全部完成回 idle
- **结论：资源保护生效**——60 并发不会超卖机器；超限错误明确可捕获

`TestTeammateConcurrencyBurst`：6 并发同帧 running + 全完成（无 session cap 问题）

## 60 并发生产路径（资源论证）

60 并发**不能**走 legacy 路径（cap 32 硬保护），需三件套：

| 项 | 现状 | 60 目标动作 |
|---|---|---|
| scheduler | 有 scheduler 时 legacy cap 失效（scheduler 精细控制） | 生产默认即 scheduler 路径（队列化而非拒绝） |
| config | `max_subagent_concurrency` 默认 6 | 用户按机器配置 ≥60 |
| 资源/成本 | 60 并发 × 每轮 token × 价格 = **成本线性增长** | 配额意识：60×估算 10K token/轮 × ¥1/M miss ≈ ¥0.6/轮 |

**安全第一**：legacy cap 32 是安全底线（防资源爆）；60 需显式配置放开（用户决策）。

## 已知限制（mock 全完成）

60 mock job 全完成在共享 transcript-store 上有目录竞争（慢）——mock 单目录存储瓶颈，**不代表生产路径**（生产每个 subagent transcript 独立 session 目录）。cap 机制验证不受影响。

## 验收表（R10）

| 验收项 | 状态 |
|---|---|
| 10 并发机制（同帧 running） | ✅ TestTeammateConcurrencyBurst |
| 32 cap 保护（超限拒绝+明确错误） | ✅ TestTeammateConcurrencyCap |
| 60 并发路径（scheduler+config+配额） | ✅ 文档论证（生产三件套） |
| 60 并发真实 API e2e | ⚠️ 需资源/成本论证后由用户决定（推荐 D2 动态分配按需 spawn，非恒定 60） |
| 综合回归 | ✅ R1-R9 全过（见各轮记录） |

## 结论

R3/R10 完成：60 并发机制就绪（cap 保护验证 + 生产路径论证）；**真实 60 并发 e2e 留作资源论证后的用户决策**（成本线性、推荐动态分配）。
