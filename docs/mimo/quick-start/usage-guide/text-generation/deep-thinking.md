# Deep Thinking

{/* feishu-style:text-align:left */}
Deep Thinking enables the model to perform deep reasoning before generating the final answer, analyzing problems step-by-step through an internal Chain of Thought (CoT), significantly improving accuracy on complex tasks. It is suitable for scenarios requiring deep analysis, such as complex reasoning, code generation, mathematical computation, and multi-step analysis.

{/* feishu-style:text-align:left */}
**Core Capabilities**

- **Deep Reasoning**: Breaks down complex problems into multiple steps for analysis, improving reasoning accuracy

- **Transparent Thinking Process**: Returns the complete thinking process, enhancing interpretability

- **Flexible Control**: Supports enabling/disabling, allowing on-demand use based on task complexity

## Supported Models

{/* feishu-style:text-align:left */}
Currently supported models: `mimo-v2.5-pro`, `mimo-v2.5`.

## Request Parameters

{/* feishu-style:text-align:left */}
Set the `thinking.type` parameter in the request to control deep thinking: `enabled` to enable deep thinking or `disabled` to disable deep thinking.

{/* feishu-style:text-align:left */}
**Default Status:** 

- Enabled by default: `mimo-v2.5-pro`, `mimo-v2.5`

## Important Notes

### Parameter Limitations

{/* feishu-style:text-align:left */}
In deep thinking, `mimo-v2.5-pro`, `mimo-v2.5` models do not support custom `temperature` and `top_p` parameters. Even if these parameters are provided, the actual effective values will be forced to use the recommended defaults of `1.0` and `0.95`.

### Multi-turn Conversation Pass-through Requirements

{/* feishu-style:text-align:left */}
When deep thinking is enabled in Agent product multi-turn conversations, and historical conversations contain tool calls, the assistant responses passed back in all subsequent user interaction rounds that contain tool calls <strong>must completely pass back the `reasoning_content` field, otherwise the API will return a 400 error</strong>. For the correct pass-back method, please refer to the "Multi-turn Tool Calls in Thinking Mode" section in the Call Examples.

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

If historical `reasoning_content` is missing, the model's context will be incomplete, which may result in decreased instruction following and increased hallucinations.

</div>

{/* feishu-style:text-align:left */}
**Affected Agent Products:** 

<table>
<colgroup>
<col style="width: 350px" />
<col style="width: 350px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">Protocol</span></th>
<th><span style="display: inline-block; text-align: left;">Affected Agent Products</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">OpenAI Compatible Protocol</span></td>
<td><span style="display: inline-block; text-align: left;">TRAE, Cursor, Roo Code, Codex, GitHub Copilot CLI, Zed, AutoGen, Goose</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Anthropic Compatible Protocol</span></td>
<td><span style="display: inline-block; text-align: left;">TRAE, GitHub Copilot CLI, AutoGen, Goose, OpenClaw, OpenCode, Kilo Code</span></td>
</tr>
</tbody>
</table>

### Other Notes

1. **Output Length Limit**: `max_completion_tokens` limits the total length of thinking content and the final answer. If the thinking process is long, the token space available for the final answer will be reduced accordingly. It is recommended to set a sufficient `max_completion_tokens` to avoid answer truncation.

1. **Response Time**: Enabling deep thinking will increase response latency, especially for complex tasks. It is recommended to use `stream: true` to view the thinking process in real-time.

## Call Examples

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

The `thinking` field is not a standard OpenAI parameter. When passing thinking-related parameters via the OpenAI Python SDK, they must be included in `extra_body`.

</div>

### Thinking Enabled

<Tab>
  <TabItem label={`Python SDK`}>

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/v1"
)

completion = client.chat.completions.create(
    model="mimo-v2.5-pro",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": "Introduce machine learning in three sentences."
        }
    ],
    max_completion_tokens=1024,
    extra_body={
        "thinking": {"type": "enabled"}
    }
)

print(completion.model_dump_json())
```

  </TabItem>
  <TabItem label={`Curl`}>

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/chat/completions' \
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
            "content": "Introduce machine learning in three sentences."
        }
    ],
    "max_completion_tokens": 1024,
    "thinking": {
        "type": "enabled"
    }
}'
```

  </TabItem>
</Tab>

{/* feishu-style:text-align:left */}
**Response Example**

