# CodeBuddy Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support CodeBuddy. Refer to this guide for configuration and usage.

## Prerequisites

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

## Use CodeBuddy IDE

### Install CodeBuddy

{/* feishu-style:text-align:left */}
Visit the [CodeBuddy website](https://www.codebuddy.ai/home) to download and install the IDE, which supports major operating systems (Windows, macOS).

### Configure MiMo Model

{/* feishu-style:text-align:left */}
**1. Configure Custom Model**

{/* feishu-style:text-align:left */}
Create or modify the configuration file `models.json` to add custom models. Example configuration:

- **macOS:** `~/.codebuddy/models.json`

- **Windows:** `User Directory\.codebuddy\models.json`

{/* feishu-style:text-align:left */}
`BASE_URL` and `MIMO_API_KEY` should be modified according to your credential acquisition method.

```json
{
 "models": [
 {
 "id": "mimo-v2.5-pro",
 "name": "mimo-v2.5-pro",
 "vendor": "MiMo",
 "apiKey": "MIMO_API_KEY",
 "url": "BASE_URL/chat/completions",
 "supportsToolCall": true,
 "supportsImages": false
 },
 {
 "id": "mimo-v2.5",
 "name": "mimo-v2.5",
 "vendor": "MiMo",
 "apiKey": "MIMO_API_KEY",
 "url": "BASE_URL/chat/completions",
 "supportsToolCall": true,
 "supportsImages": true
 }
 ]
}
```

{/* feishu-style:text-align:left */}
**2. View and Switch Models**

{/* feishu-style:text-align:left */}
After configuration, turn off `Auto mode` and open the model list to see the configured MiMo models.

### Use MiMo Model

{/* feishu-style:text-align:left */}
Select the configured model to start conversations, coding, and other operations.

## Use CodeBuddy IDE Plugin

### Install Plugin

{/* feishu-style:text-align:left */}
Search for `Tencent Cloud CodeBuddy` in the VS Code extension marketplace and install the plugin.

### Configure MiMo Model

{/* feishu-style:text-align:left */}
Refer to the `models.json` configuration file in the "Use CodeBuddy IDE" section. If previously configured, it will be automatically loaded.

## Use CodeBuddy CLI

### Install CodeBuddy CLI

{/* feishu-style:text-align:left */}
**Install via npm (requires Node.js 18.20 or newer):** 

```bash
npm install -g @tencent-ai/codebuddy-code
```

{/* feishu-style:text-align:left */}
**Verify installation (if a version number is displayed, the installation was successful):** 

```bash
codebuddy --version
```

### Configure MiMo Model

The `BASE_URL` and `API Key` for **Pay-as-you-go MiMo API** and **Token Plan** are different. Please configure accordingly.

{/* feishu-style:text-align:left */}
Refer to the `models.json` configuration file in the "Use CodeBuddy IDE" section. If previously configured, it will be automatically loaded.

### Use CodeBuddy CLI

{/* feishu-style:text-align:left */}
After configuration, navigate to your project directory and run:

```bash
codebuddy
```

{/* feishu-style:text-align:left */}
After startup, use `/model` to view or switch models, and `/status` to check the current model.

## FAQ

### Model not appearing in the dropdown after configuration? 

- Check if the JSON syntax is correct

- If the `availableModels` field is configured, ensure the model id is included