# Rate Limit

This page lists all the models currently supported by the Xiaomi MiMo API Open Platform and their rate-limiting quotas, helping you plan your request frequency before integration. 

### Rate Limiting Instructions

The platform sets a model concurrency limit for each account. When the server load is high, response delays or ` 429 ` error may occur. We recommend that you plan your request frequency reasonably and implement request retry and backoff strategies in high-concurrency scenarios to avoid triggering rate limits. 

- **RPM (Requests Per Minute)** : The maximum number of requests initiated per minute. The calculation scope is the sum of the total number of requests from all API Keys under a single account when calling the same model.
- **TPM (Tokens Per Minute)** : The maximum number of Tokens that can be interacted with per minute. The calculation scope is the sum of the total number of requested Tokens for all API Keys under a single account when calling the same model.

### Text Generation Model

| Model ID | RPM | TPM |
|---|---|---|
| `mimo-v2.5-pro` | 100 | 10M |
| `mimo-v2.5` | 100 | 10M |

### Automatic Speech Recognition Model (ASR) 

| Model ID | RPM | TPM |
|---|---|---|
| `mimo-v2.5-asr` | 100 | 10K |

### Text-to-Speech (TTS) Model

| Model ID | RPM | TPM |
|---|---|---|
| `mimo-v2.5-tts` | 100 | 10M |
| `mimo-v2.5-tts-voiceclone` | 100 | 10M |
| `mimo-v2.5-tts-voicedesign` | 100 | 10M |