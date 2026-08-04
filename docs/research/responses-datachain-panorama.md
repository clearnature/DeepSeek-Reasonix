# Responses 数据链全景报告（Panorama）

> 以 PR #7234 数据链为主线，8 个 iss/pr 交叉印证，三协议横向对比。
> 组成：problem-panorama-matrix.md（问题×层）+ datachain-7234-trace.md（数据链）
>      + protocol-comparison-matrix.md（三协议）+ cross-validation-panorama.md（交叉印证）

## 一、全景结论（一页）

1. **问题不是孤立的**：8 个 iss/pr（#7357/#7273/#7234/#7168/#4711/#6259/#7200/#7191）
   归属 4 层（配置/协议/会话/架构），但共同根因只有三个架构缺陷：
   **配置合并语义错配（#7357）/ vendor 判定散落（#7273/#7234/#7168）/
   协议抽象层缺失（#4711/#6259）**。

2. **问题演变有清晰轨迹**：配置层初始缺陷（7de6a2474，2026 初）潜伏 →
   协议层 vendor 差异集中爆发（6-17 #4711 起）→ 会话层鲁棒性（#6259/#7191）→
   架构层标准化诉求（#7200）。每层问题的修复都暴露下一层。

3. **#7234 数据链六环节**（配置→请求→wire→流式→会话→工具）每一环都有
   vendor 差异与问题锚点——vendor 能力表（6 能力单点）是当前最优解。

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
