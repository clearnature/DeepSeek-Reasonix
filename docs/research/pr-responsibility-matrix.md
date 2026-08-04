# PR 责任清单：主题归属与拆分决策（2026-08-04）

> 原则：当前 PR 只有 #7234（MiMo wire）/#7168（DashScope 完成语义）——
> 只在各自主题内解决；主题外问题独立 PR；用户使用行为（填错余额地址等）
> 与代码无关，不列为问题。

## 一、已在 #7234/#7168 主题内（PR 已含）

| 修复 | 提交 | 主题 |
|---|---|---|
| summary 隔离（能力表） | d287298c1 | #7234+#7168 共享 |
| reasoning id/status 贯通（含 broker） | aeb5b70f4/3d7d1528a/3a4da243a | #7234+#7168 |
| vendor 能力表（6 能力） | 79ec3effd | #7234 |
| JSON 输出（text.format） | d1ee719be | #7234 |
| budget 表驱动（mimo 65536） | 1e8d25ca9 | #7234 |
| multimodal（input_image） | 9dcfca8e4 | #7234 |
| tool name 空值（#4711） | 373b5fd5d | #7234 |
| TTL 24h（#7168 评审第 4 点） | fbf86b8d1/a8ee0fd10 | #7168 |
| singleSegment / 警告 scoped | 96330b565/4076a68e9 | #7234 |

## 二、P0-P2c 与 PR 主题的归属

| 修复 | 主题 | 归属决策 |
|---|---|---|
| **P2c 全零+stop 完成语义**（1e6119053，responses.go） | **#7168 完成语义**（评审"完成语义保留"完整实现） | ✅ **并入 #7168**（responses 主题内） |
| P0 #7451 effort 回退（efc1fe786，config） | 配置层 effort | ❌ 独立 PR（effort 主题） |
| P1a/P1b 中断计费（1caca8a96，agent/anthropic） | 计费/会话层 | ❌ 独立 PR（计费主题，#7184） |
| P2a vendor 逃生口（d181fc307，openai） | Chat 协议层 | ❌ 独立 PR（openai effort） |
| P2b #4814 Price 检查（997d2bdbb，config pricing） | 配置层定价 | ❌ 独立 PR（定价主题） |
| #7357/#7358 配置合并（e2e2dae0c，config load） | 配置层合并 | ❌ 独立 PR（配置合并） |

## 三、统一分支 fix7234and7168 的验证任务（职责内）

P2c（1e6119053）应 cherry-pick 进统一分支：
- 它是 #7168 评审"完成语义与计费数据分离"的完整实现（原实现只做一半）
- 属于 responses 主题内（responses.go），不越界
- 验证点：全零+stop 发送 → agent reasoningOnly 完成 honored；计费层 0 成本

## 四、非问题清单（用户使用行为，与代码无关）

1. **mimo 余额地址**：用户自行配置非官方 balance_url / 填错地址 =
   用户使用行为（代码只防"继承默认"——#7357 已修；主动配置属自担准确性）
2. 不属于任何 PR 的配置层/计费层问题 → 独立 PR 待后续（上游 PR 不可预期，
   本地 dev 已全部修复，优先级由用户定）

## 五、独立 PR 建议拆分（按主题，待用户确认）

| 独立 PR | 内容 | 状态 |
|---|---|---|
| effort 主题 | P0（#7451）+ P2a（#4099/#3561 逃生口） | dev 已修，待 PR |
| 计费主题 | P1a/P1b（#7184 中断计费）+ P2c 剩余 | dev 已修，待 PR |
| 配置合并主题 | #7357/#7358 + P2b（#4814） | dev 已修，待 PR |
