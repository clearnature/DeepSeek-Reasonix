# 衡量标准体系（docs/team/presets/metrics.md）

> 15 智能体 #4 决策：迭代效果/预置智能体的量化评估。回答用户"你建立测试和衡量的标准了吗"。

## 一、6 维度指标体系

| 维度 | 定义 | 采集方式 | 目标值 |
|---|---|---|---|
| 机制正确性 | 无竞态/死锁/回归 | `go test -race` + 并发测试 | 0 竞态 |
| 效果（任务成功率） | 产出正确性 | 3 基准任务（fizzbuzz/bugfix-discount-int/ambiguous-hard） | ≥90% |
| 延迟 | 任务耗时 | probe 记录（分配→完成） | 每轮不劣于基线 |
| 成本 | token/API 用量 | stats 遥测 | 每轮不劣于基线 |
| 吞吐（并发） | 同时 running/产出率 | 并发 probe（spawn N→add N） | N 档达标 |
| 竞态/停滞 | race 命中/停滞数 | -race + stall 遥测 | 0 / 阈值内 |

## 二、基准任务集（R1 复用，3 类型）

| 类型 | 任务 | 验证 |
|---|---|---|
| 写文件 | 在 worktree 写指定文件 | 文件存在+内容对 |
| 算法实现 | 实现快速排序（含 main 演示） | 编译+运行正确 |
| 数据分析 | 读 results.tsv 找最优 | 结论正确+数字证据 |

## 三、评估流程（每轮 5 步闭环）

```
L1 机制（race 门）→ L2 效果（e2ebench）→ L3 吞吐（并发 probe）
→ 记录（JSON+Markdown）→ 对比上一轮
```

## 四、R1 基线记录格式

```json
{
  "round": 1,
  "tasks": [{"type": "write", "ok": true, "ms": 123}],
  "concurrency": {"n": 5, "all_running": true, "outputs": 5},
  "cost": {"tokens": 12345, "cache_hit_rate": 0.88},
  "race": {"failures": 0}
}
```

## 五、CI team-gate（建议）

```makefile
team-gate:
	go test -race -run 'Team|Teammate|Fleet' ./internal/...
	golden 字节对比
```
