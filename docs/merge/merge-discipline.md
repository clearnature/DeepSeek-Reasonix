# 合并纪律（Merge Discipline）

> 沉淀自 2026-08-04 → 08-14 的合并事故链（#7168 割裂误删共享基础 / 自动合并 format 语义丢失 / 7488 整包 checkout 污染 / PR #7566 gofmt 三连败 / merge 遥测静默丢失 / 0 文本冲突≠0 语义风险）。
> 每次合并（dev 同步上游、PR 分支更新）必须执行，不可跳过。

## 0. 前置风险分析（合并前必做——team-planner）

合并前必须使用 **team-planner 技能**做前置风险分析，建立风险识别防线：

**风险识别领域（四域）**：
1. **内核安全**——agent/provider 内核（编译、语义、并发、请求路径）
2. **压缩主题**——compact/投影/估算/触发（我们的核心修复域）
3. **缓存主题**——前缀字节稳定 / 命中率 / 写放大
4. **统计准确性**——stats/遥测字段落盘（四方审计）

**三级别风险预警**：
- 🟢 **低（绿）**：无上述四域重叠 / 无内核文件改动 → 常规验证链（§6）
- 🟡 **中（黄）**：同文件双改（非核心）/ docs / 前端 → 定向核对 + 相关守卫测试
- 🔴 **高（红）**：压缩/缓存/统计/内核**同域同文件** → team-planner 深入分析 + 守卫套件全量 + 遥测四方审计（§8）+ 风险文档归档

**产出**：风险等级 + 核对点清单 → 归档 `docs/merge/YYYYMMDD-<版本>-merge-risk.md`（先读本目录已有分析，按主题矩阵核对点执行）。

## 1. 合并前侦察

```bash
git rev-list --count HEAD..origin/main-v2          # 落后量
git log --oneline HEAD..origin/main-v2             # 上游提交列表（按主题分组）
git merge-base dev/clearnature main-v2             # 共同祖先
git merge-tree --write-tree --name-only dev main-v2 | grep CONFLICT   # dry-run 冲突面
# 对每个潜在冲突文件：
git log --oneline HEAD..origin/main-v2 -- <file>   # 上游是否改过它
comm -23 <(git grep -o 'func [A-Za-z]*' <ours>) <(...)   # 函数级超集对比（判断哪边是超集）
```

## 2. 逐块精确处理（绝不整文件/整包 checkout）

- **禁止**：`git checkout <other-branch> -- <file/dir>`（整文件覆盖会带入对方分支的无关演进）、`git merge <branch>` 到 PR 分支、`git checkout <commit> -- .` 探测历史。
- **正确**：`awk`/`sed` 打印每个冲突块 → 判断 ours/theirs → `edit_file` 逐块解决（只补主题相关行）。
- **铁律**："7488 + 基线 ≠ dev"——dev 还有后续开发，整包 checkout 会拉入无关优化（如 max_output_tokens 抖动、注释改写）。

## 3. 自动合并语义丢失复查

git 标"自动合并成功"的文件也可能保留旧版语义（曾：自动合并保留 dev 旧版 `runRefTurn`（无 format 绑定），丢上游 `runRefTurnWithFormat`，测试失败才暴露）。核心功能文件（controller.go / run_loop.go / responses.go / compact*.go / stats）merge 后：

```bash
git diff origin/main-v2 -- <file>   # 复查——确认没保留过时实现、没丢新修复
```

## 4. 冲突方向确认再 edit

`<<<<<<< HEAD` 侧**未必是正确版本**（曾把 gofmt 缩进冲突解决反）。先对比哪边是新修复（版本、语义、host 匹配），再决定取舍。

## 5. 生成文件三件套

```bash
git checkout --theirs   # 清冲突标记
go run ./cmd/remote-protocol-gen   # 重新生成
go run ./cmd/remote-protocol-gen -check   # 验证
```
不要手改 schema/generated 文件。

## 6. 合并后验证链（顺序固定）

1. `gofmt -l .` 全量（CI 同款；排除 desktop/.direnv/本地实验目录）
2. `go build ./...`
3. 核心包测试（provider / agent / control / config / tool / protocol）
4. `remote-protocol-gen -check`
5. `go run ./tools/repolint`（看完整退出码，不能 `grep -c "over its"` 计数——会漏 file-size 违规）
6. **desktop 边界**（merge 后必跑——前端 breakage 只在这里暴露）：
   ```bash
   cd desktop/frontend && ./node_modules/.bin/tsc --noEmit
   cd desktop && CI=true ~/go/bin/wails build -tags webkit2_41 -ldflags "-X main.version=$(date +%Y%m%d-%H%M)"
   ```

## 7. 隐藏产物排查清单

- 重复声明（`grep -c 'var xxx'`——合并可能带两遍）
- 多余括号 / 丢失缩进（编译错误行号定位 / `gofmt -d` 看 diff）
- `validate`/`omitempty` 类语义差异（对照 dev 已修复版本 `git show dev/clearnature:<file>`）

## 8. 遥测四方审计（合并涉及遥测文件后必做）

涉及 `internal/agent/*telemetry*.go`、`internal/stats/*` 的合并：

```
CompactionTelemetry 结构 ↔ emit 的 detail 键 ↔ recordCompaction/setCompactionInt 解析 ↔ CompactionRecord JSON tag
```
**任一环缺 = 断链，一次补完（禁止零敲碎打）**。失败路径必须落盘（`status=failed` + `err_type=`，禁止 return 吞 notice）。历史教训：PrefixHash / CompactionRecord / Est / Results / SavedChars / ElapsedMs 曾静默丢失——**编译通过 ≠ 无丢失**，必须行为级核对（跑 FastCompress / QueryCompactions 测试 + 实际运行验证 stats 落盘字段）。详见 `docs/telemetry-audit-20260813.md`。

## 9. 分支纪律

- `main-v2` = 只读镜像：只 `git fetch origin` + `git merge --ff-only origin/main-v2`，**绝不在上面提交**。
- `dev/clearnature` = 唯一本地开发分支（文档/决策/代码都在这）。
- PR 分支更新：上游基线重建 + 逐个 cherry-pick 主题 commit（一次一个，冲突逐个解决）；**禁止 merge dev 全量**。
- merge 优先于 rebase：rebase 逐提交重放产生伪冲突（上游没改的文件也被标记）；merge 一次性解决。

## 10. 0 文本冲突 ≠ 0 语义风险

`git merge-tree` 报 0 冲突不代表安全——**同文件双方都改 = 逐字段核对**（主题 B 类：同域修复合并后跑守卫测试套件确认双修复并存）。合并前先读 `docs/merge/` 的风险分析（按主题矩阵核对点执行）。
