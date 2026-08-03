# MiMo Responses API 官方示例与 Schema（用户提供完整版）

> 来源: 官方文档（api.xiaomimimo.com/v1/responses），2026-08-03 用户提供

## 1. 基础调用（非流式，effort=none 关闭思考）

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/responses' \
--header "api-key: $MIMO_API_KEY" \
--header 'Content-Type: application/json' \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
 "input": "please introduce yourself",
 "max_output_tokens": 1024,
 "stream": false,
 "reasoning": {
 "effort": "none"
 }
}'
```

响应（effort=none 时 **无 reasoning item**，reasoning_tokens=0）：
```json
{
 "id": "resp_5fcaac1af26a4b449f30e1eeeaa8c48f",
 "object": "response",
 "status": "completed",
 "model": "mimo-v2.5-pro",
 "output": [{
 "id": "msg_1fa8f5011ebf47adb158ecf13c99d06c",
 "type": "message", "status": "completed", "role": "assistant",
 "content": [{"type": "output_text", "text": "Hello! I am MiMo, ...", "annotations": []}]
 }],
 "output_text": "Hello! I am MiMo, ...",
 "usage": {
 "input_tokens": 57, "input_tokens_details": {},
 "output_tokens": 46, "output_tokens_details": {"reasoning_tokens": 0},
 "total_tokens": 103
 }
}
```

## 2. 流式调用（SSE 事件序列）

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/responses' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "instructions": "...",
 "input": [{"role": "user", "content": [{"type": "input_text", "text": "please introduce yourself"}]}],
 "max_output_tokens": 1024,
 "stream": true
}'
```

SSE 事件流（**无 data: [DONE]**，以 response.completed 结尾）：
```
event:response.created
data:{"type":"response.created","sequence_number":0,"response":{...}}

event:response.in_progress
data:{"type":"response.in_progress","sequence_number":1,...}

event:response.output_item.added
data:{"type":"response.output_item.added","sequence_number":2,"output_index":0,
 "item":{"id":"rs_54dee72a...","type":"reasoning","summary":[],"content":[],"status":"in_progress"}}

event:response.content_part.added
data:{"type":"response.content_part.added","sequence_number":3,"item_id":"rs_...",
 "output_index":0,"content_index":0,"part":{"type":"reasoning_text","text":""}}

...（reasoning_text.delta × N）

event:response.output_text.done
data:{"type":"response.output_text.done","sequence_number":84,"item_id":"msg_...",
 "output_index":1,"content_index":0,"text":"Hello! I'm MiMo, ..."}

event:response.output_item.done
data:{"type":"response.output_item.done","sequence_number":86,"output_index":1,
 "item":{"id":"msg_...","type":"message","status":"completed","role":"assistant","content":[...]}}

event:response.completed
data:{"type":"response.completed","sequence_number":87,"response":{
 "id":"resp_646d7c96...","status":"completed",
 "output":[
 {"id":"rs_54dee72a...","type":"reasoning","summary":[],"content":[{"type":"reasoning_text","text":"Okay, the user just asked me to introduce myself..."}],"status":"completed"},
 {"id":"msg_513be1...","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"Hello! I'm MiMo, ..."}]}
 ],
 "usage":{"input_tokens":55,"input_tokens_details":{},"output_tokens":177,
 "output_tokens_details":{"reasoning_tokens":110},"total_tokens":232}}}
```

**关键**：
- reasoning item 输出带 `summary: []`（空数组）——**服务端输出有 summary 字段**
- 流式事件无 `[DONE]`
- completed 事件里 reasoning item 的 `content` 是**完整推理文本**（delta 聚合）

