# api-reference/models/anthropic/list-models.md

> 来源: https://platform.minimaxi.com/docs/api-reference/models/anthropic/list-models.md

# 获取模型列表

> 获取平台支持的所有模型列表，兼容 Anthropic API 规范。

## OpenAPI

````yaml api-reference/models/anthropic/api/list-models.json GET /anthropic/v1/models
openapi: 3.1.0
info:
 title: MiniMax Models API
 description: MiniMax models API compatible with Anthropic API specification.
 version: 1.0.0
servers:
 - url: https://api.minimaxi.com
security:
 - apiKeyAuth: []
paths:
 /anthropic/v1/models:
 get:
 tags:
 - Models
 summary: 获取模型列表
 description: 获取平台支持的所有模型列表。该接口兼容 Anthropic API 规范。
 operationId: anthropicListModels
 parameters:
 - name: limit
 in: query
 required: false
 description: 每页返回的模型数量
 schema:
 type: integer
 - name: after_id
 in: query
 required: false
 description: 分页游标，返回该 ID 之后的模型
 schema:
 type: string
 - name: before_id
 in: query
 required: false
 description: 分页游标，返回该 ID 之前的模型
 schema:
 type: string
 responses:
 '200':
 description: 可用模型列表。
 content:
 application/json:
 schema:
 type: object
 properties:
 data:
 type: array
 description: 模型对象数组
 items:
 type: object
 properties:
 id:
 type: string
 description: 模型标识符
 created_at:
 type: string
 description: 模型创建时间（ISO 8601 格式）
 display_name:
 type: string
 description: 模型显示名称
 type:
 type: string
 description: 模型类型，固定为 `"model"`
 first_id:
 type: string
 description: 返回数据中的第一个模型 ID
 has_more:
 type: boolean
 description: 是否还有更多数据
 last_id:
 type: string
 description: 返回数据中的最后一个模型 ID
 examples:
 默认响应:
 value:
 data:
 - id: MiniMax-M3
 created_at: '2026-06-01T00:00:00Z'
 display_name: MiniMax-M3
 type: model
 - id: MiniMax-M2.7
 created_at: '2026-03-18T02:00:00Z'
 display_name: MiniMax-M2.7
 type: model
 - id: MiniMax-M2.5
 created_at: '2026-02-13T02:00:00Z'
 display_name: MiniMax-M2.5
 type: model
 first_id: MiniMax-M3
 has_more: false
 last_id: MiniMax-M2.5
components:
 securitySchemes:
 apiKeyAuth:
 type: apiKey
 in: header
 name: X-Api-Key
 description: |-
 `API Key Auth`
 - Security Scheme Type: apiKey
 - API Key Header: X-Api-Key，用于验证账户信息，可在 [账户管理>接口密钥](https://platform.minimaxi.com/user-center/basic-information/interface-key) 中查看

````