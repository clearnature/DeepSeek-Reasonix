# 交叉印证：架构缺陷归类与问题演变全景

## 1. 三个问题的同源验证（#7357 / #7273 / #4711）

| 问题 | 现象 | 根因模式 | 同源点 |
|---|---|---|---|
| #7357 | 自定义 provider 继承 DeepSeek 默认（balance/price/cw） | Default()+decode 按 index 合并 slice | **默认注入与用户输入共用合并路径**——无"用户声明即替换"边界 |
| #7273 | supported_efforts 被 DeepSeek thinking 硬编码校验覆盖 | 官方 DeepSeek 判定（officialProviderKind/host 匹配）后强加校验 | **"官方判定"模式**——按 host 认 vendor 后无条件覆盖用户配置 |
| #4711 | tool message name 空值 400 | chatMessage.Name omitempty 省略 → MiMo 严格校验打回 | **协议严格性未抽象**——厂商严格度差异无能力表表达 |

**共同根因**：三问题都源于"**vendor 判定/默认注入过于自信**"——
代码假定"我知道这是谁"（host 匹配 → 默认/校验），
但对"用户自定义/第三方代理/严格厂商"的边界场景没有兜底。

## 2. 架构缺陷归类（三类）

### A. 配置合并语义错配（#7357）
- 模式：Default() 预填 + decode 按 index 合并（7de6a2474 初始）
- 缺陷：slice 不能参与分层覆盖（用户声明数组应替换而非融合）
- 修复：decode 前检测用户声明 → 清空默认（2db41eb95）

### B. vendor 判定散落（#7273 + #7234/#7168 前置状态）
- 模式：DetectVendor/host 匹配散落 5 处；能力（summary 必须/温度无效/
  stateless/effort 档）无单一事实来源
- 缺陷：新增 vendor 时漏判 → wire 错/校验覆盖
- 修复：vendor 能力表（9bbc0b819）——6 能力单点

### C. 协议抽象层缺失（#4711 + #6259 + #7168 完成语义）
- 模式：三协议各自实现 wire，严格度/完成语义差异无统一表达
- 缺陷：厂商严格校验（name/summary/签名）逐一打回；缺 thinking 无标准降级
- 修复：条件序列化（*string name/summaryRequired）+ 完成语义合成（stop）

## 3. 问题演变全景（数据链视角）

```
配置层（#7357 合并缺陷 7de6a2474 初始）
  └─ 影响所有协议（provider 是协议入口）——但被"官方预设"掩盖
协议层（#7234 MiMo wire / #7168 DashScope——vendor 差异集中爆发）
  └─ 三协议都出现"严格 vendor 打回"（#4711 Chat name / #7168 summary /
      Anthropic 签名）——严格校验阶梯
会话层（#6259 缺 thinking / #7191 corrupt meta）
  └─ 多轮工具循环鲁棒性——服务端偶发 + 本地容错
架构层（#7200 Responses 标准化诉求）
  └─ 生态级：各厂商 Responses 支持差异——vendor 能力表是代码内对应物
```

## 4. 修复路线分级

| 优先级 | 项 | 状态 |
|---|---|---|
| P0 | #7357 配置合并（隐私外呼） | ✅ 2db41eb95（本地 dev） |
| P0 | #4711 name 严格校验 | ✅ c7cbed768 |
| P1 | vendor 能力表收敛（summary/预算/TTL/effort） | ✅ 9bbc0b819 + 统一分支 |
| P1 | 完成语义（stop 合成/zero-usage） | ✅ #7168 统一实现 |
| P2 | 协议抽象层长期演进（三协议能力统一表达） | 待 #7200 生态共识 |
