// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, waitFor } from "@testing-library/react";
import "./testkit";
import { Composer } from "./Composer";
import { MockPort } from "../port/mock";
import type { AgentPort, ApprovalMode, Preset, SessionStatus } from "../port/port";

afterEach(cleanup);

const status = (over: Partial<SessionStatus> = {}) =>
  ({ preset: "balanced" as Preset, effort: "auto", toolApprovalMode: "ask" as ApprovalMode,
     plan: false, modelRef: "deepseek/deepseek-v4-pro", ...over } as SessionStatus);

function draw(st: SessionStatus | null = status()) {
  const r = render(
    <Composer port={new MockPort() as unknown as AgentPort} status={st} running={false}
      focus={0} onSubmit={vi.fn()} onChanged={vi.fn()} onError={vi.fn()} />,
  );
  // The separators are the shelf's own, so counting them says whether anything
  // was left behind by a control that receded.
  const seps = () => r.container.querySelectorAll(".turntools > .sep").length;
  return { ...r, seps };
}

describe("what the composer shows when nothing is unusual", () => {
  // The defect: four controls on every turn, three of them reciting a value the
  // session already had before anyone touched it.
  it("shows no default-state noise", () => {
    const { container } = draw();
    expect(container.querySelector(".policy")).toBeNull();
    expect(container.textContent).not.toMatch(/均衡|询问/);
  });

  // A status may recede to its baseline. The only way into a mode may not: the
  // toggle is where plan mode is discovered, and Shift+Tab is a shortcut for
  // people who already know. It stays, saying it is off.
  it("keeps the plan toggle discoverable while plan is off", () => {
    const { container } = draw();
    const tog = container.querySelector('[data-action="plan.mode"]');
    expect(tog).toBeTruthy();
    expect(tog?.getAttribute("aria-pressed")).toBe("false");
    expect(tog?.textContent).toMatch(/计划/);
  });

  // The model has no baseline to fall back to, so every value of it is a real
  // choice and it stays.
  it("keeps the model, which has no default to recede to", async () => {
    const { container } = draw();
    await waitFor(() => expect(container.querySelector('[data-action="model.select"]')).toBeTruthy());
  });

  // Vacuously true if the selector is wrong, so the deviating case has to show
  // the same query counting one more. The policy control brings its own
  // separator and takes it away again; the plan toggle's stays either way.
  it("takes its separator with it when it recedes", () => {
    expect(draw().seps()).toBe(1);
    expect(draw(status({ preset: "delivery" as Preset })).seps()).toBe(2);
  });
});

describe("every deviation is visible", () => {
  it.each([

    ["preset", status({ preset: "delivery" as Preset }), /交付/],
    ["effort", status({ effort: "high" }), /High/],
    ["approval", status({ toolApprovalMode: "auto" as ApprovalMode }), /自动批准/],
    ["a stricter approval", status({ toolApprovalMode: "dontAsk" as ApprovalMode }), /不询问/],
  ])("surfaces %s", (_what, st, want) => {
    const { container } = draw(st);
    expect(container.textContent).toMatch(want);
  });

  // Plan is a control, not a readout, so what deviates is its state.
  it("shows plan mode as engaged once it is on", () => {
    const { container } = draw(status({ plan: true }));
    expect(container.querySelector('[data-action="plan.mode"]')?.getAttribute("aria-pressed")).toBe("true");
  });

  it("surfaces several at once without losing any", () => {
    const { container } = draw(
      status({ preset: "delivery" as Preset, effort: "high", toolApprovalMode: "yolo" as ApprovalMode, plan: true }),
    );
    for (const want of [/交付/, /High/, /全部放行/, /计划/]) expect(container.textContent).toMatch(want);
  });
});

// Sabotage. Collapsing the shelf is allowed to hide a default. It is never
// allowed to hide the one state a person can forget they are in and lose a
// workspace to.
describe("the quiet may not swallow a fully-permitted session", () => {
  it("shows it, marks it, and does so even with everything else at baseline", () => {
    const { container } = draw(status({ toolApprovalMode: "yolo" as ApprovalMode }));
    expect(container.querySelector(".polrisk")?.textContent).toContain("全部放行");
    expect(container.querySelector(".polwarn")?.textContent).toBe("⚠");
  });

  it("shows it while a plan is running too", () => {
    const { container } = draw(status({ toolApprovalMode: "yolo" as ApprovalMode, plan: true }));
    expect(container.textContent).toMatch(/全部放行/);
    expect(container.textContent).toMatch(/计划/);
  });
});
