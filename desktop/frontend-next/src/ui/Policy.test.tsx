// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { Policy } from "./Policy";
import { MockPort } from "../port/mock";
import type { AgentPort, ApprovalMode, Preset, SessionStatus } from "../port/port";

afterEach(cleanup);

const status = (preset: Preset, effort: string | undefined, mode: ApprovalMode) =>
  ({ preset, effort, toolApprovalMode: mode } as SessionStatus);

function draw(over: {
  status?: SessionStatus | null;
  efforts?: string[];
  port?: Partial<AgentPort>;
} = {}) {
  const port = { ...(new MockPort() as unknown as AgentPort), ...over.port };
  const onChanged = vi.fn();
  const r = render(
    <Policy
      port={port}
      status={over.status === undefined ? status("balanced", "auto", "ask") : over.status}
      efforts={over.efforts ?? ["auto", "low", "medium", "high"]}
      onChanged={onChanged}
    />,
  );
  // The trigger by its own element: getByRole would move to the menu's
  // buttons the moment it is open, which is exactly when this is asked.
  return { ...r, port, onChanged, shelf: () => r.container.querySelector(".policy .vl")?.textContent };
}

describe("what the closed control says", () => {
  it("is one word for a session nobody has configured", () => {
    const { shelf } = draw();
    expect(shelf()).toBe("均衡");
  });

  it("carries the whole reading on hover, so nothing is actually lost", () => {
    const { container } = draw();
    expect(container.querySelector("button")?.getAttribute("title")).toBe("均衡 · auto · 询问");
  });

  it("names the control, not a posture, before the kernel has answered", () => {
    const { shelf, container } = draw({ status: null });
    expect(shelf()).toBe("策略");
    // And the hover text must not quietly supply the posture either.
    expect(container.querySelector("button")?.getAttribute("title")).not.toMatch(/均衡|询问/);
  });

  it("tones a fully-permitted session and marks it, without repainting the control", () => {
    const { container } = draw({ status: status("balanced", "auto", "yolo") });
    const risk = container.querySelector(".vl .polrisk");
    expect(risk?.textContent).toContain("全放行");
    expect(container.querySelector(".vl .polwarn")?.textContent).toBe("⚠");
    // The button itself stays quiet; only the segment that must be remembered
    // is toned, or every ordinary turn is watched too.
    expect(container.querySelector("button")?.className).not.toMatch(/risk|danger/);
  });

  it("leaves a stricter-than-default posture visible but untoned", () => {
    const { container } = draw({ status: status("balanced", "auto", "dontAsk") });
    expect(container.querySelector(".vl")?.textContent).toBe("均衡 · 不打扰");
    expect(container.querySelector(".vl .polrisk")).toBeNull();
  });
});

describe("the shelf reads committed fact, never the answer being waited on", () => {
  // Sabotage C. A summary that moves on click says the session is running under
  // something the kernel has not accepted — and may refuse.
  it("does not adopt a rung the kernel has not answered for", async () => {
    let release!: () => void;
    const setEffort = vi.fn(() => new Promise<void>((ok) => { release = ok; }));
    const { shelf } = draw({ port: { setEffort } });
    await userEvent.click(screen.getByRole("button", { name: "均衡" }));
    await userEvent.click(screen.getByRole("button", { name: "high" }));
    expect(setEffort).toHaveBeenCalledWith("high");
    // Asked for, not yet true.
    expect(shelf()).toBe("均衡");
    release();
    await waitFor(() => expect(shelf()).toBe("均衡"));
  });

  it("moves only once the kernel's own reading says so", async () => {
    const { rerender, shelf } = draw();
    expect(shelf()).toBe("均衡");
    rerender(
      <Policy port={new MockPort() as unknown as AgentPort} status={status("balanced", "high", "ask")}
        efforts={["auto", "high"]} onChanged={() => {}} />,
    );
    expect(shelf()).toBe("均衡 · High");
  });

  it("leaves the summary alone when the kernel refuses", async () => {
    const setApprovalMode = vi.fn(() => Promise.reject(new Error("not while a turn is running")));
    const { shelf, container } = draw({ port: { setApprovalMode } });
    await userEvent.click(screen.getByRole("button", { name: "均衡" }));
    await userEvent.click(screen.getByRole("button", { name: "全放行" }));
    await waitFor(() => expect(container.querySelector(".segbad")).toBeTruthy());
    expect(shelf()).toBe("均衡");
  });
});

describe("a model that publishes no rungs", () => {
  // That the open menu draws no ladder is policy.interaction's; what is this
  // file's is the other half — a shelf claiming a control the menu will not
  // draw, from a rung left over in a status nobody cleared.
  it("claims no rung on the shelf either", () => {
    const { shelf } = draw({ status: status("balanced", "high", "ask"), efforts: [] });
    expect(shelf()).toBe("均衡");
  });
});
