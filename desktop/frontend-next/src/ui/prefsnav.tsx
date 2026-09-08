import type { ReactNode } from "react";

/** The settings table of contents: which sections exist, what each is called,
 *  what mark it carries and which question it answers. Kept out of Settings
 *  itself because it is a table, not a screen — the screen reads it. */
export type Section = "session" | "model" | "tools" | "hooks" | "ext" | "network" | "remote" | "memory" | "usage" | "storage" | "account" | "versions" | "appearance" | "advanced";

// Drawn on the same 16-unit grid at 1.45 stroke as the rest of this screen's
// marks. The rail in Nav.tsx keeps its own set on purpose: those name panes to
// open, these name sections of one page, and the two lists barely overlap.
export const ICON: Record<Section, ReactNode> = {
  session: <path d="M2.6 8h10.8M8 2.6v10.8" />,
  model: (
    <>
      <circle cx="8" cy="8" r="2.4" />
      <path d="M8 1.8v2.2M8 12v2.2M1.8 8h2.2M12 8h2.2" />
    </>
  ),
  tools: (
    <>
      <path d="M8 1.9 13.8 4.4v4.2c0 3-2.4 5.1-5.8 5.7-3.4-.6-5.8-2.7-5.8-5.7V4.4Z" />
      <path d="M6.4 8.1 7.6 9.3l2.4-2.4" />
    </>
  ),
  hooks: (
    <>
      <path d="M4.4 2.6v6.2a3.2 3.2 0 0 0 6.4 0V6.2" />
      <circle cx="10.8" cy="4.4" r="1.6" />
    </>
  ),
  ext: (
    <>
      <rect x="2.4" y="2.4" width="5" height="5" rx="1.2" />
      <rect x="8.6" y="8.6" width="5" height="5" rx="1.2" />
      <path d="M7.4 5h6.2M5 7.4v6.2" />
    </>
  ),
  network: (
    <>
      <circle cx="8" cy="8" r="5.8" />
      <path d="M2.4 8h11.2M8 2.2c1.6 1.7 2.4 3.6 2.4 5.8S9.6 12.1 8 13.8C6.4 12.1 5.6 10.2 5.6 8s.8-4.1 2.4-5.8Z" />
    </>
  ),
  remote: (
    <>
      <rect x="2.2" y="2.6" width="11.6" height="4.4" rx="1.3" />
      <rect x="2.2" y="9" width="11.6" height="4.4" rx="1.3" />
      <path d="M4.8 4.8h.01M4.8 11.2h.01" />
    </>
  ),
  storage: (
    <>
      <ellipse cx="8" cy="4" rx="5.4" ry="2.1" />
      <path d="M2.6 4v8c0 1.2 2.4 2.1 5.4 2.1s5.4-.9 5.4-2.1V4" />
      <path d="M2.6 8c0 1.2 2.4 2.1 5.4 2.1s5.4-.9 5.4-2.1" />
    </>
  ),
  memory: <path d="M8 3.2c-1.4-1.3-4.6-1-4.6 1.9 0 2.4 2.6 4.4 4.6 6 2-1.6 4.6-3.6 4.6-6 0-2.9-3.2-3.2-4.6-1.9Z" />,
  usage: <path d="M2.6 12.4V9M6.2 12.4V5.4M9.8 12.4V7.2M13.4 12.4V3.4" />,
  account: (
    <>
      <circle cx="8" cy="5.6" r="2.6" />
      <path d="M2.9 13.4c.8-2.4 2.7-3.6 5.1-3.6s4.3 1.2 5.1 3.6" />
    </>
  ),
  versions: <path d="M8 2.4v7.4M5.2 7.2 8 10l2.8-2.8M3 12.8h10" />,
  appearance: (
    <>
      <circle cx="8" cy="8" r="3.1" />
      <path d="M8 1.4v1.8M8 12.8v1.8M1.4 8h1.8M12.8 8h1.8M3.4 3.4l1.3 1.3M11.3 11.3l1.3 1.3M12.6 3.4l-1.3 1.3M4.7 11.3l-1.3 1.3" />
    </>
  ),
  advanced: <path d="M3 5h5.2M11.2 5H13M3 11h2.2M8.2 11H13M9.4 3.4v3.2M6.4 9.4v3.2" />,
};

// Grouped by the question each answers, in the order they get asked: what this
// turn does, where it runs, what it keeps, what this machine is. Fourteen flat
// rows made the list something to scan; four short ones make it something to
// aim at.
export const NAV: [string, [Section, string][]][] = [
  [
    "这一轮",
    [
      ["session", "会话"],
      ["model", "模型"],
      ["tools", "工具与权限"],
      ["hooks", "自动化"],
      ["ext", "扩展"],
    ],
  ],
  [
    "在哪里跑",
    [
      ["network", "网络"],
      ["remote", "远程"],
      ["storage", "存储"],
    ],
  ],
  [
    "它记得什么",
    [
      ["memory", "记忆"],
      ["usage", "用量"],
    ],
  ],
  [
    "这台机器",
    [
      ["account", "账号"],
      ["versions", "版本"],
      ["appearance", "外观"],
      ["advanced", "高级"],
    ],
  ],
];

/** When a change to this setting is in force.
 *
 *  Three answers, and they are declared rather than derived: nothing here may
 *  be read off a handler's name, a hint's wording or a component's type. Each
 *  value below was taken from what the endpoint behind the control actually
 *  does — whether its handler reaches the kernel's runtime rebuild — not from
 *  what the screen says about itself.
 *
 *  immediate       canonical state changed and the running runtime already
 *                  shows it; nothing is reassembled.
 *  runtime-rebuild canonical state changed and this session's runtime is
 *                  rebuilt to adopt it. Refused while a turn is running.
 *  restart         written and kept now; this process goes on without it and
 *                  the next launch starts with it. Saved is not the same fact
 *                  as in force, and the row says both.
 *  none            nothing here changes anything: the block only reports. */
