# Models

{/* feishu-style:text-align:left */}
This page lists all the models currently supported by the Xiaomi MiMo API Open Platform, including model capabilities, length limits, and rate-limiting quotas, to help you select the appropriate model based on your usage scenario. 

### Rate Limiting Instructions

{/* feishu-style:text-align:left */}
The platform sets a model concurrency limit for each account. When the server load is high, response delays or ` 429 ` error may occur. We recommend that you reasonably plan your request frequency and implement request retry and backoff strategies in high-concurrency scenarios to avoid triggering rate limits. 

- **RPM (Requests Per Minute)** : The maximum number of requests initiated per minute. The calculation scope is the sum of the total number of requests from all API Keys under a single account when calling the same model.

- **TPM (Tokens Per Minute)** : The maximum number of Tokens that can be interacted with per minute. The calculation scope is the sum of the total number of requested Tokens for all API Keys under a single account when calling the same model.

{/* feishu-style:text-align:left */}

### Text Generation Model

**Note:** `mimo-v2-pro`, `mimo-v2-omni`, `mimo-v2-flash`and `mimo-v2-tts` **have been officially deprecated on June 30 00:00, 2026. Please switch to the new models as soon as possible.**

{/* feishu-style:text-align:left */}

### Automatic Speech Recognition (ASR) Model 

{/* feishu-style:text-align:left */}

### Text-to-Speech (TTS) Model

{/* feishu-style:text-align:left */}

### Quick Selection Guide