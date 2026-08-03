# OpenAI Responses API（zh-CN 官方）

> 来源: 官方文档 zh-CN

## 请求地址

```
https://api.xiaomimimo.com/v1/responses
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

当前仅 `mimo-v2.5` 模型支持图像输入。",
 "children": [
 {
 "name": "image_url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "发送至模型的图片地址，可为完整可访问 URL，或 Base64 编码的 data URL 格式图片。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "输入项的类型。
可选值：`input_image`"
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
 "description": "消息输入的角色。
可选值：`user`，`assistant`，`system`，`developer`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "消息输入的类型。
可选值：`message`"
 }
 ]
 },
 {
 "name": "Message",
 "type": "object",
 "isBold": false,
 "description": "带有角色标识指令优先级层级的模型消息输入。",
 "children": [
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "required": true,
 "description": "模型的一个或多个输入项列表，可包含多种内容类型。",
 "children": [
 {
 "name": "ResponseInputText",
 "type": "object",
 "isBold": false,
 "description": "模型的文本输入。",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "向模型输入的文本内容。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "输入项的类型。
可选值：`input_text`"
 }
 ]
 },
 {
 "name": "ResponseInputImage",
 "type": "object",
 "isBold": false,
 "description": "模型的图像输入。
当前仅 `mimo-v2.5` 模型支持图像输入。",
 "children": [
 {
 "name": "image_url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "发送至模型的图片地址，可为完整可访问 URL，或 Base64 编码的 data URL 格式图片。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "输入项的类型。
可选值：`input_image`"
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
 "description": "消息输入的角色。
可选值：`user`，`system`，`developer`"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "项目状态，通过 API 返回数据时填充。
可选值：`in_progress`，`completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "消息输入的类型。
可选值：`message`"
 }
 ]
 },
 {
 "name": "ResponseOutputMessage",
 "type": "object",
 "isBold": false,
 "description": "模型返回的输出消息。",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "输出消息的唯一标识 ID。"
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "required": true,
 "description": "输出消息的内容。",
 "children": [
 {
 "name": "ResponseOutputText",
 "type": "object",
 "isBold": false,
 "description": "模型输出的文本内容。",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "模型输出的文本。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "输出文本类型。
可选值：`output_text`"
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
 "description": "输出消息的角色。
可选值：`assistant`"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "消息状态，通过 API 返回输入项时填充。
可选值：`in_progress`，`completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "输出消息类型。
可选值：`message`"
 }
 ]
 },
 {
 "name": "FunctionCall",
 "type": "object",
 "isBold": false,
 "description": "用于执行函数的工具调用。",
 "children": [
 {
 "name": "arguments",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "传递给函数的参数（JSON 字符串格式）。"
 },
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "模型生成的函数工具调用唯一标识。"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "要执行的函数名称。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "函数工具调用类型。
可选值：`function_call`"
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "函数工具调用的唯一标识。"
 },
 {
 "name": "namespace",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "要执行的函数所属命名空间。"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "条目状态，通过 API 返回数据时填充。
可选值：`in_progress`，`completed`"
 }
 ]
 },
 {
 "name": "FunctionCallOutput",
 "type": "object",
 "isBold": false,
 "description": "函数工具调用的返回结果。",
 "children": [
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "模型生成的函数工具调用唯一标识。"
 },
 {
 "name": "output",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "函数工具调用的输出结果（JSON 字符串格式）。"
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "函数工具调用输出的唯一标识，API 返回该字段时填充。"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "条目状态，API 返回该字段时填充。
可选值：`in_progress`，`completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "函数工具调用输出的类型。
可选值：`function_call_output`"
 }
 ]
 },
 {
 "name": "Reasoning",
 "type": "object",
 "isBold": false,
 "description": "推理模型生成响应时的思维过程描述。",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "推理内容的唯一标识。"
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "推理文本内容。",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "模型输出的推理文本。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "对象类型。
可选值：`reasoning_text`"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "对象类型。
可选值：`reasoning`"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "条目状态，API 返回数据时填充。
可选值：`in_progress`，`completed`"
 }
 ]
 }
 ]
 }
 ]
},
{
 "name": "instructions",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "注入模型上下文的系统（或开发者）指令。"
},
{
 "name": "max_output_tokens",
 "type": "integer",
 "isBold": true,
 "required": false,
 "description": "响应可生成的 token 数的上限，包含可见输出 token 数和推理 token 数。
- `mimo-v2.5-pro` 的默认值 `131072`
- `mimo-v2.5` 的默认值为 `32768`
所需范围：`[1, 131072]`"
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
 "name": "stream",
 "type": "boolean",
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "设为 `true` 时，模型响应将通过服务端推送事件（SSE）流式实时返回给客户端。"
},
{
 "name": "reasoning",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "推理模型相关配置项。
注意：在思考模式下的多轮工具调用过程中，模型会在返回工具调用字段的同时返回思考内容。若要继续对话，建议在后续每次请求的 `input` 数组中保留所有历史思考内容，以获得最佳表现。在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `temperature` 和 `top_p` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `1.0` 和 `0.95`。",
 "children": [
 {
 "name": "effort",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "控制推理模型的思考投入强度。降低推理投入可加快响应速度、减少推理环节消耗的 token。
当前暂不支持自定义调节推理投入档位：参数设为 `none` 时关闭推理，其余合法取值均开启推理。- `mimo-v2.5-pro`，`mimo-v2.5` 默认值为 `enabled`
可选值：`none`, `low`, `medium`, `high`"
 }
 ]
},
{
 "name": "temperature",
 "type": "number",
 "isBold": true,
 "required": false,
 "description": "采样温度，介于 0 和 1.5 之间。较高的值（如 0.8）会让输出更随机；较低的值（如 0.2）会让输出更集中、确定性更强。通常建议仅调整该参数或 `top_p` 其中一项，不要同时修改。
在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `temperature` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `1.0`。- `mimo-v2.5-pro`，`mimo-v2.5` 默认值为 `1.0`
所需范围：`[0, 1.5]`"
},
{
 "name": "text",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "模型文本响应的配置项，支持纯文本或结构化 JSON 数据输出。",
 "children": [
 {
 "name": "format",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "用于指定模型必须输出的格式对象。默认格式为 `{"type": "text"}`，无额外配置项。",
 "children": [
 {
 "name": "ResponseFormatText",
 "type": "object",
 "isBold": false,
 "description": "默认响应格式，用于生成文本类回复。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "定义的响应格式类型。
可选值：`text`"
 }
 ]
 },
 {
 "name": "ResponseFormatJSONObject",
 "type": "object",
 "isBold": false,
 "description": "JSON 对象响应格式。
注意：若未通过系统指令或用户指令要求输出 JSON，模型不会主动生成 JSON。",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "定义的响应格式类型。
可选值：`json_object`"
 }
 ]
 }
 ]
 }
 ]
},
{
 "name": "tool_choice",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "控制模型调用工具的方式。
注意：当 `tool_choice` 传入非 `auto` 值时，后端会默认移除该字段，模型响应行为仍等同于 `auto` 模式（该逻辑保留调整的可能性）。可选值：`auto`"
},
{
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "模型生成响应时可调用的工具数组。可通过 `tool_choice` 参数指定使用的工具。
注意：在思考模式下的多轮工具调用过程中，模型会在返回工具调用字段的同时返回思考内容。若要继续对话，建议在后续每次请求的 `input` 数组中保留所有历史思考内容，以获得最佳表现。",
 "children": [
 {
 "name": "Function",
 "type": "object",
 "isBold": false,
 "description": "在你自己的代码中定义一个模型可以选择调用的函数。",
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
 "name": "parameters",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "描述函数参数的 JSON 模式对象。"
 },
 {
 "name": "strict",
 "type": "boolean",
 "isBold": true,
 "required": true,
 "defaultValue": "false",
 "description": "生成函数调用时是否启用严格遵循参数模式校验。"
 },
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "函数描述，供模型判断是否调用该函数。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具类型。
可选值：`function`"
 }
 ]
 },
 {
 "name": "Namespace",
 "type": "object",
 "isBold": false,
 "description": "将功能工具归在一个共享的命名空间下。",
 "children": [
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "展示给模型的命名空间描述。
最小长度：`1`"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "工具调用时使用的命名空间名称。
最小长度：`1`"
 },
 {
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": true,
 "description": "该命名空间内可用的函数工具。",
 "children": [
 {
 "name": "Function",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "所需字符串长度：`1 - 64`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "可选值：`function`"
 },
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "required": false
 },
 {
 "name": "parameters",
 "type": "object",
 "isBold": true,
 "required": false
 },
 {
 "name": "strict",
 "type": "boolean",
 "isBold": true,
 "required": false
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
 "description": "工具类型。
可选值：`namespace`"
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
 "description": "核采样，是温度采样的替代方案。通常建议仅调整该参数或 `temperature` 其中一项，不要同时修改。
在思考模式下，`mimo-v2.5-pro`、`mimo-v2.5` 模型不支持自定义 `top_p` 参数。即使传入该参数，实际生效值也会被模型强制采用其推荐默认值 `0.95`。所需范围：`[0.01, 1.0]`"
}
]">

## Response 对象（非流式输出）

- 输出数组内元素的长度与顺序由模型响应决定。
- 不建议直接取数组第一项并默认其为助手消息；在支持的 SDK 中，可优先使用 `output_text` 属性获取内容。
",
 "children": [
 {
 "name": "ResponseOutputMessage",
 "type": "object",
 "isBold": false,
 "description": "模型输出的消息。",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "输出消息的唯一标识。"
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "description": "输出消息的内容。",
 "children": [
 {
 "name": "ResponseOutputText",
 "type": "object",
 "isBold": false,
 "description": "模型输出的文本内容。",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "description": "模型输出的文本。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "仅为 `output_text`。"
 }
 ]
 }
 ]
 },
 {
 "name": "role",
 "type": "string",
 "isBold": true,
 "description": "输出消息的角色，仅为 `assistant`。"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "消息状态。
可选值：`in_progress`，`completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "仅为 `message`。"
 }
 ]
 },
 {
 "name": "FunctionCall",
 "type": "object",
 "isBold": false,
 "description": "函数工具的调用指令。",
 "children": [
 {
 "name": "arguments",
 "type": "string",
 "isBold": true,
 "description": "模型为工具调用生成的参数，格式为 JSON 字符串。"
 },
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "description": "用于回传工具调用结果时的标识。"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "description": "待调用的工具名称。"
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "该工具调用的唯一标识。"
 },
 {
 "name": "namespace",
 "type": "string",
 "isBold": true,
 "description": "待执行函数所属的命名空间。"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "条目状态。
可选值：`in_progress`，`completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "仅为 `function_call`。"
 }
 ]
 },
 {
 "name": "FunctionCallOutput",
 "type": "object",
 "isBold": false,
 "description": "函数工具调用的输出结果。",
 "children": [
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "description": "模型生成的函数工具调用唯一标识。"
 },
 {
 "name": "output",
 "type": "string",
 "isBold": true,
 "description": "函数工具调用的输出内容，为 JSON 字符串格式。"
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "函数工具调用输出的唯一标识。"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "条目状态。
可选值：`in_progress`，`completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "函数工具调用输出的类型。
仅为 `function_call_output`。"
 }
 ]
 },
 {
 "name": "Reasoning",
 "type": "object",
 "isBold": false,
 "description": "推理模型生成响应过程中使用的思维链描述。",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "推理内容的唯一标识。"
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "description": "推理文本内容。",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "description": "模型输出的推理文本。"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "推理文本的类型。
仅为 `reasoning_text`。"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "对象类型。
仅为 `reasoning`。"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "条目状态。
可选值：`in_progress`，`completed`"
 }
 ]
 }
 ]
},
{
 "name": "output_text",
 "type": "string",
 "isBold": true,
 "description": "仅 SDK 提供的便捷属性，用于聚合输出数组中所有 `output_text` 项的文本内容（存在时返回）。"
},
{
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "响应状态。
可选值：`completed`，`in_progress`，`incomplete`"
},
{
 "name": "usage",
 "type": "object",
 "isBold": true,
 "description": "本次响应的用量统计信息。",
 "children": [
 {
 "name": "ResponseUsage",
 "type": "object",
 "isBold": false,
 "children": [
 {
 "name": "input_tokens",
 "type": "integer",
 "isBold": true,
 "description": "输入 token 数量。"
 },
 {
 "name": "input_tokens_details",
 "type": "object",
 "isBold": true,
 "description": "输入 token 详情。",
 "children": [
 {
 "name": "cached_tokens",
 "type": "integer",
 "isBold": true,
 "description": "缓存命中的输入 token 数量。"
 }
 ]
 },
 {
 "name": "output_tokens",
 "type": "integer",
 "isBold": true,
 "description": "输出 token 数量。"
 },
 {
 "name": "output_tokens_details",
 "type": "object",
 "isBold": true,
 "description": "输出 token 详情。",
 "children": [
 {
 "name": "reasoning_tokens",
 "type": "integer",
 "isBold": true,
 "description": "推理过程消耗的 token 数量。"
 }
 ]
 },
 {
 "name": "total_tokens",
 "type": "integer",
 "isBold": true,
 "description": "总 token 数量。"
 }
 ]
 }
 ]
}
]">

## Response chunk 对象（流式输出）

当你创建响应并将 `stream` 设为 `true` 时，服务端会在响应生成过程中向客户端推送服务端发送事件（SSE）。

### response.created

响应创建时触发的事件。

### response.in_progress

响应处于生成中状态时触发。

### response.completed

模型响应生成完成时触发。

### response.incomplete

响应以未完成状态结束时触发该事件。

### response.output_item.added

当新增一条输出项时触发。

### response.output_item.done

当某条输出项标记为完成时触发。

### response.content_part.added

新增内容片段时触发。

### response.content_part.done

内容片段生成完成时触发。

### response.output_text.delta

产生增量文本片段时触发。

### response.output_text.done

文本内容最终生成完成时触发。

### response.function_call_arguments.delta

产生函数调用参数增量片段时触发。

### response.function_call_arguments.done

函数调用参数最终生成完成时触发。

### response.reasoning_text.delta

推理文本产生增量片段时触发。

### response.reasoning_text.done

推理文本生成完成时触发。

curlpython基础调用流式响应函数调用图像输入深度思考结构化输出curl --location --request POST 'https://api.xiaomimimo.com/v1/responses' \
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
}'响应基础调用流式响应函数调用图像输入深度思考结构化输出{
 "id": "resp_5fcaac1af26a4b449f30e1eeeaa8c48f",
 "object": "response",
 "created_at": 1782215659,
 "status": "completed",
 "error": null,
 "incomplete_details": null,
 "model": "mimo-v2.5-pro",
 "metadata": null,
 "output": [
 {
 "id": "msg_1fa8f5011ebf47adb158ecf13c99d06c",
 "type": "message",
 "status": "completed",
 "role": "assistant",
 "content": [
 {
 "type": "output_text",
 "text": "Hello! I am MiMo, a large language model developed by Xiaomi's LLM Core Team. I'm here to help answer your questions, generate text, and assist with various tasks. How can I assist you today?",
 "annotations": []
 }
 ]
 }
 ],
 "output_text": "Hello! I am MiMo, a large language model developed by Xiaomi's LLM Core Team. I'm here to help answer your questions, generate text, and assist with various tasks. How can I assist you today?",
 "usage": {
 "input_tokens": 57,
 "input_tokens_details": {},
 "output_tokens": 46,
 "output_tokens_details": {
 "reasoning_tokens": 0
 },
 "total_tokens": 103
 }
}更新时间 2026 年 07 月 17 日OpenAI Chat Completion APIAnthropic API目录curlpython基础调用流式响应函数调用图像输入深度思考结构化输出curl --location --request POST 'https://api.xiaomimimo.com/v1/responses' \
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
}'响应基础调用流式响应函数调用图像输入深度思考结构化输出{
 "id": "resp_5fcaac1af26a4b449f30e1eeeaa8c48f",
 "object": "response",
 "created_at": 1782215659,
 "status": "completed",
 "error": null,
 "incomplete_details": null,
 "model": "mimo-v2.5-pro",
 "metadata": null,
 "output": [
 {
 "id": "msg_1fa8f5011ebf47adb158ecf13c99d06c",
 "type": "message",
 "status": "completed",
 "role": "assistant",
 "content": [
 {
 "type": "output_text",
 "text": "Hello! I am MiMo, a large language model developed by Xiaomi's LLM Core Team. I'm here to help answer your questions, generate text, and assist with various tasks. How can I assist you today?",
 "annotations": []
 }
 ]
 }
 ],
 "output_text": "Hello! I am MiMo, a large language model developed by Xiaomi's LLM Core Team. I'm here to help answer your questions, generate text, and assist with various tasks. How can I assist you today?",
 "usage": {
 "input_tokens": 57,
 "input_tokens_details": {},
 "output_tokens": 46,
 "output_tokens_details": {
 "reasoning_tokens": 0
 },
 "total_tokens": 103
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