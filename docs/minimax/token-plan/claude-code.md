# token-plan/claude-code.md

> 来源: https://platform.minimaxi.com/docs/token-plan/claude-code.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Claude Code

> 在 Claude Code 中使用最新的 MiniMax M 系列模型进行 AI 编程。

<div style={{background:"#fffbeb",borderLeft:"4px solid #d97706",padding:"12px 16px",borderRadius:"6px",margin:"16px 0"}}>[**Claude Code**](https://github.com/anthropics/claude-code) 是 Anthropic 出品的官方终端原生编程 Agent，由 Claude 驱动。</div>

## 安装 Claude Code

可参考 [Claude Code 文档](https://code.claude.com/docs/en/setup) 进行安装。

## 配置 MiniMax API

<Warning>
  **重要提示**：

  在配置前，请确保清除以下 Anthropic 相关的环境变量，以免影响 MiniMax API 的正常使用：

  * `ANTHROPIC_AUTH_TOKEN`
  * `ANTHROPIC_BASE_URL`

  ```bash theme={null}
  unset ANTHROPIC_AUTH_TOKEN
  unset ANTHROPIC_BASE_URL
  ```

  若以上变量在 `~/.bashrc` / `~/.zshrc` 中被永久导出，请同步删除对应行，否则新开 shell 会再次注入。
</Warning>

<Steps>
  <Step title="API 配置">
    <Tabs>
      <Tab title="手动编辑配置文件（推荐）">
        **1. 编辑配置文件**

        编辑或创建 Claude Code 的配置文件（MacOS & Linux 为 `~/.claude/settings.json`，Windows 为 `用户目录/.claude/settings.json`），将 `MINIMAX_API_KEY` 替换为您的 MiniMax API Key。环境变量 `ANTHROPIC_AUTH_TOKEN` 和 `ANTHROPIC_BASE_URL` 优先级高于配置文件。

        ```json theme={null}
        {
          "env": {
            "ANTHROPIC_BASE_URL": "https://api.minimaxi.com/anthropic",
            "ANTHROPIC_AUTH_TOKEN": "<MINIMAX_API_KEY>",
            "CLAUDE_CODE_AUTO_COMPACT_WINDOW": "1000000",
            "ANTHROPIC_MODEL": "MiniMax-M3[1m]",
            "ANTHROPIC_DEFAULT_SONNET_MODEL": "MiniMax-M3[1m]",
            "ANTHROPIC_DEFAULT_OPUS_MODEL": "MiniMax-M3[1m]",
            "ANTHROPIC_DEFAULT_HAIKU_MODEL": "MiniMax-M3[1m]"
          }
        }
        ```

        其中 `CLAUDE_CODE_AUTO_COMPACT_WINDOW` 用于将 Claude Code 的自动压缩阈值设为 1000000，与 MiniMax-M3 当前的上下文窗口保持一致。

        **2. 新增 onboarding 标记**

        编辑或新增 `.claude.json` 文件（MacOS & Linux 为 `~/.claude.json`，Windows 为 `用户目录/.claude.json`），新增 `hasCompletedOnboarding` 参数。

        ```json theme={null}
        {
          "hasCompletedOnboarding": true
        }
        ```
      </Tab>

      <Tab title="使用 cc-switch">
        [cc-switch](https://github.com/farion1231/cc-switch) 是一个便捷的工具，可以快速切换 Claude Code 的 API 配置。

        **1. 安装 cc-switch**

        <Tabs>
          <Tab title="macOS / Linux">
            ```bash theme={null}
            brew tap farion1231/ccswitch
            brew install --cask cc-switch
            brew upgrade --cask cc-switch
            ```
          </Tab>

          <Tab title="Windows">
            前往 [cc-switch GitHub Releases](https://github.com/farion1231/cc-switch/releases) 页面下载最新版本的安装包。
          </Tab>
        </Tabs>

        **2. 添加 MiniMax 配置**

        启动 cc-switch，点击右上角 **"+"** ，选择预设的 MiniMax 供应商，并填写从 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/payment/token-plan) (国际用户可访问 [MiniMax Developer Platform](https://platform.minimax.io/user-center/payment/token-plan)) 获取的 MiniMax API Key。 <img src="https://filecdn.minimax.chat/public/0acbfee9-8871-4171-af19-e318476456a4.png" alt="choose" />

        **3. 配置模型名称**

        将模型名称全部改为 `MiniMax-M3`，完成后点击右下角的 **"添加"**。 <img src="https://filecdn.minimax.chat/public/1ceadee0-5488-44a1-82bb-94af0fc8d3b7.png" alt="add" />

        如需将自动压缩阈值与 MiniMax-M3 当前的 1M 上下文窗口对齐，可参照「手动编辑配置文件」标签页，在 `~/.claude/settings.json` 的 `env` 中加入 `"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "1000000"`。

        **4. 启用配置**

        回到首页，点击 **"启用"** <img src="https://filecdn.minimax.chat/public/0c5cbe27-1a6d-4583-9ad9-b48222055c3b.png" alt="start" />

        **5. 编辑配置文件**

        编辑或新增 `.claude.json` 文件（MacOS & Linux 为 `~/.claude.json`，Windows 为 `用户目录/.claude.json`），新增 `hasCompletedOnboarding` 参数。

        ```json theme={null}
        {
          "hasCompletedOnboarding": true
        }
        ```
      </Tab>
    </Tabs>
  </Step>

  <Step title="启动 Claude Code">
    配置完成后，进入工作目录，在终端中运行 `claude` 命令以启动 Claude Code
  </Step>

  <Step title="信任文件夹">
    启动后，选择 **信任此文件夹 (Trust This Folder)**，以允许 Claude Code 访问该文件夹中的文件，随后开始在 Claude Code 中使用 MiniMax-M3

    ![](https://filecdn.minimax.chat/public/7ca00f05-81bd-4058-a357-3bb79eabd738.jpg)
  </Step>
</Steps>

<Warning>
  **重要提示**：

  在配置完成后，如果您还想要使用 网络搜索 能力，则需要根据 [此教程](https://platform.minimaxi.com/docs/token-plan/mcp-guide) 来配置 网络搜索 MCP
</Warning>

## 验证配置生效

启动 `claude` 后，在 TUI 中依次输入以下 slash 命令，确认已切换到 MiniMax：

```text theme={null}
/status
/model
```

* `/status` 应显示 `ANTHROPIC_BASE_URL` 指向 `api.minimaxi.com/anthropic`（国际用户为 `api.minimax.io/anthropic`）。
* `/model` 应显示当前模型为 `MiniMax-M3`。

<Tip>
  MiniMax-M3 支持 Claude Code 的扩展思考（Extended Thinking），默认开启。如需开启或关闭，运行 `/config` 并将 **Thinking mode** 设为 `true` 或 `false`。也可随时通过 `Option+T`（macOS）或 `Alt+T`（Windows/Linux）切换。
</Tip>
