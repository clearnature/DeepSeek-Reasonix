# 合并纪律（Merge Discipline）

> 沉淀自 2026-08-04 → 08-14 的合并事故链（#7168 割裂误删共享基础 / 自动合并 format 语义丢失 / 7488 整包 checkout 污染 / PR #7566 gofmt 三连败 / merge 遥测静默丢失 / 0 文本冲突≠0 语义风险）。
> 每次合并（dev 同步上游、PR 分支更新）必须执行，不可跳过。

## 0. 前置风险分析（合并前必做——team-planner）

合并前必须使用 **team-planner 技能**做前置风险分析，建立风险识别防线：

**风险识别领域（四域）**——**上下文压缩与缓存 = 第一风险（优先级最高）**：

| 优先级 | 领域 | 破坏后果 |
|---|---|---|
| ★ **第一风险** | **上下文压缩与缓存**（compact/投影/估算/触发/前缀字节稳定/命中率/写放大——我们的核心价值与成本优势之源） | 缓存打穿 → 全价付费；压缩误触发 → 数据丢失；前缀漂移 → 成本暴增 |
| 2 | **内核安全**（agent/provider 内核：编译、语义、并发、请求路径） | 崩溃 / 语义错误 / 死锁 |
| 3 | **统计准确性**（stats/遥测字段落盘——四方审计） | 诊断失效（8/13 教训：失败不落盘=盲修） |
| 4 | 前端 / docs / 其余 | 体验 / 文档 |

**三级别风险预警**：
- 🟢 **低（绿）**：不触压缩/缓存/内核 / 无同文件双改 → 常规验证链（§8）
- 🟡 **中（黄）**：同文件双改（非压缩缓存）/ docs / 前端 → 定向核对 + 相关守卫测试
- 🔴 **高（红）**：**触及压缩或缓存主题**（同域同文件）→ **必红**——team-planner 深入分析 + 守卫套件全量（cachehit_e2e / healthy-window / RejectsExhausted / 投影/校准测试）+ 遥测四方审计（§10）+ 风险文档归档

**产出**：风险等级 + 核对点清单 → 归档 `docs/merge/YYYYMMDD-<版本>-merge-risk.md`（先读本目录已有分析，按主题矩阵核对点执行）。

## 1. 主题合并模式（代替版本合并）

**核心原则：以主题合并代替版本合并**——不一次性 merge 整个版本（风险面大），按主题逐项评估、逐项决定。

**主题合并流程**（每个主题独立执行）：
1. **主题合并历史追溯**：`git log --oneline -- <主题文件>`（上游 + dev 双方历史——识别长期冲突模式 / 重复修复 / 谁先谁后）
2. **issue/PR 社区追踪**：该主题的社区 issue/PR 状态（已修复？仍复现？——hold 判断依据，如 v1.25.0 的 #8739/#8741 是否被 v1.25.1 解决）
3. **风险等级**（第 0 条四域识别 + 三级别预警）
4. **合并决策**：
   - ✅ 立即合并（低风险 / 社区已确认修复 / 与 dev 无重叠）
   - ⏸ **推迟合并**：新架构和功能合并（不急需——等稳定）；上游剧烈变动（等社区验证沉淀）
   - ⛔ 不合并（hold——社区 bug 未清 / 与我们的修复冲突）
5. 每主题独立执行验证链（§7）+ 遥测四方审计（§9）

**版本合并 vs 主题合并**：
- 版本合并（旧）：一次吞整个版本（26 提交）——红级风险面大、hold 无法局部化（坏主题拖累好主题）
- 主题合并（新）：按主题独立评估/决策/验证——**hold 可局部化**（坏主题 hold、好主题吸收）、风险可控、追溯清晰

## 2. 本地功能保护识别 + 合并风险丢失交叉分析

**目的**：dev 的 508 提交成果（本地功能）不被上游合并覆盖/丢失。

**本地功能保护清单**（合并前盘点——常驻）：
| 本地功能 | 所在文件（代表） | 守护测试 |
|---|---|---|
| est 实测优先（f105a0454） | output_budget.go / sampling_request.go | RejectsExhaustedSharedWindow |
| fork 继承（27d6cb0f5） | subagent_fork.go / task.go | TestCaptureForkInheritance |
| 投影语义哈希（a37f7c054/44b827e9c） | projection.go / compact_commit.go | TestProjectionContentValidToleratesPruneRewrite |
| summarize 预算（c35e7569f） | compact_fold_input.go | TestSummaryBudgetAccountsForPrefix |
| 失败落盘 + elapsed_ms | context_receipt.go / compact_projection.go | TestCompactionTelemetrySourceTokens |
| 遥测字段（status/tpc/reason/prefix_hash/est/results/saved_chars/action） | stats/record.go + recorder.go | 四方审计（§9） |
| P6 team | teammate_store.go / controller.go | team 测试套件 |
| 检索系统（web_search/知识缓存） | internal/provider/responses/ | retrieve 测试 |
| 校准持久化 | calibration.go | 校准测试 |
| resume 遥测（C1 门控） | budget.go | resume 测试 |

