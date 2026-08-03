# Anthropic API（zh-CN 官方）

> 来源: 官方文档 zh-CN

## 请求地址

```
https://api.xiaomimimo.com/anthropic/v1/messages
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

仅 `mimo-v2.5` 模型支持图像输入。",
 "children": [
 {
 "name": "Text",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "文本块的内容。
最小值：`1`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`text`"
 }
 ]
 },
 {
 "name": "Image",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "source",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "图像数据通过 URL 或 Base64 提供。",
 "children": [
 {
 "name": "Base64ImageSource",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "data",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Base64 编码的图像数据。"
 },
 {
 "name": "media_type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "媒体类型。
可选值：`image/jpeg`，`image/png`，`image/gif`，`image/webp`，`image/bmp`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像源类型。
可选值：`base64`"
 }
 ]
 },
 {
 "name": "URLImageSource",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像的 URL。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像源类型。
可选值：`url`"
 }
 ]
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`image`"
 }
 ]
 },
 {
 "name": "Tool use",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具使用的唯一标识符。"
 },
 {
 "name": "input",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "使用工具时传入的参数对象。"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具名称。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`tool_use`"
 }
 ]
 },
 {
 "name": "Tool result",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "tool_use_id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "与本次结果对应的 `tool_use` 的 ID。"
 },
 {
 "name": "content",
 "type": [
 "string",
 "array"
 ],
 "isBold": true,
 "description": "工具执行后返回的结果。",
 "children": [
 {
 "name": "Text content",
 "type": "string",
 "isBold": false,
 "description": "消息的文本内容。"
 },
 {
 "name": "Array of content parts",
 "type": "array",
 "isBold": false,
 "description": "一个包含多个具有特定类型的内容部分的数组。例如，文本和图像。",
 "children": [
 {
 "name": "Text",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "文本块的内容。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`text`"
 }
 ]
 },
 {
 "name": "Image",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "source",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "图像数据通过 URL 或 Base64 提供。",
 "children": [
 {
 "name": "Base64ImageSource",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "data",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Base64 编码的图像数据。"
 },
 {
 "name": "media_type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "媒体类型。
可选值：`image/jpeg`，`image/png`，`image/gif`，`image/webp`，`image/bmp`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像源类型。
可选值：`base64`"
 }
 ]
 },
 {
 "name": "URLImageSource",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像的 URL。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "图像源类型。
可选值：`url`"
 }
 ]
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`image`"
 }
 ]
 }
 ]
 }
 ]
 },
 {
 "name": "is_error",
 "type": "boolean",
 "isBold": true
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`tool_result`"
 }
 ]
 },
 {
 "name": "Thinking",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "signature",
 "type": "string",
 "isBold": true,
 "description": "思考块的签名。"
 },
 {
 "name": "thinking",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "思考内容。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`thinking`"
 }
 ]
 }
 ]
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
 "description": "使用的模型名称。
可选值：`mimo-v2.5-pro`，`mimo-v2.5`"
},
{
 "name": "max_tokens",
 "type": "integer",
 "isBold": true,
 "required": false,
 "description": "停止前生成的最大 token 数。
请注意，我们的模型可能在达到此最大值之前就停止。此参数仅指定要生成的绝对最大 token 数。
- `mimo-v2.5-pro` 的默认值 `131072`
- `mimo-v2.5` 的默认值为 `32768`
所需范围：`[1, 131072]`"
},
{
 "name": "stop_sequences",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "使模型停止生成的自定义文本序列。
我们的模型通常会在自然完成一轮对话后停止，这将导致响应的 `stop_reason` 为 `end_turn`。
如果您希望模型在遇到自定义文本字符串时停止生成，可以使用 `stop_sequences` 参数。"
},
{
 "name": "stream",
 "type": "boolean",
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "是否以流式输出方式回复。"
},
{
 "name": "system",
 "type": [
 "string",
 "array"
 ],
 "isBold": true,
 "required": false,
 "description": "系统提示词是向模型提供上下文与指令的一种方式，例如为模型指定特定目标或角色。",
 "children": [
 {
 "name": "Text content",
 "type": "string",
 "isBold": false,
 "description": "系统提示词的内容。"
 },
 {
 "name": "Array of content parts",
 "type": "array",
 "isBold": false,
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "文本内容。
最小长度：`1`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "内容的类型。
可选值：`text`"
 }
 ]
 }
 ]
},
{
 "name": "temperature",
 "type": "number",
 "isBold": true,
 "required": false,
 "description": "采样温度，控制模型生成文本的多样性。
`temperature` 越高，生成的文本更多样，反之，生成的文本更确定。
在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `temperature` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `1.0`。- `mimo-v2.5-pro`，`mimo-v2.5` 默认值为 `1.0`
所需范围：`[0, 1.5]`"
},
{
 "name": "thinking",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "启用模型扩展思维的配置。
注意：在思考模式下的多轮工具调用过程中，模型会在返回 `tool_use` 内容块的同时返回 `thinking` 内容块。若要继续对话，建议在后续每次请求的 `messages` 数组中保留所有历史 `thinking` 内容块，以获得最佳表现。在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `temperature` 和 `top_p` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `1.0` 和 `0.95`。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "- `mimo-v2.5-pro`，`mimo-v2.5` 默认值为 `enabled`
可选值：`enabled`，`disabled`"
 }
 ]
},
{
 "name": "tool_choice",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "控制模型如何使用提供的工具。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "- `auto` 意味着模型将自动决定是否使用工具。
注意：当 `type` 传入非 `auto` 值时，后端会默认移除该字段，模型响应行为仍等同于 `auto` 模式（该逻辑保留调整的可能性）。可选值：`auto`"
 },
 {
 "name": "disable_parallel_tool_use",
 "type": "boolean",
 "isBold": true,
 "defaultValue": "false",
 "description": "是否禁用并行工具使用。
如果设置为 true：
- 当类型为 `auto` 时，模型将输出至多一个工具使用。
"
 }
 ]
},
{
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "模型可能会使用的工具的定义。
如果在 API 请求中包含工具，则模型可能会返回 `tool_use` 内容块，表示模型对这些工具的使用。您可以使用模型生成的工具输入运行这些工具，然后选择性地返回结果给模型，使用 `tool_result` 内容块。
注意：在思考模式下的多轮工具调用过程中，模型会在返回 `tool_use` 内容块的同时返回 `thinking` 内容块。若要继续对话，建议在后续每次请求的 `messages` 数组中保留所有历史 `thinking` 内容块，以获得最佳表现。工具定义包括：
- `name`：工具的名称。
- `description`：可选，但强烈推荐填写工具描述。
- `input_schema`：工具输入形状的 JSON 模式，模型将在 `tool_use` 输出内容块中生成。
",
 "children": [
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具名称。
模型将通过它调用该工具，并是在 `tool_use` 块中使用的名称。"
 },
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "description": "工具的描述。
工具描述应尽可能详细。模型关于工具是什么以及如何使用的信息越多，执行表现就越好。您可以使用自然语言描述来强化工具输入 JSON 模式中的重要信息。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "可选值：`custom`"
 },
 {
 "name": "input_schema",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "工具输入形状的 JSON 模式，模型将在 `tool_use` 输出内容块中生成。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "`input_schema` 的类型，仅为 `object`。
可选值：`object`"
 },
 {
 "name": "properties",
 "type": [
 "object",
 "null"
 ],
 "isBold": true,
 "description": "工具输入的属性。"
 },
 {
 "name": "required",
 "type": [
 "array",
 "null"
 ],
 "isBold": true,
 "description": "工具输入中必须包含的属性列表。"
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
 "description": "启用核采样。
在核采样机制中，我们会按概率从高到低的顺序，为生成每个后续 token 的所有候选结果计算累积概率分布，当累积概率达到 `top_p` 参数指定的阈值时，便会截断后续候选。请注意，你应仅调整 `temperature` 或 `top_p` 二者其一，不可同时修改。
此采样方式仅建议用于高级使用场景。通常情况下，你只需调整 `temperature` 参数即可满足需求。
在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `top_p` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `0.95`。所需范围：`[0.01, 1.0]`"
}
]">

## 非流式响应

- `end_turn`：模型达到自然停止点。
- `max_tokens`：超过请求的 `max_tokens` 或模型的最大限制。
- `tool_use`：模型调用了一个或多个工具。
- `content_filter`：内容因触发过滤策略而被拦截。
- `repetition_truncation`：模型检测到了复读。
可选值：`end_turn`，`max_tokens`，`tool_use`，`content_filter`，`repetition_truncation`"
},
{
 "name": "usage",
 "type": "object",
 "isBold": true,
 "description": "计费和限流相关的使用量统计。",
 "children": [
 {
 "name": "input_tokens",
 "type": "integer",
 "isBold": true,
 "description": "使用的输入 token 数量。"
 },
 {
 "name": "output_tokens",
 "type": "integer",
 "isBold": true,
 "description": "使用的输出 token 数量。"
 },
 {
 "name": "cache_read_input_tokens",
 "type": [
 "integer",
 "null"
 ],
 "isBold": true,
 "description": "从缓存读取的输入 token 数量。"
 }
 ]
}
]">

## 流式响应

curlpython基础调用流式响应函数调用图像输入深度思考curl --location --request POST 'https://api.xiaomimimo.com/anthropic/v1/messages' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "max_tokens": 1024,
 "system": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
 "messages": [
 {
 "role": "user",
 "content": [
 {
 "type": "text",
 "text": "please introduce yourself"
 }
 ]
 }
 ],
 "top_p": 0.95,
 "stream": false,
 "temperature": 1.0,
 "stop_sequences": null,
 "thinking": {
 "type": "disabled"
 }
}'响应基础调用流式响应函数调用图像输入深度思考{
 "id": "b966dbcad38c48b59d16d8c1f313681b",
 "type": "message",
 "role": "assistant",
 "model": "mimo-v2.5-pro",
 "stop_reason": "end_turn",
 "content": [
 {
 "type": "text",
 "text": "Hello! I'm MiMo, an AI assistant developed by Xiaomi. I'm here to help answer your questions, provide information, or assist with various tasks. My knowledge is up to date until December 2024. How can I help you today?"
 }
 ],
 "usage": {
 "input_tokens": 57,
 "output_tokens": 54
 }
}更新时间 2026 年 07 月 17 日OpenAI Responses API语音识别（MiMo‑V2.5-ASR）- OpenAI API 兼容目录curlpython基础调用流式响应函数调用图像输入深度思考curl --location --request POST 'https://api.xiaomimimo.com/anthropic/v1/messages' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
 "model": "mimo-v2.5-pro",
 "max_tokens": 1024,
 "system": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
 "messages": [
 {
 "role": "user",
 "content": [
 {
 "type": "text",
 "text": "please introduce yourself"
 }
 ]
 }
 ],
 "top_p": 0.95,
 "stream": false,
 "temperature": 1.0,
 "stop_sequences": null,
 "thinking": {
 "type": "disabled"
 }
}'响应基础调用流式响应函数调用图像输入深度思考{
 "id": "b966dbcad38c48b59d16d8c1f313681b",
 "type": "message",
 "role": "assistant",
 "model": "mimo-v2.5-pro",
 "stop_reason": "end_turn",
 "content": [
 {
 "type": "text",
 "text": "Hello! I'm MiMo, an AI assistant developed by Xiaomi. I'm here to help answer your questions, provide information, or assist with various tasks. My knowledge is up to date until December 2024. How can I help you today?"
 }
 ],
 "usage": {
 "input_tokens": 57,
 "output_tokens": 54
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