```json
{
    "id": "2b92b0964c9b4335bffad7c2f75cfe9e",
    "choices": [
        {
            "finish_reason": "stop",
            "index": 0,
            "message": {
                "content": "Machine learning is a branch of artificial intelligence that enables systems to automatically learn and improve from experience without being explicitly programmed. It works by identifying patterns in data to make predictions or decisions. This technology powers a wide range of applications, from recommendation systems and speech recognition to autonomous vehicles and medical diagnosis.",
                "role": "assistant",
                "tool_calls": null,
                "reasoning_content": "Hmm, the user wants a concise three-sentence introduction to machine learning. This seems like a straightforward request for a clear, high-level explanation. \n\nI should focus on the core idea without technical jargon, mention its practical use, and end with its significance. The first sentence can define it simply, the second can give an example, and the third can highlight its impact. \n\nKeeping it neutral and informative fits the user's likely need for a quick overview. No need for extra details or fluff since they specifically asked for brevity."
            }
        }
    ],
    "created": 1781233054,
    "model": "mimo-v2.5-pro",
    "object": "chat.completion",
    "usage": {
        "completion_tokens": 171,
        "prompt_tokens": 60,
        "total_tokens": 231,
        "completion_tokens_details": {
            "reasoning_tokens": 110
        },
        "prompt_tokens_details": {}
    }
}
```

### Thinking Disabled

<Tab>
  <TabItem label={`Python SDK`}>

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/v1"
)

completion = client.chat.completions.create(
    model="mimo-v2.5-pro",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": "Write a short paragraph about the beauty of nature."
        }
    ],
    max_completion_tokens=1024,
    extra_body={
        "thinking": {"type": "disabled"}
    }
)

print(completion.model_dump_json())
```

  </TabItem>
  <TabItem label={`Curl`}>

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/chat/completions' \
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
            "content": "Write a short paragraph about the beauty of nature."
        }
    ],
    "max_completion_tokens": 1024,
    "thinking": {
        "type": "disabled"
    }
}'
```

  </TabItem>
</Tab>

{/* feishu-style:text-align:left */}
**Response Example**

```json
{
    "id": "f914c393444e4a35a4f7b1e337e032cb",
    "choices": [
        {
            "finish_reason": "stop",
            "index": 0,
            "message": {
                "content": "From the gentle rustle of leaves in an ancient forest to the fiery spectacle of a sunset painting the sky, nature's beauty is a symphony for the senses. It is found in the delicate symmetry of a snowflake, the vibrant hues of a wildflower meadow, and the silent majesty of a mountain range draped in morning mist. This ever-changing tapestry offers a profound sense of peace and wonder, reminding us of a world that exists beyond our own making. Whether in a vast, untouched wilderness or a single dewdrop clinging to a spider's web, nature's artistry is a constant, humbling source of inspiration and renewal.",
                "role": "assistant",
                "tool_calls": null
            }
        }
    ],
    "created": 1781233927,
    "model": "mimo-v2.5-pro",
    "object": "chat.completion",
    "usage": {
        "completion_tokens": 131,
        "prompt_tokens": 64,
        "total_tokens": 195,
        "completion_tokens_details": {
            "reasoning_tokens": 0
        },
        "prompt_tokens_details": {}
    }
}
```

### Streaming Response (Thinking Enabled)

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

During streaming responses, thinking content and answer content are output sequentially: first, the thinking process is returned step-by-step via `reasoning_content`, and after thinking is complete, the final answer is output step-by-step via `content`.

</div>

<Tab>
  <TabItem label={`Python SDK`}>

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/v1"
)

completion = client.chat.completions.create(
    model="mimo-v2.5-pro",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": "Give me some tips for improving work efficiency."
        }
    ],
    max_completion_tokens=1024,
    stream=True,
    extra_body={
        "thinking": {"type": "enabled"}
    }
)

for chunk in completion:
    print(chunk.model_dump_json())
```

  </TabItem>
  <TabItem label={`Curl`}>

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/chat/completions' \
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
            "content": "Give me some tips for improving work efficiency."
        }
    ],
    "max_completion_tokens": 1024,
    "stream": true,
    "thinking": {
        "type": "enabled"
    }
}'
```

  </TabItem>
</Tab>

{/* feishu-style:text-align:left */}
**Response Example**

