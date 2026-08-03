# Codex Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Codex. Refer to this guide for configuration and usage.

MiMo now supports the Responses API. Users who are using the Chat Completions API can follow this document to update.

## Prerequisites

### Obtain Credentials

{/* feishu-style:text-align:left */}
Two usage methods are supported, but the corresponding credential acquisition methods differ:

## Install Codex

{/* feishu-style:text-align:left */}
**Prerequisites:** Node.js 18 or a later version must be installed first.

{/* feishu-style:text-align:left */}
**Installation command:** 

```bash
npm install -g @openai/codex
```

{/* feishu-style:text-align:left */}
**Verify the installation (a version number output indicates success):** 

```bash
codex --version
```

## Edit Configuration File

**Notes**

- When configuring basic information, first check if the `MIMO_API_KEY` environment variable exists. If it does, please clear it or replace the value with the API Key obtained through the corresponding usage method.

- To switch models directly via the `/model` command, refer to the "Further Reading - Custom MiMo Model Configuration" section and configure the `model-catalogs.json` file.

{/* feishu-style:text-align:left */}
Configuration file paths:

- macOS/Linux: `~/.codex/config.toml`

- Windows: `User directory\.codex\config.toml`

- Below are the complete configuration examples. The `model` field can be changed to other supported models (e.g., `mimo-v2.5`) as needed.

### Pay-as-you-go

{/* feishu-style:text-align:left */}
**1. Edit or create the configuration file** `config.toml`

```bash
model = "mimo-v2.5-pro"
model_provider = "mimo"
model_reasoning_effort = "high"

# Enable model reasoning summaries; if set to false, model_reasoning_effort will not take effect even if configured
model_supports_reasoning_summaries = true
model_reasoning_summary = "none"
model_context_window = 1048576

web_search = "disabled"

[model_providers.mimo]
name = "mimo"
base_url = "https://api.xiaomimimo.com/v1"
env_key = "MIMO_API_KEY"
wire_api = "responses"
```

{/* feishu-style:text-align:left */}
**2. Configure the environment variable** `MIMO_API_KEY`

{/* feishu-style:text-align:left */}
Pay-as-you-go uses an API Key starting with `sk-`.

- macOS/Linux

 ```bash
 echo 'export MIMO_API_KEY="sk-your-api-key-here"' >> ~/.bashrc
 source ~/.bashrc
 ```

- Windows (CMD)

 ```bash
 # Run the following command in CMD
 setx MIMO_API_KEY "sk-your-api-key-here"

 # After success, open a new command prompt and run the following to verify the variable is set.
 echo %MIMO_API_KEY%
 ```

### Token Plan

{/* feishu-style:text-align:left */}
**1. Edit or create the configuration file** `config.toml`

```bash
model = "mimo-v2.5-pro"
model_provider = "mimo"
model_reasoning_effort = "high"

# Enable model reasoning summaries; if set to false, model_reasoning_effort will not take effect even if configured
model_supports_reasoning_summaries = true
model_reasoning_summary = "none"
model_context_window = 1048576

web_search = "disabled"

[model_providers.mimo]
name = "mimo"
base_url = "https://token-plan-cn.xiaomimimo.com/v1"
env_key = "MIMO_API_KEY"
wire_api = "responses"
```

{/* feishu-style:text-align:left */}
**2. Configure the environment variable** `MIMO_API_KEY`

{/* feishu-style:text-align:left */}
Token Plan uses an API Key starting with `tp-`.

- macOS/Linux

 ```bash
 echo 'export MIMO_API_KEY="tp-your-api-key-here"' >> ~/.bashrc
 source ~/.bashrc
 ```

- Windows (CMD)

 ```bash
 # Run the following command in CMD
 setx MIMO_API_KEY "tp-your-api-key-here"

 # After success, open a new command prompt and run the following to verify the variable is set.
 echo %MIMO_API_KEY%
 ```

## Use Codex CLI

{/* feishu-style:text-align:left */}
After completing the above configuration, open a new terminal and run the following command to start Codex.

```bash
codex
```

## Use Codex IDE Plugin

{/* feishu-style:text-align:left */}
Codex provides a VS Code plugin. Search for "Codex" in the VS Code Extension Marketplace to install it. The plugin automatically reuses the existing local Codex configuration. If you have never used the Codex command-line tool, please follow the specifications in the "Edit Configuration File" section to complete the configuration.

## Further Reading

### Custom MiMo Model Configuration

{/* feishu-style:text-align:left */}
Codex supports custom model parameter configuration, allowing fine-grained and personalized adaptation for MiMo models. After configuration, type `/model` in the Codex CLI to see MiMo models and their corresponding reasoning levels in the model list, and switch between them at any time.

#### Key Field Descriptions

#### Configuration Steps

{/* feishu-style:text-align:left */}
1. Edit or create the `.codex/model-catalogs/model-catalogs.json` file for detailed MiMo model configuration (Take `mimo-v2.5-pro` and `mimo-v2.5` as examples)

```json
{
 "models": [
 {
 "slug": "mimo-v2.5-pro",
 "display_name": "mimo-v2.5-pro",
 "description": "MiMo-v2.5-Pro: Trillion-parameter Flagship Agent Foundation",
 "default_reasoning_level": "high",
 "supported_reasoning_levels": [
 {
 "effort": "none",
 "description": "Disable Thinking"
 },
 {
 "effort": "high",
 "description": "Enabled Thinking"
 }
 ],
 "shell_type": "shell_command",
 "visibility": "list",
 "supported_in_api": true,
 "priority": 0,
 "base_instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today's date: {date} {week}. Your knowledge cutoff date is December 2024.",
 "supports_reasoning_summaries": true,
 "default_reasoning_summary": "none",
 "support_verbosity": false,
 "truncation_policy": {
 "mode": "bytes",
 "limit": 10000
 },
 "supports_parallel_tool_calls": false,
 "supports_image_detail_original": false,
 "context_window": 1048576,
 "max_context_window": 1048576,
 "effective_context_window_percent": 95,
 "experimental_supported_tools": [],
 "input_modalities": ["text"],
 "supports_search_tool": false
 },
 {
 "slug": "mimo-v2.5",
 "display_name": "mimo-v2.5",
 "description": "MiMo-V2.5: Native Omni-modal Perception Model",
 "default_reasoning_level": "high",
 "supported_reasoning_levels": [
 {
 "effort": "none",
 "description": "Disable Thinking"
 },
 {
 "effort": "high",
 "description": "Enabled Thinking"
 }
 ],
 "shell_type": "shell_command",
 "visibility": "list",
 "supported_in_api": true,
 "priority": 1,
 "base_instructions": "You are MiMo, an AI assistant developed by Xiaomi. Today's date: {date} {week}. Your knowledge cutoff date is December 2024.",
 "supports_reasoning_summaries": true,
 "default_reasoning_summary": "none",
 "support_verbosity": false,
 "truncation_policy": {
 "mode": "bytes",
 "limit": 10000
 },
 "supports_parallel_tool_calls": false,
 "supports_image_detail_original": true,
 "context_window": 1048576,
 "max_context_window": 1048576,
 "effective_context_window_percent": 95,
 "experimental_supported_tools": [],
 "input_modalities": ["text", "image"],
 "supports_search_tool": false
 }
 ]
}

```

{/* feishu-style:text-align:left */}
2. Add the following configuration at the root level of the `config.toml` file

- macOS/Linux

 ```bash
 model_catalog_json = "~/.codex/model-catalogs/model-catalogs.json"
 ```

- Windows

 ```bash
 model_catalog_json = "User directory/.codex/model-catalogs/model-catalogs.json"
 ```