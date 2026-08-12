# P13 R5：fork 深拷贝/transcript 内存基准（2026-08-12）

> R5（内存）：fork 深拷贝（cloneForkMessages）成本 + transcript 增长——大前缀（长无压缩历史）fork 的延迟/内存是否病态。

## 基准结果

| 前缀大小 | 克隆耗时 | 吞吐 | 内存 | 分配 |
|---|---|---|---|---|
| 200KB（2000 消息） | 0.97ms | 205 MB/s | 993KB | 2001 allocs |
| **1MB（10000 消息）** | **4.8ms** | 207 MB/s | 4.96MB | 10001 allocs |

（BenchmarkCloneForkMessages，-benchtime 100ms）

## 结论

- **无瓶颈，无优化必要**：1MB 前缀深拷贝 4.8ms——fork 总延迟（捕获+继承+首请求网络 ~秒级）占比 <1%
- **线性而非二次**：每消息恒定 1 alloc（ToolCalls append([]nil)），1MB=10001 allocs（非 10000²）
- 深拷贝语义正确：clone 的 ToolCalls 独立 backing array（测试验证）
- transcript 增长：append-only 线性增长为设计特性（fork-inheritance 已让子代理继承父 lastUsage/投影，压缩由各 agent 的 ContextManager 管理）

## 交付

- `internal/agent/subagent_fork_bench_test.go`：1MB 前缀克隆基准 + 深拷贝语义防回归
- 本文件
