# Hermes Agent Configuration

{/* feishu-style:text-align:left */}
**Pay-as-you-go MiMo API** and **Token Plan** both support Hermes Agent. Refer to this guide for configuration and usage.

## Prerequisites

### Obtain Credentials 

{/* feishu-style:text-align:left */}
Supports two usage methods, but the corresponding credential acquisition methods are different:

## Install Hermes Agent

{/* feishu-style:text-align:left */}
Hermes Agent supports Linux, macOS, WSL2 (Windows), and more. For more information, refer to the [Hermes Agent Official Documentation](https://hermes-agent.nousresearch.com/docs/).

- Linux / macOS: No additional steps required.

- Windows: Refer to [Install WSL](https://learn.microsoft.com/en-us/windows/wsl/install) to install WSL2, then run the commands below in WSL2.

{/* feishu-style:text-align:left */}
**Installation Command:** 

```bash
curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash
```

{/* feishu-style:text-align:left */}
**After installation, reload the terminal environment:** 

```bash
source ~/.bashrc # or source ~/.zshrc
```

{/* feishu-style:text-align:left */}
**Verify installation (if a version number is displayed, the installation was successful):** 

```bash
hermes --version
```

{/* feishu-style:text-align:left */}
After installation, the following interface will appear:

## Configure a Predefined Provider

{/* feishu-style:text-align:left */}
**1. Select Quick Setup**

{/* feishu-style:text-align:left */}
Choose Quick setup for initial configuration.

> If not configured initially, you can re-enter the setup wizard via `hermes setup`.

{/* feishu-style:text-align:left */}
**2. Select Provider** `Xiaomi MiMo`

{/* feishu-style:text-align:left */}
**3. Fill in Configuration**

{/* feishu-style:text-align:left */}
Set API Key, Base URL, and default model as guided. The API Key and Base URL should be filled according to your credential type.

{/* feishu-style:text-align:left */}
Follow the remaining steps as needed.

**If you previously configured Pay-as-you-go MiMo API and need to switch to Token Plan:** 

- **Method 1:** Edit `~/.hermes/.env` file, replace `XIAOMI_API_KEY` and `XIAOMI_BASE_URL` with Token Plan credentials (open a new terminal after configuration).

- **Method 2:** Use a custom provider for configuration.

{/* feishu-style:text-align:left */}
**4. After configuration, the following interface will appear:** 

## Configure a Custom Provider

### Configure Basic Settings

{/* feishu-style:text-align:left */}
Replace `BASE_URL` and `MIMO_API_KEY` in the following methods with your actual credentials.

{/* feishu-style:text-align:left */}
**Method 1: Quick configuration via terminal commands**

Here `model.provider` can only be set to `custom`. Custom names like `xiaomi-coding` will be invalid.

```bash
hermes config set model.provider custom
hermes config set model.base_url BASE_URL
hermes config set model.api_key MIMO_API_KEY
hermes config set model.default mimo-v2.5-pro
```

{/* feishu-style:text-align:left */}
After configuration, you can view the settings in `~/.hermes/config.yaml`.

{/* feishu-style:text-align:left */}
**Method 2: Manually edit configuration file**

{/* feishu-style:text-align:left */}
Edit `~/.hermes/config.yaml` manually:

```bash
model:
 provider: custom
 base_url: BASE_URL
 api_key: MIMO_API_KEY
 default: mimo-v2.5-pro
```

### Verify Configuration

{/* feishu-style:text-align:left */}
After configuration, run the following command to verify:

```bash
hermes doctor
```

## Use Hermes Agent

{/* feishu-style:text-align:left */}
After configuration, run the following command to start:

```bash
hermes # Classic CLI mode
hermes --tui # Modern TUI mode
```