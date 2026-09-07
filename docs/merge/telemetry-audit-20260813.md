# 压缩遥测四方审计纪律（2026-08-13 确立）

## 背景

8/13 用户质疑"修复是否在制造新 bug"。完整数据链管道追踪发现：
**根因已解决**（投影保持/校准持久化/est 系数——均有实测验证），但
**8/12 合并后未做遥测四方审计**，导致 `CompactionTelemetry` 26 字段中
detail/recorder/record 三方只接了一部分——用户发现一个缺口、补一个，
**补了 6 个提交才接近补完**（elapsed_ms → tpc → results/saved_chars →
失败落盘 → fold/spans/reason）。这是流程失误，不是技术根因。

## 四方审计方法（合并后必做）

合并涉及遥测文件（`internal/agent/*telemetry*.go`、`internal/stats/*`）
后，用三方字段对照一次性列出全部缺口：

1. `CompactionTelemetry` 结构字段（`internal/agent/projection.go`）
2. `emitCompactionTelemetry` 的 detail 键（`context_receipt.go`）
3. `recordCompaction`/`setCompactionInt` 解析的键（`recorder.go`）
4. `CompactionRecord` JSON tag（`internal/stats/record.go`）

对照规则：**结构有值 → detail 必须发 → recorder 必须解析 → record 必须有
字段**，任一环缺 = 断链，一次补完。同样适用于 Resume/Retrieval 遥测。

## 8/13 修复分层结论

### 真根因修复（已验证）

| 根因 | 修复 | 证据 |
|---|---|---|
| 投影版本漂移→失效→全量折叠→摘要 miss（10.2%） | `6158a6317` 投影保持 | 13:17 摘要 99.9% 命中 ¥0.007 |
| est 0.25 fallback 低估中文（误触发压缩） | `e015533dc` 官方系数 0.6/0.3 | tpc=0.411 |
| 校准重启丢失（tpc=0.000） | `bfb8029a8` 持久化 | tpc=0.411 落盘 |
| 1M+ 摘要 90s 超时卡死 | `73019b6d5` 超时自适应 | 1.2M 压缩成功未卡死 |

### 合并遗留遥测断链（8/12 merge 未做四方审计）

`56d150d9e`→`51fcd0b1c`→`d1e3b1db1`→`4c19472bf`→`a123407e8`：
elapsed_ms 结构/计时/record/recorder、detail 漏改、tpc 未填、results/
saved_chars 断链、非 degraded 失败被 return 吞（不落盘）、fold/spans/
reason 断链——**全部补齐**。

### 最终状态：compaction 遥测 22 键全通

trigger/mode/status/cache/est/src/fold/spans/proj/in/out/hit/miss/write/
reqs/tpc/elapsed_ms/reason/user_kept/user_dropped/pref_hash/results/
saved_chars + 条件附加（err_type/provider_request_id）。唯一有意省略：
`Native`（bool，低价值）。失败也落盘（`status=failed` + `err_type=`）。

## 摘要命中率规律（数据链验证）

- 增量折叠（fold=新增段，主请求刚发过）→ **99.9% 命中**（13:17 实测）
- 新会话/新前缀首次全量折叠（fold 历史区主请求未发过）→ miss（13:2x）
- 这是设计权衡：投影省 N 轮主请求成本，压缩时 fold 区付一次全价；
  增量折叠 + 投影保持（6158a6317）让"fold=新增段"常态化命中。

## 前缀大小分层（DeepSeek 服务端缓存）

| 前缀大小 | 空闲保留 | 命中率 |
|---|---|---|
| 小 <50K | 5-15 分钟 | 仅 9.6%（优先淘汰） |
| 大（118 万） | 天级 | 99.2%（8/7 三天重放实测） |

C1 门控可考虑按前缀大小分级（大前缀 24h warm 重放；小前缀按 5-15min
判定或冷处理）——未实施，观察项。
