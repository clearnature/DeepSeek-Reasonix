# 任务分类器与资源映射（docs/team/presets/task-classifier.md）

> 15 智能体 #1/#10 决策综合：识别任务类型与复杂度，细粒度分配资源。

## 一、分类器（基座：internal/taskintent）

**现有基座**（已冻结导出面）：
- 意图分类（5 类）：simple / write / research 等
- Goal 预算：simple=10 轮 / write=20 轮 / research=40 轮

**P13 扩展**：复杂度分档（L1-L5）——在意图之上叠加：

| 档 | 特征（关键词/规模/引用） | 默认配置 |
|---|---|---|
| L1 简单 | 单文件/查询/格式化 | 1 teammate · effort low |
| L2 中等 | 单模块实现/测试 | 1-2 teammate · effort medium |
| L3 复杂 | 多文件/跨模块/需要设计 | 2-3 teammate · effort high |
| L4 极复杂 | 架构/性能/并行 | 3-5 teammate · effort high |
| L5 战略 | 全仓/多团队 | 5+ teammate · effort high |

## 二、资源映射（复杂度 → teammate 数/effort/模型/超时）

| 档 | teammate 数 | effort | 模型 | worktree | 超时 |
|---|---|---|---|---|---|
| L1 | 1 | low | flash（快） | 否 | 5m |
| L2 | 1-2 | medium | flash | 可选 | 15m |
| L3 | 2-3 | high | pro（如配置） | 是 | 30m |
| L4 | 3-5 | high | pro | 是 | 60m |
| L5 | 5+ | high | pro | 是 | 120m |

**成本红线**：L4-L5 全 pro + high 成本线性叠加（devil's advocate 警告）——缓解：
1. L1-L2 一律 flash/low
2. 同类型任务复用 teammate（前缀缓存命中）
3. 并发按复杂度动态（D2），默认 ≤6，60 为 roadmap

## 三、动态调度流程

```
任务文本 → taskintent 分类（意图+复杂度档）
→ 资源映射（teammate 数/effort/模型/worktree）
→ /team-spawn N <prefix>（按需创建）
→ /team-add ×N（并行分配）
→ 监控（/team-status 轮询）→ 停滞检测（P10）→ 完成评估（衡量标准）
```

## 四、预置引用

分类器输出的"角色"直接对应预置库（analyst/perf/algo/fixer/guard）：
`/team-create <name>@<preset>` → 按预置的 tools/effort/worktree 配置生成 teammate。
