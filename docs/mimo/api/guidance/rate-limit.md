# Rate Limit

<div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">This page lists all the models currently supported by the Xiaomi MiMo API Open Platform and their rate-limiting quotas, helping you plan your request frequency before integration. </div>

### Rate Limiting Instructions

<div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">The platform sets a model concurrency limit for each account. When the server load is high, response delays or ` 429 ` error may occur. We recommend that you plan your request frequency reasonably and implement request retry and backoff strategies in high-concurrency scenarios to avoid triggering rate limits. </div>

<div className='mdx-highlight'>

- **RPM (Requests Per Minute)** : The maximum number of requests initiated per minute. The calculation scope is the sum of the total number of requests from all API Keys under a single account when calling the same model.
- **TPM (Tokens Per Minute)** : The maximum number of Tokens that can be interacted with per minute. The calculation scope is the sum of the total number of requested Tokens for all API Keys under a single account when calling the same model.

</div>

### Text Generation Model

<table>
<colgroup>
<col style="width: 183px" />
<col style="width: 183px" />
<col style="width: 183px" />
</colgroup>
<thead>
<tr>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">Model ID</div></th>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">RPM</div></th>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">TPM</div></th>
</tr>
</thead>
<tbody>
<tr>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">`mimo-v2.5-pro`</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">100</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">10M</div></td>
</tr>
<tr>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">`mimo-v2.5`</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">100</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">10M</div></td>
</tr>
</tbody>
</table>

### Automatic Speech Recognition Model (ASR) 

<table>
<colgroup>
<col style="width: 244px" />
<col style="width: 244px" />
<col style="width: 244px" />
</colgroup>
<thead>
<tr>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">Model ID</div></th>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">RPM</div></th>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">TPM</div></th>
</tr>
</thead>
<tbody>
<tr>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">`mimo-v2.5-asr`</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">100</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">10K</div></td>
</tr>
</tbody>
</table>

### Text-to-Speech (TTS) Model

<table>
<colgroup>
<col style="width: 261px" />
<col style="width: 244px" />
<col style="width: 244px" />
</colgroup>
<thead>
<tr>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">Model ID</div></th>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">RPM</div></th>
<th><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">TPM</div></th>
</tr>
</thead>
<tbody>
<tr>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">`mimo-v2.5-tts`</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">100</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">10M</div></td>
</tr>
<tr>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">`mimo-v2.5-tts-voiceclone`</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">100</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">10M</div></td>
</tr>
<tr>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">`mimo-v2.5-tts-voicedesign`</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">100</div></td>
<td><div class="mb-[16px] w-full text-[14px] leading-[24px] lg:text-[16px] lg:leading-[28px] font-normal text-[#5C5C62]" style="text-align: left;">10M</div></td>
</tr>
</tbody>
</table>
