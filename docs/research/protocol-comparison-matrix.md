# 三协议数据流对比矩阵（Chat Completions / Responses / Anthropic）


| 维度 | Chat Completions（openai.go） | Responses（responses.go） | Anthropic（anthropic.go） |
|---|---|---|---|
| **消息形态** | messages 数组（role/content） | input items（text/tool_call/function_call/reasoning） | messages 数组 + system 独立 |
| **思考表达** | thinking.type + reasoning_effort（DeepSeek flavor）；vendor 分支（M3 binary/GLM toggle） | reasoning item + effort（mimo none 关闭）；id/status 必须回传 | thinking block（Anthropic **签名**回放；DeepSeek 端点 unsigned + output_config.effort） |
| **多轮状态** | 无状态（每次全量历史） | stateless（全量历史）vs stateful（previous_response_id）——vendor 表 | 无状态（全量历史 + thinking 签名回放） |
| **完成语义** | finish_reason 直接 | response.completed 常缺 finish_reason → **合成 stop**（#7168）；zero-usage 抑制 | stop_reason 直接 |
| **缓存** | prefix cache（24h 档，cache_ttl_minutes） | 同（vendor 表 TTL）；sessionCacheHeader（dashscope） | 无特殊（HTTP 缓存） |
| **严格校验点** | name 空值 400（#4711 MiMo）；thinking 覆盖第三方（#7273） | summary 缺 400（dashscope）；未知字段折回膨胀（mimo）；max_output_tokens 截断 | 签名回放缺失报错；redacted_thinking 兼容 |
| **失败模式** | 400 字段校验 | incomplete 截断（预算）；thinking 缺（#6259 服务端偶发）；工具 JSON 截断 | 签名不匹配/思考重放错 |

## 关键交叉印证

1. **vendor 差异模式一致**：三协议都出现"同一协议、不同 vendor 行为"（Chat 的
   thinking 三形态；Responses 的 summary/stateless；Anthropic 的签名 vs unsigned）
   ——根因：**官方 DeepSeek 判定/硬编码 + vendor 分支散落**（#7273/#7234/#7168 同源）
2. **严格校验阶梯**：#4711（Chat name）→ #7168（Responses summary）→
   Anthropic 签名——三协议都曾被"严格 vendor"打回，修复模式统一为
   "能力表/条件序列化"
3. **多轮状态差异**：Chat/Anthropic 无状态全量；Responses 有 stateless/stateful
   双模式——这是 Responses 特有的 vendor 分裂点（mimo/deepseek stateless，
   dashscope 兼容态）
4. **完成语义**：Responses 需合成 stop（#7168），Chat/Anthropic 原生——协议
   抽象层最薄弱的点
