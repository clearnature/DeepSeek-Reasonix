# MiMo Code Released and Open-Sourced | Model Agent Collaborative Optimization, Stepping Towards the Self-Evolution Era

Today, we officially release and open source **MiMoCode V0.1.0** — an exploratory AI programming assistant running in the terminal.

MiMo Code **starts with programming but goes beyond it** . It is not just a useful AI Coding tool, but also an AI teammate who lives in your computer and understands you better the more you use it. 

It comes with a built-in top-tier MultiModal Machine Learning model that is free for a limited time **MiMo-V2.5,** whose performance is comparable to Claude Sonnet 4.6; at the same time, it supports the integration of mainstream models such as DeepSeek, Kimi, and GLM, as well as third-party Token Plans, to meet the needs of different developers. 

MiMo Code is based on the open-source project **OpenCode** for secondary development, **released and open-sourced under the MIT License** . 

## Core Competencies 

**Persistent Memory System + Infinite Context: Solving "AI Amnesia" at the Root**

MiMo Code incorporates a built-in and original persistent memory system, which uses three mechanisms - project memory, conversation checkpoints, and task progress - to address the long-standing challenge of "forgetting as you use" in long conversations. Even in long conversations spanning hundreds of rounds, it can maintain output quality and retain key information.

Most mainstream Code Agents (Claude Code, Codex, etc.) mostly operate on the principle of "letting AI take its own notes", but the model does not initiate this action on its own; whether and when to take notes depends entirely on its self-awareness. So we changed our approach: **Let the main agent focus on its tasks, and outsource the record-keeping entirely** - an independent subagent automatically saves the state, and when the window is almost full, it creates a clean summary, allowing the main agent to continue working instead of starting from scratch. 

One sentence: **Don't rely solely on the model's self-awareness; use engineering to support it.** 

![图片](https://mimo.mi.com/static/RMetbv6RWouvkJxIfP5ck9aKnKe.1bc21f29ab39583f.jpeg)

**Model Agent Collaborative Optimization + Original Compose Mode**

Most Coding Agents work in a way that is like "getting a requirement and then burying themselves in writing code", just like a driver who sets off without looking at the navigation - seemingly efficient, but actually prone to going off track.

However, models are not all the same: different models each have their own "personalities" and "endowments", and there are also natural differences in the level of "compatibility" with different Agent frameworks. Simply combining models and frameworks often fails to bring out their true capabilities.

MiMo Code has specifically designed a dedicated Harness system for the MiMo series of models, enabling the capabilities of the models to be deeply integrated with the framework; when combined with the original **Compose mode,** it achieves a synergistic effect of 1+1>2. 

When using it, simply press the **Tab key** to switch to Compose mode, give it a simple idea, and it can automatically complete the entire process of design, planning, coding, testing, and review, ultimately delivering an industrial-grade finished product.

**Actual measurement comparison**

We gave the same instructions to both tools:

"Help me implement a Redis using Golang, which needs to support connection via redis-cli." 

**Claude Code**moves quickly, and the code runs out soon—but there are almost no accompanying tests. The functionality works but is not robust enough, and the risk of subsequent rework is quite high.

**MiMoCode** uses the Compose pattern, which initially took more time for planning and seemed to be a bit "slower"; however, in terms of results, it has achieved more comprehensive functionality and is accompanied by complete and detailed testing, truly demonstrating what industrial-grade code should look like.

Interestingly,**MiMoCode is actually faster when it comes to calculating the general ledger:** it spends time thinking things through in the early stage and verifying them steadily in the later stage - slow writing, fast verification, resulting in a more worry-free overall experience.

**Dream: Memory settles, the more you use it, the better it understands you**

MiMoCode has a built-in unique `/dream `command. It is automatically triggered every 7 days, with an independent Agent reading historical conversations and existing memory files, performing merging, deduplication, validating path effectiveness, and compression, converging scattered memories into a compact current state, and updating the global memory.

By the next use, it will automatically recall these memories at the appropriate time. This means that MiMo Code does not start from scratch every time, but continues to grow with an understanding of you and your project—truly becoming more user-friendly the more you use it.

**Supports voice input: "A gentleman uses his words, not his hands."** 

MiMo Code comes with built-in voice input and control capabilities, powered by the robust speech recognition capabilities of MiMo-V2.5-ASR. Just speak, and the work gets done.

What it can do is not just "reading out the prompt": you can verbally correct a wrongly written instruction, or directly issue operation commands such as "send" or "execute" - from input to control, without touching the keyboard throughout the process, and efficiency naturally takes another step up. 

## Let Data Do the Talking: Same Model, Stronger Performance

On two authoritative test sets **SWE-Bench** and **Terminal Bench** that target real-world programming scenarios, we conducted a set of Controlled Experiments: we let MiMo Code and Claude Code **use the same MiMo model,** and only compare their respective Agent systems themselves. 

Results show that MiMo Code achieved **62% (** Claude Code achieved 57%) on SWE-Bench Pro, and **73% (** Claude Code achieved 68%) on Terminal Bench 2 —— on the premise that the models are exactly the same, MiMo Code obtained a better score through the synergy of its exclusive Harness and Compose mode.

![图片](https://mimo.mi.com/static/Ki8Bb4voCosxwTxqbhScDg6ynTe.32608041eeeb3006.jpeg)

## How to Use: Zero Configuration, Out Of The Box 

**Installation and Startup:** Open the terminal

- Recommended for Mac and Linux users: `curl -fsSL https://mimo.xiaomi.com/install | bash`

- Windows users are recommended to use npm:`npm install -g @mimo-ai/cli`

After installation, enter ` mimo in the terminal ` to start. For the best experience, it is highly recommended **Mac users to use it in iTerm or the vscode terminal.** 

- **Model Configuration**

 - Built-in MiMo-V2.5 Limited-Time**Free Channel,** Available Without Registration

 - Compatible with mainstream model APIs such as DeepSeek / Kimi / GLM, as well as third-party Token Plans 

- **Usage:** 

 - Input `/ ` View Each Configuration 

 - All settings are fully Chinese localized, making it locally friendly

 - The right side of the TUI page has a permanent status dashboard for observing work progress at any time 

For more technical details, please follow our team Blog:https://mimo.xiaomi.com/mimocode

## Open Source and Outlook 

MiMo Code **was released and open-sourced** , under the permissive MIT License — which means it is open to almost everyone: 

- **Individual developers** can freely use, modify, and distribute, and do whatever they want;

- **Enterprises** can integrate it into their own development toolchains without worrying about licensing constraints; 

- **The community** can build vertical programming assistants based on it, giving rise to more possibilities. 

We believe that the value of a good tool lies not only in its functionality but also in how many people can contribute to its refinement and where it is headed. We look forward to working with you to make MiMo Code even better.