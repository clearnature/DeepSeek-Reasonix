// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { Compaction } from "./Compaction";
import { MockPort } from "../port/mock";
import type { AgentPort, CompactionSettings } from "../port/port";

afterEach(cleanup);

const port = (over: Partial<CompactionSettings> = {}) => {
  const p = new MockPort() as unknown as AgentPort;
  const base = {
    soft_limit_tokens: 0, default_soft_limit: 160000, ratio: 0.85,
    context_window: 1_000_000, trigger: 160000, path: "~/.reasonix/config.toml",
    ...over,
  };
  p.compaction = async () => ({ ...base });
  return p;
};

const open = async (p: AgentPort) => {
  render(<Compaction port={p} onChanged={() => {}} />);
  await screen.findByText("下次整理");
};

describe("what the default view of context maintenance says", () => {
  // The setting used to be one number in a text box. A threshold is only
  // meaningful beside the session's own usage, which is what turns it from a
  // constant into a distance.
  it("leads with the distance, not with the configured number", async () => {
    await open(port());
    expect(screen.getByText("当前")).toBeTruthy();
    expect(screen.getByText("160k")).toBeTruthy();
    expect(screen.getByText("1000k")).toBeTruthy();
  });

  // The whole reason a 1M session folds at 160k, said once, on the screen where
  // the number lives.
  it("names which of the two bounds is the one that fires", async () => {
    await open(port());
    expect(screen.getByText(/经济维护阈值会先到/)).toBeTruthy();
  });

  it("names capacity instead when the window is the lower bound", async () => {
    await open(port({ context_window: 128000, trigger: 108800 }));
    expect(screen.getByText(/模型窗口的容量保护会先到/)).toBeTruthy();
  });

  // A number box is the kernel's configuration language. The default view is
  // the product's, and it has no free-text threshold in it at all.
  it("puts no number field in front of someone who did not ask for one", async () => {
    const { container } = render(<Compaction port={port()} onChanged={() => {}} />);
    await screen.findByText("下次整理");
    expect(container.querySelector("input")).toBeNull();
  });
});

describe("how the three answers are given", () => {
  // "填负数关闭" was the defect: the intent is a choice among three, and a
  // minus sign is not one of the three.
  it("offers the three intents rather than asking for a signed integer", async () => {
    const p = port();
    await open(p);
    await userEvent.click(screen.getByRole("button", { name: "高级设置" }));
    expect(screen.getByRole("button", { name: "使用默认值" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "自定义" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "只按模型容量保护" })).toBeTruthy();
  });

  it("stores the default as the default, not as today's copy of it", async () => {
    const p = port({ soft_limit_tokens: 90000 });
    const save = vi.fn(async (n: number) => ({ ...(await p.compaction()), soft_limit_tokens: n, trigger: 160000 }));
    p.saveCompaction = save;
    await open(p);
    await userEvent.click(screen.getByRole("button", { name: "使用默认值" }));
    expect(save).toHaveBeenCalledWith(0);
  });

  // The negative is still what the kernel stores; it is no longer what a person
  // has to type, or decode.
  it("keeps the negative as storage and never as vocabulary", async () => {
    const p = port();
    const save = vi.fn(async (n: number) => ({ ...(await p.compaction()), soft_limit_tokens: n, trigger: 850000 }));
    p.saveCompaction = save;
    await open(p);
    await userEvent.click(screen.getByRole("button", { name: "高级设置" }));
    await userEvent.click(screen.getByRole("button", { name: "只按模型容量保护" }));
    expect(save).toHaveBeenCalledWith(-1);
    expect(screen.queryByText("-1")).toBeNull();
  });

  // Choosing "custom" must open the field. Deriving the mode from the stored
  // value alone made this click show nothing, so a value could never be typed.
  it("opens the field on the choice, before there is a value to derive it from", async () => {
    const p = port();
    await open(p);
    await userEvent.click(screen.getByRole("button", { name: "高级设置" }));
    await userEvent.click(screen.getByRole("button", { name: "自定义" }));
    expect(screen.getByRole("textbox")).toBeTruthy();
  });

  it("saves a custom threshold once, on the keystroke aimed at it", async () => {
    const p = port();
    const save = vi.fn(async (n: number) => ({ ...(await p.compaction()), soft_limit_tokens: n, trigger: n }));
    p.saveCompaction = save;
    await open(p);
    await userEvent.click(screen.getByRole("button", { name: "高级设置" }));
    await userEvent.click(screen.getByRole("button", { name: "自定义" }));
    const box = screen.getByRole("textbox");
    await userEvent.clear(box);
    await userEvent.type(box, "90000{Enter}");
    await waitFor(() => expect(save).toHaveBeenCalledWith(90000));
    expect(save).toHaveBeenCalledTimes(1);
  });
});

describe("what turning the economic bound off is allowed to claim", () => {
  // The defect behind the rename: "关闭" read as "this will never compact
  // again", while the window share went on firing at 85%. The word for that
  // state now says what it protects, and capacity is stated beside it.
  it("says capacity still applies rather than that maintenance is off", async () => {
    await open(port({ soft_limit_tokens: -1, trigger: 850000 }));
    expect(screen.getByText("容量保护")).toBeTruthy();
    const guard = screen.getByText(/始终生效/).closest(".lrow");
    // With the economic bound retired the two figures coincide, so the capacity
    // row is read on its own rather than by searching the whole panel.
    expect(guard?.querySelector(".sc")?.textContent).toBe("850k");
  });

  // A session running under something other than the default must not have that
  // fact hidden behind a disclosure.
  it("opens the advanced block for a session not on the default", async () => {
    await open(port({ soft_limit_tokens: -1, trigger: 850000 }));
    expect(screen.getByRole("button", { name: "只按模型容量保护" })).toBeTruthy();
  });

  // Nothing folds without a declared window, and that outranks either bound.
  it("says an undeclared window is what actually turns maintenance off", async () => {
    render(<Compaction port={port({ context_window: 0, trigger: 0 })} onChanged={() => {}} />);
    expect(await screen.findByText(/未声明上下文窗口/)).toBeTruthy();
    expect(screen.queryByText("下次整理")).toBeNull();
  });
});

describe("what the block says while it is closed", () => {
  // Folded, the panel used to look identical whether the session ran on the
  // default or on something someone had changed months ago.
  it("names the mode in force without being expanded", async () => {
    render(<Compaction port={port({ soft_limit_tokens: 90000, trigger: 90000 })} onChanged={() => {}} />);
    await screen.findByText("下次整理");
    expect(screen.getByText("自定义 90k")).toBeTruthy();
  });

  it("says so when the session is on the default", async () => {
    await open(port());
    expect(screen.getByText("使用默认值")).toBeTruthy();
  });
});
