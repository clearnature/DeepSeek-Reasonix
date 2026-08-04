# Responses 数据链全景报告（Panorama）

> 以 PR #7234 数据链为主线，8 个 iss/pr 交叉印证，三协议横向对比。
> 组成：problem-panorama-matrix.md（问题×层）+ datachain-7234-trace.md（数据链）
>      + protocol-comparison-matrix.md（三协议）+ cross-validation-panorama.md（交叉印证）
      + deepdive-family{A,B,C}-*.md（三家族代码级深挖）

## 一、全景结论（一页）

1. **问题不是孤立的**：8 个 iss/pr（#7357/#7273/#7234/#7168/#4711/#6259/#7200/#7191 + 深度追踪扩展
   #7099/#7184/#7111/#7304/#4099/#3561/#7451/#7410/#7358 + 货币计价家族
   #4814/#3883/#4498/#4546/#4565/#3527/#4617/#724 = 18 条）
   归属 4 层（配置/协议/会话/架构），但共同根因只有三个架构缺陷：
   **配置合并语义错配（#7357）/ vendor 判定散落（#7273/#7234/#7168）/
   协议抽象层缺失（#4711/#6259）**。

2. **问题演变有清晰轨迹**：配置层初始缺陷（7de6a2474，2026 初）潜伏 →
   协议层 vendor 差异集中爆发（6-17 #4711 起）→ 会话层鲁棒性（#6259/#7191）→
   架构层标准化诉求（#7200）。每层问题的修复都暴露下一层。

3. **#7234 数据链六环节**（配置→请求→wire→流式→会话→工具）每一环都有
   vendor 差异与问题锚点——vendor 能力表（6 能力单点）是当前最优解。

3b. **effort 校验家族 4 成员**（#4099/#3561/#7273/#7451）实证"vendor 判定
   过于自信"模式持续复发；**计费链路**（#7184 ↔ #7168 ↔ MiMo 截断）实证
   计费依赖流走完；**#7410** 显示检索系统已走上游化流程。

3c. **货币计价家族 8 条**（#4814/#3883/#4498/#4546/#4565/#3527/#4617/#724）：
   #4814（RMB 硬编码覆盖 USD）是 #7357 在官方 provider 路径的货币变体
   ——"默认注入覆盖用户配置"模式在计价体系的完整实证；#7358 是
   #7357 的 context_window 症状（已一并修复）。

4. **三协议横向印证**：Chat/Responses/Anthropic 都出现"同一协议不同 vendor
   行为"（thinking 三形态/stateless 分裂/签名回放）——协议抽象层是长期缺口。

5. **修复路线**：P0（#7357 隐私外呼 + #4711 严格校验）已修；
   P1（vendor 能力表/完成语义）已收敛；P2（协议抽象层）待 #7200 生态共识。

## 二、数据链全景图

```
配置层 ──→ 请求层 ──→ wire 层 ──→ 流式层 ──→ 会话层 ──→ 工具层
preset     build        messages     readStream   Chunk meta   web_search
entry      RequestBody  ToInput      reasoning    Agent.stream (检索系统
vendor 表  (预算/温度/   (string/     id/status    run_loop     最大消费者)
           effort)      array+       FinishReason 持久化回传
                        summary)     合成 stop
   ↑            ↑            ↑            ↑            ↑
 #7357      #7273       b0c1d8cb5    #7168 评审3  #7234 评审1
 #7273      (effort)    #4711        #6259        #7168 评审2
```

## 三、与 #7200 的生态呼应

#7200 主张"Responses 是事实标准接口"——本报告证明：协议标准化只是起点，
**vendor 差异是长期现实**（同一协议三套行为）。vendor 能力表是代码内
对生态现实的正确响应：能力声明而非行为猜测。

## 四、可执行建议

