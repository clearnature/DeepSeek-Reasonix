import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { Trajectory } from "./Trajectory";
import { initialTraj, reduceTraj } from "../state/trajectory";
import type { TrajRow } from "../state/trajectory";
import type { TrajectoryAvailability } from "../port/wire";

const row: TrajRow = { seq: 1, at: 0, kind: "event", payload: [{ t: "turn_started" }], subs: [] };

// Rendered in the source language: the English side is the i18n catalogue's
// own gate, not this one's.
const draw = (availability?: TrajectoryAvailability) =>
  renderToStaticMarkup(<Trajectory rows={[row]} availability={availability} onSave={async () => null} />);

// The table draws the same rows whatever they cover, so the sentence under it
// is the only place a reader learns that the last row may not be the end. It
// used to say one thing always — and the thing it said was wrong: these frames
// are replayed from disk, not lost when the window closes.
describe("what the pane says its rows cover", () => {
  it("marks a prefix as a prefix", () => {
    expect(draw("truncated")).toContain('data-coverage="truncated"');
    expect(draw("truncated")).toContain("记录到容量上限就停了");
  });

  it("does not present an unrecorded run's live events as a record", () => {
    const html = draw("not_recorded");
    expect(html).toContain('data-coverage="not_recorded"');
    expect(html).toContain("这次运行没有记录轨迹");
  });

  it("says nothing is missing only when the host said so", () => {
    expect(draw("complete")).toContain('data-coverage="complete"');
    expect(draw(undefined)).toContain('data-coverage="unread"');
    expect(draw(undefined)).not.toContain('data-coverage="complete"');
  });
});

// Coverage is state the pane holds, not a prop threaded past the reducer: a
// session switch clears it with everything else, and until the next read lands
// the table must not go on claiming the previous session's extent.
describe("coverage in the trajectory state", () => {
  it("starts unknown and is cleared with the session", () => {
    expect(initialTraj.availability).toBeUndefined();
    const covered = reduceTraj(initialTraj, { kind: "__coverage", availability: "truncated" });
    expect(covered.availability).toBe("truncated");
    expect(reduceTraj(covered, { kind: "__clear" }).availability).toBeUndefined();
  });
});
