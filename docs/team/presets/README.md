# Reasonix Team 智能体预置库（docs/team/presets/）

> 临时可复用的智能体配置——**不内置系统智能体**，用完解散（/team-remove）。
> 格式：YAML frontmatter + Markdown body（复用 Skill 文件惯例，零新解析）。
> 加载：`/team-create <name>@<preset>`（预置引用，详见集成设计）。

## 目录

| 文件 | 预置 | 用途 |
|---|---|---|
| `analyst.md` | `analyst` | 数据分析师：数据清洗/统计/结论 |
| `perf-engineer.md` | `perf` | 性能工程师：基准/profile/瓶颈定位 |
| `algorithm-engineer.md` | `algo` | 算法工程师：设计/复杂度分析/优化 |
| `concurrency-guard.md` | `guard` | 并发守护：race 审计/锁序/TOCTOU |
| `bug-fixer.md` | `fixer` | Bug 修复专家（技能转化示例） |
| `task-classifier.md` | — | 任务分类器 + 复杂度→资源映射（系统文档） |
| `ga-iteration.md` | — | 遗传算法迭代框架（系统文档） |
| `metrics.md` | — | 衡量标准体系（系统文档） |

## 预置 frontmatter 契约

```yaml
---
name: <短名>
description: 一句话用途
role: <角色标签>
prompt: |-
  <系统提示词（方法论）——首 fork 时替换 teammate system>
tools: [bash, read_file, grep, ...]   # 工具白名单
effort: low|medium|high               # 模型 effort
writable: false                       # 默认只读
worktree: false                       # true = 分支并行模式
model: ""                             # 空 = 跟随 leader（模型无关）
---
```

## 原则

1. **不内置**：预置只是 docs/team/ 下的数据文件，不进系统智能体/工具注册表
2. **模型无关**：model 空 = 跟随 leader 接入的任何模型
3. **前缀稳定**：预置 prompt 首 fork 时一次性替换 system（父缓存 miss 一次，之后字节稳定）；批量 spawn 同预置摊薄
4. **用完解散**：/team-remove 清理（含 worktree）
5. **进化复用**：任务评估 → 优胜配置更新预置版本 → 下代复用（见 ga-iteration.md）