```json
data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":"","role":"assistant","tool_calls":null,"reasoning_content":null},"finish_reason":null,"index":0}],"created":1781234029,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":null,"role":null,"tool_calls":null,"reasoning_content":"The user is asking"},"finish_reason":null,"index":0}],"created":1781234029,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":null,"role":null,"tool_calls":null,"reasoning_content":" for tips on"},"finish_reason":null,"index":0}],"created":1781234029,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":null,"role":null,"tool_calls":null,"reasoning_content":" improving work efficiency."},"finish_reason":null,"index":0}],"created":1781234029,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

...

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":null,"role":null,"tool_calls":null,"reasoning_content":", etc"},"finish_reason":null,"index":0}],"created":1781234030,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":null,"role":null,"tool_calls":null,"reasoning_content":"."},"finish_reason":null,"index":0}],"created":1781234030,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":"# Tips","role":null,"tool_calls":null,"reasoning_content":null},"finish_reason":null,"index":0}],"created":1781234030,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":" for Improving Work","role":null,"tool_calls":null,"reasoning_content":null},"finish_reason":null,"index":0}],"created":1781234030,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

...

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":" on any specific","role":null,"tool_calls":null,"reasoning_content":null},"finish_reason":null,"index":0}],"created":1781234037,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":" area?","role":null,"tool_calls":null,"reasoning_content":null},"finish_reason":null,"index":0}],"created":1781234037,"model":"mimo-v2.5-pro","object":"chat.completion.chunk"}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[{"delta":{"content":null,"role":null,"tool_calls":null,"reasoning_content":null},"finish_reason":"stop","index":0}],"created":1781234037,"model":"mimo-v2.5-pro","object":"chat.completion.chunk","usage":null}

data: {"id":"4e57d676fe464c09aa2f27fa652abc40","choices":[],"created":1781234037,"model":"mimo-v2.5-pro","object":"chat.completion.chunk","usage":{"completion_tokens":339,"prompt_tokens":61,"total_tokens":400,"completion_tokens_details":{"reasoning_tokens":41},"prompt_tokens_details":{}}}

data: [DONE]
```

### Multi-turn Tool Calls in Thinking Mode

{/* feishu-style:text-align:left */}
In multi-turn conversations with deep thinking enabled, passing back `reasoning_content` when tool calls are involved ensures thinking continuity and improves model output quality.

```python
import os
import json
from openai import OpenAI

# Initialize client
client = OpenAI(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/v1"
)

# Define tools
tools = [
    {
        "type": "function",
        "function": {
            "name": "get_current_weather",
            "description": "Get the current weather for a given city",
            "parameters": {
                "type": "object",
                "properties": {
                    "location": {"type": "string", "description": "City name, e.g. Beijing"},
                    "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
                },
                "required": ["location"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "get_time",
            "description": "Get the current time in a given timezone",
            "parameters": {
                "type": "object",
                "properties": {
                    "timezone": {"type": "string", "description": "Timezone, e.g. Asia/Shanghai"}
                },
                "required": ["timezone"]
            }
        }
    }
]

# Tool execution functions (replace with real API calls in production)
def get_current_weather(location: str, unit: str = "celsius") -> str:
    weather_data = {"Beijing": "Sunny 25°C", "Shanghai": "Cloudy 22°C", "Shenzhen": "Rainy 28°C"}
    return weather_data.get(location, f"Weather unknown for {location}")

def get_time(timezone: str) -> str:
    from datetime import datetime
    return datetime.now().strftime(f"%Y-%m-%d %H:%M:%S ({timezone})")

TOOL_MAP = {
    "get_current_weather": lambda **kw: get_current_weather(**kw),
    "get_time": lambda **kw: get_time(**kw)
}

def run_turn(messages, turn_num):
    """Execute a single user turn: call model, run tools in a loop until final answer."""
    request_num = 0
    while True:
        request_num += 1
        print(f"\nRequest {turn_num}-{request_num}:")

        response = client.chat.completions.create(
            model="mimo-v2.5-pro",
            messages=messages,
            tools=tools,
            extra_body={"thinking": {"type": "enabled"}}
        )

        assistant_message = response.choices[0].message
        messages.append(assistant_message)

        # Print full model response
        print(f"reasoning_content: {assistant_message.reasoning_content}")
        print(f"content: \"{assistant_message.content}\"")
        print(f"tool_calls: {assistant_message.tool_calls}")

        # If no tool calls, we have the final answer
        if not assistant_message.tool_calls:
            break

        # Execute each tool call and append results
        for tool_call in assistant_message.tool_calls:
            func_name = tool_call.function.name
            func_args = json.loads(tool_call.function.arguments)
            result = TOOL_MAP[func_name](**func_args)

            print(f"-> Tool result [{func_name}]: {result}")
            messages.append({
                "role": "tool",
                "tool_call_id": tool_call.id,
                "content": result
            })

# --- Multi-turn conversation ---
messages = []

# Turn 1
print("=== Turn 1 ===")
messages.append({"role": "user", "content": "How is the weather in Beijing today? What time is it now?"})
run_turn(messages, turn_num=1)

# Turn 2: reasoning_content from Turn 1 is already in messages via assistant_message
print("\n=== Turn 2 ===")
messages.append({"role": "user", "content": "How about Shanghai? And is it hotter or colder than Beijing?"})
run_turn(messages, turn_num=2)
```

