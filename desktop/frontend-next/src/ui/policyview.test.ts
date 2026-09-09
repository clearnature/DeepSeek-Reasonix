import { describe, expect, it } from "vitest";
import { policySummary, type PolicySummary } from "./policyview";
import type { ApprovalMode, Preset, SessionStatus } from "../port/port";

const status = (preset: Preset, effort: string | undefined, mode: ApprovalMode): SessionStatus =>
  ({ preset, effort, toolApprovalMode: mode } as SessionStatus);

// What the shelf actually reads, assembled the way Policy.tsx assembles it.
const shelf = (s: PolicySummary) =>
  s.ready ? [s.preset, s.effort, s.approval].filter(Boolean).join(" · ") : s.label;

const at = (preset: Preset, effort: string | undefined, mode: ApprovalMode, hasEffort = true) =>
  policySummary(status(preset, effort, mode), hasEffort);

describe("the posture a session runs as by default", () => {
  // The defect: three kernel facts recited forever, on every turn where nothing
  // was unusual, next to a model picker and everything else the composer holds.
  it("spends one word on a session nobody has configured", () => {
    expect(shelf(at("balanced", "auto", "ask"))).toBe("均衡");
  });

  // Sabotage A guards this from the other side: putting the baseline values
  // back must not read as a summary anyone would accept.
  it("never recites a value the session already had", () => {
    const s = at("balanced", "auto", "ask");
    expect(s.ready && s.effort).toBeUndefined();
    expect(s.ready && s.approval).toBeUndefined();
  });

  // A missing effort is the kernel not naming one, which is what auto means.
  it("treats an absent rung as the baseline rather than as unknown", () => {
    expect(shelf(at("balanced", undefined, "ask"))).toBe("均衡");
  });

  it("still names the preset, which is never baseline-hidden", () => {
    expect(shelf(at("delivery", "auto", "ask"))).toBe("交付");
  });
});

describe("what a deviation is allowed to cost the shelf", () => {
  it.each([
    ["balanced", "high", "ask", "均衡 · High"],
    ["delivery", "high", "ask", "交付 · High"],
    ["balanced", "auto", "auto", "均衡 · 自动批准"],
    ["balanced", "auto", "dontAsk", "均衡 · 不打扰"],
    ["balanced", "auto", "yolo", "均衡 · 全放行"],
    ["delivery", "high", "yolo", "交付 · High · 全放行"],
  ] as [Preset, string, ApprovalMode, string][])("reads %s/%s/%s as %s", (p, e, m, want) => {
    expect(shelf(at(p, e, m))).toBe(want);
  });

  // The menu can say 自动 because it sits under a 工具权限 heading; the shelf
  // cannot, with a reasoning rung standing right next to it.
  it("says what is automatic, not merely that something is", () => {
    const s = at("balanced", "auto", "auto");
    expect(s.ready && s.approval).toBe("自动批准");
    expect(s.ready && s.approval).not.toBe("自动");
  });

  // A rung table would need a row per rung an endpoint might publish.
  it("cases the kernel's own spelling instead of translating it", () => {
    const s = at("balanced", "xhigh", "ask");
    expect(s.ready && s.effort).toBe("Xhigh");
  });
});

describe("the two postures that must never be quiet", () => {
  // Sabotage B: dropping yolo from the projection. It is the state a person can
  // forget they are in, and forgetting it is what costs a workspace.
  it("keeps a fully-permitted session visible and toned", () => {
    const s = at("balanced", "auto", "yolo");
    expect(s.ready && s.approval).toBe("全放行");
    expect(s.ready && s.danger).toBe(true);
  });

  // dontAsk is stricter than ask, not looser: it refuses what it would have
  // asked about. Hidden, it reads as "Reasonix suddenly changes nothing".
  it("keeps a session that refuses rather than asks visible, and not as danger", () => {
    const s = at("balanced", "auto", "dontAsk");
    expect(s.ready && s.approval).toBe("不打扰");
    expect(s.ready && s.danger).toBeUndefined();
  });

  it("tones nothing else as danger", () => {
    for (const m of ["ask", "auto", "dontAsk"] as ApprovalMode[]) {
      const s = at("balanced", "auto", m);
      expect(s.ready && s.danger).toBeUndefined();
    }
  });
});

describe("what the shelf may claim before the kernel has answered", () => {
  // The context gauge's defect, in the one place where the fact is what the
  // agent may do to a workspace unasked.
  it("names the control rather than a posture it has not been told", () => {
    const s = policySummary(null, true);
    expect(s.ready).toBe(false);
    expect(shelf(s)).toBe("策略");
  });

  it("carries no posture at all to be read by mistake", () => {
    const s = policySummary(null, true);
    expect("preset" in s).toBe(false);
    expect("danger" in s).toBe(false);
  });
});

describe("a rung the endpoint does not publish", () => {
  // The open menu draws no ladder when the model names no levels; a summary
  // claiming one would advertise a control that is not there.
  it("is not named on the shelf even when a stale status carries one", () => {
    expect(shelf(at("balanced", "high", "ask", false))).toBe("均衡");
  });

  it("is left out of the hover reading too", () => {
    const s = at("balanced", "high", "ask", false);
    expect(s.ready && s.reading).toBe("均衡 · 询问");
  });
});

describe("what the projection hides is one pointer away", () => {
  // Nothing is deleted. The baseline stops occupying the shelf and stays on the
  // control, which is the difference between quiet and lossy.
  it("keeps the whole reading for hover", () => {
    const s = at("balanced", "auto", "ask");
    expect(s.ready && s.reading).toBe("均衡 · auto · 询问");
  });
});