1. 提交 #7357 修复 PR（2db41eb95 已在本地 dev）
2. 汇总 #7234/#7168 统一分支（已含全部 P1 修复）
3. #7200 社区回复：附本全景报告链接（vendor 差异实证）
4. 长期：协议抽象层（三协议能力统一表达）随生态共识演进
5. 检索系统上游化：#7410 已提交 feature 请求，代码在 tools-server-web-search
   分支（含场景覆盖优化）——PR 待用户决策


## 五、三家族代码级深挖汇总（2026-08-04 第二轮）

### A. 默认注入家族（#7357/#7358/#4814/#7273 + 货币 8 条）

```
✅ 7 个注入函数全清单 + 引入提交（1a87cd4cf RMB 硬编码 → 2fefeef06 provenance）
✅ #7357/#7358 修复已合入 dev（e2e2dae0c，cherry-pick 自 main-v2 2db41eb95）
✅ #4814 上游主体已修（provenance），残余：标准模板+只改价格组合
⚠️ 教训：每个"官方注入"必须有"用户已声明则跳过"守卫（声明即替换）
```

### B. vendor 硬校验家族（#7451/#4099/#3561/#7273/#4711/#7168）

```
🔑 两层不对称（总根因）：配置层 NormalizeEffort default fail-closed（拒配未知端点）
   vs 协议层 openai.New() generic 分支 fail-open（接受 low/medium/high）
❌ 未修：#7451（TokenHub——配置层拒配、协议层实测支持）
⚠️ 残余：minimax/zhipu/longcat 分支无逃生口
✅ 已修：#4711（*string 条件序列化）/ #7168（能力表）/ #7273（白名单）
修复方向：#7451 = default 分支对 kind=openai 未知端点回退 generic 词汇
```

### C. 计费/完成语义家族（#7184/#7168/MiMo 截断）

```
✅ #7184 主体已修（70097ea08 emitTurnUsage + bestEffortStreamUsage）
❌ 残留缺口 1：agent.go:2862-2864 StreamInterrupted 分支不对齐（计费完全丢失）
❌ 残留缺口 2：best-effort 不估 prompt tokens（中断时计费大头丢失）
⚠️ 完成语义与计费分离"架构成立、实现不完整"（全零+stop 场景 reasoningOnly 失效）
✅ MiMo 截断已修（da3aadd34 65536）
```

### D. 会话鲁棒性家族（#6259/#7191/#7111）——已修 2/3（#7111 open）

### E. 生态/架构（#7200/#7099/#7410）——#7410 检索上游化进行中

## 六、未闭环缺口清单（可执行优先级）

| 优先级 | 缺口 | 位置 | 修复方向 |
|---|---|---|---|
| P0 | #7451 配置层 fail-closed 拦截未知 OpenAI 端点 | effort.go:246-247 | kind=openai 未知端点回退 generic 词汇（与 openai.go:206 对齐） |
| P1 | StreamInterrupted 计费缺口 | agent.go:2862-2864 | 对齐 ctx.Done 分支（best-effort + request count） |
| P1 | best-effort 不估 prompt tokens | agent.go:2877-2904 | 按 input 估算或 provider 留存 usage |
| P2 | minimax/zhipu/longcat 无逃生口 | openai.go:160-205 | hasExplicitSupportedEfforts 补全 |
| P2 | #4814 残余（标准模板+改价格被刷新） | pricing.go:186-188 | markPersisted 增加 Price 检查 |
| P2 | 全零+stop 完成语义耦合 | responses.go:734-738 | 完成语义与计费数据彻底解耦 |

## 七、分类体系最终版

```
18 条问题 → 5 家族 + 1 生态：
A. 默认注入（4+货币 8）：#7357/#7358/#4814/#7273 + 币种 8 条
B. vendor 硬校验（6）：#7451/#4099/#3561/#7273/#4711/#7168
C. 计费/完成语义（3）：#7184/#7168/MiMo 截断
D. 会话鲁棒性（3）：#6259/#7191/#7111
E. 生态/架构（3）：#7200/#7099/#7410
（#7273/#4711/#7168 跨家族——多模式归属）
```
