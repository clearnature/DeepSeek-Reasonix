# 摘要请求缓存打穿终局修复审计（2026-08-31）

## 背景：用户面板数据的三个阶段

| 时间 | 面板数据 | 判读 |
|---|---|---|
| 08-31 早 | 9 次摘要，1,071,000 token，命中 13.45% | 修复前基线：9 次全部 system-only（141,568÷9≈15,730≈system 前缀） |
| 08-31 上午 | 10 次摘要，1,329,974 token，命中 12.11% | 0514 版仍 system-only——修复只生效一半 |
| 08-31 晚 | 11 次摘要，1,588,287 token，命中 26.07% | 10 次修复前 + **1 次修复后（97.5%）**，均值被历史拖累 |

面板 11 次 = 08-30 19:44 ~ 08-31 07:20 移动窗口，与 stats 逐 token 吻合
（in=1,564,662 / hit=407,936 / miss=1,156,726）。

## 根因链（三层，逐层剥离）

### 第 1 层（08-30 已修，d9e235177）：视图分叉

投影失效时普通请求走降级分支（投影+tail），压缩规划 fail-closed 回退 canonical
全量——摘要 fold 与采样前缀从第一条非 system 消息就字节不同。命中率崩到 0-4%
（08-30 19:44 实测 3.6%）。

### 第 2 层（08-31 定案，51e5d6e9b + f1cce4a6b）：冻结单元不完整

`saveMainRequest` 只冻结 `Messages` 不冻结 `Tools`。DeepSeek 服务端缓存单元是
**system+tools+messages 完整前缀**，desktop 6 个 MCP 服务器（puppeteer/context7/
zai 等）resume 后异步重连注册工具 → 主请求与摘要请求的 tools 集合不同 → 前缀在
tools 处打穿 → 永远 system-only。

证据链（06:30:32，0514 版）：

1. sidecar `last_wire_messages`（907 条）指纹 = 192bace1 = 06:29:36 主请求
   `wire_fp`——messages 逐字节一致，saved 分支确实走了
2. 摘要 in=256,122 仅命中 16,896（= system 前缀）——分叉点必在 system 与
   messages 之间 → 只剩 tools 参数

### 第 3 层（fd34b1008，验收前复查发现）：legacy sidecar 无 tools → 永久 miss

旧版 sidecar（无 `last_wire_tools`）resume 后 `saved.tools=nil`，原判定只看
`len(saved.messages)>0` → 发送空工具列表 → tools 处打穿，且 commit 把空列表写回
sidecar → **永久性 system-only**。修复：冻结 tools 为空时回退 live registry。

## 修复 commit

| commit | 内容 |
|---|---|
| `51e5d6e9b` | `saveMainRequest` 冻结主请求 wire（messages 全量）；fold 不重发、用锚点指令；sidecar 持久化 `last_wire_messages`，resume 恢复 → 首次压缩命中父进程单元 |
| `f1cce4a6b` | 冻结单元升级为 messages+tools；`summaryRequest` 重放冻结前缀时复用冻结 tools；sidecar 新增 `last_wire_tools`；`compactSidecarRawMessages` 解决 `MarshalIndent` 美化 `json.RawMessage`（工具 Parameters/响应 items/server-search payload）导致的字节漂移 |
| `fd34b1008` | legacy sidecar（有 messages 无 tools）回退 live registry；`summaryRequestToolsForCommit` 镜像同一判定，保证 sidecar 存储与实际上行字节一致 |

## 验证数据

### 同一会话前后对照（desktop 实测，view_fp=192bace1）

| 时间 | 版本 | in | hit | 命中率 |
|---|---|---|---|---|
| 06:30:32 | 0514（修复前） | 256,122 | 16,896 | 6.6%（system-only） |
| 07:20:36 | 0700（修复后） | 255,795 | 249,472 | **97.5%** |

07:20 摘要 miss=6,323 ≈ fold（4,697）+ 指令——完全符合预期。压缩后 sidecar
升级为 907 msgs + 14 tools，后续 resume 直接恢复完整单元。

### 11 次全量明细（面板窗口）

| # | 时间 | 输入 | 命中 | 命中率 | 判读 |
|---|---|---|---|---|---|
| 1 | 08-30 19:44 | 348,684 | 12,544 | 3.6% | system 附近（旧 system） |
| 2 | 08-30 21:47 | 222,690 | 11,008 | 4.9% | system 附近 |
| 3 | 08-30 22:16 | 73,770 | 16,768 | 22.7% | system-only |
| 4 | 08-30 23:11 | 76,008 | 16,768 | 22.1% | system-only |
| 5 | 08-30 23:40 | 85,961 | 16,896 | 19.7% | system-only |
| 6 | 08-30 23:49 | 33,974 | 16,896 | 49.7% | system-only |
| 7 | 08-31 00:26 | 86,355 | 16,896 | 19.6% | system-only |
| 8 | 08-31 00:52 | 74,191 | 16,896 | 22.8% | system-only |
| 9 | 08-31 00:58 | 51,112 | 16,896 | 33.1% | system-only |
| 10 | 08-31 06:30 | 256,122 | 16,896 | 6.6% | system-only（0514） |
| 11 | 08-31 07:20 | 255,795 | **249,472** | **97.5%** | 完整单元（0700） |