## 3. 工具调用（思考模式，工具轮返回 reasoning + function_call）

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/responses' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "instructions": "...",
 "input": [{"role": "user", "content": "What is the weather like in Boston today?"}],
 "max_output_tokens": 1024,
 "tools": [{
 "type": "function",
 "name": "get_current_weather",
 "description": "Get the current weather in a given location",
 "parameters": {"type": "object", "properties": {...}, "required": ["location"]},
 "strict": true
 }],
 "tool_choice": "auto"
}'
```

响应（**工具轮带 reasoning**，reasoning_tokens=186）：
```json
{
 "id": "resp_cd7be5c8...",
 "status": "completed",
 "output": [
 {"id": "rs_80ce68b8...", "type": "reasoning", "summary": [],
 "content": [{"type": "reasoning_text", "text": "The user wants to know the weather in Boston today. I have access to the get_current_weather tool..."}],
 "status": "completed"},
 {"id": "fc_a6b2cb43...", "type": "function_call",
 "call_id": "call_cab078c9...", "name": "get_current_weather",
 "arguments": "{\"location\": \"Boston, MA\", \"unit\": \"fahrenheit\"}",
 "status": "completed"}
 ],
 "output_text": "",
 "usage": {"input_tokens": 349, "output_tokens": 221, "output_tokens_details": {"reasoning_tokens": 186}, "total_tokens": 570}
}
```

**关键**：工具调用轮 `function_call` 的 `call_id` 是 `call_xxx`（与 OpenAI 一致），arguments 是 JSON 字符串。

## 4. 多模态（input_image）

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/responses' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5",
 "instructions": "...",
 "input": [{
 "role": "user",
 "content": [
 {"type": "input_image", "image_url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"},
 {"type": "input_text", "text": "please describe the content of the image"}
 ]
 }],
 "max_output_tokens": 1024
}'
```

响应（含 reasoning 310 tokens + 多模态文本）：
```json
"usage": {"input_tokens": 1085, "input_tokens_details": {"cached_tokens": 1024},
 "output_tokens": 559, "output_tokens_details": {"reasoning_tokens": 310},
 "total_tokens": 1644}
```

**关键**：多模态请求 **input_tokens_details.cached_tokens=1024**（非流式返回缓存命中！与流式 completed 事件 input_tokens_details={} 不同）

## 5. 深度思考（effort=high）

```bash
"reasoning": {"effort": "high"}
```

响应 reasoning item（effort=high 时 reasoning_tokens=15，简短思考）：
```json
{"id": "rs_e344b356...", "type": "reasoning", "summary": [],
 "content": [{"type": "reasoning_text", "text": "The user is asking for a brief introduction to machine learning in three sentences."}],
 "status": "completed"}
```

## 6. 结构化输出（text.format=json_object）

```bash
"text": {"format": {"type": "json_object"}}
```

响应（effort 默认，reasoning_tokens=0——json 模式可能无思考）：
```json
{"type": "output_text", "text": "{\n \"name\": \"Zhang San\",\n \"age\": 28,\n \"email\": \"zhangsan@test.com\",\n \"birthday\": \"1996-05-12\"\n}"}
```

---

# MiMo Responses API 完整 Schema（官方，用户提供 1151 行）

## 请求体

| 字段 | 类型 | 必选 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 可选值：mimo-v2.5-pro，mimo-v2.5 |
| `input` | string \| array | ✅ | 文本输入或 InputItemList |
| `instructions` | string | | 系统（开发者）指令 |
| `max_output_tokens` | integer | | pro 默认 131072 / v2.5 默认 32768，范围 [1, 131072] |
| `stream` | boolean | | 默认 false |
| `reasoning` | object | | 推理配置；多轮工具调用需保留历史思考内容 |
| `reasoning.effort` | string | ✅ | **可选值：none, low, medium, high**；none 关闭推理，其余均开启（暂不支持自定义档位） |
| `temperature` | number | | 思考模式强制 1.0（传入无效） |
| `top_p` | number | | 思考模式强制 0.95（传入无效） |
| `text` | object | | 响应格式配置 |
| `text.format` | object | | 默认 {"type": "text"}；json_object 结构化 |
| `tool_choice` | string | | **仅 auto**（非 auto 后端移除，等同 auto） |
| `tools` | array | | 工具数组；思考模式多轮工具调用需保留历史思考内容 |
| `tools.name` | string | ✅ | 1-64 字符，a-z/A-Z/0-9/_/- |
| `tools.parameters` | object | ✅ | JSON Schema |
| `tools.strict` | boolean | ✅ | 默认 false |
| `tools.description` | string | | 函数描述 |
| `tools.type` | string | ✅ | 仅 function |

