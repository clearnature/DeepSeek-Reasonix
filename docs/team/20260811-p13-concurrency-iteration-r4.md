# P13 R4：steer 消息吞吐基准（2026-08-12）

> R4（消息吞吐）：P1 信封/P8 消息注入吞吐——量化 per-agent steer 队列（agent.go steerQueue/steerMu）的注入/消费吞吐，判定是否有瓶颈需优化。

## 基准结果

| 场景 | 吞吐 | 说明 |
|---|---|---|
| 8 并发注入 50k 条 + 全消费 | **1,214,262 msg/s**（41ms/50k） | TestSteerQueueThroughputFloor |
| 1 worker（Benchmark） | 702 ns/op（142 MB/s） | 单调用者 |
| 8 worker | 513 ns/op（195 MB/s） | 锁竞争下反而更快（批效应） |
| 32 worker | 437 ns/op（229 MB/s） | 高并发无退化 |

## 结论

- **无瓶颈，无优化必要**：steer 队列 O(1) 追加 + 消费切片头移动（无拷贝）+ 短临界区——吞吐比 team 场景需求（1-10 msg/turn）高 5 个数量级
- 多 worker 下吞吐不退化（锁竞争可忽略）——team 多 caller 注入（jobs/mailbox/P1 信封）安全
- **防回归测试**：`TestSteerQueueThroughputFloor`（5000 msg/s 下限，现 121 万 = 243 倍余量）——队列若回归拷贝/长临界区路径即 FAIL
- Benchmark：`BenchmarkSteerConcurrentInject`（1/8/32 workers）

## 交付

- `internal/agent/steer_throughput_test.go`：吞吐下限测试 + 并发 benchmark
- 本文件
