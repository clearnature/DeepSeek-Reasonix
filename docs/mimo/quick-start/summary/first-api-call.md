# First API Call

## Supported API Types

{/* feishu-style:text-align:left */}
Xiaomi MiMo API Open Platform is compatible with OpenAI API and Anthropic API formats. You can use existing SDKs to access model inference services.

## Preparation Before Calling

### Log in to Xiaomi MiMo API Open Platform

{/* feishu-style:text-align:left */}
Currently, the platform only provides personal account login. You need to use a Xiaomi account to log in. If you already have a Xiaomi account, you can log in directly. If you don't have a Xiaomi account, you can visit the [Console](https://platform.xiaomimimo.com/#/console/usage) to register, or register in advance at [id.mi.com](https://id.mi.com/).

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

<table>
<colgroup>
<col style="width: 143px" />
<col style="width: 191px" />
<col style="width: 700px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: center;">Usage Method</span></th>
<th><span style="display: inline-block; text-align: center;">Description</span></th>
<th><span style="display: inline-block; text-align: center;">Acquisition Method (BASE_URL and API Key below are examples) </span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">Pay-as-you-go MiMo API</span></td>
<td><span style="display: inline-block; text-align: left;">Charged based on actual usage, suitable for light use</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://api.xiaomimimo.com/v1`</li><li>Anthropic Compatibility Protocol: `https://api.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `sk-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">Go to [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) to create an API Key</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: center;">Token Plan</span></td>
<td><span style="display: inline-block; text-align: left;">Fixed subscription fee, with limited calls based on the package</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/v1`</li><li>Anthropic Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `tp-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">After successful subscription, go to [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) to obtain the exclusive Base URL and API Key </span></td>
</tr>
</tbody>
</table>

> Please keep your API Key safe to avoid leakage that may result in quota theft. It is recommended to configure the API Key in environment variables.

## Quick Integration Examples

{/* feishu-style:text-align:left */}
You can copy the following API example code and replace the API Key value to quickly make calls. If you use the Token Plan, you need to replace the BASE_URL and use the dedicated API Key.

{/* feishu-style:text-align:left */}
The following system prompts are highly recommended, please choose from English and Chinese version.

> Chinese version
>
> ```json
> 你是MiMo（中文名称也是MiMo），是小米公司研发的AI智能助手。
> 今天的日期：{date} {week}，你的知识截止日期是2024年12月。
> ```

> English version
>
> ```json
> You are MiMo, an AI assistant developed by Xiaomi.
> Today's date: {date} {week}. Your knowledge cutoff date is December 2024.
> ```

### OpenAI Chat Completions API Compatibility

{/* feishu-style:text-align:left */}
MiMo models are compatible with the OpenAI Chat Completions API and support invocation via the Python SDK and Curl. Integration examples are provided below.

<Tab>
  <TabItem label={`Python SDK`}>

{/* feishu-style:text-align:left */}
1. Install the OpenAI Python SDK by running the following command:

```shell
# If the run fails, you can replace pip with pip3 and run again
pip install -U openai
```

{/* feishu-style:text-align:left */}
2. Call the API:

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
            "content": "please introduce yourself"
        }
    ],
    max_completion_tokens=1024,
    temperature=1.0,
    top_p=0.95,
    stream=False,
    stop=None,
    frequency_penalty=0,
    presence_penalty=0
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
            "content": "please introduce yourself"
        }
    ],
    "max_completion_tokens": 1024,
    "temperature": 1.0,
    "top_p": 0.95,
    "stream": false,
    "stop": null,
    "frequency_penalty": 0,
    "presence_penalty": 0
}'
```

  </TabItem>
</Tab>

### Anthropic Messages API Compatibility

{/* feishu-style:text-align:left */}
MiMo models are compatible with the Anthropic Messages API and support invocation via the Python SDK and Curl. Integration examples are provided below.

<Tab>
  <TabItem label={`Python SDK`}>

{/* feishu-style:text-align:left */}
1. Install the Anthropic Python SDK by running the following command:

```shell
# If the run fails, you can replace pip with pip3 and run again
pip install -U anthropic
```

{/* feishu-style:text-align:left */}
2. Call the API:

```python
import os
from anthropic import Anthropic

client = Anthropic(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/anthropic"
)

message = client.messages.create(
    model="mimo-v2.5-pro",
    max_tokens=1024,
    system="You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
    messages=[
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
    top_p=0.95,
    stream=False,
    temperature=1.0,
    stop_sequences=None
)

print(message.content)
```

  </TabItem>
  <TabItem label={`Curl`}>

```bash
curl --location --request POST 'https://api.xiaomimimo.com/anthropic/v1/messages' \
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
    "stop_sequences": null
}'
```

  </TabItem>
</Tab>

### Make Multi-turn Tool Calls in Thinking Mode

{/* feishu-style:text-align:left */}
During the multi-turn tool calls process in thinking mode, the model returns a `reasoning_content` field alongside `tool_calls`. To continue the conversation, it is recommended to keep all previous `reasoning_content` in the `messages` array for each subsequent request to achieve the best performance.

{/* feishu-style:text-align:left */}
The requested example is as follows:

```bash
curl --location --request POST 'https://api.xiaomimimo.com/v1/chat/completions' \
--header "api-key: $MIMO_API_KEY" \
--header "Content-Type: application/json" \
--data-raw '{
    "messages": [
        {
            "role": "assistant",
            "content": "Hello! I am MiMo.",
            "reasoning_content": "Okay, the user just asked me to introduce myself. That is a pretty straightforward request, but I should think about why they are asking this."
        },
        {
            "role": "user",
            "content": "What is the weather like in Hebei?"
        }
    ],
    "model": "mimo-v2.5-pro",
    "max_completion_tokens": 1024,
    "temperature": 1.0,
    "stream": false,
    "tools": [
        {
            "type": "function",
            "function": {
                "name": "get_current_weather",
                "description": "Get the current weather in a given location",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "location": {
                            "type": "string",
                            "description": "The city and state, e.g. San Francisco, CA"
                        },
                        "unit": {
                            "type": "string",
                            "enum": [
                                "celsius",
                                "fahrenheit"
                            ]
                        }
                    },
                    "required": [
                        "location"
                    ]
                }
            }
        }
    ],
    "tool_choice": "auto"
}'
```

## Check Usage Information

{/* feishu-style:text-align:left */}
On the [Usage Information](https://platform.xiaomimimo.com/#/console/usage) page, you can view and export detailed data of your account's model Token usage and request counts by date.
