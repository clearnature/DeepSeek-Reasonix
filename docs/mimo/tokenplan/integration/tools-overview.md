# Overview of AI Tools

**The pay-as-you-go MiMo API and Token Plan subscription packages,**  are both supported for use in the following mainstream AI programming tools (the tool list is continuously updated), click to view the detailed access and usage guide for the corresponding tool.

## Use Tools

<ToolGrid />

## Configuration Methods for Other Tools

> Core Steps:
>
> 1. Find a compatible OpenAI protocol or Anthropic protocol, and support custom configuration Provider 
>
> 1. Replace or add as the Base URL for the corresponding protocol
>
> 1. Enter API Key, select or add MiMo model 

Supports two usage methods, but the corresponding credential acquisition methods are different:

<table>
<colgroup>
<col style="width: 143px" />
<col style="width: 191px" />
<col style="width: 700px" />
</colgroup>
<thead>
<tr>
<th>Usage Method</th>
<th>Description</th>
<th>Acquisition Method (BASE_URL and API Key below are examples)</th>
</tr>
</thead>
<tbody>
<tr>
<td>Pay-as-you-go MiMo API</td>
<td>Charged based on actual usage, suitable for light use</td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://api.xiaomimimo.com/v1`</li><li>Anthropic Compatibility Protocol: `https://api.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `sk-xxxxx`</li></ul></li></ul><br />Go to [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) to create an API Key</td>
</tr>
<tr>
<td>Token Plan</td>
<td>Fixed subscription fee, with limited calls based on the package</td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/v1`</li><li>Anthropic Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `tp-xxxxx`</li></ul></li></ul><br />After successful subscription, go to [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) to obtain the exclusive Base URL and API Key</td>
</tr>
</tbody>
</table>
