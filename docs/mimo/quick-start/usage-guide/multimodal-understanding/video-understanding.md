# Video Understanding

{/* feishu-style:text-align:left */}
The video understanding model can answer based on the video you provide, supporting both video URL and Base64 encoding as input methods, and is suitable for scenarios such as video analysis. 

## Quick Start

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

For preparations such as obtaining an API Key, please refer to [First API Call](https://platform.xiaomimimo.com/#/docs/quick-start/first-api-call).

</div>

{/* feishu-style:text-align:left */}
Quickly experience the video understanding effect by passing the model through the video URL method. The sample code is as follows.

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
                    "type": "video_url",
                    "video_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/video/video_example.mp4"
                    },
                    "fps": 2,
                    "media_resolution": "default"
                },
                {
                    "type": "text",
                    "text": "please describe the content of the video"
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
                    "type": "video_url",
                    "video_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/video/video_example.mp4"
                    },
                    "fps": 2,
                    "media_resolution": "default"
                },
                {
                    "type": "text",
                    "text": "please describe the content of the video"
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
    "id": "a7a6fe52ffeb40ab9e699fe018592b16",
    "choices": [
        {
            "finish_reason": "stop",
            "index": 0,
            "message": {
                "content": "The video features a small, fluffy white kitten with bright blue eyes sitting on a polished wooden floor. The kitten is positioned near a window draped with sheer, light-colored curtains, through which sunlight is streaming, casting warm shadows on the floorboards. In the background to the right, a potted green plant, resembling a Monstera, adds a touch of color to the room.\\n\\nThroughout the clip, the kitten is meowing repeatedly. It opens its mouth wide to let out high-pitched meows, looking slightly to its left before turning its gaze directly toward the camera. The audio consists entirely of the kitten's vocalizations.",
                "role": "assistant",
                "tool_calls": null,
                "reasoning_content": "The user wants me to describe the video provided.\\n\\n1.  **Analyze the video:**\\n*   **Subject:** A small white kitten with striking blue eyes.\\n*   **Action:** The kitten is sitting on a wooden floor near a window. It is meowing. Its mouth opens and closes repeatedly. It looks slightly to its left and then more directly forward.\\n*   **Setting:** Indoors. There's a window on the left with light-colored curtains. Sunlight is streaming in, casting shadows on the floor. In the background, there's a green plant (looks like a Monstera) in a pot. The walls are light-colored.\\n*   **Audio:** The audio consists of the kitten meowing. It's a series of high-pitched meows.\\n*   **Visual details:** The kitten is fluffy. The floor is polished wood. The lighting is warm and natural.\\n\\n2.  **Break down the description chronologically:**\\n*   **Start (00:00 - 00:01):** The video opens with a white kitten sitting on a wooden floor. It's positioned near a window with sheer curtains. Sunlight is hitting the floor. The kitten is looking slightly to the side.\\n*   **Middle (00:01 - 00:02):** The kitten opens its mouth wide to meow. It does this multiple times. Its gaze shifts a bit. The sunlight highlights its white fur.\\n*   **End (00:02 - 00:03):** The kitten continues to meow. It looks directly at the camera/viewer. The background remains consistent with the plant and the wall.\\n\\n3.  **Synthesize the description:**\\n*   Start with the main subject: A white kitten with blue eyes.\\n*   Describe the environment: Sitting on a wooden floor near a window with curtains. Sunlight is present. A plant is in the background.\\n*   Describe the action: The kitten is meowing repeatedly. It looks around and then at the camera.\\n*   Mention the audio: The sound of the kitten meowing.\\n\\n4.  **Refine the language:**\\n*   \\\"A white kitten with blue eyes is sitting on a wooden floor.\\\"\\n*   \\\"It is positioned next to a window with light curtains.\\\"\\n*   \\\"Sunlight is streaming in, creating shadows.\\\"\\n*   \\\"The kitten is meowing, opening its mouth wide.\\\"\\n*   \\\"In the background, there is a potted green plant.\\\"\\n*   \\\"The kitten looks to the side and then towards the camera while meowing.\\\"\\n\\n5.  **Final Polish:** Combine these points into a coherent paragraph or a list of key observations. The prompt asks to \\\"describe the content of the video,\\\" so a narrative description is appropriate.\\n\\n*   *Draft:* The video shows a cute white kitten with blue eyes sitting on a polished wooden floor. It's near a window with white curtains that are letting in bright sunlight. To the right, in the background, there is a large green plant. The kitten is meowing repeatedly, opening its mouth wide. It looks around a bit and then stares directly at the camera. The audio is just the sound of the kitten's meows.\\n\\n*   *Structure:*\\n*   **Scene:** White kitten, wooden floor, window, sunlight, plant.\\n*   **Action:** Meowing, looking around.\\n*   **Audio:** Meowing sounds.\\n\\nThis covers all the essential aspects of the video."
            }
        }
    ],
    "created": 1776850708,
    "model": "mimo-v2.5",
    "object": "chat.completion",
    "usage": {
        "completion_tokens": 849,
        "prompt_tokens": 1260,
        "total_tokens": 2109,
        "completion_tokens_details": {
            "reasoning_tokens": 719
        },
        "prompt_tokens_details": {
            "audio_tokens": 19,
            "cached_tokens": 1256,
            "video_tokens": 1144
        }
    }
}
```

## Supported models

{/* feishu-style:text-align:left */}
Currently, only the `mimo-v2.5` model is supported.

## Video Input Method

{/* feishu-style:text-align:left */}
Supported video input methods are as follows:

- Video URL Input: A publicly accessible video URL address must be provided. 

- Base64 Encoding Input: Convert the video to a Base64-encoded string before inputting it.

### Video URL Input

{/* feishu-style:text-align:left */}
Videos can be directly passed in via a publicly accessible video URL address, which is suitable for scenarios where the video is already stored in a publicly accessible environment. The size of a single video file cannot exceed 300 MB. 

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
                    "type": "video_url",
                    "video_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/video/video_example.mp4"
                    },
                    "fps": 2,
                    "media_resolution": "default"
                },
                {
                    "type": "text",
                    "text": "please describe the content of the video"
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
                    "type": "video_url",
                    "video_url": {
                        "url": "https://example-files.cnbj1.mi-fds.com/example-files/video/video_example.mp4"
                    },
                    "fps": 2,
                    "media_resolution": "default"
                },
                {
                    "type": "text",
                    "text": "please describe the content of the video"
                }
            ]
        }
    ],
    "max_completion_tokens": 1024
}'
```

  </TabItem>
