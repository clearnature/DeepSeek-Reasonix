// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render } from "@testing-library/react";
import "./testkit";
import { Transcript } from "./Transcript";
import type { Item } from "../state/session";
// Not through import.meta.glob: vite's CSS handling intercepts ?raw there and
// hands back an empty string, which every rule below would then pass.
import css from "../styles/app.css?raw";

afterEach(cleanup);

const user = (id: string, text: string): Item => ({ t: "user", id, text });

function draw(items: Item[], entering: string[], hidden = false) {
  const onEntered = vi.fn();
  const noop = async () => undefined as never;
  const view = render(
    <Transcript
      items={items} entering={entering} onEntered={onEntered} revision={1}
      waiting={{}} scroll={{ current: null }} hidden={hidden} onPinned={() => {}}
      jump={0} focus={null}
      onApprove={noop} onPlan={noop} onAnswer={noop} onSuggest={() => {}} onForget={noop}
      onCancelQueued={() => {}} onExtInvoke={() => {}} onExtSubmit={noop}
      checkpoints={new Map()} onPrepareRewind={noop} onCommitRewind={noop} onUndoRewind={noop}
      onPrepareFileRevert={noop} onCommitFileRevert={noop}
      needsProject={false} onOpenProject={() => {}} onKeepHere={() => {}}
    />,
  );
  const entered = () => [...document.querySelectorAll(".enterbox[data-enter]")].length;
  return { view, onEntered, entered };
}

describe("a card's one entrance", () => {
  it("is marked on the card that has just arrived", () => {
    const { entered } = draw([user("i1", "hi")], ["i1"]);
    expect(entered()).toBe(1);
  });

  it("is spent by the render that drew it, not by the animation ending", () => {
    const { onEntered } = draw([user("i1", "hi")], ["i1"]);
    // No animation has run — jsdom has none, and reduced motion runs none
    // either. The debt is settled all the same.
    expect(onEntered).toHaveBeenCalledWith(["i1"]);
  });

  // What virtualization does: the same card, a new element, after the
  // projection has already spent the debt.
  it("is not given again when the same card mounts a second time", () => {
    const first = draw([user("i1", "hi")], ["i1"]);
    expect(first.entered()).toBe(1);
    cleanup();
    const again = draw([user("i1", "hi")], []);
    expect(again.entered()).toBe(0);
  });

  it("survives its own card re-rendering while it is still owed", () => {
    const { view, entered } = draw([user("i1", "hi")], ["i1"]);
    expect(entered()).toBe(1);
    // The projection spends the debt on the next commit; the mark has to stay
    // for this element, or a streaming answer loses its entrance halfway.
    view.rerender(view.container.firstChild ? (
      <Transcript
        items={[user("i1", "hi")]} entering={[]} onEntered={() => {}} revision={2}
        waiting={{}} scroll={{ current: null }} hidden={false} onPinned={() => {}}
        jump={0} focus={null}
        onApprove={(async () => undefined) as never} onPlan={(async () => undefined) as never}
        onAnswer={(async () => undefined) as never} onSuggest={() => {}} onForget={(async () => undefined) as never}
        onCancelQueued={() => {}} onExtInvoke={() => {}} onExtSubmit={(async () => undefined) as never}
        checkpoints={new Map()} onPrepareRewind={(async () => undefined) as never}
        onCommitRewind={(async () => undefined) as never} onUndoRewind={(async () => undefined) as never}
        onPrepareFileRevert={(async () => undefined) as never} onCommitFileRevert={(async () => undefined) as never}
        needsProject={false} onOpenProject={() => {}} onKeepHere={() => {}}
      />
    ) : null);
    expect(entered()).toBe(1);
  });

  // A pane that is not the one on screen still holds its transcript. Coming
  // back to it is not a fact arriving.
  it("is spent even while the pane is not the one being looked at", () => {
    const { onEntered } = draw([user("i1", "hi")], ["i1"], true);
    expect(onEntered).toHaveBeenCalledWith(["i1"]);
  });
});

// The rule underneath both of the above, held on the stylesheet itself: a
// selector that fires an entrance the moment an element exists has taken
// mount for a cause. Written here rather than left to review, because the
// tempting line is one character short of the correct one.
describe("mount is not a cause", () => {
  /** Rules that give this selector an entrance animation, by exact term. */
  const entrances = (term: string) =>
    css
      .split("\n")
      // Turning one off is not giving one: the reduced-motion block names the
      // same selectors to say they must not move.
      .filter((l) => l.includes("animation:") && !/animation:\s*none/.test(l) && !l.trimStart().startsWith("/*"))
      .filter((l) => l.split("{")[0].split(",").some((sel) => sel.trim() === term));

  it("does not animate a transcript card into being", () => {
    expect(entrances(".call"), "a card animates whenever one exists — including replays").toEqual([]);
  });

  it("does not animate a settings block into being", () => {
    expect(entrances(".grp"), "every block flies in on a section change, and nothing happened").toEqual([]);
  });

  it("still animates the one that really did just arrive", () => {
    expect(entrances(".enterbox[data-enter] > .call").length).toBe(1);
    expect(entrances(".prefs-col").length).toBe(1);
  });
});
