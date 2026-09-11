// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { Settings } from "./Settings";
import type { AgentPort, SessionStatus } from "../port/port";
import { MockPort } from "../port/mock";
import { MockHub } from "../port/mock_hub";

afterEach(cleanup);

function draw() {
  const onClose = vi.fn();
  render(
    <Settings
      hub={new MockHub() as never}
      port={new MockPort() as unknown as AgentPort}
      status={{ preset: "balanced", toolApprovalMode: "ask" } as SessionStatus}
      theme="light" onTheme={() => {}} contrast="" onContrast={() => {}} weight="" onWeight={() => {}}
      look={{} as never} onLook={() => {}} reloadThemes={() => {}}
      onClose={onClose} onChanged={() => {}} onError={() => {}}
      account={null} accountUnread="" reloadAccount={() => {}}
    />,
  );
  return { onClose };
}

describe("what Escape takes back", () => {
  // The sheet listens for Escape on the window. An inline form that does not
  // claim the key first goes down with the whole sheet — and takes what was
  // typed into it, which is the one thing pressing Escape there cannot mean.
  it("closes the form it was pressed in, not the sheet around it", async () => {
    const { onClose } = draw();
    await userEvent.click(screen.getByRole("tab", { name: /扩展/ }));
    const add = await screen.findByRole("button", { name: "添加" });
    await userEvent.click(add);
    await screen.findByRole("textbox", { name: /粘贴|安装/ }).catch(() => null);

    await userEvent.keyboard("{Escape}");
    expect(onClose).not.toHaveBeenCalled();
    await waitFor(() => expect(screen.queryByRole("button", { name: "添加" })).not.toBeNull());

    // With nothing open above it, the same key does close the sheet.
    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalled();
  });
});

describe("the section list as one tab stop", () => {
  // role=tablist means arrows move inside the list and Tab steps over it.
  // Twelve stops made Tab the wrong way to get anywhere on this screen.
  it("keeps every unselected section out of the tab order", async () => {
    draw();
    const tabs = screen.getAllByRole("tab");
    expect(tabs.length).toBeGreaterThan(6);
    const stops = tabs.filter((t) => t.tabIndex === 0);
    expect(stops).toHaveLength(1);
    expect(stops[0].getAttribute("aria-selected")).toBe("true");
  });
});

describe("where focus is when the sheet goes", () => {
  // Closing used to leave focus on <body>: a reader on the keyboard came out
  // of settings with no position and had to tab in from the top of the window.
  it("puts focus back on whatever opened it", async () => {
    const opener = document.createElement("button");
    document.body.append(opener);
    opener.focus();
    const { unmount } = render(
      <Settings
        hub={new MockHub() as never}
        port={new MockPort() as unknown as AgentPort}
        status={{ preset: "balanced", toolApprovalMode: "ask" } as SessionStatus}
        theme="light" onTheme={() => {}} contrast="" onContrast={() => {}} weight="" onWeight={() => {}}
        look={{} as never} onLook={() => {}} reloadThemes={() => {}}
        onClose={() => {}} onChanged={() => {}} onError={() => {}}
        account={null} accountUnread="" reloadAccount={() => {}}
      />,
    );
    expect(document.activeElement).not.toBe(opener);
    unmount();
    expect(document.activeElement).toBe(opener);
    opener.remove();
  });
});
