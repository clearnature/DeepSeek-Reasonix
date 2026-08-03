# Qwen Code Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Qwen Code. Refer to this guide for configuration and usage.

## Prerequisites

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

<table>
<colgroup>
<col style="width: 143px" />
<col style="width: 191px" />
<col style="width: 700px" />
</colgroup>
<thead>
<tr>
<th><span style="display: inline-block; text-align: center;">Usage Method</span></th>
<th><span style="display: inline-block; text-align: center;">Description</span></th>
<th><span style="display: inline-block; text-align: center;">Acquisition Method (BASE_URL and API Key below are examples) </span></th>
</tr>
</thead>
<tbody>
<tr>
<td><span style="display: inline-block; text-align: left;">Pay-as-you-go MiMo API</span></td>
<td><span style="display: inline-block; text-align: left;">Charged based on actual usage, suitable for light use</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://api.xiaomimimo.com/v1`</li><li>Anthropic Compatibility Protocol: `https://api.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `sk-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">Go to [API Keys](https://platform.xiaomimimo.com/#/console/api-keys) to create an API Key</span></td>
</tr>
<tr>
<td><span style="display: inline-block; text-align: left;">Token Plan</span></td>
<td><span style="display: inline-block; text-align: left;">Fixed subscription fee, with limited calls based on the package</span></td>
<td><ul><li>BASE_URL<ul><li>OpenAI Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/v1`</li><li>Anthropic Compatibility Protocol: `https://token-plan-cn.xiaomimimo.com/anthropic`</li></ul></li><li>API Key<ul><li>Format: `tp-xxxxx`</li></ul></li></ul><br /><span style="display: inline-block; text-align: left;">After successful subscription, go to [Token Plan](https://platform.xiaomimimo.com/#/console/plan-manage) to obtain the exclusive Base URL and API Key </span></td>
</tr>
</tbody>
</table>

## Use Qwen Code CLI

### Install Qwen Code CLI

{/* feishu-style:text-align:left */}
**Installation commands:** 

- macOS/Linux

   ```bash
   bash -c "$(curl -fsSL https://qwen-code-assets.oss-cn-hangzhou.aliyuncs.com/installation/install-qwen.sh)" -s --source bailian
   ```

- Windows

   ```bash
   curl -fsSL -o %TEMP%\install-qwen.bat https://qwen-code-assets.oss-cn-hangzhou.aliyuncs.com/installation/install-qwen.bat && %TEMP%\install-qwen.bat --source bailian
   ```

{/* feishu-style:text-align:left */}
**Verify the installation (a version number output indicates success):** 

```bash
qwen --version
```

### Configure Settings

{/* feishu-style:text-align:left */}
**1.**  **Select API Key -> Custom API Key to enter custom configuration**

<img src="https://mimo.mi.com/static/KFCmbZCRkoH1IOxx6x0cEzcEnAc.e5a1b0df5d9a9b96.png" alt="图片" style="margin: 16px auto;" />

<img src="https://mimo.mi.com/static/AlRibRrhIoD2S3x3YhRcS4JFnSd.b7cfac14e661c925.png" alt="图片" style="margin: 16px auto;" />

{/* feishu-style:text-align:left */}
**2.**  **Edit the configuration file**

<img src="https://mimo.mi.com/static/XPfhbXATyoSJcOx4EsaclpSTnYc.50a9828f35e788a1.png" alt="图片" style="margin: 16px auto;" />

<div className='mdx-highlight mdx-highlight-info' data-highlight-icon='info'>

For more detailed configuration information, visit the [Qwen Code official configuration documentation](https://qwenlm.github.io/qwen-code-docs/en/users/configuration/model-providers/).

</div>

{/* feishu-style:text-align:left */}
Edit or create the `settings.json` file at the following path:

- macOS/Linux: `~/.qwen/settings.json`

- Windows: `User directory\.qwen\settings.json`

{/* feishu-style:text-align:left */}
Copy the following content into the configuration file (replace with your actual settings when using):

<div className='mdx-highlight mdx-highlight-warning' data-highlight-icon='warning'>

When configuring basic information, you need to first check if the `MIMO_API_KEY` environment variable exists. If it does, please clear it or replace the value with the API Key obtained through the corresponding usage method.

</div>

```bash
{
  "env": {
    "MIMO_API_KEY": "MIMO_API_KEY"
  },
  "modelProviders": {
    "openai": [
      {
        "id": "mimo-v2.5-pro",
        "name": "mimo-v2.5-pro",
        "baseUrl": "BASE_URL",
        "envKey": "MIMO_API_KEY"
      }
    ]
  },
  "security": {
    "auth": {
      "selectedType": "openai"
    }
  },
  "model": {
    "name": "mimo-v2.5-pro"
  },
  "$version": 3
}
```

### Use Qwen Code CLI

{/* feishu-style:text-align:left */}
After completing the above configuration, open a new terminal and run the following command to start Qwen Code CLI:

```bash
qwen
```

{/* feishu-style:text-align:left */}
Once started, you can use MiMo models in Qwen Code CLI.

<img src="https://mimo.mi.com/static/DZQYb5P6EoqfmAxCScDcRIkwnWc.711ef024c64aca4b.png" alt="图片" style="margin: 16px auto;" />

## Use the Qwen Code IDE Plugin

### Install the Plugin

{/* feishu-style:text-align:left */}
Search for and install the **Qwen Code Companion** plugin from the VS Code Extensions marketplace.

<img src="https://mimo.mi.com/static/Fe4qbzUcnoJswXxiLw0cgpcBncb.817b38485b942fea.png" alt="图片" style="margin: 16px auto;" />

### Configure Settings

{/* feishu-style:text-align:left */}
Follow the same steps as described in the Qwen Code CLI configuration section above.

### Use the Qwen Code Plugin

{/* feishu-style:text-align:left */}
Click the Qwen Code icon in the top-right corner to open the dialog.

<img src="https://mimo.mi.com/static/TZEyb0zIDodYvgxHbqscutfAnff.312163f2e1e2886f.png" alt="图片" style="margin: 16px auto;" />

{/* feishu-style:text-align:left */}
Type or click `/`, then select `Switch model` to change the model.

<img src="https://mimo.mi.com/static/VQftbMiUKoFYYWx2Xsycbj0FnGf.bc73a7c00de4f7b7.png" alt="图片" style="margin: 16px auto;" />