**合并风险丢失交叉分析**（方法）：
1. `comm -23 <(git ls-tree dev --name-only) <(git ls-tree main-v2 --name-only)` —— **dev 独有文件**（上游没改 = 安全区）
2. **上游也改的文件 ∩ 本地保护功能所在文件 = 红区**（丢失风险点——合并后逐个核对）
3. 红区核对：`git diff <保护基准分支> -- <file>` 逐字段 + 跑守护测试 + 遥测四方审计

**保护执行**：
- 合并前：保护清单 + 交叉矩阵（红区列表——并入主题风险文档）
- 合并后：红区逐个核对（守护测试 + 字段对照）——**编译通过 ≠ 无丢失**（历史教训：PrefixHash / Est / Results / SavedChars / ElapsedMs 静默丢失，靠测试暴露——见 §10）

### 2.1 本地修复 vs 社区修补——差别识别（每个主题合并前必做）

对每个主题（尤其 🟡/🔴 级），识别**本地已完成修复**与**社区修补方向**，做三方差别对比（本地 / 社区 / 基线）：

| 差别类型 | 判定 | 合并策略 |
|---|---|---|
| **同向互补** | 本地与上游补不同缺口（如：本地 est 实测优先 + 上游 web_search 排除） | ✅ 合并后**双修复共存**——守卫测试确认两方都在 |
| **同向重复** | 本地已覆盖上游的修复（相同语义） | ⚪ 上游冗余——可跳过；吸收则确认不破坏本地 |
| **方向冲突** | 语义相反（本地容忍 vs 上游严格 / 本地保留 vs 上游删除） | 🔴 红——team-planner 深入分析 + 守卫全量（如：投影版本漂移——本地容忍 vs 上游门控） |
| **上游更完整** | 上游覆盖本地且更多 | 🔶 评估替代——保护清单逐字段核对（遥测字段/行为级），缺一不换 |

**判定方法**：三方对比 `git show dev:<file>` / `git show main-v2:<file>` / 基线——函数级超集对比（`comm` 函数列表）+ 逐字段核对。**差别识别结论并入主题风险文档**（每主题标注：本地已修什么 / 社区修什么 / 差别类型 / 合并策略）。

## 3. 合并前侦察

```bash
git rev-list --count HEAD..origin/main-v2          # 落后量
git log --oneline HEAD..origin/main-v2             # 上游提交列表（按主题分组）
git merge-base dev/clearnature main-v2             # 共同祖先
git merge-tree --write-tree --name-only dev main-v2 | grep CONFLICT   # dry-run 冲突面
# 对每个潜在冲突文件：
git log --oneline HEAD..origin/main-v2 -- <file>   # 上游是否改过它
comm -23 <(git grep -o 'func [A-Za-z]*' <ours>) <(...)   # 函数级超集对比（判断哪边是超集）
```

## 4. 逐块精确处理（绝不整文件/整包 checkout）

- **禁止**：`git checkout <other-branch> -- <file/dir>`（整文件覆盖会带入对方分支的无关演进）、`git merge <branch>` 到 PR 分支、`git checkout <commit> -- .` 探测历史。
- **正确**：`awk`/`sed` 打印每个冲突块 → 判断 ours/theirs → `edit_file` 逐块解决（只补主题相关行）。
- **铁律**："7488 + 基线 ≠ dev"——dev 还有后续开发，整包 checkout 会拉入无关优化（如 max_output_tokens 抖动、注释改写）。

## 5. 自动合并语义丢失复查

git 标"自动合并成功"的文件也可能保留旧版语义（曾：自动合并保留 dev 旧版 `runRefTurn`（无 format 绑定），丢上游 `runRefTurnWithFormat`，测试失败才暴露）。核心功能文件（controller.go / run_loop.go / responses.go / compact*.go / stats）merge 后：

```bash
git diff origin/main-v2 -- <file>   # 复查——确认没保留过时实现、没丢新修复
```

## 6. 冲突方向确认再 edit

`<<<<<<< HEAD` 侧**未必是正确版本**（曾把 gofmt 缩进冲突解决反）。先对比哪边是新修复（版本、语义、host 匹配），再决定取舍。

## 7. 生成文件三件套

```bash
git checkout --theirs   # 清冲突标记
go run ./cmd/remote-protocol-gen   # 重新生成
go run ./cmd/remote-protocol-gen -check   # 验证
```
不要手改 schema/generated 文件。