{/* feishu-style:text-align:left */}
**Example Output**

{/* feishu-style:text-align:left */}
**Turn 1**: The user asks about the weather in Beijing and the current time. After receiving the user message, the model thinks and decides to call both `get_current_weather` and `get_time` tools simultaneously (Request 1-1). The client executes the tools and appends the results as `role: "tool"` messages to `messages`, then requests the model again. The model generates the final answer based on the tool results (Request 1-2).

```bash
=== Turn 1 ===

Request 1-1:
reasoning_content: The user wants to know two things: 1. The current weather in Beijing 2. The current time in Beijing I can call both functions at the same time since they are independent of each other.
content: ""
tool_calls: [ChatCompletionMessageFunctionToolCall(id='call_dd34ce1810be4afbaaa11c9a', function=Function(arguments='{"location": "Beijing"}', name='get_current_weather'), type='function'), ChatCompletionMessageFunctionToolCall(id='call_cf4c667abd094ce090b40f00', function=Function(arguments='{"timezone": "Asia/Shanghai"}', name='get_time'), type='function')]
-> Tool result [get_current_weather]: Sunny 25°C
-> Tool result [get_time]: 2026-05-12 16:37:26 (Asia/Shanghai)

Request 1-2:
reasoning_content: I got the results for both calls. Let me present this information in a friendly way.
content: "Here's the information for Beijing: ☀️ **Weather:** Sunny, 25°C — a lovely day! 🕒 **Current Time:** May 12, 2026 16:37 (Beijing Time) Looks like a beautiful afternoon in Beijing! Perfect weather for being outdoors. Is there anything else you'd like to know? 😊"
tool_calls: None
```

{/* feishu-style:text-align:left */}
**Turn 2**: The user follows up asking about Shanghai's weather and comparing it with Beijing. Since the first turn's `assistant` message (containing `reasoning_content`, `content`, `tool_calls`) has been accumulated into the conversation history via `messages.append()`, the model can directly access Beijing's 25°C information from the context, and only needs to call `get_current_weather` for Shanghai (Request 2-1), then provides a comparison answer based on both cities' weather data (Request 2-2).

```bash
=== Turn 2 ===

Request 2-1:
reasoning_content: The user wants to know the weather in Shanghai and compare it with Beijing. I already know Beijing is 25°C, so I just need to get Shanghai's weather.
content: ""
tool_calls: [ChatCompletionMessageFunctionToolCall(id='call_f4fc7fdbfbd14cb497026d21', function=Function(arguments='{"location": "Shanghai"}', name='get_current_weather'), type='function')]
-> Tool result [get_current_weather]: Cloudy 22°C

Request 2-2:
reasoning_content: Shanghai is 22°C and cloudy, while Beijing is 25°C and sunny. So Shanghai is 3°C cooler than Beijing.
content: "Here's the weather for Shanghai: ☁️ **Weather:** Cloudy, 22°C And comparing the two cities:
| City | Temperature | Conditions |
|------|------------|------------|
| Beijing | 25°C | ☀️ Sunny |
| Shanghai | 22°C | ☁️ Cloudy |

**Beijing is 3°C warmer** than Shanghai right now! Beijing also has clearer skies, while Shanghai is a bit cloudier. Both are pleasant temperatures though — great weather in both cities! 😊 Is there anything else you'd like to check?"
tool_calls: None
```
