# 任务分类器（docs/team/presets/task-classifier.md）

> 15 智能体 #1/#10 决策 + devil's advocate 修正（A2）：**不写规则表/关键词词典**——分类本身就是 LLM 擅长的，用 runAs:subagent skill 输出类型+复杂度（3 档），避免伪精度死代码。

## 一、分类器设计（LLM skill，非规则系统）

```yaml
---
name: task-classifier
description: 任务分类——识别类型 + 复杂度 3 档，输出资源建议
runAs: subagent
allowed-tools: []          # 纯推理，无工具
effort: low
---
你是任务分类器。给定任务文本，输出：
type: write | bugfix | review | algorithm | data-analysis | creative | research | other
complexity: simple | complex | very-complex   # 3 档足够
resources: {teammates: 1|2|3-5, effort: low|medium|high, worktree: bool}
成本红线：L1-L2 一律 flash/low；并发默认 ≤6。
```

**复杂度 3 档**（替代原 5 档）：
- `simple`：单文件/查询/格式化 → 1 teammate · low
- `complex`：单模块/跨文件 → 2-3 teammate · high
- `very-complex`：架构/性能/多域 → 3-5 teammate · high · worktree

## 二、资源映射（复杂度 → teammate 数/effort/worktree）

| 档 | teammate | effort | worktree | 超时 |
|---|---|---|---|---|
| simple | 1 | low | 否 | 5m |
| complex | 2-3 | high | 是 | 30m |
| very-complex | 3-5 | high | 是 | 60m |

## 三、动态调度流程

```
任务文本 → task-classifier skill（LLM 分类：type + complexity 3 档）
→ 资源映射（teammate 数/effort/worktree）
→ /team-spawn N <prefix> + /team-add ×N（并行）
→ 监控/停滞检测（P10）→ 完成评估（metrics.md）
```

## 四、预置引用

分类器输出角色 → 预置库（analyst/perf/algo/fixer）：
`/team-create <name>@<preset>`。