## InputItemList 类型

### EasyInputMessage / Message
- `content`: string | array（**必选**）
 - TextInput: string（纯文本简写）
 - ResponseInputMessageContentList: array
 - ResponseInputText: `{"type": "input_text", "text": "..."}`
 - ResponseInputImage: `{"type": "input_image", "image_url": "..."}`
- `role`: **必选**，user/assistant/system/developer
- `type`: message（可选）

### FunctionCall（输入）
- `arguments`: string **必选**（JSON 字符串）
- `call_id`: string **必选**
- `name`: string **必选**
- `type`: function_call **必选**
- `id`/`namespace`/`status`: 可选（API 返回填充）

### FunctionCallOutput（输入）
- `call_id`: string **必选**
- `output`: string **必选**（JSON 字符串）
- `type`: function_call_output **必选**
- `id`/`status`: 可选

### Reasoning（输入）
- `id`: string **必选**
- `content`: array
 - `{"type": "reasoning_text", "text": "..."}`
- `type`: reasoning **必选**
- `status`: in_progress/completed

**注意**：Reasoning 输入对象**无 summary 字段**（与输出不同，输出带 summary:[]）

### ResponseOutputMessage（输入回传 assistant）
- `id`: string **必选**
- `content`: array **必选**（ResponseOutputText: `{"type": "output_text", "text": "..."}`）
- `role`: assistant **必选**
- `status`: in_progress/completed **必选**
- `type`: message **必选**

## Response 对象（非流式输出）

- `id` / `created_at` / `object`("response") / `model`
- `status`: completed/in_progress/incomplete
- `error`: {code, message}
- `incomplete_details`: {reason: max_output_tokens/content_filter}
- `output`: array（message/function_call/function_call_output/reasoning）
 - ResponseOutputMessage: content[output_text], role=assistant, status, type=message
 - FunctionCall: arguments/call_id/name/type=function_call/status
 - Reasoning: content[reasoning_text], status, type=reasoning, **summary: []**
- `output_text`: SDK 便捷聚合属性
- `usage`: {input_tokens, input_tokens_details{cached_tokens}, output_tokens, output_tokens_details{reasoning_tokens}, total_tokens}

## Response chunk 对象（流式 SSE 事件）

事件序列（**无 [DONE]**，以 completed/incomplete 结尾）：

| 事件 | 关键字段 |
|------|---------|
| `response.created` | response |
| `response.in_progress` | response |
| `response.completed` | response（含完整 output + usage） |
| `response.incomplete` | response |
| `response.output_item.added` | item, output_index |
| `response.output_item.done` | item（完整），output_index |
| `response.content_part.added` | part（output_text/reasoning_text）, item_id, content_index |
| `response.content_part.done` | part（完整 text）, item_id, content_index |
| `response.output_text.delta` | delta, item_id, content_index |
| `response.output_text.done` | text（完整）, item_id, content_index |
| `response.function_call_arguments.delta` | delta, item_id, output_index |
| `response.function_call_arguments.done` | arguments（完整）, name, item_id |
| `response.reasoning_text.delta` | delta, item_id, content_index |
| `response.reasoning_text.done` | text（完整）, item_id, content_index |

每个事件都带 `sequence_number` 和 `type`。

## 与我们实现的关键对照

| 项 | 官方文档 | 我们的实现 |
|----|---------|-----------|
| reasoning 输入无 summary | ✅ 文档确认 | ✅ summaryRequired 仅 dashscope（MiMo 不发） |
| 流式无 [DONE] | ✅ | ✅ readStream 兼容（completed 终止） |
| reasoning item 输出 summary:[] | ✅ | ✅ 读取时忽略 |
| tool_choice 仅 auto | ✅ | ✅ 不发=auto |
| 多轮工具调用保留历史 reasoning | ✅ 文档强调 | ✅ RequiresToolCallReasoning |
| max_output_tokens 默认 32768（v2.5） | ✅ | ✅ 已提升 65536（da3aadd34） |
| effort none/low/medium/high | ✅ | ✅ mimoEffortCapability |