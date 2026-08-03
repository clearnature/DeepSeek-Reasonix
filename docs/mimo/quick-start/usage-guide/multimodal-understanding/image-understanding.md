# Image Understanding

{/* feishu-style:text-align:left */}
The image understanding model can answer based on the images you provide, supporting both image URL and Base64 encoding as input methods, and is suitable for scenarios such as image description and classification. 

## Quick Start

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

For preparations such as obtaining an API Key, please refer to [First API Call](https://platform.xiaomimimo.com/#/docs/quick-start/first-api-call).

</div>

{/* feishu-style:text-align:left */}
Quickly experience the effect of image understanding by passing the model through the image URL method. The sample code is as follows. 

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
    model="mimo-v2.5",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ],
    max_completion_tokens=1024
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
    "model": "mimo-v2.5",
    "messages": [
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ],
    "max_completion_tokens": 1024
}'
```

  </TabItem>
</Tab>

{/* feishu-style:text-align:left */}
**Response**

```json
{
    "id": "be319773af414195b27a9b1cefc3fe6f",
    "choices": [
        {
            "finish_reason": "stop",
            "index": 0,
            "message": {
                "content": "This is a serene, lush temperate old-growth forest scene, bathed in warm, golden hour light, likely early morning or late afternoon:\\n\\n1.  **Foreground & Stream**: A shallow, clear stream winds through the scene, its surface catching the sunlight to create shimmering highlights. Smooth, rounded river rocks, many covered in thick, vibrant green moss, line the stream's banks and sit partially submerged in the water. On the left, a large, detailed fern with full, feathery fronds anchors the mossy shore.\\n2.  **Midground & Forest Floor**: The forest floor is dense with verdant, mossy growth, small leafy shrubs, and undergrowth, creating a rich, textured green landscape. Ancient, gnarled trees with thick, textured trunks and sprawling, twisted branches rise up, their bases rooted in the mossy earth.\\n3.  **Light & Atmosphere**: Soft, dramatic sunbeams, crepuscular rays, filter through the misty forest canopy, cutting through the hazy air. Tiny particles like pollen or dust catch the light, adding an ethereal, magical quality. Mist hangs between the trees, softening the background and lending the scene a quiet, tranquil, almost enchanted mood.\\n\\nThe image conveys a sense of peace, untouched natural beauty, and the quiet magic of an old, undisturbed forest.",
                "role": "assistant",
                "tool_calls": null,
                "reasoning_content": "Got it, let's break down this image step by step. First, the setting is a lush, misty old-growth forest, early morning or late afternoon (golden hour, since the sunbeams are warm).\\n\\nStart with the foreground: a shallow, clear stream with smooth, moss-covered rocks lining its banks. The rocks are rounded, some partially submerged, with bright green moss growing thickly on them and the shore. On the left, a large, vibrant fern with detailed fronds sits on the mossy bank.\\n\\nThen the midground: the stream winds through the forest, with small shrubs and more mossy growth along its edges. The trees are ancient, with thick, gnarled trunks and sprawling, twisted branches, their bark textured and aged. The forest floor is covered in moss and undergrowth, feeling dense and verdant.\\n\\nThe background and lighting: soft, golden sunbeams (crepuscular rays) filter through the misty forest canopy, illuminating the air—you can see tiny particles (like dust or pollen) catching the light, creating a magical, ethereal glow. The mist hangs in the air between the trees, softening the background and adding depth, making the forest feel quiet, serene, and almost enchanted.\\n\\nOverall, it's a tranquil, lush temperate rainforest scene, full of natural detail, with warm, diffused light creating a peaceful, mystical atmosphere."
            }
        }
    ],
    "created": 1776850561,
    "model": "mimo-v2.5",
    "object": "chat.completion",
    "usage": {
        "completion_tokens": 574,
        "prompt_tokens": 1085,
        "total_tokens": 1659,
        "completion_tokens_details": {
            "reasoning_tokens": 288
        },
        "prompt_tokens_details": {
            "cached_tokens": 1081,
            "image_tokens": 1024
        }
    }
}
```

## Supported models

{/* feishu-style:text-align:left */}
Currently, only the `mimo-v2.5` model is supported.

## Image Input Method

{/* feishu-style:text-align:left */}
Supported ways to upload images are as follows:

- Image URL Input: A publicly accessible image URL address must be provided. 

- Base64 Encoding Input: Convert the image to a Base64-encoded string before passing it in.

### Image URL Input

{/* feishu-style:text-align:left */}
Directly pass in the image via the publicly accessible image URL, which is suitable for scenarios where the image is already stored in a publicly accessible environment. The file size of a single image cannot exceed 50 MB.

#### OpenAI Chat Completions API

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
    model="mimo-v2.5",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ],
    max_completion_tokens=1024
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
    "model": "mimo-v2.5",
    "messages": [
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ],
    "max_completion_tokens": 1024
}'
```

  </TabItem>
