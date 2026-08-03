# Models

{/* feishu-style:text-align:left */}
This page lists all the models currently supported by the Xiaomi MiMo API Open Platform, including model capabilities, length limits, and rate-limiting quotas, to help you select the appropriate model based on your usage scenario. 

### Rate Limiting Instructions

{/* feishu-style:text-align:left */}
The platform sets a model concurrency limit for each account. When the server load is high, response delays or ` 429 ` error may occur. We recommend that you reasonably plan your request frequency and implement request retry and backoff strategies in high-concurrency scenarios to avoid triggering rate limits. 

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

- **RPM (Requests Per Minute)** : The maximum number of requests initiated per minute. The calculation scope is the sum of the total number of requests from all API Keys under a single account when calling the same model.

- **TPM (Tokens Per Minute)** : The maximum number of Tokens that can be interacted with per minute. The calculation scope is the sum of the total number of requested Tokens for all API Keys under a single account when calling the same model.

</div>

{/* feishu-style:text-align:left */}
<br></br>

### Text Generation Model

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

**Note:** `mimo-v2-pro`, `mimo-v2-omni`, `mimo-v2-flash`and `mimo-v2-tts` **have been officially deprecated on June 30 00:00, 2026. Please switch to the new models as soon as possible.**

</div>

<table>
<colgroup>
<col style="width: 214px" />
<col style="width: 214px" />
<col style="width: 214px" />
<col style="width: 156px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">**Model ID (Model ID)** </span></th>
<th><span style="display: inline-block; text-align: left;">**Capability Support**</span></th>
<th><span style="display: inline-block; text-align: left;">**Length Limit (token)** </span></th>
<th><span style="display: inline-block; text-align: left;">**Rate Limiting**</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-pro`</span></td>
<td><ul><li>Text Generation</li><li>Deep Thinking</li><li>Streaming Output</li><li>Function Call</li><li>Structured Output</li><li>Web Search</li></ul></td>
<td><span style="display: inline-block; text-align: left;">Context Window: 1M</span><br /><span style="display: inline-block; text-align: left;">Maximum Output: 128K</span></td>
<td rowspan="2"><span style="display: inline-block; text-align: left;">Maximum RPM: 100</span><br /><span style="display: inline-block; text-align: left;">Maximum TPM: 10M</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5`</span></td>
<td><ul><li>Text Generation</li><li>Full-modal Understanding</li><li>Deep Thinking</li><li>Streaming Output</li><li>Function Call</li><li>Structured Output</li><li>Web Search</li></ul></td>
<td><span style="display: inline-block; text-align: left;">Context Window: 1M</span><br /><span style="display: inline-block; text-align: left;">Maximum Output: 128K</span></td>
</tr>
</tbody>
</table>

{/* feishu-style:text-align:left */}
<br></br>

### Automatic Speech Recognition (ASR) Model 

<table>
<colgroup>
<col style="width: 273px" />
<col style="width: 219px" />
<col style="width: 171px" />
<col style="width: 156px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">**Model ID**</span></th>
<th><span style="display: inline-block; text-align: left;">**Capability Support**</span></th>
<th><span style="display: inline-block; text-align: left;">**Length Limit (token)** </span></th>
<th><span style="display: inline-block; text-align: left;">**Rate Limiting**</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-asr`</span></td>
<td><span style="display: inline-block; text-align: left;">Speech Recognition</span></td>
<td><span style="display: inline-block; text-align: left;">Context Window: 8k</span><br /><span style="display: inline-block; text-align: left;">Maximum Output: 2k </span></td>
<td><span style="display: inline-block; text-align: left;">Maximum RPM: 100</span><br /><span style="display: inline-block; text-align: left;">Maximum TPM: 10k</span></td>
</tr>
</tbody>
</table>

{/* feishu-style:text-align:left */}
<br></br>

### Text-to-Speech (TTS) Model

<table>
<colgroup>
<col style="width: 273px" />
<col style="width: 219px" />
<col style="width: 171px" />
<col style="width: 156px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">**Model ID (Model ID)** </span></th>
<th><span style="display: inline-block; text-align: left;">**Capability Support**</span></th>
<th><span style="display: inline-block; text-align: left;">**Length Limit (token)** </span></th>
<th><span style="display: inline-block; text-align: left;">**Rate Limiting**</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-tts`</span></td>
<td><span style="display: inline-block; text-align: left;">Speech Synthesis</span></td>
<td rowspan="3"><span style="display: inline-block; text-align: left;">Context Window: 8K</span><br /><span style="display: inline-block; text-align: left;">Maximum Output: 8K </span></td>
<td rowspan="3"><span style="display: inline-block; text-align: left;">Maximum RPM: 100</span><br /><span style="display: inline-block; text-align: left;">Maximum TPM: 10M</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-tts-voiceclone`</span></td>
<td><span style="display: inline-block; text-align: left;">Speech Synthesis</span><br /><span style="display: inline-block; text-align: left;">Timbre Cloning</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-tts-voicedesign`</span></td>
<td><span style="display: inline-block; text-align: left;">Speech Synthesis</span><br /><span style="display: inline-block; text-align: left;">Timbre Design</span></td>
</tr>
</tbody>
</table>

{/* feishu-style:text-align:left */}
<br></br>

### Quick Selection Guide

<table>
<colgroup>
<col style="width: 406px" />
<col style="width: 350px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">Requirement Scenario</span></th>
<th><span style="display: inline-block; text-align: left;">Recommendation Model</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">Complex reasoning, in-depth analysis, long document processing</span></td>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-pro`</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Understanding of image, audio, and video content</span></td>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5`</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Speech to Text (Supports both Chinese and English)</span></td>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-asr`</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Text-to-Speech (Standard Preset Voice)</span></td>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-tts`</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Voice Cloning (Upload Audio Sample) </span></td>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-tts-voiceclone`</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Customized Tone Design</span></td>
<td><span style="display: inline-block; text-align: left;">`mimo-v2.5-tts-voicedesign`</span></td>
</tr>
</tbody>
</table>
