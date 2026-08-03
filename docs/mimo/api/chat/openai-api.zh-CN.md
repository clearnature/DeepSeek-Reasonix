# OpenAI Chat Completions API（zh-CN 官方）

> 来源: 官方文档 zh-CN

## 请求地址

```
https://api.xiaomimimo.com/v1/chat/completions
```

## 请求头

接口支持以下两种认证方式，请选择其中一种添加到请求头中：

```
api-key: $MIMO_API_KEY
Content-Type: application/json
```

```
Authorization: Bearer $MIMO_API_KEY
Content-Type: application/json
```

## 请求体

当前仅 `mimo-v2.5` 模型支持图像、音频或视频输入。",
 "children": [
 {
 "name": "Text content part",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "文本内容。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容部分的类型。"
 }
 ]
 },
 {
 "name": "Image content part",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "image_url",
 "type": "object",
 "isBold": true,
 "required": true,
 "children": [
 {
 "name": "url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像的 URL 或 Base64 编码的图像数据。"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容部分的类型。
可选值：`image_url`"
 }
 ]
 },
 {
 "name": "Audio content part",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "input_audio",
 "type": "object",
 "isBold": true,
 "required": true,
 "children": [
 {
 "name": "data",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "音频的 URL 或 Base64 编码的音频数据。"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容部分的类型。
可选值：`input_audio`"
 }
 ]
 },
 {
 "name": "Video content part",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "video_url",
 "type": "object",
 "isBold": true,
 "required": true,
 "children": [
 {
 "name": "url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "视频的 URL 或 Base64 编码的视频数据。"
 }
 ]
 },
 {
 "name": "fps",
 "type": "number",
 "isBold": true,
 "required": false,
 "defaultValue": "2",
 "description": "每秒抽帧数。
所需范围：`[0.1, 10.0]`"
 },
 {
 "name": "media_resolution",
 "type": "string",
 "isBold": true,
 "required": false,
 "defaultValue": "default",
 "description": "分辨率档次。
可选值：`default`，`max`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容部分的类型。
可选值：`video_url`"
 }
 ]
 }
 ]
 }
 ]
 },
 {
 "name": "role",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "用户消息的角色。
可选值：`user`"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "参与者的可选名称。为模型提供信息以区分相同角色的参与者。"
 }
 ]
 },
 {
 "name": "Assistant message",
 "type": "object",
 "isBold": false,
 "description": "模型响应用户消息发送的消息。",
 "children": [
 {
 "name": "role",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "助手消息的角色。
可选值：`assistant`"
 },
 {
 "name": "content",
 "type": [
 "string",
 "array"
 ],
 "isBold": true,
 "required": false,
 "description": "助手消息的内容。除非指定了 `tool_calls`，否则为必需。",
 "children": [
 {
 "name": "Text content",
 "type": "string",
 "isBold": false,
 "description": "助手消息的内容。"
 },
 {
 "name": "Array of content parts",
 "type": "array",
 "isBold": false,
 "description": "一个由指定类型的内容部分组成的数组。可以是一个或多个 `text` 类型。",
 "children": [
 {
 "name": "Text content part",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "文本内容。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容部分的类型。"
 }
 ]
 }
 ]
 }
 ]
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "参与者的可选名称。为模型提供信息以区分相同角色的参与者。"
 },
 {
 "name": "tool_calls",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "模型生成的工具调用，如函数调用。",
 "children": [
 {
 "name": "Function tool call",
 "type": "object",
 "isBold": false,
 "description": "模型创建的对函数工具的调用。",
 "children": [
 {
 "name": "function",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "模型调用的函数。",
 "children": [
 {
 "name": "arguments",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "调用函数时所需的参数，由模型以 JSON 格式生成。请注意，模型生成的内容并非总是有效的 JSON，还可能虚构出函数 schema 中未定义的参数。在调用函数前，请在代码中验证这些参数。"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "要调用的函数名称。"
 }
 ]
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具调用的 ID。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具类型。目前仅支持 `function`。"
 }
 ]
 }
 ]
 }
 ]
 },
 {
 "name": "Tool message",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "content",
 "type": [
 "string",
 "array"
 ],
 "isBold": true,
 "required": true,
 "description": "工具消息的内容。",
 "children": [
 {
 "name": "Text content",
 "type": "string",
 "isBold": false,
 "description": "工具消息的内容。"
 },
 {
 "name": "Array of content parts",
 "type": "array",
 "isBold": false,
 "description": "一个由指定类型的内容部分组成的数组。对于工具消息，仅支持 `text` 类型。",
 "children": [
 {
 "name": "Text content part",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "文本内容。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容部分的类型。"
 }
 ]
 }
 ]
 }
 ]
 },
 {
 "name": "role",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "消息作者的角色。
可选值：`tool`"
 },
 {
 "name": "tool_call_id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "此消息响应的工具调用。"
 }
 ]
 }
 ]
},
{
 "name": "model",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "用于生成响应的模型 ID。
可选值：`mimo-v2.5-pro`，`mimo-v2.5`"
},
{
 "name": "frequency_penalty",
 "type": [
 "number",
 "null"
 ],
 "isBold": true,
 "required": false,
 "defaultValue": "0",
 "description": "取值范围在 -2.0 到 2.0 之间的数值。如果该值为正，那么新 token 会根据其在已有文本中的出现频率受到相应的惩罚，降低模型重复相同内容的可能性。
所需范围：`[-2.0, 2.0]`"
},
{
 "name": "max_completion_tokens",
 "type": [
 "integer",
 "null"
 ],
 "isBold": true,
 "required": false,
 "description": "对话补全中可以生成的 token 数的上限，包括可见的输出 token 数和推理 token 数。
- `mimo-v2.5-pro` 的默认值 `131072`
- `mimo-v2.5` 的默认值为 `32768`
所需范围：`[1, 131072]`"
},
{
 "name": "presence_penalty",
 "type": [
 "number",
 "null"
 ],
 "isBold": true,
 "required": false,
 "defaultValue": "0",
 "description": "取值范围在 -2.0 到 2.0 之间的数值。如果该值为正，那么新 token 会根据其是否已在已有文本中出现受到相应的惩罚，从而增加模型谈论新主题的可能性。
所需范围：`[-2.0, 2.0]`"
},
{
 "name": "response_format",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "一个指定模型必须输出的格式的对象。",
 "children": [
 {
 "name": "Text",
 "type": "object",
 "isBold": false,
 "description": "默认响应格式。用于生成文本响应。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "所定义的响应格式类型。仅为 `text`。"
 }
 ]
 },
 {
 "name": "JSON object",
 "type": "object",
 "isBold": false,
 "description": "JSON 对象响应格式。请注意，若系统消息或用户消息中未指示模型生成 JSON，模型将不会输出 JSON 格式内容。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "所定义的响应格式类型。仅为 `json_object`。"
 }
 ]
 }
 ]
},
{
 "name": "stop",
 "type": [
 "string",
 "array",
 "null"
 ],
 "isBold": true,
 "required": false,
 "defaultValue": "null",
 "description": "最多 4 个序列，当 API 生成到这些序列时会停止继续生成 token。返回的文本中不会包含这些停止序列。"
},
{
 "name": "stream",
 "type": [
 "boolean",
 "null"
 ],
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "如果设置为 `true`，模型的响应数据会在生成过程中通过SSE（server-sent events）的形式流式传输到客户端。"
},
{
 "name": "thinking",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "这个参数用于控制模型是否启用思维链。
注意：在思考模式下的多轮工具调用过程中，模型会在返回 `tool_calls` 字段的同时返回 `reasoning_content` 字段。若要继续对话，建议在后续每次请求的 `messages` 数组中保留所有历史 `reasoning_content`，以获得最佳表现。在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `temperature` 和 `top_p` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `1.0` 和 `0.95`。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "是否启用思维链
- `mimo-v2.5-pro`，`mimo-v2.5` 默认值为 `enabled`
可选值：`enabled`，`disabled`"
 }
 ]
},
{
 "name": "temperature",
 "type": "number",
 "isBold": true,
 "required": false,
 "description": "要使用的采样温度，介于 0 和 1.5 之间。较高的值（如 0.8）会使输出更加随机，而较低的值（如 0.2）会使其更加集中和确定性。我们通常建议更改此值或 `top_p`，但不要同时更改。
在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `temperature` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `1.0`。- `mimo-v2.5-pro`，`mimo-v2.5` 默认值为 `1.0`
所需范围：`[0, 1.5]`"
},
{
 "name": "tool_choice",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "控制模型如何选择工具。
注意：当 `tool_choice` 传入非 `auto` 值时，后端会默认移除该字段，模型响应行为仍等同于 `auto` 模式（该逻辑保留调整的可能性）。可选值：`auto`"
},
{
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "模型可能调用的工具列表。目前仅支持函数作为工具。
注意：在思考模式下的多轮工具调用过程中，模型会在返回 `tool_calls` 字段的同时返回 `reasoning_content` 字段。若要继续对话，建议在后续每次请求的 `messages` 数组中保留所有历史 `reasoning_content`，以获得最佳表现。",
 "children": [
 {
 "name": "Function tool",
 "type": "object",
 "isBold": false,
 "description": "可用于生成响应的函数工具。",
 "children": [
 {
 "name": "function",
 "type": "object",
 "isBold": true,
 "required": true,
 "children": [
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具函数的名称。必须由 `a-z`、`A-Z`、`0-9` 组成，或包含下划线（`_`）和连字符（`-`），最大长度为64。
所需字符串长度：`1 - 64`"
 },
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "函数功能的描述，供模型判断何时以及如何调用该函数。"
 },
 {
 "name": "parameters",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "函数接受的参数，以 JSON 模式对象的形式描述。
若省略 `parameters`，则表示该函数的参数列表为空。"
 },
 {
 "name": "strict",
 "type": "boolean",
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "生成函数调用时是否启用严格的模式遵循。若设为 true，模型将严格遵循 `parameters` 字段中定义的确切模式。当 `strict` 为 `true` 时，仅支持 JSON 模式的一个子集。"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具类型。目前仅支持 `function`。"
 }
 ]
 },
 {
 "name": "Web search tool",
 "type": "object",
 "isBold": false,
 "description": "可用于生成响应的联网搜索工具。详情请参考 联网搜索。
注意：使用前需要开通 联网服务插件。",
 "children": [
 {
 "name": "user_location",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "用户地理位置，用于优化模型输出结果",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "approximate"
 },
 {
 "name": "country",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "国家"
 },
 {
 "name": "region",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "省份"
 },
 {
 "name": "city",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "城市"
 },
 {
 "name": "district",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "区县"
 },
 {
 "name": "longitude",
 "type": "long",
 "isBold": true,
 "required": false,
 "description": "经度"
 },
 {
 "name": "latitude",
 "type": "long",
 "isBold": true,
 "required": false,
 "description": "维度"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具类型。目前仅支持 `web_search`。"
 },
 {
 "name": "force_search",
 "type": "string",
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "是否启用强制搜索。`true` 强制搜索，`false` 由模型判断是否需要搜索。"
 },
 {
 "name": "max_keyword",
 "type": "integer",
 "isBold": true,
 "required": false,
 "defaultValue": "5",
 "description": "限制单轮搜索中可使用的最大关键词数量。
所需范围：`[1, 50]`"
 },
 {
 "name": "limit",
 "type": "integer",
 "isBold": true,
 "required": false,
 "defaultValue": "5",
 "description": "限制单次搜索操作返回的最大结果条数。
所需范围：`[1, 50]`"
 }
 ]
 }
 ]
},
{
 "name": "top_p",
 "type": "number",
 "isBold": true,
 "required": false,
 "defaultValue": "0.95",
 "description": "核采样的概率阈值，用于控制模型生成文本的多样性。`top_p` 值越高，生成的文本多样性越强；`top_p` 值越低，生成的文本确定性越高。
由于 `temperature` 和 `top_p` 均用于控制生成文本的多样性，建议仅设置其中一个参数。
在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `top_p` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `0.95`。所需范围：`[0.01, 1.0]`"
}
]">

