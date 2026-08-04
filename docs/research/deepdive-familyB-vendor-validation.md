# 深挖报告 B：vendor 硬校验家族根因链（#7451/#4099/#3561/#7273/#4711/#7168）

## 核心结论
**两层不对称**是家族总根因：配置层对未知端点 fail-closed（拒配），
协议层对同一端点 fail-open（接受）——#7451（TokenHub）是最新实证。

## 判定链（effort.go NormalizeEffort 8 级优先级）

| 优先级 | 依据 | 类型 | 对未知端点 |
|---|---|---|---|
| 1-2 | reasoning_protocol / supported_efforts | 用户声明 | 有声明才通过 |
| 3 | 模型能力表（硬编码 flash/pro） | 硬编码 | ❌ 不认 |
| 4-7 | protocol 推断 / host 匹配（miniMax/zhipu/longcat/ollama） | host 匹配 | ❌ 不认 |
| **8** | **default → fail-closed（不支持）** | 兜底 | **❌ 拒绝（#7451 根因）** |

## 协议层对照（openai.go New() effort 校验）

| 分支 | 硬校验 | 逃生口 |
|---|---|---|
| deepseek | low/high/max/disabled | ✅ hasExplicitSupportedEfforts（#7273 修） |
| minimax | adaptive/disabled | ❌ 无 |
| zhipu/longcat | enabled/disabled | ❌ 无 |
| ollamaCloud | none..max | ❌ 无 |
| **generic（未知）** | **low/medium/high** | ✅ 有 |

## #7451 错误传播链（TokenHub）

```
task.go:821 resolveSubSessionRuntime 失败
→ boot.go:796 NormalizeEffort → effort.go:246 default → effortNotConfigurableError
→ boot.go:798 无 ProviderResolver → 报错
→ 但协议层 openai.go:206 generic 分支接受 low/medium/high（已实测 TokenHub 支持）
结论：配置层 default 分支应对 kind=openai 未知端点回退 generic 词汇
```

## 严格校验阶梯（#4711 name / #7168 summary）

| 校验 | 修复 | 模式 |
|---|---|---|
| name 空值 400（MiMo） | c7cbed768 *string 条件序列化 | 角色判断 |
| summary 缺 400（DashScope） | 9bbc0b819 能力表 + b0c1d8cb52 | 能力表 |
| effort 词汇 | e36b6aa80 白名单 | 逃生口 |

## 未闭环缺口
1. **#7451**（open）：配置层 default fail-closed——修复方向：kind=openai 未知端点回退 generic OpenAI 词汇
2. minimax/zhipu/longcat 协议层分支无逃生口（#4099/#3561 残余）
