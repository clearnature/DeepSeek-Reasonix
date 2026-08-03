# Audio Understanding

{/* feishu-style:text-align:left */}
The audio understanding model can answer based on the audio you provide, supporting both audio URL and Base64 encoding as input methods, and is suitable for scenarios such as audio analysis. 

## Quick Start

For preparations such as obtaining an API Key, please refer to [First API Call](https://platform.xiaomimimo.com/#/docs/quick-start/first-api-call).

{/* feishu-style:text-align:left */}
Quickly experience the audio understanding effect by passing the audio URL into the model. The sample code is as follows.

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
 "type": "input_audio",
 "input_audio": {
 "data": "https://example-files.cnbj1.mi-fds.com/example-files/audio/audio_example.wav"
 }
 },
 {
 "type": "text",
 "text": "please describe the content of the audio"
 }
 ]
 }
 ],
 max_completion_tokens=1024
)

print(completion.model_dump_json())
```

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
 "type": "input_audio",
 "input_audio": {
 "data": "https://example-files.cnbj1.mi-fds.com/example-files/audio/audio_example.wav"
 }
 },
 {
 "type": "text",
 "text": "please describe the content of the audio"
 }
 ]
 }
 ],
 "max_completion_tokens": 1024
}'
```

{/* feishu-style:text-align:left */}
**Response**

```json
{
 "id": "550a678a6c2046a29128883eaaf849e7",
 "choices": [
 {
 "finish_reason": "stop",
 "index": 0,
 "message": {
 "content": "",
 "role": "assistant",
 "tool_calls": null,
 "reasoning_content": "Good morning. Could you tell me what the weather will be like today?"
 }
 }
 ],
 "created": 1776850627,
 "model": "mimo-v2.5",
 "object": "chat.completion",
 "usage": {
 "completion_tokens": 17,
 "prompt_tokens": 86,
 "total_tokens": 103,
 "completion_tokens_details": {
 "reasoning_tokens": 15
 },
 "prompt_tokens_details": {
 "audio_tokens": 25,
 "cached_tokens": 82
 }
 }
}
```

## Supported models

{/* feishu-style:text-align:left */}
Currently, only the `mimo-v2.5` model is supported.

## Audio Input method

{/* feishu-style:text-align:left */}
Supported audio input methods are as follows:

- Audio URL Input: A publicly accessible audio URL address must be provided.

- Base64 Encoding Input: Convert the audio to a Base64-encoded string before passing it in.

### Audio URL Input

{/* feishu-style:text-align:left */}
Audio files can be directly passed in via a publicly accessible audio URL address, which is suitable for scenarios where the audio files are already stored in a publicly accessible environment. The size of a single audio file cannot exceed 100 MB.

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
 "type": "input_audio",
 "input_audio": {
 "data": "https://example-files.cnbj1.mi-fds.com/example-files/audio/audio_example.wav"
 }
 },
 {
 "type": "text",
 "text": "please describe the content of the audio"
 }
 ]
 }
 ],
 max_completion_tokens=1024
)

print(completion.model_dump_json())
```

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
 "type": "input_audio",
 "input_audio": {
 "data": "https://example-files.cnbj1.mi-fds.com/example-files/audio/audio_example.wav"
 }
 },
 {
 "type": "text",
 "text": "please describe the content of the audio"
 }
 ]
 }
 ],
 "max_completion_tokens": 1024
}'
```

### Base64 Encoding Input

{/* feishu-style:text-align:left */}
Convert the audio file to a Base64-encoded string and then pass it in, which is suitable for scenarios where the audio file cannot be accessed via a public network URL. The size of the converted Base64-encoded string cannot exceed 50 MB.

Please include the prefix before Base64 encoding: `data:{MIME_TYPE};base64,$BASE64_AUDIO`

- `{MIME_TYPE}`: The MIME type (media type) of the audio, used to identify the audio format, which needs to be replaced with the MIME value corresponding to the actual audio.

- `$BASE64_AUDIO`: A pure Base64-encoded string of the audio file (without any prefix).

> In the following example, both `{MIME_TYPE}` and `$BASE64_AUDIO` are placeholders, please replace them with actual valid data before running.

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
 "type": "input_audio",
 "input_audio": {
 "data": "data:{MIME_TYPE};base64,$BASE64_AUDIO"
 }
 },
 {
 "type": "text",
 "text": "please describe the content of the audio"
 }
 ]
 }
 ],
 max_completion_tokens=1024
)

print(completion.model_dump_json())
```

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
 "type": "input_audio",
 "input_audio": {
 "data": "data:{MIME_TYPE};base64,$BASE64_AUDIO"
 }
 },
 {
 "type": "text",
 "text": "please describe the content of the audio"
 }
 ]
 }
 ],
 "max_completion_tokens": 1024
}'
```

## Audio Restrictions

- Audio Formats: MP3, WAV, FLAC, M4A, OGG.

> Audio Formats variants are numerous, and it cannot be guaranteed that all files can be recognized. Please verify through testing that the files can be recognized normally.

- Audio Size:

- When passed in as a URL: File size does not exceed 100 MB. 

- When passed in as Base64 encoding: The size of the Base64 encoded string of a single audio file does not exceed 50 MB. 

- Number of audios: When multiple audio files are input, the number of audio files is limited by the model's context length, and the total number of tokens for all audio and text must be less than the model's context length.

> Note: For calculating audio tokens, please refer to [Explanation of Audio Token Usage](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/audio-understanding?target=explanation-of-audio-token-usage). For the model context length, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing). 

## Explanation of Audio Token Usage

{/* feishu-style:text-align:left */}
For the Token conversion of audio, please refer to the following code. The estimated results are for reference only, and the actual usage is subject to the API response. 

```bash
Total tokens ≈ Audio duration (in seconds, e.g., 10.6 seconds) * 6.25
```

## Price

- Billing: The total cost is calculated based on the number of input, input (cache hits), and output tokens; for pricing, please refer to [Pricing and Rate Limits](https://platform.xiaomimimo.com/#/docs/pricing). 

- Audio Token consumption can be calculated through [Explanation of Audio Token Usage](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/audio-understanding?target=explanation-of-audio-token-usage). The estimated results are for reference only, and the actual usage is subject to the API response. 

- View Bill: You can view your bill and usage on the [Billing](https://platform.xiaomimimo.com/#/console/usage) page in the Console. 

## FAQ

### Does it support local file upload?

{/* feishu-style:text-align:left */}
`mimo-v2.5` model does not currently support uploading local audio files. For supported upload methods, please refer to [Audio Input Method](https://platform.xiaomimimo.com/#/docs/usage-guide/multimodal-understanding/audio-understanding?target=audio-input-method).