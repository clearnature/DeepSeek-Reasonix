# 研究域流水线验证（2026-08-12）

> 满格矩阵第三域：researcher → validator → reporter（dependsOn 依赖链）。
> Probe 缺陷修复后重跑（产物在 ctrl.Close 前复制到 /tmp，worktree 清理不丢产出）。

## 任务

研究"LLM 前缀缓存（prompt caching）机制与成本模型"——官方文档 + 项目实测交叉。

## 流水线执行

| 阶段 | 智能体（坐标） | 依赖 | 产出 |
|---|---|---|---|
| 研究 | res（researcher [2,2,1,0,1,1]） | — | findings.md（12.5KB：机制/成本/三家对比/项目实测） |
| 验证 | val（validator [2,1,2,0,1,1]） | depends:task-1 | validation.md（9.7KB：逐条 ✅/⚠️/❌ + 官方文档在线实取比对） |
| 报告 | rep（reporter [2,0,0,1,0,2]） | depends:task-2 | report.md（8.5KB：结论→证据→存疑→行动建议） |

## 环节价值实证（三域呼应）

**researcher**：产出机制+成本模型（三家对比表 + 引用项目 telemetry 实测）。
**validator**（关键）：web_fetch 官方文档逐条实证（kv_cache 指南/pricing 页），
**抓到 researcher 3 类错误**：S1 算术错误 1000 倍（100K 全 miss $14 实为 100M 口径）、
S2 写放大方向矛盾（hit 0.3-1.3% vs 4 节反向）、S3 memory 源文 hit/miss 误写。
**reporter**：综合定稿——6 条已证实结论 + 证据表（代码行级）+ S1-S7 存疑标注 +
P0-P3 行动建议（10 条实证驱动）。

## 三域对照（满格矩阵验证完成）

| 域 | 流水线 | 评审/验证环节抓到 |
|---|---|---|
| 代码域 | architect→coder→guard | P0 Add 位对齐 bug（16#FF+2#1 错位） |
| 创作域 | creator→critic→copywriter | P0 韵脚出韵（重=二冬≠一东） |
| 研究域 | researcher→validator→reporter | S1 算术 1000 倍 + S2 方向矛盾 + S3 源文错误 |

三个域的**审查环节全部抓到创作者的真实错误**——"执行后必须独立验证"跨域成立。

## Probe 缺陷修复

- 缺陷：probe 在 ctrl.Close() 后才打印验证——Close 移除 teammate 并清理 worktree，
  产物丢失（上次研究域 3 文件全没）。
- 修复：verifyAndCopy 在 Close 前把 3 产物复制到 /tmp/team-research-final/。
- 验证：重跑后 /tmp/team-research-final/ 3 文件完整（findings 12.5KB/validation 9.7KB/report 8.5KB）。

## 完整产出存档

/tmp/team-research-final/（findings.md + validation.md + report.md）
