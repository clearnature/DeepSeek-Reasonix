# Web Search

{/* feishu-style:text-align:left */}
Web Search is a basic online search tool that helps your large model obtain real-time public online information (such as news, products, weather, etc.).

{/* feishu-style:text-align:left */}
**Core Capabilities**

- **Flexible search modes**: Supports forced search and intent recognition. With intent recognition enabled, the model will autonomously decide whether to perform an online search without manual triggering.

- **Early search source return**: In the streaming response, the first packet will return all search sources.

- **Hybrid multi-tool invocation**: Can work with custom functions and tools; the model will automatically determine invocation priority and necessity.

- **Flexible response modes**: Supports both streaming and non-streaming responses, and both methods will return search and summary content.

## Quick Start

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

The [Web Search Plugin](https://platform.xiaomimimo.com/#/console/plugin) must be activated prior to use.

</div>

### Enable the Service

1. Go to [Console → Plugin Management](https://platform.xiaomimimo.com/#/console/plugin), and activate the Web Search Plugin.

1. Web Search Plugin fee, refer to the [Pricing](https://platform.xiaomimimo.com/#/docs/pricing). Note: **Search invocation is determined by the model. A single search round (if required by the model) may initiate multiple keywords concurrently, resulting in multiple invocations of the Internet Content Plugin. You may use the** `max_keyword` **parameter to limit the maximum number of keywords per search round, thereby further controlling invocation frequency and costs**.

{/* feishu-style:text-align:left */}
**Note**：For preparations such as obtaining an API Key, please refer to [First API Call](https://platform.xiaomimimo.com/#/docs/quick-start/first-api-call).

### Sample Code

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
            "content": "武汉明天天气怎么样？"
        }
    ],
    max_completion_tokens=1024,
    temperature=1.0,
    top_p=0.95,
    stream=False,
    stop=None,
    frequency_penalty=0,
    presence_penalty=0,
    extra_body={
        "thinking": {"type": "disabled"}
    },
    tools=[
        {
            "type": "web_search",
            "max_keyword": 3,
            "force_search": True,
            "limit": 1,
            "user_location": {
                "type": "approximate",
                "country": "China",
                "region": "Hubei",
                "city": "Wuhan"
            }
        }
    ],
    tool_choice="auto"
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
            "role": "user",
            "content": "武汉明天天气怎么样？"
        }
    ],
    "tools": [
        {
          "type": "web_search",
          "max_keyword": 3,
          "force_search": true,
          "limit": 1,
          "user_location": {
            "type": "approximate",
            "country": "China",
            "region": "Hubei",
            "city": "Wuhan"
          }
        }
    ],
    "max_completion_tokens": 1024,
    "temperature": 1.0,
    "top_p": 0.95,
    "stream": false,
    "stop": null,
    "frequency_penalty": 0,
    "presence_penalty": 0,
    "thinking": {
        "type": "disabled"
    }
}'
```

  </TabItem>
</Tab>

{/* feishu-style:text-align:left */}
**Response**

```json
{
    "id": "d9cbdd74d5384247a3b9f03580901588",
    "choices": [
        {
            "finish_reason": "stop",
            "index": 0,
            "message": {
                "content": "根据搜索结果，武汉明天（2026年4月23日，周四）的天气情况如下：\\n\\n*   **天气状况**：白天为阴天，夜间转为晴天。\\n*   **气温范围**：最高气温18℃，最低气温10℃。\\n*   **风力风向**：北风，风力较小，为微风（风力小于3级）。\\n\\n**综合来看**，明天武汉白天阴天，夜间放晴，气温相比今天（4月22日）有所回升，但昼夜温差仍达8℃左右。建议您根据早晚和午后的温差，采用“洋葱式穿衣法”，方便随时增减衣物。明天无需携带雨具，适合进行户外活动。",
                "role": "assistant",
                "annotations": [
                    {
                        "type": "url_citation",
                        "url": "https://news.qq.com/rain/a/20260422A03GDF00",
                        "title": "小雨转晴再迎小雨!武汉未来三天阴晴交替,湿度大温差显_腾讯新闻",
                        "summary": "今天是2026年4月22日,武汉白天天气为小雨,北风微风,夜晚天气为多云,北风微风,最高气温15°C,最低气温11°C,空气湿度92%,体感温度9.5°C,空气质量优。雨天道路湿滑,出行请携带雨具,注意防滑,驾车保持安全车距。明日武汉天气为阴,微风,夜间晴,微风,最高气温18°C,最低气温10°C。未来三天,武汉天气以阴到多云为主,24日夜间转小雨,25日白天有小雨,气温逐步回升,最高气温从18°C升至25°C,最低气温从10°C升至13°C,昼夜风力均为微风。降雨时段需注意低洼路段可能短时积水,建议提前检查排水设施,避免涉水通行。近期武汉天气总体平稳,但阴雨相间,湿度偏高,体感偏凉;24日起气温明显回升,昼夜温差达11°C左右。建议采用洋葱式穿衣法,兼顾早晚清凉与午后温和;室内注意通风除湿,防范衣物、食品受潮霉变;雨天晾晒条件不佳,可优先使用烘干设备。此稿由AI生成(来源:极目新闻)",
                        "site_name": "腾讯网",
                        "publish_time": "2026-04-22T11:24:12+08:00",
                        "logo_url": "https://th.bochaai.com/favicon?domain_url=https://news.qq.com/rain/a/20260422A03GDF00"
                    },
                    {
                        "type": "url_citation",
                        "url": "https://bocha.cn/share/e79b4068-66c6-4f13-bae2-ecbd48336bc5",
                        "title": "2026年04月22日武汉天气预报",
                        "summary": "2026年04月22日武汉天气预报:\\n04/22 (周三):\\n天气:小雨转多云,温度:16/11°C,风向风力:北风<3级\\n04/23 (周四):\\n天气:阴转晴,温度:18/10°C,风向风力:北风<3级\\n04/24 (周五):\\n天气:小雨,温度:22/13°C,风向风力:北风<3级\\n04/25 (周六):\\n天气:多云转晴,温度:25/13°C,风向风力:北风<3级\\n04/26 (周日):\\n天气:多云转阴,温度:28/17°C,风向风力:北风<3级\\n04/27 (周一):\\n天气:阴转晴,温度:28/18°C,风向风力:北风<3级\\n04/28 (周二):\\n天气:多云转阴,温度:29/19°C,风向风力:北风<3级",
                        "site_name": "博查",
                        "publish_time": "2026-04-22T00:00:00+08:00",
                        "logo_url": "https://th.bochaai.com/favicon?domain_url=https://bocha.cn/share/e79b4068-66c6-4f13-bae2-ecbd48336bc5"
                    },
                    {
                        "type": "url_citation",
                        "url": "https://news.qq.com/rain/a/20260421A06R9300",
                        "title": "【明日天气预报】武汉2026年04月22日天气预报,小雨转多云,北风转北风<3级_腾讯新闻",
                        "summary": "武汉04月22日(周三)天气预报,天气现象小雨转多云,\\n风向风力:\\n北风转北风<3级。最高气温16°C摄氏度,最低气温11摄氏度。\\n感冒指数:\\n少发,\\n无明显降温,感冒机率较低。运动指数:\\n适宜,\\n天气较好,尽情感受运动的快乐吧。过敏指数:\\n易发,\\n应减少外出,外出需采取防护措施。穿衣指数:\\n较冷,\\n建议着厚外套加毛衣等服装。洗车指数:\\n较适宜,\\n无雨且风力较小,易保持清洁度。紫外线指数:\\n最弱,\\n辐射弱,涂擦SPF8-12防晒护肤品。\\n【来源:综合自中国气象局】\\n更多出行游玩、民生资讯、办事服务等精彩内容,欢迎下载九派新闻APP查看。声明:此文版权归原作者所有,若有来源错误或者侵犯您的合法权益,您可通过邮箱与我们取得联系,我们将及时进行处理。邮箱地址:jpbl@jp.jiupainews.com",
                        "site_name": "腾讯网",
                        "publish_time": "2026-04-21T19:32:10+08:00",
                        "logo_url": "https://th.bochaai.com/favicon?domain_url=https://news.qq.com/rain/a/20260421A06R9300"
                    }
                ],
                "tool_calls": null
            }
        }
    ],
    "created": 1776850783,
    "model": "mimo-v2.5-pro",
    "object": "chat.completion",
    "usage": {
        "completion_tokens": 204,
        "prompt_tokens": 2106,
        "total_tokens": 2310,
        "completion_tokens_details": {
            "reasoning_tokens": 0
        },
        "prompt_tokens_details": {
            "cached_tokens": 192
        },
        "web_search_usage": {
            "tool_usage": 3,
            "page_usage": 3
        }
    }
}
```

{/* feishu-style:text-align:left */}
For detailed parameters and invocation instructions, please refer to the [OpenAI Chat Completions API](https://platform.xiaomimimo.com/#/docs/api/text-generation/openai-api). Other API protocols are not supported for the time being.

## Supported models

{/* feishu-style:text-align:left */}
Currently supports `mimo-v2.5-pro`, `mimo-v2.5` models.

## Price

{/* feishu-style:text-align:left */}
The billing for the Web Search Plugin consists of the following two parts:

- **Usage of Web Search**: The number of times internet resources appear in one response of the Internet Service Plugin.

   - Cost per 1,000 calls from the Web Search tool: China ¥16 / 1K requests、Overseas $5 / 1K requests.

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

When invoking Web Search via API, one search round will initiate concurrent keyword searches according to the `max_keyword` value, resulting in multiple uses of this plugin.

</div>

- **Model token fee**: The webpage content from internet search will be appended to the prompt, increasing the model’s input tokens. Billing is based on the model’s standard price. For price details, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing).

## FAQ

{/* feishu-style:text-align:left */}
**Why doesn’t the model perform a web search after enabling online search?** 

{/* feishu-style:text-align:left */}
There may be three reasons:

- **Cache**: There is a 5-minute cache period after enabling / disabling online search. The online search switch will not take effect immediately within 5 minutes.

- **Model determines no need for search**: The model judges that the current query does not involve real-time information and can be answered directly with its own knowledge. To force a search, set `force_search: true`.

- **Model not supported**: Currently `mimo-v2.5-pro`, `mimo-v2.5` supports online search.
