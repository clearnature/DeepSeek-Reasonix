# MiMo Credits 积分算法拆解

> 来源: 官方账单 xlsx（`/home/yanli/文档/deepseek/mimo8月份.xlsx`）+ MiMo 官方文档
> 日期: 2026-08-07

## 一、结论速览

MiMo 的 Credits 与人民币存在**统一换算关系**：

```
1 Credit = 1e-8 元（0.00000001 元）
Credits = 人民币费用 × 1e8
```

MiMo 页面显示的"费用"是 **Credits 折算值**（×1e8 缩放），实际扣费是人民币。

## 二、MiMo 双轨计费体系

MiMo 有**两套**计费：

| 轨道 | 计费单位 | 适用 |
|---|---|---|
| **API 按量** | 人民币直计（hit/miss/out 单价） | 普通 API 调用 |
| **Token Plan 套餐** | Credits 折算（按 token 消耗） | 订阅套餐用户 |

用户实际走 **API 按量**（无套餐），但 MiMo 页面把费用折算成 Credits 显示。

## 三、Credits 倍率表（官方）

| 模型 | 输入（命中缓存） | 输入（未命中缓存） | 输出 |
|---|---|---|---|
| **mimo-v2.5-pro** | 2.5 Credits | 300 Credits | 600 Credits |
| **mimo-v2.5** | 2 Credits | 100 Credits | 200 Credits |

**模型分级**：pro（高级）≈ v2.5（快速）的 **3 倍**（miss/out），hit 为 1.25 倍。

## 四、Credits ↔ 人民币换算验证

| 模型 | 类型 | API 价（¥/1M） | Credits/token | ¥/Credit |
|---|---|---|---|---|
| pro | hit | 0.025 | 2.5 | **1e-8** |
| pro | miss | 3.0 | 300 | **1e-8** |
| pro | out | 6.0 | 600 | **1e-8** |
| v2.5 | hit | 0.02 | 2.0 | **1e-8** |
| v2.5 | miss | 1.0 | 100 | **1e-8** |
| v2.5 | out | 2.0 | 200 | **1e-8** |

**所有档位 ¥/Credit 完全一致 = 1e-8**，证明 Credits 就是人民币 × 1e8 的整数化表达。

## 五、账单列定义（xlsx）

`Model usage detail` 表列：

```
Date, Model, API Key, Currency, Consumed Amount,
Input Hit Amount, Input Miss Amount, Output Amount,
Total Tokens, Input Hit Tokens, Input Miss Tokens, Output Tokens,
Total audio duration, Request Count
```

**注意**：列顺序是 `Input Hit / Input Miss / Output`，不是 `Input / Output / Cache`。
之前分析把 `Input Miss` 和 `Output` 列搞反，导致错误结论"差一半"。

## 六、API 按量单价反推（用户账单验证）

08-06 mimo-v2.5-pro 行：

```
Consumed Amount = 10.157051
Input Hit  = 3.969626 / 158785024 × 1e6 = 0.025 ✓
Input Miss = 5.236731 / 1745577  × 1e6 = 3.0   ✓
Output     = 0.950694 / 158449   × 1e6 = 6.0   ✓
```

**与 Reasonix 代码表（`mimoDomesticPrices`）100% 一致**：

| 模型 | hit 价 | miss 价 | out 价 | 代码表 |
|---|---|---|---|---|
| mimo-v2.5 | 0.020 | 1.000 | 2.000 | ✅ 0.02/1/2 |
| mimo-v2.5-pro | 0.025 | 3.000 | 6.000 | ✅ 0.025/3/6 |

## 七、与 DeepSeek 对照（统一建模）

| 维度 | DeepSeek | MiMo |
|---|---|---|
| 计费单位 | 人民币直计 | API 按量：人民币；Token Plan：Credits |
| Credits 换算 | 无 | 1 Credit = 1e-8 元 |
| 模型分级 | flash/pro（2 级） | v2.5/pro（2 级） |
| hit/miss/out | flash 0.02/1/2、pro 0.025/3/6 | 与 DeepSeek 完全一致 |
| 缓存折扣 | hit = miss × 1/50 | 同 DeepSeek |

**关键洞察**：MiMo 与 DeepSeek 的价格结构惊人一致（flash↔v2.5、pro↔pro），
疑似国内 API 定价的市场锚定，非巧合。

## 八、Reasonix 意义

1. **代码表正确**：`mimoDomesticPrices`（0.02/1/2、0.025/3/6）无需修改。
2. **用量统计准确**：`Usage.CacheHitTokens / CacheMissTokens` × 价格表即可反映真实扣费。
3. **Credits 折算**：如需在 UI 显示 Credits，用 `费用 × 1e8` 即可；不需要单独建模。
4. **黑盒已拆解**：MiMo 页面显示 Credits（×1e8），实际扣费人民币，两者可互相换算。