## Chat 响应对象（非流式输出）

## Chat 响应 chunk 对象（流式输出）

curlpython基础调用流式响应函数调用联网搜索图像输入音频输入视频输入结构化输出深度思考curl --location --request POST 'https://api.xiaomimimo.com/v1/chat/completions' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "messages": [
 {
 "role": "system",
 "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
 },
 {
 "role": "user",
 "content": "please introduce yourself"
 }
 ],
 "max_completion_tokens": 1024,
 "temperature": 1.0,
 "top_p": 0.95,
 "stream": false,
 "stop": null,
 "frequency_penalty": 0,
 "presence_penalty": 0,
 "thinking": {
 "type": "disabled"
 }
}'响应基础调用流式响应函数调用联网搜索图像输入音频输入视频输入结构化输出深度思考{
 "id": "8b51f9e0515949cb8207fbd35ea6ea5c",
 "choices": [
 {
 "finish_reason": "stop",
 "index": 0,
 "message": {
 "content": "Hello! I'm MiMo, Xiaomi's AI assistant created by the Xiaomi LLM-Core team. I'm here to chat, help answer questions, and assist with various tasks—whether it's providing information, brainstorming ideas, or just having a friendly conversation. Feel free to ask me anything, and I'll do my best to help! 😊",
 "role": "assistant",
 "tool_calls": null
 }
 }
 ],
 "created": 1776848906,
 "model": "mimo-v2.5-pro",
 "object": "chat.completion",
 "usage": {
 "completion_tokens": 72,
 "prompt_tokens": 57,
 "total_tokens": 129,
 "completion_tokens_details": {
 "reasoning_tokens": 0
 },
 "prompt_tokens_details": null
 }
}更新时间 2026 年 07 月 17 日错误码OpenAI Responses API目录curlpython基础调用流式响应函数调用联网搜索图像输入音频输入视频输入结构化输出深度思考curl --location --request POST 'https://api.xiaomimimo.com/v1/chat/completions' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "messages": [
 {
 "role": "system",
 "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
 },
 {
 "role": "user",
 "content": "please introduce yourself"
 }
 ],
 "max_completion_tokens": 1024,
 "temperature": 1.0,
 "top_p": 0.95,
 "stream": false,
 "stop": null,
 "frequency_penalty": 0,
 "presence_penalty": 0,
 "thinking": {
 "type": "disabled"
 }
}'响应基础调用流式响应函数调用联网搜索图像输入音频输入视频输入结构化输出深度思考{
 "id": "8b51f9e0515949cb8207fbd35ea6ea5c",
 "choices": [
 {
 "finish_reason": "stop",
 "index": 0,
 "message": {
 "content": "Hello! I'm MiMo, Xiaomi's AI assistant created by the Xiaomi LLM-Core team. I'm here to chat, help answer questions, and assist with various tasks—whether it's providing information, brainstorming ideas, or just having a friendly conversation. Feel free to ask me anything, and I'll do my best to help! 😊",
 "role": "assistant",
 "tool_calls": null
 }
 }
 ],
 "created": 1776848906,
 "model": "mimo-v2.5-pro",
 "object": "chat.completion",
 "usage": {
 "completion_tokens": 72,
 "prompt_tokens": 57,
 "total_tokens": 129,
 "completion_tokens_details": {
 "reasoning_tokens": 0
 },
 "prompt_tokens_details": null
 }
}回到顶部公众号开放平台微信群开放平台飞书群Token Plan
微信群Token Plan
飞书群
###### 产品
Xiaomi MiMo APIXiaomi MiMo StudioXiaomi MiMo ClawXiaomi MiMo Code
###### 开放平台
文档中心常见问题API 定价Token Plan
###### 动态
新闻更新日志MiMo Blog
###### 关于我们
服务协议隐私政策
###### 产品
Xiaomi MiMo APIXiaomi MiMo StudioXiaomi MiMo ClawXiaomi MiMo Code
###### 开放平台
文档中心常见问题API 定价Token Plan
###### 动态
新闻更新日志MiMo Blog
###### 关于我们
服务协议隐私政策