export type ApplySemantics = "immediate" | "runtime-rebuild" | "restart" | "none";

export interface SettingEntry {
  /** Which page it is on. */
  section: Section;
  /** The id the block renders, and what a search result scrolls to. */
  anchor: string;
  title: string;
  /** Words someone might look for that the title does not contain. These buy
   *  discoverability and nothing else: an alias never becomes the setting's
   *  name, its identity, or anything a judgement is made on. */
  keywords?: string[];
  apply: ApplySemantics;
}

// One row per block the settings screen renders, checked both ways against
// what it really renders — a block with no row here fails, and a row nothing
// renders fails too.
export const SETTINGS: SettingEntry[] = [
  { section: "session", anchor: "preset", title: "执行设定", apply: "immediate", keywords: ["均衡", "交付", "完成判定"] },
  { section: "session", anchor: "plan-mode", title: "计划模式", apply: "immediate", keywords: ["只读", "先规划"] },
  { section: "session", anchor: "session-dir", title: "这个会话在哪写", apply: "none", keywords: ["工作目录", "路径"] },

  { section: "model", anchor: "roles", title: "分工", apply: "runtime-rebuild", keywords: ["子代理", "规划", "执行", "审查"] },
  { section: "model", anchor: "model", title: "模型", apply: "runtime-rebuild", keywords: ["切换", "端点"] },
  { section: "model", anchor: "effort", title: "推理强度", apply: "runtime-rebuild", keywords: ["思考", "reasoning", "档位"] },
  { section: "model", anchor: "context", title: "上下文维护", apply: "runtime-rebuild", keywords: ["上下文窗口", "压缩", "compaction"] },
  // Adding a source does not rebuild; changing which protocol a source is
  // reached through switches the model, and that does. The stronger of the two
  // is what the row promises, because the weaker one would be a promise this
  // block cannot keep.
  { section: "model", anchor: "providers", title: "连接", apply: "runtime-rebuild", keywords: ["提供商", "api key", "密钥", "协议", "地址"] },

  { section: "tools", anchor: "approval", title: "工具批准", apply: "immediate", keywords: ["权限", "yolo", "询问", "放行"] },
  { section: "tools", anchor: "rules", title: "明确的规矩", apply: "runtime-rebuild", keywords: ["permissions", "允许", "拒绝", "配方"] },
  { section: "tools", anchor: "sandbox", title: "沙箱", apply: "runtime-rebuild", keywords: ["隔离", "联网", "写权限", "ssh-agent"] },
  { section: "tools", anchor: "shell", title: "命令交给谁执行", apply: "runtime-rebuild", keywords: ["bash", "powershell", "解释器"] },

  { section: "hooks", anchor: "hooks", title: "自动化", apply: "immediate", keywords: ["钩子", "hook", "触发"] },

  { section: "ext", anchor: "ext-runtime", title: "运行时", apply: "immediate", keywords: ["扩展", "沙盒"] },
  { section: "ext", anchor: "plugins", title: "插件包", apply: "immediate", keywords: ["安装", "市场"] },
  { section: "ext", anchor: "mcp", title: "外部工具", apply: "immediate", keywords: ["mcp", "服务器", "连接外部"] },
  { section: "ext", anchor: "skills", title: "技能", apply: "immediate", keywords: ["skill", "技能包"] },

  { section: "network", anchor: "network", title: "网络", apply: "immediate", keywords: ["代理", "proxy", "抓取", "超时"] },
  { section: "remote", anchor: "remote", title: "远程", apply: "immediate", keywords: ["ssh", "机器", "远端工作区"] },
  { section: "account", anchor: "account", title: "账号", apply: "immediate", keywords: ["登录", "社区"] },
  { section: "versions", anchor: "versions", title: "版本", apply: "immediate", keywords: ["更新", "升级"] },
  { section: "memory", anchor: "memory", title: "记忆", apply: "immediate", keywords: ["记住", "忘记", "事实"] },
  { section: "usage", anchor: "usage", title: "用量与成本", apply: "none", keywords: ["token", "花费", "缓存命中"] },
  { section: "storage", anchor: "storage", title: "存储", apply: "restart", keywords: ["搬家", "迁移", "磁盘", "位置"] },
  { section: "advanced", anchor: "elsewhere", title: "还不在这一版里", apply: "none", keywords: ["配置文件"] },

  { section: "appearance", anchor: "language", title: "语言", apply: "restart", keywords: ["中文", "english", "界面语言"] },
  { section: "appearance", anchor: "window", title: "窗口", apply: "immediate", keywords: ["托盘", "关闭行为"] },
  { section: "appearance", anchor: "size", title: "大小", apply: "immediate", keywords: ["缩放", "字号"] },
  { section: "appearance", anchor: "font", title: "字体", apply: "immediate", keywords: ["等宽", "mono"] },
  { section: "appearance", anchor: "wallpaper", title: "壁纸", apply: "immediate", keywords: ["背景", "图片"] },
  { section: "appearance", anchor: "weight", title: "文字粗细", apply: "immediate", keywords: ["加粗", "字重"] },
  { section: "appearance", anchor: "contrast", title: "文字对比度", apply: "immediate", keywords: ["柔和", "对比"] },
  { section: "appearance", anchor: "mode", title: "明暗", apply: "immediate", keywords: ["深色", "浅色", "跟随系统"] },
  { section: "appearance", anchor: "scheme", title: "配色", apply: "immediate", keywords: ["主题", "theme", "色板"] },
];

export const SETTING_AT = (anchor: string) => SETTINGS.find((s) => s.anchor === anchor);
