# api-reference/responses-create.md

> 来源: https://platform.minimaxi.com/docs/api-reference/responses-create.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# 对话生成

> OpenAI Responses API 兼容的主接口调用MiniMax 模型，生成模型回复，支持流式与非流式。

## 推理控制

对于 `MiniMax-M3`，`reasoning` 字段用于控制响应是否包含推理输出。

* 如果省略 `reasoning`，默认关闭推理，响应不会包含 `type: "reasoning"` 的输出项。
* `reasoning: {"effort": "none"}` 是默认行为，可关闭 `MiniMax-M3` 的推理输出。
* `minimal`、`low`、`medium` 和 `high` 这些取值会被兼容接收并开启推理输出，但不会调节 MiniMax-M3 的推理深度。
* 对于 M2.x 模型，推理无法关闭；即使传入 `reasoning: {"effort": "none"}`，推理仍会保持开启。

```json theme={null}
{
  "model": "MiniMax-M3",
  "input": "9.11 和 9.9 哪个更大？"
}
```

```json theme={null}
{
  "model": "MiniMax-M3",
  "input": "9.11 和 9.9 哪个更大？",
  "reasoning": {
    "effort": "minimal"
  }
}
```


## OpenAPI

````yaml api-reference/text/api/openapi-responses.json POST /v1/responses
openapi: 3.1.0
info:
  title: MiniMax Responses API
  description: MiniMax OpenAI Responses API 兼容接口，支持对话生成和 Token 估算
  license:
    name: MIT
  version: 1.0.0
servers:
  - url: https://api.minimaxi.com
security:
  - bearerAuth: []
paths:
  /v1/responses:
    post:
      tags:
        - Responses
      summary: 对话生成
      operationId: createResponse
      parameters:
        - name: Content-Type
          in: header
          required: true
          description: 请求体的媒介类型，请设置为 `application/json`，确保请求数据的格式为 JSON
          schema:
            type: string
            enum:
              - application/json
            default: application/json
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateResponseReq'
            examples:
              SimpleText:
                summary: 简单文本输入
                value:
                  model: MiniMax-M3
                  input: 你好！
              ConversationHistory:
                summary: 完整对话历史
                value:
                  model: MiniMax-M3
                  instructions: 你是一位中文技术写作助手。
                  input:
                    - role: user
                      content: >-
                        我们公司要做一个海外社交产品，技术栈我倾向于 Go + React Native +
                        PostgreSQL，你怎么看？
              Streaming:
                summary: 流式输出
                value:
                  model: MiniMax-M3
                  input: 你好！
                  stream: true
              FunctionCall:
                summary: 工具调用
                value:
                  model: MiniMax-M3
                  input: What is the weather in Boston today?
                  tools:
                    - type: function
                      name: get_current_weather
                      description: Get the current weather in a given location
                      parameters:
                        type: object
                        properties:
                          location:
                            type: string
                            description: The city and state, e.g. San Francisco, CA
                          unit:
                            type: string
                            enum:
                              - celsius
                              - fahrenheit
                        required:
                          - location
                          - unit
              MultiTurnFunctionCall:
                summary: 多轮工具调用
                value:
                  model: MiniMax-M3
                  input:
                    - type: message
                      role: user
                      content: What is the weather in Beijing?
                    - type: function_call
                      call_id: call_abc123
                      name: get_weather
                      arguments: '{"city":"Beijing"}'
                    - type: function_call_output
                      call_id: call_abc123
                      output: Sunny, 22°C
                  tools:
                    - type: function
                      name: get_weather
                      description: Get current weather for a city
                      parameters:
                        type: object
                        properties:
                          city:
                            type: string
                        required:
                          - city
        required: true
      responses:
        '200':
          description: 成功响应
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CreateResponseResp'
              examples:
                Response:
                  value:
                    id: abc123
                    object: response
                    created_at: 1764000000
                    model: MiniMax-M3
                    status: completed
                    output:
                      - id: abc123_msg
                        type: message
                        status: completed
                        role: assistant
                        content:
                          - type: output_text
                            text: 你好！我是 MiniMax，请问有什么可以帮你的？
                            annotations: []
                    output_text: 你好！我是 MiniMax，请问有什么可以帮你的？
                    usage:
                      input_tokens: 8
                      input_tokens_details:
                        cached_tokens: 0
                      output_tokens: 18
                      output_tokens_details:
                        reasoning_tokens: 0
                      total_tokens: 26
                    parallel_tool_calls: true
                    store: false
                    truncation: disabled