## 8. 合并后验证链（顺序固定）

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

### 8.1 验证链与验收纪律对齐

merge §8 的六步链与 `docs/team/20260812-discipline-charter.md` 验收项 4
（回归）命令口径统一如下——合并事务必须完整执行，缺一步即驳回：

```
gofmt -l .（排除 desktop/.direnv）→ go build ./... → 核心包测试
（provider/agent/control/config/tool/protocol）→ remote-protocol-gen -check
→ repolint（完整退出码）→ desktop tsc + 【前端 tsx 测试（desktop/frontend
node_modules/.bin/tsx src/__tests__/——滚动/水合/历史加载等——tsc 不跑测试，
冲突块采用可能丢 helper/import，必须 tsx 实测）】→ wails build
```

## 9. 隐藏产物排查清单

- 重复声明（`grep -c 'var xxx'`——合并可能带两遍）
- 多余括号 / 丢失缩进（编译错误行号定位 / `gofmt -d` 看 diff）
- `validate`/`omitempty` 类语义差异（对照 dev 已修复版本 `git show dev/clearnature:<file>`）

## 10. 遥测四方审计（合并涉及遥测文件后必做）

涉及 `internal/agent/*telemetry*.go`、`internal/stats/*` 的合并：

```
CompactionTelemetry 结构 ↔ emit 的 detail 键 ↔ recordCompaction/setCompactionInt 解析 ↔ CompactionRecord JSON tag
```
**任一环缺 = 断链，一次补完（禁止零敲碎打）**。失败路径必须落盘（`status=failed` + `err_type=`，禁止 return 吞 notice）。历史教训：PrefixHash / CompactionRecord / Est / Results / SavedChars / ElapsedMs 曾静默丢失——**编译通过 ≠ 无丢失**，必须行为级核对（跑 FastCompress / QueryCompactions 测试 + 实际运行验证 stats 落盘字段）。详见 `docs/merge/telemetry-audit-20260813.md`。

## 11. 分支纪律

- `main-v2` = 只读镜像：只 `git fetch origin` + `git merge --ff-only origin/main-v2`，**绝不在上面提交**。
- `dev/clearnature` = 唯一本地开发分支（文档/决策/代码都在这）。
- PR 分支更新：上游基线重建 + 逐个 cherry-pick 主题 commit（一次一个，冲突逐个解决）；**禁止 merge dev 全量**。
- merge 优先于 rebase：rebase 逐提交重放产生伪冲突（上游没改的文件也被标记）；merge 一次性解决。

## 12. 0 文本冲突 ≠ 0 语义风险

`git merge-tree` 报 0 冲突不代表安全——**同文件双方都改 = 逐字段核对**（主题 B 类：同域修复合并后跑守卫测试套件确认双修复并存）。合并前先读 `docs/merge/` 的风险分析（按主题矩阵核对点执行）。


## 13. 验收纪律（合并事务必过——引用纪律团章程）

合并事务的完成必须通过 `docs/team/20260812-discipline-charter.md`
第二部分的**验收项 1-6**（每条附可验证证据，审查员不得无证据通过）：

| 验收项 | 内容 | 合并事务的落点 |
|---|---|---|
| 1 | fable5 合规（执行队是否跳步） | 合并三件套（plan/execution/review）齐全 |
| 2 | 幻觉检测（证据真实性） | 冲突清单/合并 diff 可复核（git diff/status） |
| 3 | **缓存红线**（D1-D7：前缀字节/D2 只 append/D5 运行值不进 schema） | 发送侧改动先问"会不会改变前缀字节" |
| 4 | 回归（验证链 §8.1） | 六步链完整执行 |
| 5 | 对抗自检（devil's advocate） | 合并方案攻击点 ≥5 个（含语义丢失/隐藏产物） |
| 6 | 防虚假完成（证据链完整性） | 每步附验证命令+结果 |

**判定规则**（charter 第二部分）：
- 验收项 3（缓存红线）FAIL → **无条件驳回**；
- 验收项 2（幻觉检测）/1（跳步未裁决）/4（回归 FAIL）/5（blocking 缺陷）→ 驳回；
- 非 blocking 缺陷 → 限期修复（驳回周期上限内）；修复后重验。

**风险联动**（merge 三级预警 ↔ 验收项）：
- 🟢 低风险：验收 1/2/6 必过 + 回归快检（build+定向测试）
- 🟡 中风险（本地功能保护交叉分析命中）：全验收 1-6 + 隐藏产物排查（§9）+ 四方审计（§10）
- 🔴 高风险（compact 核心/desktop 前端/third_party 换代）：全验收 1-6 + D 组缓存红线专项 + 双人复核（执行+独立纪律）