</Tab>

#### Anthropic Messages API

<Tab>
  <TabItem label={`Python SDK`}>

```python
import os
from anthropic import Anthropic

client = Anthropic(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/anthropic"
)

message = client.messages.create(
    model="mimo-v2.5",
    max_tokens=1024,
    system="You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
    messages=[
        {
            "role": "user",
            "content": [
                {
                    "type": "image",
                    "source": {
                        "type": "url",
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ]
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
    "model": "mimo-v2.5",
    "max_tokens": 1024,
    "system": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
    "messages": [
        {
            "role": "user",
            "content": [
                {
                    "type": "image",
                    "source": {
                        "type": "url",
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ]
}'
```

  </TabItem>
</Tab>

### Base64 Encoded Input

{/* feishu-style:text-align:left */}
Convert the image file to a Base64-encoded string and then pass it in, which is suitable for scenarios where the image cannot be accessed via a public network URL. The size of the converted Base64-encoded string cannot exceed 50 MB.

> In the following example, both `{MIME_TYPE}` and `$BASE64_IMAGE` are placeholders, please replace them with actual valid data before running.

#### OpenAI Chat Completions API

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

Please include the prefix before Base64 encoding: `data:{MIME_TYPE};base64,$BASE64_IMAGE`

- `{MIME_TYPE}`: The MIME type (media type) of the image, used to identify the image format, needs to be replaced with the MIME value corresponding to the actual image.

- `$BASE64_IMAGE`: A pure Base64-encoded string of the image file (without any prefix).

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
    model="mimo-v2.5",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:{MIME_TYPE};base64,$BASE64_IMAGE"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ],
    max_completion_tokens=1024
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
    "model": "mimo-v2.5",
    "messages": [
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:{MIME_TYPE};base64,$BASE64_IMAGE"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ],
    "max_completion_tokens": 1024
}'
```

  </TabItem>
</Tab>

#### Anthropic Messages API

<Tab>
  <TabItem label={`Python SDK`}>

```python
import os
from anthropic import Anthropic

client = Anthropic(
    api_key=os.environ.get("MIMO_API_KEY"),
    base_url="https://api.xiaomimimo.com/anthropic"
)