components:
  schemas:
    CreateResponseReq:
      type: object
      required:
        - model
        - input
      properties:
        model:
          type: string
          description: 调用的模型名称，如 `MiniMax-M3`
          example: MiniMax-M3
        service_tier:
          type: string
          description: >-
            请求准入服务层级。支持的取值为 `standard` 和 `priority`。省略时默认使用
            `standard`。`priority` 的[价格](/guides/pricing-paygo)为 `standard` 的 1.5
            倍，并会确保请求获得优先准入，使其排在其他请求之前处理，从而带来更快响应并减少失败。
          enum:
            - standard
            - priority
          default: standard
        input:
          description: 对话内容，支持简单文本或完整对话历史数组
          oneOf:
            - type: string
              description: 简单文本输入
            - type: array
              description: 完整对话历史
              items:
                $ref: '#/components/schemas/InputItem'
        instructions:
          type: string
          description: 系统指令
        max_output_tokens:
          type: integer
          description: 最大输出 token 数
        temperature:
          type: number
          format: float
          description: 采样温度，取值范围 (0, 1]
          default: 1
          minimum: 0
          maximum: 1
        top_p:
          type: number
          format: float
          description: 核采样，取值范围 (0, 1]
          default: 0.95
          minimum: 0
          maximum: 1
        stream:
          type: boolean
          description: 设置为 `true` 启用 SSE 流式响应
          default: false
        tools:
          type: array
          description: 工具列表
          items:
            $ref: '#/components/schemas/Tool'
        tool_choice:
          type: string
          enum:
            - none
            - auto
          description: 工具选择策略：`none` 表示不调用任何工具；`auto` 表示由模型自动判断是否调用工具
        metadata:
          type: object
          description: 请求元数据，key-value 均为字符串
          additionalProperties:
            type: string
        prompt_cache_key:
          type: string
          description: Prompt 缓存路由标识
        text:
          type: object
          description: 输出格式控制
          properties:
            format:
              type: object
              properties:
                type:
                  type: string
                  enum:
                    - text
                  default: text
                  description: 输出格式类型
        reasoning:
          type: object
          description: >-
            推理控制。对于 MiniMax-M3，默认为 `none`，即关闭推理。将 `effort` 设置为非 `none`
            值（`minimal`、`low`、`medium` 或 `high`），即可开启 Adaptive Thinking，但不会调节
            MiniMax-M3 的推理深度。对于 M2.x 模型，推理无法关闭。
          properties:
            effort:
              type: string
              enum:
                - minimal
                - low
                - medium
                - high
                - none
              default: none
          required: []
    CreateResponseResp:
      type: object
      required:
        - id
        - object
        - created_at
        - model
        - status
        - output
      properties:
        id:
          type: string
          description: 响应 ID
          example: abc123
        object:
          type: string
          enum:
            - response
          description: 对象类型，固定为 `response`
        created_at:
          type: integer
          description: 响应创建时间（Unix 秒）
        model:
          type: string
          description: 实际处理请求的模型名称
        status:
          type: string
          enum:
            - completed
            - incomplete
            - failed
          description: 响应状态
        output:
          type: array
          description: 模型输出列表
          items:
            $ref: '#/components/schemas/OutputItem'
        output_text:
          type: string
          nullable: true
          description: 便利字段，所有文本输出拼接后的结果
        usage:
          $ref: '#/components/schemas/Usage'
        error:
          type: object
          nullable: true
          description: 错误信息，仅在 `status=failed` 时返回
          properties:
            code:
              type: string
              description: 错误码
            message:
              type: string
              description: 人类可读的错误描述
        incomplete_details:
          type: object
          nullable: true
          description: 未完成原因，仅在 `status=incomplete` 时返回
          properties:
            reason:
              type: string
              enum:
                - max_output_tokens
                - content_filter
        parallel_tool_calls:
          type: boolean
          description: 是否支持并行工具调用
        store:
          type: boolean
          description: 响应是否持久化
        truncation:
          type: string
          enum:
            - disabled
          description: 上下文截断策略
    InputItem:
      type: object
      description: >-
        对话历史条目。`type` 字段决定具体形态：`message`（默认）/ `function_call` /
        `function_call_output` / `reasoning`
      properties:
        type:
          type: string
          enum:
            - message
            - function_call
            - function_call_output
            - reasoning
          default: message
          description: 条目类型
        role:
          type: string
          enum:
            - user
            - assistant
            - system
            - developer
            - tool
          description: 消息角色（仅当 `type` 为 `message` 时使用）
        content:
          description: 消息内容，可以是字符串或多模态片段数组（仅当 `type` 为 `message` 时使用）
          oneOf:
            - type: string
            - type: array
              items:
                $ref: '#/components/schemas/ContentPart'
        call_id:
          type: string
          description: 工具调用 ID（仅当 `type` 为 `function_call` 或 `function_call_output` 时使用）
        name:
          type: string
          description: 函数名（仅当 `type` 为 `function_call` 时使用）
        arguments:
          type: string
          description: 函数参数的 JSON 字符串（仅当 `type` 为 `function_call` 时使用）
        output:
          description: 工具返回结果（仅当 `type` 为 `function_call_output` 时使用）
          oneOf:
            - type: string
            - type: array
              items:
                $ref: '#/components/schemas/ContentPart'
        summary:
          type: array
          description: 思维链段落数组（仅当 `type` 为 `reasoning` 时使用）
          items:
            type: object
            properties:
              type:
                type: string
                enum:
                  - summary_text
              text:
                type: string
                description: 思维链文本
    Tool:
      type: object
      required:
        - type
        - name
      properties:
        type:
          type: string
          enum:
            - function
          description: 工具类型
        name:
          type: string
          description: 函数名称
        description:
          type: string
          description: 函数说明，用于模型判断何时调用
        parameters:
          type: object
          description: 函数参数定义，使用 JSON Schema 格式
    OutputItem:
      oneOf:
        - title: Message
          type: object
          description: 助手回复
          properties:
            id:
              type: string
              description: '`<id>_msg` 格式'
            type:
              type: string
              enum:
                - message
            status:
              type: string
              enum:
                - completed
            role:
              type: string
              enum:
                - assistant
            content:
              type: array
              items:
                type: object
                properties:
                  type:
                    type: string
                    enum:
                      - output_text
                  text:
                    type: string
                    description: 模型生成的文本
                  annotations:
                    type: array
                    description: 引用标注
        - title: Reasoning
          type: object
          description: 思维链输出（仅开启推理时返回）
          properties:
            id:
              type: string
              description: '`<id>_rs` 格式'
            type:
              type: string
              enum:
                - reasoning
            status:
              type: string
              enum:
                - completed
            summary:
              type: array
              description: 思维链摘要
            content:
              type: array
              items:
                type: object
                properties:
                  type:
                    type: string
                    enum:
                      - reasoning_text
                  text:
                    type: string
                    description: 思维链文本
        - title: Function Call
          type: object
          description: 工具调用
          properties:
            id:
              type: string
              description: '`<id>_fc_<index>` 格式'
            type:
              type: string
              enum:
                - function_call
            status:
              type: string
              enum:
                - completed
            call_id:
              type: string
              description: 工具调用 ID，用于关联 `function_call_output`
            name:
              type: string
              description: 函数名称
            arguments:
              type: string
              description: 函数参数的 JSON 字符串
    Usage:
      type: object
      properties:
        input_tokens:
          type: integer
          description: 输入 token 数
        input_tokens_details:
          type: object
          properties:
            cached_tokens:
              type: integer
              description: Prompt 缓存命中的 token 数
        output_tokens:
          type: integer
          description: 输出 token 数
        output_tokens_details:
          type: object
          properties:
            reasoning_tokens:
              type: integer
              description: 思维链消耗的 token 数（仅开启推理时计入）
        total_tokens:
          type: integer
          description: 总 token 数
    ContentPart:
      type: object
      required:
        - type
      description: 消息内容片段
      properties:
        type:
          type: string
          enum:
            - input_text
            - output_text
            - input_image
            - input_video
          description: |-
            内容片段类型：
            - `input_text` / `output_text`: 文本片段
            - `input_image`: 图片输入
            - `input_video`: 视频输入
        text:
          type: string
          description: 文本内容（当 `type` 为 `input_text` / `output_text` 时）
        image_url:
          description: 图片输入（当 `type` 为 `input_image` 时）。支持的格式：JPEG、PNG、GIF、WEBP
          oneOf:
            - type: string
            - type: object
              required:
                - url
              properties:
                url:
                  type: string
                  description: 图片链接或 Base64 编码
                detail:
                  type: string
                  enum:
                    - low
                    - default
                    - high
                  default: default
                  description: 图片理解的精细度档位
        video_url:
          description: 视频输入（当 `type` 为 `input_video` 时）。支持的格式：MP4、AVI、MOV、MKV
          oneOf:
            - type: string
            - type: object
              required:
                - url
              properties:
                url:
                  type: string
                  description: 视频链接或 Base64 编码，大文件建议使用 File API 上传
                fps:
                  type: number
                  format: float
                  default: 1
                  minimum: 0.2
                  maximum: 5
                  description: 抽帧频率
                detail:
                  type: string
                  enum:
                    - low
                    - default
                    - high
                  default: default
                  description: 视频理解的精细度档位
                max_long_side_pixel:
                  type: integer
                  description: 视频帧最长边的像素约束
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: |-
        `HTTP: Bearer Auth`
         - Security Scheme Type: http
         - HTTP Authorization Scheme: Bearer API_key，用于验证账户信息，可在 [账户管理>接口密钥](https://platform.minimaxi.com/user-center/basic-information/interface-key) 中查看

````