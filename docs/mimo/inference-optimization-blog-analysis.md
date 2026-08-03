# MiMo-V2.5 推理全链路优化与技术解析

> 来源: https://mimo.xiaomi.com/zh/blog/mimo-v2-5-inference（Xiaomi MiMo Team, 2026）
> 整理: 2026-08-03，含 Reasonix 实测数据交叉验证

## 概述

MiMo-V2.5 系列（MiMo-V2.5、MiMo-V2.5-Pro）融合三大架构：**Hybrid SWA**（混合滑动窗口注意力）+ **MoE**（稀疏激活）+ **多模态 Encoder**。本文记录小米团队围绕该系列的一次推理系统全链路工程化实践，覆盖 KVCache 管理、分级缓存、SWA 前缀缓存树、调度策略、Prefill/Decode 执行链路及多模态优化。

---

## 一、Hybrid SWA 架构的推理效率优势

### 1.1 计算量分析

- MiMo-V2.5-Pro：**70 层**，其中 **10 层 Full Attention + 60 层 SWA**，SWA 窗口 **W=128**
- SWA 层占比 6/7 → 计算量约为 Full Attention 的 **1/7**
- Chunk Prefill 场景近似 compute-bound → Prefill 成本理论缩减 ~7×

### 1.2 KVCache 存储分析

- SWA 层仅需保留滑动窗口内 KV → KVCache 占用降至接近 **1/7**
- Decode 阶段近似 memory-bandwidth-bound → 长序列下 KVCache 减少直接等价 decode 成本降低
- KVCache 效率国产模型排名第二（仅次于 DeepSeek-V4）

**关键数据**：
- Attention 计算量（FLOPs）差距：~7.0×
- KVCache 显存占用差距：~7.0×

---

## 二、KVCache 系统重构

### 2.1 SWA KVCache 管理

**核心矛盾**：Full Attention 层需 O(N) 存储，SWA 层仅需 O(W)。传统单一 KV pool 按 O(N) 统一分配 → SWA 稀疏性被浪费。

**双池设计**：
- 物理层：Full KV pool（O(N)）+ SWA KV pool（O(W)），SWA pool 支持基于 window 的独立 eviction
- 逻辑层：暴露单一序列视图，Full Attention 索引为权威索引
- 调度约束：接入时同时校验 Full/SWA 容量
- 数据搬运：跨层传输仅基于 SWA mask，只搬有效窗口内数据
- 效果：**KVCache 容量效率提升 ~7×**

**KVCache 按层异步拉取**：SWA 层只需 prefetch 极少 KVCache，layerwise 粒度调度实现完美 overlap，Cache 读取成本 ≈ 0。

**SWA-aware 前缀缓存树**（最核心创新）：

传统 RadixAttention 假设"token 相等 → KV 相等"在 SWA 下失效——前缀树逻辑生命周期 ≠ SWA KV 物理生命周期。三层改造：

1. **匹配规则 = "窗口安全长度"**：除 token 相等外，保证尾部至少 W 个 token 在 SWA 池有有效 slot；越过边界的部分按 miss 处理
2. **淘汰与请求生命周期绑定**：chunk 完成/请求结束/decode 每 N token 都触发窗口外 SWA 释放 → SWA 池占用恒定 O(W)
3. **双索引节点**：每个节点记录 Full Attention 段索引 + SWA 段映射，可独立淘汰（只淘汰窗口外 SWA 保留 Full 段）

**缓存命中率优化**（分布式一致性）：

| 场景 | 处理 |
|------|------|
| Device 完备、Host 空缺 | 主动检查差集，host 端补分配 SWA slot，异步 D2H 写入 |
| Host 完备、Device 空缺 | 等下次 H2D 自然对齐 |
| 高频序列 L3 过期 | 定期 query L3 防止过早淘汰 |
| 中短序列 SWA 留存 | 按请求 pattern 固定长度留存稠密 SWA KV |

### 2.2 GCache：高性能分布式缓存基础设施

- 同时支持文件/KV 语义，内存/磁盘/远程多级缓存
- shm 内存持久化、全链路零拷贝、高并发非阻塞 IO、RDMA 通信
- 非中心化元数据（一致性哈希 + Raft Master 只管心跳）
- 网络优化：GPU 网卡优先，1MB IO 单进程 RDMA 读 **170 GB/s**（280us 延迟），GDR 场景 **350 GB/s**
- 存储成本：GPU 机器混布，额外存储成本 **0**；单副本足够（故障处理 + 数据迁移 + SDK 超时重算兜底）

### 2.3 缓存命中率

- 主流 harness 框架平均：**93%**
- 高强度、长周期个人用户：**95%+**
- 核心机制：SWA 极小存储 → 相同成本承载更多缓存 → 淘汰压力小 → **TTL 可显著延长**

---

## 三、调度优化

### 3.1 KVCache 与负载亲和调度

```
score(worker) = matchWeight × prefix_match_percentage − normalized_load
```
- 效果：L2 命中率 **+25%**，单机输入吞吐 **+30%**

### 3.2 TTFT 优化

