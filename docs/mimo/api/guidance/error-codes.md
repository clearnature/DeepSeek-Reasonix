# Error Codes

{/* feishu-style:text-align:left */}
When using API calls to the MiMo model, common error codes and solutions are as follows:

| **Error Code** | **Causes** | **Solutions** |
|---|---|---|
| 400 - Invalid Format | Invalid request format | Check if the JSON format is correctCheck if all required parameters are includedCheck if parameter values are within the valid rangeCheck if the message format meets the interface requirementsCheck if the model existsCheck if the fields are entered correctlyCheck multimodal file input for compliance with format, size and other restrictions.Check if multimodal file input is publicly accessibleIn multi-turn conversations under thinking mode, the `reasoning_content` field must be fully passed back to the API. |
| 401 - Authentication Fails | Missing or invalid API Key, or incorrect Authorization request header formatAPI Key that mixes Token Plan and Pay-as-you-go API | Check if the API key and request header format are correctCheck if a dedicated Base URL and API Key are used when using the Token Plan |
| 402 - Insufficient Balance | Insufficient account balance | Check your account balance and recharge in a timely manner |
| 403 - Forbidden Access | The service is currently not available in the current region, or the API Key has been restricted by risk control | Create a new API Key and pay attention to the security of input content |
| 404 - Not Found | The requested endpoint or model does not support image input capability | Verify that the model / endpoint being used supports image input capability |
| 421 - Content Filter | Content moderation and blocking | Avoid entering unsafe or sensitive content |
| 429 - Too Many Requests | Requests are too frequent, or the quota of Token Plan has been exhausted | Implement exponential backoff and retry logic, or reduce the request frequencyUpgrade the Token Plan package or switch to pay-as-you-go API |
| 500 - Server Error | Our server encounters an issue | Please try again later, or contact us for resolution |
| 503 - Server Overloaded | The server is overloaded due to high traffic | Please try again later |