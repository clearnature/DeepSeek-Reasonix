# 摘要调用开销闭环（2026-08-13 数据实证）

> 与 `docs/research/compaction-evolution.md` §六b 同步的决策记录。

## 结论（永久事实）

1. **summarize 低命中 ≠ prefix 漂移**。`pref_hash` 三样本（10:26/10:30/11:30）完全一致（`8ff12faef6b0`）——prefix 字节稳定。
2. **低命中 = fold 区（从未发送的新内容）的必然 miss**——命中率差异的结构归因：全量折叠（fold=165K）3.3% vs 增量折叠（fold≈0）99.4%。
3. **成本规律**：miss 成本集中在"重放→全量折叠"事件（8/13：07:00 ¥0.97、10:00 ¥0.56）；稳态增量折叠 ≈ 0（09:00 ¥0.02）。
4. **根因**：重放后投影失效（版本计数漂移 + 尚未 append → `projectionContentValid` 的 `n < len(msgs)` 分支拒绝）→ 首轮折叠走全量。
5. **修复（6158a6317）**：covered 前缀哈希已验证后，版本漂移不再单独使投影失效（哈希 fail-closed 保持——内容改写仍失配重算）。

## 验证样本（修复后）

```
11:30:16 trigger=pressure mode=summarized status=installed cache=warm
src=290,056 fold=21,583（增量） spans=1 proj=274,729
in=34,968 hit=13,184 miss=21,784 write=0
user_kept=131 user_dropped=0 pref_hash=8ff12faef6b0
```

- fold 165K→21.6K（↓7.6 倍）；prefix 命中 98.5%；miss 成本 ¥0.18→¥0.022/次（↓8 倍）；零丢弃、零重写。

## 诊断方法（可复用）

1. 先 `pref_hash` 自对比（两次压缩）——排除/确认漂移；
2. 再按 fold 大小归因：fold ≈ miss → "新增内容必然 miss"，非缺陷；
3. 服务端 amount 数据（按小时）验证整体命中率与成本分布——本地遥测与服务端统计一致（8/13 全天 97.9% 命中，miss 主体 = 摘要+写放大）。

## 提交链

- `739253b0f` pref_hash 指纹遥测（诊断 1）
- `7b490a1fd` ②b resume 超窗首轮先压缩
- `6158a6317` 投影跨版本漂移保持有效（方案 A）
- `521eb2bf2` compaction-evolution.md §六b