message = client.messages.create(
    model="mimo-v2.5",
    max_tokens=1024,
    system="You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
    messages=[
        {
            "role": "user",
            "content": [
                {
                    "type": "image",
                    "source": {
                        "type": "base64",
                        "media_type": "{MIME_TYPE}",
                        "data": "$BASE64_IMAGE"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ]
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
    "model": "mimo-v2.5",
    "max_tokens": 1024,
    "system": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024.",
    "messages": [
        {
            "role": "user",
            "content": [
                {
                    "type": "image",
                    "source": {
                        "type": "base64",
                        "media_type": "{MIME_TYPE}",
                        "data": "$BASE64_IMAGE"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the content of the image"
                }
            ]
        }
    ]
}'
```

  </TabItem>
</Tab>

### Multi-image Input

{/* feishu-style:text-align:left */}
Supports simultaneously passing in public network URLs or Base64-encoded strings of multiple images, and the model can parse the image content and return responses that match the image semantics.

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
    model="mimo-v2.5",
    messages=[
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:{MIME_TYPE};base64,$BASE64_IMAGE"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the connections and differences between these two pictures"
                }
            ]
        }
    ],
    max_completion_tokens=1024
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
    "model": "mimo-v2.5",
    "messages": [
        {
            "role": "system",
            "content": "You are MiMo, an AI assistant developed by Xiaomi. Today is date: Tuesday, December 16, 2025. Your knowledge cutoff date is December 2024."
        },
        {
            "role": "user",
            "content": [
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/image/image_example.png"
                    }
                },
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:{MIME_TYPE};base64,$BASE64_IMAGE"
                    }
                },
                {
                    "type": "text",
                    "text": "please describe the connections and differences between these two pictures"
                }
            ]
        }
    ],
    "max_completion_tokens": 1024
}'
```

  </TabItem>
</Tab>

## Image Restrictions

- Image Formats: JPEG, PNG, GIF, WebP, BMP. 

- Image Size: 

   - When passed in as a URL: single image file size does not exceed 50 MB.

   - When passed in as Base64 encoding: The size of the Base64 encoded string of a single image does not exceed 50 MB. 

- Number of images: When multiple images are passed in, the number of images is limited by the model's context length, and the total number of Tokens for all images and text must be less than the model's context length.

> Note: For calculating image tokens, please refer to [Explanation of Image Token Usage](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/image-understanding?target=explanation-of-image-token-usage-and-scaling-rules). For the model context length, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing). 

## Explanation of Image Token Usage and Scaling Rules

{/* feishu-style:text-align:left */}
The calculation rules for images are relatively complex. For Token conversion and scaling rules, please refer to the following code. The estimated results are for reference only, and the actual usage shall be subject to the API response. 

```python
import math
from PIL import Image

PATCH_SIZE = 16
SPATIAL_MERGE_SIZE = 2
TEMPORAL_PATCH_SIZE = 2
IMAGE_MIN_PIXELS = 8192
IMAGE_MAX_PIXELS = 8388608

def calc_image_tokens(image_path: str) -> dict:
    image = Image.open(image_path)
    height = image.height
    width = image.width

    factor = PATCH_SIZE * SPATIAL_MERGE_SIZE  # 32

    h_bar = round(height / factor) * factor
    w_bar = round(width / factor) * factor

    if h_bar * w_bar > IMAGE_MAX_PIXELS:
        beta = math.sqrt((height * width) / IMAGE_MAX_PIXELS)
        h_bar = math.floor(height / beta / factor) * factor
        w_bar = math.floor(width / beta / factor) * factor
    elif h_bar * w_bar < IMAGE_MIN_PIXELS:
        beta = math.sqrt(IMAGE_MIN_PIXELS / (height * width))
        h_bar = math.ceil(height / beta / factor) * factor
        w_bar = math.ceil(width / beta / factor) * factor

    grid_t = 1
    grid_h = h_bar // PATCH_SIZE
    grid_w = w_bar // PATCH_SIZE
    num_tokens = (grid_t * grid_h * grid_w) // (SPATIAL_MERGE_SIZE ** 2)
    return num_tokens

if __name__ == "__main__":
   token = calc_image_tokens(image_path="xxx/test.jpg")
   print(token)
```

## Price

- Billing: Total cost is calculated based on the number of input, input (cache hits), and output tokens; for pricing, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing). 

   - The Token consumption of images can be calculated through [Explanation of Image Token Usage](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/image-understanding?target=explanation-of-image-token-usage-and-scaling-rules). The estimated results are for reference only, and the actual usage shall be subject to the API response. 

- View Bill: You can view your bill and usage on the [Billing](https://platform.xiaomimimo.com/#/console/usage) page in the Console. 

## FAQ

### Does it support local file upload?

{/* feishu-style:text-align:left */}
`mimo-v2.5` model does not currently support uploading local image files. For supported upload methods, please refer to [Image Input Method](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/image-understanding?target=image-input-method).
