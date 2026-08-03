# Error Codes

{/* feishu-style:text-align:left */}
When using API calls to the MiMo model, common error codes and solutions are as follows:

<table>
<colgroup>
<col style="width: 267px" />
<col style="width: 465px" />
<col style="width: 564px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: left;">**Error Code**</span></th>
<th><span style="display: inline-block; text-align: left;">**Causes**</span></th>
<th><span style="display: inline-block; text-align: left;">**Solutions**</span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">400 - Invalid Format</span></td>
<td><span style="display: inline-block; text-align: left;">Invalid request format</span></td>
<td><ul><li>Check if the JSON format is correct</li><li>Check if all required parameters are included</li><li>Check if parameter values are within the valid range</li><li>Check if the message format meets the interface requirements</li><li>Check if the model exists</li><li>Check if the fields are entered correctly</li><li>Check multimodal file input for compliance with format, size and other restrictions.</li><li>Check if multimodal file input is publicly accessible</li><li>In multi-turn conversations under thinking mode, the `reasoning_content` field must be fully passed back to the API.</li></ul></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">401 - Authentication Fails</span></td>
<td><ul><li>Missing or invalid API Key, or incorrect Authorization request header format</li><li>API Key that mixes Token Plan and Pay-as-you-go API</li></ul></td>
<td><ul><li>Check if the API key and request header format are correct</li><li>Check if a dedicated Base URL and API Key are used when using the Token Plan</li></ul></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">402 - Insufficient Balance</span></td>
<td><span style="display: inline-block; text-align: left;">Insufficient account balance</span></td>
<td><span style="display: inline-block; text-align: left;">Check your account balance and recharge in a timely manner</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">403 - Forbidden Access </span></td>
<td><span style="display: inline-block; text-align: left;">The service is currently not available in the current region, or the API Key has been restricted by risk control </span></td>
<td><span style="display: inline-block; text-align: left;">Create a new API Key and pay attention to the security of input content</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">404 - Not Found</span></td>
<td><span style="display: inline-block; text-align: left;">The requested endpoint or model does not support image input capability</span></td>
<td><span style="display: inline-block; text-align: left;">Verify that the model / endpoint being used supports image input capability</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">421 - Content Filter</span></td>
<td><span style="display: inline-block; text-align: left;">Content moderation and blocking</span></td>
<td><span style="display: inline-block; text-align: left;">Avoid entering unsafe or sensitive content</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">429 - Too Many Requests</span></td>
<td><span style="display: inline-block; text-align: left;">Requests are too frequent, or the quota of Token Plan has been exhausted</span></td>
<td><ul><li>Implement exponential backoff and retry logic, or reduce the request frequency</li><li>Upgrade the Token Plan package or switch to pay-as-you-go API</li></ul></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">500 - Server Error</span></td>
<td><span style="display: inline-block; text-align: left;">Our server encounters an issue</span></td>
<td><span style="display: inline-block; text-align: left;">Please try again later, or contact us for resolution</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">503 - Server Overloaded</span></td>
<td><span style="display: inline-block; text-align: left;">The server is overloaded due to high traffic</span></td>
<td><span style="display: inline-block; text-align: left;">Please try again later</span></td>
</tr>
</tbody>
</table>
