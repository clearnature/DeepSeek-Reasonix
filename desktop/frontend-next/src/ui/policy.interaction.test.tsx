// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import "./testkit";
import { Policy } from "./Policy";
import { HttpError } from "../port/port";
import type { AgentPort, SessionStatus } from "../port/port";

afterEach(cleanup);

function deferred<T>() {
  let settle!: (v: T) => void;
  let fail!: (e: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    settle = res;
    fail = rej;
  });
  // A promise nobody rejects still counts as unhandled if the component drops
  // it, so the test has to be the one holding it.
  promise.catch(() => {});
  return { promise, settle, fail };
}

const STATUS = { preset: "balanced", effort: "medium", toolApprovalMode: "ask" } as SessionStatus;

function draw(calls: Partial<AgentPort> = {}, status: SessionStatus = STATUS) {
  const onChanged = vi.fn();
  const port = {
    setPreset: vi.fn(async () => {}),
    setEffort: vi.fn(async () => {}),
    setApprovalMode: vi.fn(async () => {}),
    ...calls,
  } as unknown as AgentPort;
  const view = render(
    <Policy port={port} status={status} efforts={["auto", "medium", "high"]} onChanged={onChanged} />,
  );
  const open = () => userEvent.click(screen.getByRole("button", { expanded: false }));
  const at = (field: string, name: string) =>
    within(screen.getByRole("group", { name: field })).getByRole("button", { name });
  const field = (name: string) => screen.getByRole("group", { name });
  return { port, onChanged, view, open, at, field, status };
}

describe("the turn policy the composer owns", () => {
  it("says what is unusual about this turn before it is opened", () => {
    draw();
    // What deviates, not the word "settings" and not the whole reading: the
    // shelf has to answer "how will the next turn run" without being opened,
    // and a shelf that recites the default forever has no attention left for
    // the turn where something is actually different. medium is a rung nobody
    // asked for by default, so it shows; ask is the baseline and does not.
    expect(screen.getByRole("button", { expanded: false }).textContent).toBe("均衡 · Medium");
  });

  it("asks the kernel once, and does not move the selection on its own", async () => {
    const call = deferred<void>();
    const setPreset = vi.fn(() => call.promise);
    const { at, open, port } = draw({ setPreset });

    await open();
    await userEvent.click(at("执行方式", "交付"));
    expect(setPreset).toHaveBeenCalledTimes(1);
    expect(setPreset).toHaveBeenCalledWith("delivery");
    // Still the kernel's answer, not the click's: nothing has come back yet.
    expect(at("执行方式", "均衡").getAttribute("aria-pressed")).toBe("true");
    expect(at("执行方式", "交付").getAttribute("aria-pressed")).toBe("false");
    // One field, one effect. Drawing three fields in one panel must not turn
    // three kernel operations into one call.
    expect(port.setEffort).not.toHaveBeenCalled();
    expect(port.setApprovalMode).not.toHaveBeenCalled();
  });

  it("shows the call is out and refuses a second one on that field", async () => {
    const call = deferred<void>();
    const setPreset = vi.fn(() => call.promise);
    const { at, open } = draw({ setPreset });

    await open();
    await userEvent.click(at("执行方式", "交付"));
    expect(at("执行方式", "交付").hasAttribute("data-asking")).toBe(true);
    expect((at("执行方式", "交付") as HTMLButtonElement).disabled).toBe(true);
    expect((at("执行方式", "均衡") as HTMLButtonElement).disabled).toBe(true);

    await userEvent.click(at("执行方式", "均衡"));
    expect(setPreset).toHaveBeenCalledTimes(1);
  });

  // The reason the three live in one panel and not one control: a panel that
  // goes grey while one value is in flight says "nothing here can be changed",
  // and that was never true.
  it("leaves the other fields alone while one is waiting", async () => {
    const call = deferred<void>();
    const setPreset = vi.fn(() => call.promise);
    const { at, open, port } = draw({ setPreset });

    await open();
    await userEvent.click(at("执行方式", "交付"));
    expect((at("思考强度", "high") as HTMLButtonElement).disabled).toBe(false);
    expect((at("工具权限", "自动") as HTMLButtonElement).disabled).toBe(false);

    await userEvent.click(at("思考强度", "high"));
    await userEvent.click(at("工具权限", "自动"));
    expect(port.setEffort).toHaveBeenCalledWith("high");
    expect(port.setApprovalMode).toHaveBeenCalledWith("auto");
    // And the one still out has not been asked a second time by either.
    expect(setPreset).toHaveBeenCalledTimes(1);
  });

  it("hands the selection back to the kernel's own answer when it lands", async () => {
    const call = deferred<void>();
    const { at, open, onChanged, view, port } = draw({ setPreset: () => call.promise });

    await open();
    await userEvent.click(at("执行方式", "交付"));
    call.settle();
    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));
    // onChanged is what makes the pane re-read /status; the selection follows
    // that read, which is why it is the prop and not local state.
    view.rerender(
      <Policy
        port={port}
        status={{ ...STATUS, preset: "delivery" } as SessionStatus}
        efforts={["auto", "medium", "high"]}
        onChanged={onChanged}
      />,
    );
    expect(at("执行方式", "交付").getAttribute("aria-pressed")).toBe("true");
    expect((at("执行方式", "交付") as HTMLButtonElement).disabled).toBe(false);
  });

  it("keeps the old value and says why, next to the field that was refused", async () => {
    const call = deferred<void>();
    const { at, field, open, onChanged } = draw({ setEffort: () => call.promise });

    await open();
    await userEvent.click(at("思考强度", "high"));
    call.fail(new HttpError(409, "a turn is running", { code: "busy.switch_model" }));

    await waitFor(() => expect(field("思考强度").parentElement?.querySelector(".segbad")).toBeTruthy());
    const said = field("思考强度").parentElement?.querySelector(".segbad");
    expect(said?.textContent).toContain("任务正在运行");
    // The refusal belongs to the field it was about, and to no other.
    expect(field("执行方式").parentElement?.querySelector(".segbad")).toBeNull();
    expect(at("思考强度", "medium").getAttribute("aria-pressed")).toBe("true");
    // Usable again, and nothing told the pane to go and re-read a change that
    // did not happen.
    expect((at("思考强度", "high") as HTMLButtonElement).disabled).toBe(false);
    expect(onChanged).not.toHaveBeenCalled();
  });

  // The kernel omits the field exactly when it would refuse every rung but one.
  it("draws no ladder for an endpoint that names no levels", async () => {
    const onChanged = vi.fn();
    render(
      <Policy
        port={{ setPreset: async () => {} } as unknown as AgentPort}
        status={STATUS}
        efforts={[]}
        onChanged={onChanged}
      />,
    );
    await userEvent.click(screen.getByRole("button", { expanded: false }));
    expect(screen.queryByRole("group", { name: "思考强度" })).toBeNull();
    expect(screen.getByRole("group", { name: "工具权限" })).toBeTruthy();
  });
});
