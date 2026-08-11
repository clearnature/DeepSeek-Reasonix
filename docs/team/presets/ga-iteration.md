# 遗传算法迭代框架（docs/team/presets/ga-iteration.md）

> 15 智能体 #3 决策：把 10 轮强化迭代（R1-R10）升级为遗传算法驱动的进化循环。

## 一、染色体编码（智能体配置模板）

染色体 = 可变参数集合（每个"基因"一个可调参数）：

| 基因 | 值域 | 作用 |
|---|---|---|
| prompt | 预置 prompt 变体 | 任务方法论 |
| tools | 工具白名单子集 | 能力范围 |
| effort | low/medium/high | 推理深度 |
| teammate 数 | 1..N | 并行度 |
| worktree | on/off | 写隔离 |
| 并发上限 | 1..32 | 调度 |

## 二、适应度函数（衡量标准加权）

```
fitness = w1·成功率 + w2·(1/耗时) + w3·(1/token成本) + w4·前缀命中率
权重建议：成功率 0.4 / 耗时 0.2 / 成本 0.2 / 命中 0.2（可调）
```

## 三、算子

| 算子 | 操作 | 约束 |
|---|---|---|
| 选择 | 锦标赛（k=2）：适应度排序取优 | 保留精英（top 20%） |
| 交叉 | 两配置交换基因（如 prompt×tools） | 不破坏 frontmatter 契约 |
| 变异 | 单基因随机扰动（如 effort low↔medium） | 值域内；前缀稳定红线 |

## 四、种群与代数（与 R1-R10 结合）

- **种群 N=3-4**（小种群——真实 API 成本红线）
- 免费 mock 前置（先 mock 验证机制，再真实 API 验证效果）
- **仅 3 代全量真实 API**（每代 = 一轮迭代）
- 淘汰抑制清单：前缀字节变化、竞态/死锁、成功率下降

## 五、进化闭环（与 15 智能体 #14 结合）

```
任务 → 评估（适应度）→ 选择/交叉/变异 → 新配置
→ 预置版本化（preset vN+1，parent+mutation_source 标注）
→ 下代复用（配置 hash == manifest.current）
```

## 六、防虚假完成自检

| 检查 | 判据 |
|---|---|
| 每任务必有成绩 | results.jsonl 行数 == 任务次数 |
| 版本必有成因 | vN+1 标注 parent + mutation_source |
| 优胜必有证据 | score_total 与父代 delta 记录 |
| 缓存红线 | 每基因落地后 golden 字节对比 + prefix_hash 新基线 |
| 复用确实发生 | 下轮 teammate 配置 hash == manifest.current |
