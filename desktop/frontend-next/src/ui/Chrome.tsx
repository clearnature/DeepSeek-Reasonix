import { useEffect, useState } from "react";
import { t } from "../i18n";
import type { AccountState, AgentPort, SessionStatus, WorkspaceInfo } from "../port/port";
import { WindowControls, zoomOnTitleBar } from "./WindowControls";
import { chord } from "./keys";

const RUN_LB: Record<string, string> = { running: "运行中", halt: "等待确认", done: "已完成", idle: "待命" };

const base = (p: string) => p.replace(/[/\\]+$/, "").split(/[/\\]/).pop() || p;
// The filename is a timestamp and a model ref — true, and useless to read. The
// title from /sessions is always present once the turn is on disk: a generated
// one when it is ready, the first message truncated until then.
const sessionName = (title?: string, p?: string) =>
  title?.trim() || (p ? base(p).replace(/\.jsonl$/, "") : t("新会话"));

interface Props {
  // Set when the focused pane is driven from another machine, naming it.
  host?: string;
  // Null while the window has no pane: the chrome still draws, and the few
  // controls that need a session simply do nothing until one is focused.
  port: AgentPort | null;
  status: SessionStatus | null;
  title?: string;
  steer: number;
  // What the focused pane is doing, as the pane already reports it upward. The
  // chrome names it because that is the one line always on screen.
  run: string;
  // 临时观看状态，不是偏好：窗口决定它，这里只负责把开关画出来。
  focus: boolean;
  onFocus: () => void;
  onSettings: (section?: string) => void;
  account: AccountState | null;
}

export function Chrome({ port, status, title, steer, run, onSettings, account, host, focus, onFocus }: Props) {
  const root = status?.workspaceRoot || status?.cwd || "";
  const project = root ? base(root) : "—";
  // Only for the "隔离" tag: the folder list and the switch itself moved to the
  // sidebar, where adding one and opening one are the same gesture.
  const [ws, setWs] = useState<WorkspaceInfo | null>(null);
  useEffect(() => {
    if (!port) {
      setWs(null);
      return;
    }
    port.workspaces().then(setWs).catch(() => setWs(null));
  }, [port, root]);

  return (
    <div className="chrome" onDoubleClick={zoomOnTitleBar}>
      <span className="brand" role="img" aria-label="Reasonix" />

      <div className="crumb">
        <span className="crumb-proj" title={root}>
          {project}
        </span>
        <span className="isolab" hidden={!ws?.isolated}>
          {t("隔离")}
        </span>
        {/* Same slot as the isolated tag: both answer "this workspace is not an
            ordinary folder on this machine", so they read as one language. */}
        <span className="isolab hostlab" hidden={!host}>
          {host}
        </span>
        <span className="crumbsep">/</span>
        <b title={status?.sessionPath}>{sessionName(title, status?.sessionPath)}</b>
      </div>

      <span className="badge" hidden={steer === 0}>
        {t("插话待送达")} <b>{steer}</b>
      </span>

      <div className="r">
        <span className="runpill" data-run={run}>
          <i />
          {t(RUN_LB[run] ?? "待命")}
        </span>
        {/* Identity sits where every app puts it, but signed out it stays an
            outline in the icon cluster: an entry point, not a pitch. Reasonix
            runs fine without an account and must not imply otherwise. */}
        <button
          className="thbtn acct-btn"
          data-action="chrome.account"
          data-on={account?.signedIn ? "" : undefined}
          onClick={() => onSettings("account")}
          aria-label={account?.signedIn ? t("账号：{name}", { name: account.user?.label ?? "" }) : t("登录")}
          title={account?.signedIn ? `${account.user?.label ?? ""} <${account.user?.email ?? ""}>` : t("登录（社区与崩溃跟进，不影响使用）")}
        >
          {account?.signedIn && account.user?.label ? (
            <span className="ini" aria-hidden="true">
              {[...account.user.label][0]?.toUpperCase()}
            </span>
          ) : (
            <svg viewBox="0 0 16 16" aria-hidden="true">
              <circle cx="8" cy="5.6" r="2.6" />
              <path d="M3.2 13.4a4.8 4.8 0 0 1 9.6 0" />
            </svg>
          )}
        </button>
        {/* The icon cluster's weight class, not the preset control's: what is
            left up here is the window's, and the turn's policy is the shelf's. */}
        {/* 进出都是这一枚。专注不是模式切换到别处去了，是同一个会话把外围收起来
            —— 所以按钮留在原地，只换它说的话。 */}
        <button
          className="thbtn"
          data-action="chrome.focus"
          onClick={onFocus}
          aria-pressed={focus}
          aria-label={focus ? t("退出专注") : t("专注")}
          title={focus ? t("退出专注") : t("专注：收起两侧栏")}
        >
          <svg viewBox="0 0 16 16" width="15" height="15" fill="none" stroke="currentColor" strokeWidth="1.3">
            {focus ? (
              <>
                <path d="M6.4 2.2v4.2H2.2M9.6 2.2v4.2h4.2M6.4 13.8V9.6H2.2M9.6 13.8V9.6h4.2" />
              </>
            ) : (
              <>
                <path d="M2.2 6.4V2.2h4.2M13.8 6.4V2.2H9.6M2.2 9.6v4.2h4.2M13.8 9.6v4.2H9.6" />
              </>
            )}
          </svg>
        </button>
        <button className="thbtn" data-action="chrome.settings" onClick={() => onSettings()} aria-label={t("设置")} title={`${t("设置")}　${chord(",")}`}>
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M8 5.9a2.1 2.1 0 1 0 0 4.2 2.1 2.1 0 0 0 0-4.2" />
            <path d="M12.7 9.8a1 1 0 0 0 .2 1.1l.04.04a1.2 1.2 0 1 1-1.7 1.7l-.04-.04a1 1 0 0 0-1.1-.2 1 1 0 0 0-.6.9v.11a1.2 1.2 0 1 1-2.4 0v-.06a1 1 0 0 0-.65-.9 1 1 0 0 0-1.1.2l-.04.04a1.2 1.2 0 1 1-1.7-1.7l.04-.04a1 1 0 0 0 .2-1.1 1 1 0 0 0-.9-.6h-.11a1.2 1.2 0 0 1 0-2.4h.06a1 1 0 0 0 .9-.65 1 1 0 0 0-.2-1.1l-.04-.04a1.2 1.2 0 1 1 1.7-1.7l.04.04a1 1 0 0 0 1.1.2h.05a1 1 0 0 0 .6-.9v-.11a1.2 1.2 0 1 1 2.4 0v.06a1 1 0 0 0 .6.9 1 1 0 0 0 1.1-.2l.04-.04a1.2 1.2 0 1 1 1.7 1.7l-.04.04a1 1 0 0 0-.2 1.1v.05a1 1 0 0 0 .9.6h.11a1.2 1.2 0 0 1 0 2.4h-.06a1 1 0 0 0-.9.6" />
          </svg>
        </button>
        <WindowControls />
      </div>
    </div>
  );
}
