# 多智能体流水线验证：12 进制计算器（2026-08-11）

> 用满格预置矩阵跑"代码域流水线"（architect→coder→guard，coordinator 依赖编排）：
> architect 设计 → coder 实现 → guard 审查——依赖图 dependsOn 串行。

## 任务

12 进制通用基础计算器：多进制输入（2/8/10/12/16）、12 进制四则运算、任意基数格式转化。

## 流水线执行

| 阶段 | 智能体 | 依赖 | 产出 |
|---|---|---|---|
| 1. 设计 | arch（architect [0,2,2,2,1,1]） | — | calculator_spec.md（模块划分/接口/12 进制 digit-slice 定点选型） |
| 2. 实现 | code（coder [0,0,1,1,0,0]） | **depends:task-1** | calculator 包 6 模块（radix/arith/lexer/format/eval/cmd）+ 编译通过 |
| 3. 审查 | rev（guard [0,1,2,0,2,1]） | **depends:task-2** | review.md（发现 Add 位对齐 bug + 修复建议） |

**coordinator 编排**：/team-add 的 `depends:<id>` 依赖图命令面（本次新增）——
task-2 等 task-1 终态、task-3 等 task-2，串行流水线自动推进（P6.1 依赖树）。

## 功能验证（修复后全部正确）

```
calc -base 10 '16#FF + 2#1'   → 10#256   （多进制输入混合运算）
calc -base 10 '12#1B3 + 2#1'  → 10#280
calc -base 12 '10+10'         → 12#20    （12 进制加法）
calc -base 12 '21/5'          → 12#5     （12 进制除法）
calc -base 16 '12#10 * 12#10' → 16#90    （12 进制输入 → 16 进制输出）
```

## Guard 环节价值实证

审查发现 **P0 Add 位对齐 bug**（16#FF+2#1 错位）——短操作数未右对齐，
digit 加到最高位。修复后全对。这证明流水线"实现后必须审查"环节的必要性。

## 产物位置

- 设计：`.reasonix/worktrees/arch/calculator_spec.md`
- 实现：`.reasonix/worktrees/code/calculator/`（含修复）
- 审查：`.reasonix/worktrees/rev/review.md`
