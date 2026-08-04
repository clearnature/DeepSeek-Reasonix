# 深挖报告 A：默认注入家族根因链（#7357/#7358/#4814/#7273）

## 核心结论
"默认注入覆盖用户配置"不是一次性 bug，而是 **7 个函数组成的注入体系**
自 2026-05-29（7de6a2474）起逐层叠加，每个函数都有"官方判定→覆盖"模式。

## 函数 → 引入提交 → 原始意图 → 行为

| 函数 | 引入提交 | 意图 | 尊重用户？ |
|---|---|---|---|
| backfillDeepSeekPro | e25fcffa2 (#2905) | 旧 wizard 补 pro 条目 | 用户显式模型列表跳过 |
| DeepSeekOfficialPricingLanguage 硬编码 zh | 1a87cd4cf (#4329) | 官方定价默认 CNY | **❌ 无条件（#4814 根源）** |
| applyDeepSeekOfficialDefaultPricing | 1a87cd4cf (#4329) | 启动刷新官方价 | isKnownOfficialPricing 守卫 |
| backfillOfficialContextWindow | fc5fe53e2 (GTC2080) | 官方回填窗口 | ContextWindow<=0 才填 |
| DeepSeekOfficialPricingCurrency/provenance | 2fefeef06 (#6945) | 区域 CNY/USD + TOML 来源保留 | **✅ 修复 #4814 主体** |
| mergeFileSnapshotWithRead 清空默认 | 2db41eb95 (#7357) | 用户声明 providers 整体替换 | **✅ 修复 #7357/#7358** |
| hasExplicitSupportedEfforts | e36b6aa80/12fdaa830 (#7328/#7273) | effort 白名单逃生口 | **✅ 修复 #7273（上游已合）** |

## 演变全景

```
7de6a2474 (05-29)  初始：Default()+decode 按 index 合并——【架构缺陷种子】
1a87cd4cf (06-16)  RMB 硬编码——#4814 根源（用户 USD 被刷成 CNY）
417366028 (05-30)  balance_url 加到默认 provider——继承面扩大（隐私外呼）
fc5fe53e2 (06-18)  context_window backfill——#7358 关联
2fefeef06 (07-28)  provenance 机制——#4814 主体修复（上游）
2db41eb95 (08-04)  清空默认——#7357/#7358 修复（本地，已合入 dev e2e2dae0c）
```

## 关键教训
1. 每个"官方默认注入"都必须有"用户已声明则跳过"的守卫——不是"注入后覆盖"
2. 修复模式统一：**声明即替换**（providers 数组）+ **来源保留**（provenance）+ **白名单逃生口**（effort）
3. #4814 残余：标准模板+只改价格组合仍会被 locale 刷新（markPersistedDeepSeekOfficialPricing 不查 Price）
