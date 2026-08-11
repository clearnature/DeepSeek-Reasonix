# 研究域流水线验证（2026-08-12）

> 满格矩阵第三域验证：researcher → validator → reporter（dependsOn 依赖链）。
> 至此三域矩阵全部验证完成（代码/创作/研究）。

## 任务

研究"LLM 前缀缓存（prompt caching）的机制与成本模型"→ 验证 → 产出最终报告。

## 流水线执行（本次全自动，无超时）

| 阶段 | 智能体（坐标） | 依赖 | 产出 |
|---|---|---|---|
| 研究 | res（researcher [2,2,1,0,1,1]） | — | findings.md（13.3KB） |
| 验证 | val（validator [2,1,2,0,1,1]） | depends:task-1 | validation.md（12.8KB） |
| 报告 | rep（reporter [2,0,0,1,0,2]） | depends:task-2 | report.md（11.7KB，35s 完成） |

**执行时间**：res ~3.5min → val ~4min → rep ~35s——三级自动推进（P6.1 依赖门控）。

## 环节价值实证

**researcher**：缓存机制（KV 前缀匹配/billable 公式）、三家对比表
（DeepSeek/Anthropic/OpenAI 触发/粒度/写入费/TTL）、**引用 Reasonix 项目实测
数据**（telemetry 08-07~10）、不确定性诚实标注。

**validator**（关键）：**web_fetch 官方文档实证核查**（DeepSeek kv_cache 文档/
OpenAI 公告）——每条断言 ✅/⚠️ 分级 + 官方原文证据；**发现 3 个 ⚠️ 修正**：
①64-token 单元是历史口径（v4 文档以"缓存前缀单元完整匹配"为准）②v4-flash
实测下限 ~256 tokens（项目代码 fork_cache_real_test.go:42 引用）③"逐位匹配"
是简化（官方"逐单元"）。成本模型独立重算通过。

**reporter**：综合 findings+validation 定稿（结论→证据→存疑→行动建议）。

## 三域矩阵验证完成

| 域 | 流水线 | 评审/验证环节抓到 |
|---|---|---|
| 代码 | architect→coder→guard | P0 Add 位对齐 bug |
| 创作 | creator→critic→copywriter | P0 韵脚出韵（重=二冬≠一东） |
| 研究 | researcher→validator→reporter | 3 处 ⚠️ 口径修正（64-token/256-token/逐单元） |

三域验证环节全部抓到创作者/研究者的真实错误——**"执行后必须审查"跨域成立**。

## 已知缺陷（probe）

- 产出在 `Ctrl.Close` 时被 worktree 清理删除（teammate removed → worktree removed）；
  verify 确认了产出存在（13.3/12.8/11.7KB）但未先复制到 /tmp。
- 修复：probe verify 应在 Close 前复制产出（下次执行）。