前 10 次命中率波动只是分母（输入 3.4 万~34.8 万）在变，分子恒为 system
大小——「每次全价」的直接证据。

### 冒烟验证（真实 DeepSeek）

- CLI（隔离环境）：66% → 52% → 80% → 99.5%
- serve 循环（同字节重复重放）：40.1% → 67.9% → 99.6% 稳定

### 服务器缓存特性：渐进落盘

同字节第一次重复重放并非立即 100% 命中（首次重复 40-68%），重复重放后
99.6% 稳定——DeepSeek 缓存写入是异步/分片的。**这不是 bug**，是缓存建立
过程；修复的意义是把「每次重放都从 0 开始」变成「重复重放逐步收敛到
99%+」。

## 已知限制（2026-08-31 实证，用户决策暂不修）

### sidecar 冻结单元陈旧 → 长时间活动后 resume 首次压缩 0% 全价

`sidecar.last_wire_messages` 只在压缩 commit 时更新，不随主请求更新。长时间
活跃会话（主请求前缀持续增长）后 resume，恢复的冻结前缀是**上次压缩时代**的
旧字节，服务器缓存中该前缀已被期间的大量新内容冲掉（TTL/挤出）→ 首次压缩
0% 命中全价。

实证（会话 20260830-035002，2026-08-31）：

| 时点 | 事件 | 命中 |
|---|---|---|
| 07:20:36 | 压缩（同进程内，前缀=刚发过的主请求 wire） | 97.5% |
| 07:22~22:40 | 持续对话 15h，68 万新字节主请求不断发送 | 每轮 99%+ |
| 23:01:50 | **resume 新进程后手动压缩**（重放 07:20 时代陈旧前缀 25.6 万 + extra 19.7 万） | **0%（in=453,516，¥0.69）** |
| 23:05:16 | 压缩后首个主请求（新前缀首次发送） | 2,816/81,650（同样全冷） |
| 23:05:18 起 | 后续主请求 | 99%+（缓存重建） |

旁证：23:05 主请求同样几乎全冷 → 当时服务器缓存整体无该前缀 → **不是摘要
请求缺陷**；23:05:18 起 99%+ → 缓存建立后一切正常。

**判定方法（区分 tools 分叉 vs 缓存冷启动）**：
- tools 分叉/冻结单元断裂：**主请求热（99%+）但摘要 system-only**（hit≈16.9k）
- 缓存冷启动/陈旧 saved：**摘要 0% 且紧接的主请求同样几乎全冷**（hit≈0-3k），
  之后同前缀重放 99%+

**设计代价**：C1 窗口外的首次重放付全价属预期；本场景（活跃 15h 后 resume）是
其自然延伸。修复候选已记录未实施（用户 2026-08-31 决策暂不修）：
1. sidecar 保鲜——主请求后节流更新冻结前缀（resume 恢复最近字节 → 缓存热）
2. resume 陈旧降级——冻结前缀超安全窗口不赌缓存，走视图重放/裁剪

### 已实施（2026-09-01）：sidecar 保鲜 + tools 分叉遥测

用户 2026-08-31 深夜决定实施（针对 23:44 system-only 案例加遥测定位）：

1. **sidecar 保鲜**（`maybePersistFreshMainRequest`，`freshWireSidecarInterval=60s`）：
   主请求后节流刷新 sidecar `last_wire_messages`/`last_wire_tools`——resume 恢复
   最近主请求的字节 → 服务器缓存必然热 → 首次压缩命中。纯本地写盘零 API 成本。
   23:01 类场景（活跃 15h 后 resume 0% 全价）不再复现。
2. **tools 分叉遥测**：detail 行新增 `tools_count`/`tools_fp`/`tools_source`
   （frozen|live|none）——23:44 类 system-only 命中可直接判定工具缝分叉源
   （冻结集 vs live 回退），无需离线指纹比对。

## 维护纪律（新增）

1. **发送侧任何改动**（tools 注册时序、MCP 重连、拦截器、工具 schema 字段）
   先问：「主请求与摘要请求之间，前缀字节会分叉吗？」——会 = 摘要打穿，
   必须同步冻结。
2. **冻结单元 = 服务器缓存单元**：system+tools+messages 一体，冻结必须三者
   完整；sidecar 持久化形态必须是 `normalizeModelRequestMessages` 规范化后的
   实际上行字节（fixpoint 测试守护）。
3. **sidecar 字节保真**：`MarshalIndent` 会美化 `json.RawMessage` 内部字节
   （工具 Parameters 等），写入前必须 compact；round-trip 字节保真测试守护。
4. **legacy 数据路径**：sidecar 字段新增后，旧文件缺失字段必须回退 live
   值而不是发送空值——空值会「写死」一个永久错误前缀。