- 优先调度真实计算 token 更少的请求 + 等待时间惩罚机制防饥饿
- 效果：长请求 TTFT P90 降低 **30.5%**，短请求基本不变

---

## 四、Prefill 优化

| 优化 | 效果 |
|------|------|
| EP 缩减（SWA 优化后） | 端到端性能 **+40%** |
| 三级长度分桶（0-64K/64K-256K/256K-1M） | 避免短请求被长请求拖慢 |
| MoE 负载均衡 | 无需额外策略（训练已学均匀路由，负载因子 0.85） |
| NUMA 冲突修复（禁用 numa_balancing） | 端到端 **+10%** |

---

## 五、Decode 优化

| 优化 | 效果 |
|------|------|
| Decode KVCache 完整 SWA | 有效容量提升 **~5×** |
| PD 分离 KVCache 预分配 GPU→CPU | 消除资源预占浪费 |
| CUDA Graph 显存调优 | 可用显存提升 |
| MTP 优化（prefill 阶段引入） | 0-128 token **2.3×** / 128-256 token **1.5×** |

---

## 六、多模态推理优化

- Encoder 吞吐翻倍：QPS **15→30**（延迟不变，P90 100.76ms→82.94ms）
- 架构：Embedding 异步复制×Prefill 重叠、Encoder TP=1+DP、跨请求组 Batch
- 预处理：图片 GPU 预处理、并行下载、下载/forward 并行、视频并行解码（1h 视频 156s→23s）
- 缓存：Encoder 一致性哈希（命中 +30%）、机内 Embedding 共享

---

## 七、与 Reasonix 实测的交叉验证

### 7.1 缓存 TTL 实测（关键决策依据）

**短前缀（646 tokens）**：
```
T+0s    cached=0    MISS（冷）
T+2s    cached=576  HIT
T+210s  cached=640  HIT
T+450s  cached=0    MISS（7.5min 失效）
```

**长前缀（8.8K tokens / 17K input）**：
```
T+0s      cached=0    MISS（冷）
T+2s      cached=0    MISS
T+30min   cached=0    MISS
T+60min   cached=0    MISS
T+90min   cached=0    MISS
T+120min  cached=17024/17034  HIT（99.9%）
```

**关键结论**：
1. 长前缀缓存建立有 **1.5-2 小时预热窗口**（异步 D2H/H2D 搬运 + GCache 跨机拉取 + 亲和调度）
2. 短前缀 7.5min 失效 ≠ TTL 短——是"缓存未建立/被 LRU 挤出"，非"TTL 到期"
3. **TTL ≥ 2 小时**，与博客"数小时"尺度吻合
4. **cacheColdAfter = 24h 决策正确**（mimo 与 DeepSeek 同档）

### 7.2 端点差异

| 端点 | cached 上报 | 结论 |
|------|------------|------|
| /v1/responses 非流式 | ✅ 可靠 | 测试/统计用 |
| /v1/responses 流式 | ❌ input_tokens_details={} | 统计不可见（异步架构固有延迟，非 bug） |
| /chat/completions | ❌ 恒 0 | 要么无缓存要么不报告 |

### 7.3 思考模式

- MiMo effort 四档（none/low/medium/high）实际只有**开/关**两种行为
- 开 = **单段固定深度思考**，不可调（low/medium/high 无差异）
- 跨轮：reasoning_content 需客户端完整回传（否则 400）——"补缴"而非"续接"
- 与 DeepSeek 差异：DeepSeek 多段思考可续接；MiMo 单段一次性

### 7.4 缓存命中机制

```
MiMo 自动缓存命中 = f(内容哈希, SWA 窗口安全长度, L1/L2/L3 一致性, 实例路由)
Reasonix prefix-cache = f(字节稳定性)  ← 客户端完全可控
```

- 93% 命中率是**服务端全局统计**，不等于单用户命中率
- Reasonix 的字节稳定 append-only 与 MiMo 服务端自动缓存是**叠加关系**（非替代）
- 非流式 cached_tokens 是 **L1 可见性快照**，不是全局缓存状态快照（预热中的长前缀 cached=0 但缓存正在建立）

---

## 八、对 Reasonix 的落地结论

| 决策 | 值 | 依据 |
|------|-----|------|
| cacheColdAfter | **24h**（mimo 已入档） | 长前缀 2h+ 99.9% 命中实测 |
| 端点选择 | responses | 功能完整（工具/思考/结构化输出）；chat cached 不可见 |
| 缓存命中统计 | 非流式 cached 可靠；流式不可见 | 多次实测 |
| effort 归一化 | none 支持（唯一有实际意义的档位） | low/medium/high 等价 |
| 工具消息 name | 始终序列化（#4711） | MiMo 严格校验 |
| 运行时观测 | 用真实请求非流式 cached 累积，无需专用探测 | 零额外成本 |

**技术路线判断**：MiMo 与 DeepSeek 同类——stateless + 服务端自动缓存 + 磁盘级长 TTL。差异：MiMo 是 Hybrid SWA 三级缓存 + 分布式 GCache（存储省 7×、长前缀建立慢、命中依赖亲和调度）。
