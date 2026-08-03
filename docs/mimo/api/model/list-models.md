# List Models

## Request Address

```bash
https://api.xiaomimimo.com/v1/models
```

## Request Headers

{/* feishu-style:text-align:left */}
The API supports the following two authentication methods. Please choose one and add it to the request headers:

<Tab>
  <TabItem label={`API Key Authentication`}>

```json
api-key: $MIMO_API_KEY
```

  </TabItem>
  <TabItem label={`Bearer Authentication`}>

```json
Authorization: Bearer $MIMO_API_KEY
```

  </TabItem>
</Tab>

## Response
<InlineSchemaV2 schema={`[
  {
    "name": "object",
    "type": "string",
    "isBold": true,
    "description": "The object type.<br />Available options: <code class=\\"schema-inline-code\\">list</code>"
  },
  {
    "name": "data",
    "type": "array",
    "isBold": true,
    "description": "An array of model objects.",
    "children": [
      {
        "name": "id",
        "type": "string",
        "isBold": true,
        "description": "The model identifier, which can be referenced in the API endpoints."
      },
      {
        "name": "object",
        "type": "string",
        "isBold": true,
        "description": "The object type.<br />Available options: <code class=\\"schema-inline-code\\">model</code>"
      },
      {
        "name": "owned_by",
        "type": "string",
        "isBold": true,
        "description": "The organization that owns the model."
      }
    ]
  }
]`} />
