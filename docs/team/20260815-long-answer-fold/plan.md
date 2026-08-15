# 立项：超长回答折叠预览（long-answer fold preview）

**日期**：2026-08-15 · **分支**：dev/develop · **状态**：已批准（用户）· **类型**：前端渲染性能根治

## 问题

transcript 向上滚动卡顿（#8200/#8845 域）——虚拟化向上滚动每帧挂载视口上方行，
巨型 markdown 回答（数千-数万 px）每帧重建 DOM（hastBlockToJsx→DOM→layout→paint）
→ 掉帧。markdown 解析有缓存，但 DOM 重建无缓存——卡顿主因。

## 方案 A1（已选）

**Virtuoso 行级折叠**：助手超长回答（>2000 字符）默认折叠为紧凑预览（纯文本截断 +
展开按钮），展开/收起走既有 foldMapWithToggle 语义。

- 折叠态行高确定性（固定预览高度）→ Virtuoso 测量稳定
- DOM 节点数骤减（O(markdown 全树) → O(preview)）
- 复用先例：MarkdownTable（>50 行折叠）、ProcessFoldHeader（关闭态零子树）、
  useCollapseAnimation（高度动画）

## 任务（T0-T7）

| T | 内容 | 验证 |
|---|---|---|
| T0 | 立项落盘 | 本文件 |
| T1 | LONG_ANSWER_FOLD_THRESHOLD_CHARS=2000 + fold 状态 | 阈值边界单测 |
| T2 | 行模型扩展（折叠态行 + renderRow 分支） | typecheck + 行单测 |
| T3 | 展开按钮 + aria-expanded/aria-controls | a11y 断言 |
| T4 | 滚动对接（row-size owner + isPinned） | scroll 测试绿 |
| T5 | 折叠信号（uiPerfSignals） | uiPerf 单测 |
| T6 | bench/transcript-long-answer-fold.mjs + 对照实验 | 真实 Chromium 阈值 |
| T7 | 全量测试 + bench 回归 + repolint + 编译 | 全绿 |

## 性能标准（bench 阈值沿用）

向上滚动 frame-gap P95 ≤ 80ms / max ≤ 250ms / longtask ≤ 250ms 单任务；
折叠后 DOM 节点数对照实验（≥2× 降幅期望）。

## 纪律

- 发送侧零变化（纯前端）——Cache-impact: none
- 折叠状态不跨会话持久化（本轮范围）
- 展开按钮显眼可逆（防误读截断——Claude Code SHOW LESS 教训）
