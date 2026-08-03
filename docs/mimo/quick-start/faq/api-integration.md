# API Integration

### How to obtain API Key? 

- **API Key for Pay-as-you-go API Calls:**  After logging in to the Xiaomi MiMo API Open Platform, apply for an API Key on the [Console - API Keys](https://platform.xiaomimimo.com/#/console/api-keys) page. When using the model via API, please include your API Key in the request header:` api-key: $MIMO_API_KEY ` or ` Authorization: Bearer $MIMO_API_KEY `. 

- **API Key of Token Plan:** After successful purchase, you can see the exclusive API Key on the [Token Plan ](https://platform.xiaomimimo.com/#/console/plan-manage)page. **Note: API Key is only visible and can be copied when created, please save it properly.** 

- **The API Key format for Token Plan is** `tp-xxxxx`**, which is only used for Token Plan subscription services; the API Key format for pay-as-you-go API calls is** `sk-xxxxx`**, used for pay-as-you-go billing. The two are independent of each other and cannot be mixed. The API Key for Token Plan is only available within the validity period of the Token Plan package you have subscribed to.** 

<br></br>

### What if the API Key is lost or leaked? 

can be reset on the [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) page. 

<br></br>

### How can I obtain the Base URL of the Token Plan? 

The Base URL provided on the Token Plan page shall prevail: Two types of Base URLs are provided, one compatible with the OpenAI interface protocol and the other compatible with the Anthropic interface protocol, which can be copied and used as needed. 

<br></br>

### Which programming tools does Token Plan support? 

Supports mainstream programming tools and model frameworks, such as Claude Code, OpenClaw, OpenCode, Kilo Code, Cline, Hermes Agent, CodeBuddy Code, etc. For specific access methods, please refer to [Overview of AI Tools](https://platform.xiaomimimo.com/#/docs/integration/tools-overview).

<br></br>

### Can Token Plan be used in multiple programming tools at the same time? 

The same package can be used across all supported tools, but the quota is shared, and usage of all tools will consume the same package quota. 

<br></br>

### What's the difference between OpenAI and Anthropic interfaces?

- OpenAI interface `/v1/chat/completions`  follows OpenAI format, including developer/system/user/assistant roles

- Anthropic interface  `/anthropic/v1/messages` follows Claude format, with a separate system parameter

<br></br>

### How to make multi-turn tool calls in thinking mode?

During the multi-turn tool calls process in thinking mode, the model returns a `reasoning_content` field alongside `tool_calls`. To continue the conversation, it is recommended to keep all previous `reasoning_content` in the `messages` array for each subsequent request to achieve the best performance.

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

<br></br>

### Why aretool_calls sometimes included in the reasoning_content field and sometimes in a separate tool_calls field?

The appearance of `tool_calls` in the reasoning content indicates instability and incomplete output caused by the model having `thinking` enabled when calling `tool`. It is recommended to disable `thinking` when calling `tool` calls and to adjust the settings according to [Model Hyperparameters](https://platform.xiaomimimo.com/#/docs/quick-start/model-hyperparameters) to achieve a more stable and better user experience.

<br></br>

### What's the response speed?

Response speed depends on:

- Request length and complexity

- Server load and geographic location

- Whether streaming response is used

<br></br>

### How to handle timeouts?

Please implement reasonable timeout handling on the client side:

- Set reasonable connection and read timeout times

- Use exponential backoff for retries

- For long responses, it's recommended to use streaming mode

<br></br>

### What if the API returns inappropriate content?

The platform has added content review for both user input and model output. If violations occur, the returned content will be automatically intercepted to ensure the content you receive is safe.

<br></br>

### Why doesn’t the model perform a web search after enabling online search? 

There may be three reasons:

- **Cache**: There is a 5-minute cache period after enabling / disabling online search. The online search switch will not take effect immediately within 5 minutes.

- **Model determines no need for search**: The model judges that the current query does not involve real-time information and can be answered directly with its own knowledge. To force a search, set `forced_search: true`.

- **Only some models are supported**: Currently only `mimo-v2.5-pro` and `mimo-v2.5` supports online search.

<br></br>

### Does it support local file upload?

Local file upload is not currently supported.
