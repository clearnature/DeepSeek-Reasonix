# token-plan/trae.md

> 来源: https://platform.minimaxi.com/docs/token-plan/trae.md

> ## Documentation Index
> Fetch the complete documentation index at: https://platform.minimaxi.com/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# TRAE

> 在 TRAE 中使用最新的 MiniMax M 系列模型进行 AI 编程。

<div style={{background:"#fffbeb",borderLeft:"4px solid #d97706",padding:"12px 16px",borderRadius:"6px",margin:"16px 0"}}>[**TRAE**](https://www.trae.ai) 是字节跳动出品的 AI-native IDE，主打智能协作与 Agent 自动化。</div>

## 安装 TRAE

<Steps>
  <Step title="下载并安装 TRAE">
    访问 [TRAE 官网](https://www.trae.cn/) 下载并安装 TRAE
  </Step>

  <Step title="完成初始设置">
    TRAE 首次启动时，你会进入以下页面，请根据指引完成初始设置

    ![](https://filecdn.minimax.chat/public/d8854624-3d7e-49b8-93df-27c1260c5f6c.PNG)
  </Step>

  <Step title="登录 TRAE">
    登录 TRAE
  </Step>

  <Step title="选择 MiniMax-M3 模型">
    TRAE 中国版内置了 MiniMax-M3 模型，你可以直接选用，开启高效编程之旅
  </Step>
</Steps>

## 在 TRAE 中使用自定义模型

TRAE 还支持通过 **API Key** 接入自定义模型，从而满足您的需求。

<Steps>
  <Step title="打开设置">
    在 AI 对话框右上角，点击 **设置** 图标
  </Step>

  <Step title="选择模型页签" />

  <Step title="添加模型">
    点击 **+ 添加模型** 按钮，选择服务商 **MiniMax-CN**，模型选择 **MiniMax-M3**
  </Step>

  <Step title="填写 API Key">
    填写从 [MiniMax 开放平台](https://platform.minimaxi.com/user-center/payment/token-plan) (国际用户可访问 [MiniMax Developer Platform](https://platform.minimax.io/user-center/payment/token-plan)) 获取的 **MiniMax API Key**
  </Step>

  <Step title="完成添加">
    点击 **添加模型** 按钮

    <Note>
      TRAE 将调用服务商的接口来检测 API Key 是否有效。可能的结果如下：

      * 若连接成功，该自定义模型会被添加。
      * 若连接失败，添加模型 窗口中展示错误信息和服务商返回的错误日志，你可以参考这些信息排查问题。
    </Note>
  </Step>
</Steps>
