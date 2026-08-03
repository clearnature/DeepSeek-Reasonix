# Anthropic Messages API Compatibility

## Request Address

```bash
https://api.xiaomimimo.com/anthropic/v1/messages
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

Only the `mimo-v2.5` model supports image input.",
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
 "description": "The content of the text block.
Minimum length: `1`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the content.
Available options: `text`"
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
 "description": "Image data is provided via URL or Base64.",
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
 "description": "Base64 encoded image data."
 },
 {
 "name": "media_type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Media type.
Available options: `image/jpeg`, `image/png`, `image/gif`, `image/webp`, `image/bmp`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Image source type.
Available options: `base64`"
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
 "description": "A URL of the image."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Image source type.
Available options: `url`"
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
 "description": "The type of the content.
Available options: `image`"
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
 "description": "The unique identifier for tool use."
 },
 {
 "name": "input",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "The parameter object passed when using the tool."
 },
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Tool name."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the content.
Available options: `tool_use`"
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
 "description": "The `tool_use` ID corresponding to this result."
 },
 {
 "name": "content",
 "type": [
 "string",
 "array"
 ],
 "isBold": true,
 "description": "The result returned after the tool is executed.",
 "children": [
 {
 "name": "Text content",
 "type": "string",
 "isBold": false,
 "description": "The text contents of the message."
 },
 {
 "name": "Array of content parts",
 "type": "array",
 "isBold": false,
 "description": "An array of content parts with a defined type. Such text and image.",
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
 "description": "The content of the text block."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the content.
Available options: `text`"
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
 "description": "Image data is provided via URL or Base64.",
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
 "description": "Base64 encoded image data."
 },
 {
 "name": "media_type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Media type.
Available options: `image/jpeg`, `image/png`, `image/gif`, `image/webp`, `image/bmp`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Image source type.
Available options: `base64`"
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
 "description": "A URL of the image."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Image source type.
Available options: `url`"
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
 "description": "The type of the content.
Available options: `image`"
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
 "description": "The type of the content.
Available options: `tool_result`"
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
 "description": "The signature of the thinking block."
 },
 {
 "name": "thinking",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Thinking content."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the content.
Available options: `thinking`"
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
 "description": "The model that will complete your prompt.
Available options: `mimo-v2.5-pro`, `mimo-v2.5`"
 },
 {
 "name": "max_tokens",
 "type": "integer",
 "isBold": true,
 "required": false,
 "description": "The maximum number of tokens to generate before stopping.
Note that our models may stop before reaching this maximum. This parameter only specifies the absolute maximum number of tokens to generate.
- `mimo-v2.5-pro`: default `131072`
- `mimo-v2.5`: default `32768`
Required range: `[1, 131072]`"
 },
 {
 "name": "stop_sequences",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "Custom text sequences that will cause the model to stop generating.
Our models will normally stop when they have naturally completed their turn, which will result in a response `stop_reason` of `end_turn`.
If you want the model to stop generating when it encounters custom strings of text, you can use the `stop_sequences` parameter."
 },
 {
 "name": "stream",
 "type": "boolean",
 "isBold": true,
 "required": false,
 "defaultValue": "false",
 "description": "Whether to incrementally stream the response using server-sent events."
 },
 {
 "name": "system",
 "type": [
 "string",
 "array"
 ],
 "isBold": true,
 "required": false,
 "description": "A system prompt is a way of providing context and instructions to model, such as specifying a particular goal or role.",
 "children": [
 {
 "name": "Text content",
 "type": "string",
 "isBold": false,
 "description": "The content of the system prompt."
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
 "description": "The text content.
Minimum length: `1`"
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of the content.
Available options: `text`"
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
 "description": "Sampling temperature controls the diversity of the text generated by the model.
The higher the temperature, the more diverse the generated text will be; conversely, the lower the temperature, the more deterministic the generated text will be.
In thinking mode, the `mimo-v2.5-pro`, `mimo-v2.5` models do not support customizing the `temperature` parameter. Even if this parameter is passed in, it will be forcibly overridden and take effect with the model's recommended default value of `1.0`.- `mimo-v2.5-pro`, `mimo-v2.5`: default `1.0`
Required range: `[0, 1.5]`"
 },
 {
 "name": "thinking",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "Configuration for enabling model's extended thinking.
Note: During the multi-turn tool calls process in thinking mode, the model returns a `thinking` content block alongside `tool_use` content block. To continue the conversation, it is recommended to keep all previous `thinking` content block in the `messages` array for each subsequent request to achieve the best performance.In thinking mode, the `mimo-v2.5-pro`, `mimo-v2.5` models do not support customizing the `temperature` and `top_p` parameters. Even if these parameters are passed in, the actual effective values will be forcibly set by the model to its recommended default values of `1.0` and `0.95`.",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "- `mimo-v2.5-pro`, `mimo-v2.5`: default `enabled`
Available options: `enabled`, `disabled`"
 }
 ]
 },
 {
 "name": "tool_choice",
 "type": "object",
 "isBold": true,
 "required": false,
 "description": "How the model should use the provided tools.",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "- `auto` means the model will automatically decide whether to use tools.
Note: When a value other than `auto` is passed to `type`, the backend will remove this field by default, and the model response behavior will still be equivalent to the `auto` mode (this logic is subject to future adjustments).Available options: `auto`"
 },
 {
 "name": "disable_parallel_tool_use",
 "type": "boolean",
 "isBold": true,
 "defaultValue": "false",
 "description": "Whether to disable parallel tool use.
If set to `true`:
- When type is `auto`, the model will output at most one tool use.
"
 }
 ]
 },
 {
 "name": "tools",
 "type": "array",
 "isBold": true,
 "required": false,
 "description": "Definitions of tools that the model may use.
If you include `tools` in your API request, the model may return `tool_use` content blocks that represent the model's use of those tools. You can then run those tools using the tool input generated by the model and then optionally return results back to the model using `tool_result` content blocks.
Note: During the multi-turn tool calls process in thinking mode, the model returns a `thinking` content block alongside `tool_use` content block. To continue the conversation, it is recommended to keep all previous `thinking` content block in the `messages` array for each subsequent request to achieve the best performance.Each tool definition includes:
- `name`: Name of the tool.
- `description`: Optional, but strongly-recommended description of the tool.
- `input_schema`: JSON schema for the tool `input` shape that the model will produce in `tool_use` output content blocks.
",
 "children": [
 {
 "name": "name",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "Name of the tool.
This is how the tool will be called by the model and in `tool_use` blocks."
 },
 {
 "name": "description",
 "type": "string",
 "isBold": true,
 "description": "Description of what this tool does.
Tool descriptions should be as detailed as possible. The more information that the model has about what the tool is and how to use it, the better it will perform. You can use natural language descriptions to reinforce important aspects of the tool input JSON schema."
 },
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "description": "Available options: `custom`"
 },
 {
 "name": "input_schema",
 "type": "object",
 "isBold": true,
 "required": true,
 "description": "JSON schema for the tool input shape that the model will produce in `tool_use` output content blocks.",
 "children": [
 {
 "name": "type",
 "type": "string",
 "isBold": true,
 "required": true,
 "description": "The type of `input_schema`, only `object` is supported.
Available options: `object`"
 },
 {
 "name": "properties",
 "type": [
 "object",
 "null"
 ],
 "isBold": true,
 "description": "The properties of the tool input."
 },
 {
 "name": "required",
 "type": [
 "array",
 "null"
 ],
 "isBold": true,
 "description": "The list of properties that must be included in the tool input."
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
 "description": "Use nucleus sampling.
In nucleus sampling, we compute the cumulative distribution over all the options for each subsequent token in decreasing probability order and cut it off once it reaches a particular probability specified by `top_p`. You should either alter `temperature` or `top_p`, but not both.
Recommended for advanced use cases only. You usually only need to use `temperature`.
In thinking mode, the `mimo-v2.5-pro`, `mimo-v2.5` models do not support customizing the `top_p` parameter. Even if this parameter is passed in, it will be forcibly overridden and take effect with the model's recommended default value of `0.95`.Required range: `[0.01, 1.0]`"
 }
]`} />

## Non-streaming Response

- `end_turn`: the model reached a natural stopping point.
- `max_tokens`: we exceeded the requested `max_tokens` or the model's maximum.
- `tool_use`: the model invoked one or more tools.
- `content_filter`: the content was omitted due to a flag from our content filters.
- `repetition_truncation`: the model detects repetition.
Available options: `end_turn`, `max_tokens`, `tool_use`, `content_filter`, `repetition_truncation`"
 },
 {
 "name": "usage",
 "type": "object",
 "isBold": true,
 "description": "Billing and rate-limit usage.",
 "children": [
 {
 "name": "input_tokens",
 "type": "integer",
 "isBold": true,
 "description": "The number of input tokens which were used."
 },
 {
 "name": "output_tokens",
 "type": "integer",
 "isBold": true,
 "description": "The number of output tokens which were used."
 },
 {
 "name": "cache_read_input_tokens",
 "type": [
 "integer",
 "null"
 ],
 "isBold": true,
 "description": "The number of input tokens read from the cache."
 }
 ]
 }
]`} />

## Streaming Response