Xiaomi MiMo : 备案号 Beijing-XiaomiMiMo-202601050182|小米大语言模型算法 : 备案号 网信算备110108916280901240011号|京ICP备17028681号-55

 .base-modal .ant-modal-content {
 border-radius: 8px;
 padding: 24px;
 box-shadow: 0px 6px 12px 0px rgba(31, 35, 41, 0.06);
 }
 .base-modal .ant-modal-header {
 margin-bottom: 24px;
 }
 .base-modal .ant-modal-body {
 max-height: calc(90vh - 160px);
 overflow-y: auto;
 }

 .base-modal .ant-modal-title {
 font-size: 16px;
 font-weight: 500;
 line-height: 24px;
 color: #1d2129;
 }
 .base-modal .ant-modal-close {
 width: 16px;
 height: 16px;
 top: 28px;
 right: 24px;
 box-shadow: none;
 }
 .base-modal .ant-modal-close:hover {
 background: transparent;
 opacity: 1;
 }
 .base-modal .ant-modal-footer {
 margin-top: 24px;
 }
 .base-modal .ant-btn {
 height: 32px;
 padding: 5px 16px;
 border-radius: 6px;
 font-size: 14px;
 font-weight: 500;
 line-height: 22px;
 }

 .base-modal .ant-btn-default {
 background: #ffffff;
 border: 1px solid #dee0e3;
 color: #1f2329;
 }
 .base-modal .ant-btn-primary:disabled {
 background: var(--color-primary-disabled);
 border-color: var(--color-primary-disabled);
 color: #ffffff;
 }

 /* large 模式：在 md 及以下时隐藏 Modal，显示 Drawer */
 @media (max-width: 1023px) {
 .base-modal-large-wrap {
 display: none !important;
 }
 }
 @media (min-width: 1024px) {
 .base-modal-drawer-wrapper {
 display: none !important;
 }
 }

 /* Drawer 样式 - 与设计稿一致 */
 .base-modal-large.base-modal {
 max-height: 90vh;
 max-height: 90svh;
 border-top-left-radius: 16px;
 border-top-right-radius: 16px;
 }
 .base-modal-large .ant-drawer-footer {
 padding: 24px;
 border-top: none;
 }
 .base-modal-large .ant-drawer-header {
 font-size: 16px;
 border-bottom: none;
 font-weight: 500;
 line-height: 24px;
 color: #1d2129;
 }
 .base-modal-large .ant-drawer-header-title{
 flex-direction: row-reverse;
 }
 .base-modal-large .ant-drawer-close{
 margin: 0 !important ;
 }

 !function(){var e,f,a,c,n,t,d,r,o,b,i,u,s,l,p={58828:function(e,f,a){Promise.all([a.e(2374),a.e(2898),a.e(1889)]).then(a.bind(a,90855))}},m={};function h(e){var f=m[e];if(void 0!==f)return f.exports;var a=m[e]={exports:{}};return p[e].call(a.exports,a,a.exports,h),a.exports}if(h.m=p,h.c=m,f=(e="function"==typeof Symbol)?Symbol("rspack queues"):"__rspack_queues",a=h.aE=e?Symbol("rspack exports"):"__webpack_exports__",c=e?Symbol("rspack error"):"__rspack_error",n=e?Symbol("rspack done"):"__rspack_done",t=h.zS=e?Symbol("rspack defer"):"__rspack_defer",h.zT=function(e){if(e.some(e=>{var f=m[e];return!f||!1===f[n]}))return{then:(f,a)=>Promise.all(e.map(h)).then(f,a)}},d=function(e){e&&e.dtypeof Symbol&&Symbol.toStringTag&&Object.defineProperty(e,Symbol.toStringTag,{value:"Module"}),Object.defineProperty(e,"__esModule",{value:!0})},h.p="/","u">typeof document){var v=function(e,f,a,c,n,t){var d=("link");return d.rel="stylesheet",d.type="text/css",h.nc&&(d.nonce=h.nc),d.href=f,0!==d.href.indexOf(window.location.origin+"/")&&(d.crossOrigin="anonymous"),d.onerror=d.onload=function(a){if(d.onerror=d.onload=null,"load"===a.type)c();else{var t=a&&("load"===a.type?"missing":a.type),r=a&&a.target&&a.target.href||f,o=Error("Loading CSS chunk "+e+" failed.\\n("+r+")");o.code="CSS_CHUNK_LOAD_FAILED",o.type=t,o.request=r,d.parentNode&&d.parentNode.removeChild(d),n(o)}},a?a.parentNode.insertBefore(d,a.nextSibling):(d),d},y=function(e,f){for(var a=("link"),c=0;c