# OpenAI Responses API Compatibility

## Request Address

```json
https://api.xiaomimimo.com/v1/responses
```

## Request Headers

{/* feishu-style:text-align:left */}
The API supports the following two authentication methods. Please choose one and add it to the request headers:

```json
api-key: $MIMO_API_KEY
Content-Type: application/json
```

```json
Authorization: Bearer $MIMO_API_KEY
Content-Type: application/json
```

## Request Body

Currently, only the `mimo-v2.5` model supports image input.",
 "children": [
 {
 "name": "image_url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the input item.
Available options: `input_image`"
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
 "description": "The role of the message input.
Available options: `user`, `assistant`, `system`, `developer`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The type of the message input.
Available options: `message`"
 }
 ]
 },
 {
 "name": "Message",
 "type": "object",
 "isBold": false,
 "description": "A message input to the model with a role indicating instruction following hierarchy.",
 "children": [
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "required": true,
 "description": "A list of one or many input items to the model, containing different content types.",
 "children": [
 {
 "name": "ResponseInputText",
 "type": "object",
 "isBold": false,
 "description": "A text input to the model.",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The text input to the model."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the input item.
Available options: `input_text`"
 }
 ]
 },
 {
 "name": "ResponseInputImage",
 "type": "object",
 "isBold": false,
 "description": "An image input to the model.
Currently, only the `mimo-v2.5` model supports image input.",
 "children": [
 {
 "name": "image_url",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the input item.
Available options: `input_image`"
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
 "description": "The role of the message input.
Available options: `user`, `system`, `developer`"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The status of item. Populated when items are returned via API.
Available options: `in_progress`, `completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The type of the message input.
Available options: `message`"
 }
 ]
 },
 {
 "name": "ResponseOutputMessage",
 "type": "object",
 "isBold": false,
 "description": "An output message from the model.",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The unique ID of the output message."
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "required": true,
 "description": "The content of the output message.",
 "children": [
 {
 "name": "ResponseOutputText",
 "type": "object",
 "isBold": false,
 "description": "A text output from the model.",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The text output from the model."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the output text.
Available options: `output_text`"
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
 "description": "The role of the output message.
Available options: `assistant`"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The status of the message input. Populated when input items are returned via API.
Available options: `in_progress`, `completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the output message.
Available options: `message`"
 }
 ]
 },
 {
 "name": "FunctionCall",
 "type": "object",
 "isBold": false,
 "description": "A tool call to run a function.",
 "children": [
 {
 "name": "arguments",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "A JSON string of the arguments to pass to the function."
 },
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The unique ID of the function tool call generated by the model."
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The name of the function to run."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the function tool call.
Available options: `function_call`"
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The unique ID of the function tool call."
 },
 {
 "name": "namespace",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The namespace of the function to run."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The status of the item. Populated when items are returned via API.
Available options: `in_progress`, `completed`"
 }
 ]
 },
 {
 "name": "FunctionCallOutput",
 "type": "object",
 "isBold": false,
 "description": "The output of a function tool call.",
 "children": [
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The unique ID of the function tool call generated by the model."
 },
 {
 "name": "output",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "A JSON string of the output of the function tool call."
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The unique ID of the function tool call output. Populated when this item is returned via API."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The status of the item. Populated when items are returned via API.
Available options: `in_progress`, `completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the function tool call output.
Available options: `function_call_output`"
 }
 ]
 },
 {
 "name": "Reasoning",
 "type": "object",
 "isBold": false,
 "description": "A description of the chain of thought used by a reasoning model while generating a response.",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The unique identifier of the reasoning content."
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "Reasoning text content.",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The reasoning text from the model."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the object.
Available options: `reasoning_text`"
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the object.
Available options: `reasoning`"
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "The status of the item. Populated when items are returned via API.
Available options: `in_progress`, `completed`"
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
 "description": "A system (or developer) message inserted into the model's context."
 },
 {
 "name": "max_output_tokens",
 "type": "integer",
 "isBold": true,
 "required": false,
 "description": "An upper bound for the number of tokens that can be generated for a response, including visible output tokens and reasoning tokens.
- `mimo-v2.5-pro`: default `131072`
- `mimo-v2.5`: default `32768`
Required range: `[1, 131072]`"
 },
 {
 "name": "model",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Model ID used to generate the response.
Available options: `mimo-v2.5-pro`, `mimo-v2.5`"
 },
 {
 "name": "stream",
 "type": "boolean",
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "If set to true, the model response data will be streamed to the client as it is generated using server-sent events."
 },
 {
 "name": "reasoning",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "Configuration options for reasoning models.
Note: During the multi-turn tool calls process in thinking mode, the model returns the reasoning content alongside the tool calls field. To continue the conversation, it is recommended to keep all previous reasoning content in the `input` array for each subsequent request to achieve the best performance.In thinking mode, the `mimo-v2.5-pro`, `mimo-v2.5` models do not support customizing the `temperature` and `top_p` parameters. Even if these parameters are passed in, the actual effective values will be forcibly set by the model to its recommended default values of `1.0` and `0.95`.",
 "children": [
 {
 "name": "effort",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Constrains effort on reasoning for reasoning models. Reducing reasoning effort can result in faster responses and fewer tokens used on reasoning in a response.
Custom tuning of reasoning effort is currently unsupported. When set to `none`, reasoning is disabled; all other valid values map to enabled reasoning.- `mimo-v2.5-pro`, `mimo-v2.5`: default `enabled`
Available options: `none`, `low`, `medium`, `high`"
 }
 ]
 },
 {
 "name": "temperature",
 "type": "number",
 "isBold": true,
 "required": false,
 "description": "What sampling temperature to use, between 0 and 1.5. Higher values like 0.8 will make the output more random, while lower values like 0.2 will make it more focused and deterministic. We generally recommend altering this or `top_p` but not both.
In thinking mode, the `mimo-v2.5-pro`, `mimo-v2.5` models do not support customizing the `temperature` parameter. Even if this parameter is passed in, it will be forcibly overridden and take effect with the model's recommended default value of `1.0`.- `mimo-v2.5-pro`, `mimo-v2.5`: default `1.0`
Required range: `[0, 1.5]`"
 },
 {
 "name": "text",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "Configuration options for a text response from the model. Can be plain text or structured JSON data.",
 "children": [
 {
 "name": "format",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "An object specifying the format that the model must output. The default format is `{ "type": "text" }` with no additional options.",
 "children": [
 {
 "name": "ResponseFormatText",
 "type": "object",
 "isBold": false,
 "description": "Default response format. Used to generate text responses.",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of response format being defined.
Available options: `text`"
 }
 ]
 },
 {
 "name": "ResponseFormatJSONObject",
 "type": "object",
 "isBold": false,
 "description": "JSON object response format.
Note: If the output of JSON is not required by system instructions or user instructions, the model will not actively generate JSON.",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of response format being defined.
Available options: `json_object`"
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
 "description": "Controls how the model calls tools.
Note: When a value other than `auto` is passed to `tool_choice`, the backend will remove this field by default, and the model response behavior will still be equivalent to the `auto` mode (this logic is subject to future adjustments).Available options: `auto`"
 },
 {
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "An array of tools the model may call while generating a response. You can specify which tool to use by setting the `tool_choice` parameter.
Note: During the multi-turn tool calls process in thinking mode, the model returns the reasoning content alongside the tool calls field. To continue the conversation, it is recommended to keep all previous reasoning content in the `input` array for each subsequent request to achieve the best performance.",
 "children": [
 {
 "name": "Function",
 "type": "object",
 "isBold": false,
 "description": "Defines a function in your own code the model can choose to call.",
 "children": [
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The name of the tool function. Must be `a-z`, `A-Z`, `0-9`, or contain underscores (`_`) and dashes (`-`), with a maximum length of 64.
Required string length: `1 - 64`"
 },
 {
 "name": "parameters",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "A JSON schema object describing the parameters of the function."
 },
 {
 "name": "strict",
 "type": "boolean",
 "isBold": true,
 "required": true,
 "defaultValue": "false",
 "description": "Whether to enable strict schema adherence when generating the function call."
 },
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "required": false,
 "description": "A description of the function. Used by the model to determine whether or not to call the function."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the tool.
Available options: `function`"
 }
 ]
 },
 {
 "name": "Namespace",
 "type": "object",
 "isBold": false,
 "description": "Groups function tools under a shared namespace.",
 "children": [
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "A description of the namespace shown to the model.
MinLength: `1`"
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The namespace name used in tool calls.
MinLength: `1`"
 },
 {
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": true,
 "description": "The function tools available inside this namespace.",
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
 "description": "Required string length: `1 - 64`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Available options: `function`"
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
 "description": "The type of the tool.
Available options: `namespace`"
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
 "description": "An alternative to sampling with temperature, called nucleus sampling. We generally recommend altering this or `temperature` but not both.
In thinking mode, the `mimo-v2.5-pro`, `mimo-v2.5` models do not support customizing the `top_p` parameter. Even if this parameter is passed in, it will be forcibly overridden and take effect with the model's recommended default value of `0.95`.Required range: `[0.01, 1.0]`"
 }
]`} />

## Response Object (non-streaming output) 

- The length and order of items in the `output` array is dependent on the model’s response.
- Rather than accessing the first item in the `output` array and assuming it’s an `assistant` message with the content generated by the model, you might consider using the `output_text` property where supported in SDKs.
",
 "children": [
 {
 "name": "ResponseOutputMessage",
 "type": "object",
 "isBold": false,
 "description": "A message output from the model.",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "The unique ID of the output message."
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "description": "The content of the output message.",
 "children": [
 {
 "name": "ResponseOutputText",
 "type": "object",
 "isBold": false,
 "description": "A text output from the model.",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "description": "The text output from the model."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "Always `output_text`."
 }
 ]
 }
 ]
 },
 {
 "name": "role",
 "type": "string",
 "isBold": true,
 "description": "The role of the output message. Always `assistant`."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "The status of the message.
Available options: `in_progress`, `completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "Always `message`."
 }
 ]
 },
 {
 "name": "FunctionCall",
 "type": "object",
 "isBold": false,
 "description": "A tool call to a function tool.",
 "children": [
 {
 "name": "arguments",
 "type": "string",
 "isBold": true,
 "description": "The arguments that the model generated for the tool call, as a JSON string."
 },
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "description": "An identifier used when responding to the tool call with output."
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "description": "The name of the tool to call."
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "The unique ID of the tool call."
 },
 {
 "name": "namespace",
 "type": "string",
 "isBold": true,
 "description": "The namespace of the function to run."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "The status of the item.
Available options: `in_progress`, `completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "Always `function_call`."
 }
 ]
 },
 {
 "name": "FunctionCallOutput",
 "type": "object",
 "isBold": false,
 "description": "The output of a function tool call.",
 "children": [
 {
 "name": "call_id",
 "type": "string",
 "isBold": true,
 "description": "The unique ID of the function tool call generated by the model."
 },
 {
 "name": "output",
 "type": "string",
 "isBold": true,
 "description": "A JSON string of the output of the function tool call."
 },
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "The unique ID of the function tool call output. Populated when this item is returned via API."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "The status of the item.
Available options: `in_progress`, `completed`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "The type of the function tool call output.
Always `function_call_output`."
 }
 ]
 },
 {
 "name": "Reasoning",
 "type": "object",
 "isBold": false,
 "description": "A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your input to the Responses API for subsequent turns of a conversation if you are manually managing context.",
 "children": [
 {
 "name": "id",
 "type": "string",
 "isBold": true,
 "description": "The unique identifier of the reasoning content."
 },
 {
 "name": "content",
 "type": "array",
 "isBold": true,
 "description": "Reasoning text content.",
 "children": [
 {
 "name": "text",
 "type": "string",
 "isBold": true,
 "description": "The reasoning text from the model."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "The type of the reasoning text.
Always `reasoning_text`."
 }
 ]
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "The type of the object.
Always `reasoning`."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "The status of the item.
Available options: `in_progress`, `completed`"
 }
 ]
 }
 ]
 },
 {
 "name": "output_text",
 "type": "string",
 "isBold": true,
 "description": "SDK-only convenience property that contains the aggregated text output from all `output_text` items in the `output` array, if any are present."
 },
 {
 "name": "status",
 "type": "string",
 "isBold": true,
 "description": "The status of the response.
Available options: `completed`, `in_progress`, `incomplete`"
 },
 {
 "name": "usage",
 "type": "object",
 "isBold": true,
 "description": "Usage statistics for the response.",
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
 "description": "The number of input tokens."
 },
 {
 "name": "input_tokens_details",
 "type": "object",
 "isBold": true,
 "description": "Details about input tokens.",
 "children": [
 {
 "name": "cached_tokens",
 "type": "integer",
 "isBold": true,
 "description": "The number of cached input tokens."
 }
 ]
 },
 {
 "name": "output_tokens",
 "type": "integer",
 "isBold": true,
 "description": "The number of output tokens."
 },
 {
 "name": "output_tokens_details",
 "type": "object",
 "isBold": true,
 "description": "Details about output tokens.",
 "children": [
 {
 "name": "reasoning_tokens",
 "type": "integer",
 "isBold": true,
 "description": "The number of reasoning tokens."
 }
 ]
 },
 {
 "name": "total_tokens",
 "type": "integer",
 "isBold": true,
 "description": "The total number of tokens."
 }
 ]
 }
 ]
 }
]`} />

## Response chunk object (streaming output) 

{/* feishu-style:text-align:left */}
When you create a Response with `stream` set to `true`, the server will emit server-sent events to the client as the Response is generated.

### response.created

> An event that is emitted when a response is created.

### response.in_progress

> Emitted when the response is in progress.

### response.completed

> Emitted when the model response is complete.

### response.incomplete

> An event that is emitted when a response finishes as incomplete.

### response.output_item.added

> Emitted when a new output item is added.

### response.output_item.done

> Emitted when an output item is marked done. 

### response.content_part.added

> Emitted when a new content part is added.

### response.content_part.done

> Emitted when a content part is done.

### response.output_text.delta

> Emitted when there is an additional text delta.

### response.output_text.done

> Emitted when text content is finalized.

### response.function_call_arguments.delta

> Emitted when there is a partial function-call arguments delta.

### response.function_call_arguments.done

> Emitted when function-call arguments are finalized.

### response.reasoning_text.delta

> Emitted when a delta is added to a reasoning text.

### response.reasoning_text.done

> Emitted when a reasoning text is completed.