</Tab>

### Base64 Encoding Input

{/* feishu-style:text-align:left */}
Convert the video file to a Base64-encoded string and then pass it in, which is suitable for scenarios where the video cannot be accessed via a public network URL. The size of the converted Base64-encoded string cannot exceed 50 MB.

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

Please include the prefix before Base64 encoding: `data:{MIME_TYPE};base64,$BASE64_VIDEO`

- `{MIME_TYPE}`: The MIME type (media type) of the video, used to identify the video format, and needs to be replaced with the MIME value corresponding to the actual video.

- `$BASE64_VIDEO`: Pure Base64-encoded string of the video file (without any prefix).

</div>

> In the following example, both `{MIME_TYPE}` and `$BASE64_VIDEO` are placeholders, please replace them with actual valid data before running.

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
                    "type": "video_url",
                    "video_url": {
                        "url": "data:{MIME_TYPE};base64,$BASE64_VIDEO"
                    },
                    "fps": 2,
                    "media_resolution": "default"
                },
                {
                    "type": "text",
                    "text": "please describe the content of the video"
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
                    "type": "video_url",
                    "video_url": {
                        "url": "data:{MIME_TYPE};base64,$BASE64_VIDEO"
                    },
                    "fps": 2,
                    "media_resolution": "default"
                },
                {
                    "type": "text",
                    "text": "please describe the content of the video"
                }
            ]
        }
    ],
    "max_completion_tokens": 1024
}'
```

  </TabItem>
</Tab>

## Instructions for Use

### Video Restrictions

- Video Formats: MP4, MOV, AVI, WMV.

> Video Formats variants are numerous, and it cannot be guaranteed that all files can be recognized. Please verify through testing that the files can be recognized normally.

- Video Size:

   - When passed in as a URL: single video file size does not exceed 300 MB. 

   - When passed in as Base64 encoding: The size of the Base64 encoded string of a single video does not exceed 50 MB. 

- Number of videos: When multiple videos are input, the number of videos is limited by the model's context length, and the total number of tokens for all video and text must be less than the model's context length.

> Note: For calculating video tokens, please refer to [Explanation of Video Token Usage](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/video-understanding?target=explanation-of-video-token-usage). For the model context length, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing). 

### Control the fineness of video understanding

{/* feishu-style:text-align:left */}
You can control the granularity of video understanding through the two fields `fps` and `media_resolution` respectively.

1. `fps` is the frame number of images extracted from the video per second, used to control the fineness of understanding the time dimension of the video. The default value is 2, with a range of `[0.1, 10]`.

   - The higher the value, the denser the frame extraction, and the more refined the model's perception of frame changes, movements, and temporal details; 

   - The lower the value, the sparser the frame extraction, the faster the processing speed, and the less Token consumption.

1. `media_resolution` refers to the resolution level of video frames, used to control the visual understanding fineness of a single frame. The default value is `default`. 

   - `default`: The default level, balancing recognition effectiveness and processing efficiency;

   - `max`: The highest resolution level, which enhances the recognition ability for small objects and detailed textures.

## Explanation of Video Token Usage

{/* feishu-style:text-align:left */}
Video tokens are divided into `video_tokens` (visual) and `audio_tokens` (audio). 

- `video_tokens` Calculation please refer to the following code. The estimated results are for reference only, and the actual usage is subject to the API response. 

   ```python
   """
   Estimate the number of tokens consumed by an API call based on video duration and resolution.
   Two parameters control the level of detail:
     - fps: Frames extracted per second. Default 2, range [0.1, 10]. Higher values yield
       finer temporal granularity at the cost of more tokens.
     - media_resolution: Per-frame resolution tier. "default" balances quality and efficiency;
       "max" improves fine-grained detail recognition.
   """
   
   import math
   
   def estimate_video_tokens(
       duration: float,
       width: int,
       height: int,
       fps: float = 2.0,
       media_resolution: str = "default",
       mute: bool = False,
   ) -> int:
       """
       Estimate the token count for a video input.
   
       Args:
           duration:         Video duration in seconds.
           width:            Video width in pixels.
           height:           Video height in pixels.
           fps:              Frame extraction rate. Default 2, range [0.1, 10].
           media_resolution: "default" or "max".
           mute:             If True, audio tokens are excluded.
   
       Returns:
           Estimated total token count.
       """
       # ---- Constants ----
       PATCH, MERGE, T_PATCH = 16, 2, 2
       SPATIAL = PATCH * MERGE                         # 32
       PIX_PER_TOKEN = SPATIAL ** 2                    # 1024
       MAX_TOTAL_TOKENS = 131072
       TOTAL_MAX_PIX = MAX_TOTAL_TOKENS * PIX_PER_TOKEN
       MIN_PIX, MAX_PIX = 8192, 8388608
       MAX_FRAMES = 2048
       DEFAULT_MAX_FRAME_TOKEN = 300
   
       # ---- 1. Number of extracted frames ----
       nframes = math.ceil(duration * fps)
       nframes = min(nframes, MAX_FRAMES)
       nframes = max(math.ceil(nframes / T_PATCH) * T_PATCH, T_PATCH)
   
       # ---- 2. Per-frame pixel budget ----
       max_pix = TOTAL_MAX_PIX * T_PATCH // nframes
       if media_resolution != "max":
           max_pix = min(max_pix, DEFAULT_MAX_FRAME_TOKEN * PIX_PER_TOKEN)
       max_pix = max(MIN_PIX, min(max_pix, MAX_PIX))
   
       # ---- 3. Resolution scaling ----
       h, w = height, width
       if min(h, w) < SPATIAL:
           if h < w:
               w = int(w * SPATIAL / h); h = SPATIAL
           else:
               h = int(h * SPATIAL / w); w = SPATIAL
       h_bar = round(h / SPATIAL) * SPATIAL
       w_bar = round(w / SPATIAL) * SPATIAL
       if h_bar * w_bar > max_pix:
           beta = math.sqrt(h * w / max_pix)
           h_bar = math.floor(h / beta / SPATIAL) * SPATIAL
           w_bar = math.floor(w / beta / SPATIAL) * SPATIAL
       elif h_bar * w_bar < MIN_PIX:
           beta = math.sqrt(MIN_PIX / (h * w))
           h_bar = math.ceil(h * beta / SPATIAL) * SPATIAL
           w_bar = math.ceil(w * beta / SPATIAL) * SPATIAL
   
       # ---- 4. Token calculation ----
       grids = nframes // T_PATCH                       # temporal grid count
       tokens_per_grid = (h_bar // PATCH) * (w_bar // PATCH) // (MERGE ** 2)
       vision = grids * tokens_per_grid
       timestamps = grids * (5 if fps > 2 else 3)       # timestamp text tokens
       special = grids * 2 + 2                           # special markers
   
       # ---- 5. Audio tokens ----
       audio = 0
       if not mute:
           spec_len = int(duration * 24000) // 240 + 1
           t = (spec_len - 1) // 2 + 1
           t = t // 2 + int(t % 2 != 0)
           audio = math.ceil(t / 4) + 2                 # +2 for audio special tokens
   
       return vision + timestamps + special + audio
   
   # ============ Example ============
   if __name__ == "__main__":
       # A 1080p, 60-second video
       tokens = estimate_video_tokens(duration=60, width=1920, height=1080)
       print(f"Default params (fps=2, default): {tokens:,} tokens")
   
       tokens = estimate_video_tokens(duration=60, width=1920, height=1080, fps=5)
       print(f"High frame rate (fps=5, default): {tokens:,} tokens")
   
       tokens = estimate_video_tokens(duration=60, width=1920, height=1080, media_resolution="max")
       print(f"High resolution (fps=2, max):     {tokens:,} tokens")
   
       tokens = estimate_video_tokens(duration=60, width=1920, height=1080, mute=True)
       print(f"Muted           (fps=2, mute):    {tokens:,} tokens")
   ```

- `audio_tokens` Calculation please refer to the following code. The estimated results are for reference only, and the actual usage is subject to the API response. 

   ```bash
   Total tokens ≈ Audio duration (in seconds) * 6.25
   ```

## Price

- Billing: The total cost is calculated based on the number of input, input (cache hits), and output tokens; for pricing, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing). 

   - Video Token consumption can be calculated through [Explanation of Video Token Usage](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/video-understanding?target=explanation-of-video-token-usage). The estimated results are for reference only, and the actual usage is subject to the API response. 

- View Bill: You can view your bill and usage on the [Billing](https://platform.xiaomimimo.com/#/console/usage) page in the Console. 

## FAQ

### Does it support local file upload?

{/* feishu-style:text-align:left */}
`mimo-v2.5` model does not currently support uploading local video files. For supported upload methods, please refer to [Video Input Method](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/video-understanding?target=video-input-method).
