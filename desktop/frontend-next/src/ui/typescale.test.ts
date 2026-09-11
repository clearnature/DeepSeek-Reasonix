import { describe, expect, it } from "vitest";
// Not through import.meta.glob: vite's CSS handling intercepts ?raw there and
// hands back an empty string, which every rule below would then pass.
import css from "../styles/app.css?raw";

// Half a pixel is not a level. The interface carried 24 sizes with 442 of its
// declarations inside the 10–11.5px band, so nothing read as one step above
// anything — only as slightly different, everywhere. Six steps replace them,
// and this keeps a seventh from being typed in by hand.
describe("a size comes from the scale", () => {
  /** The size slot of every type declaration: before the `/line-height`. */
  const sizes = (): { where: string; slot: string }[] =>
    [...css.matchAll(/(?<![\w-])(font-size|font):\s*([^;}]+)/g)].map((m) => ({
      where: m[1],
      slot: m[2].split("/")[0],
    }));

  const RAW = /(?<![\w.-])(\d+(?:\.\d+)?)px(?![\w-])/g;
  // A length inside calc() against the reading token adjusts that token; it is
  // not a size of its own, and reading it as one puts 1.5px under the floor.
  const typedByHand = (slot: string) => !slot.includes("calc(") && !slot.includes("var(--read)");

  it("never types one into the interface", () => {
    // 24px and up is a brand moment — a splash wordmark is one size for one
    // thing, not a step other text is measured against.
    const typed = sizes().flatMap(({ where, slot }) =>
      [...slot.matchAll(RAW)]
        .filter((m) => Number(m[1]) < 24)
        .filter(() => typedByHand(slot))
        .map((m) => `${where}: …${m[0]}…`),
    );
    expect([...new Set(typed)], "hand-typed sizes: the scale says each one once").toEqual([]);
  });

  it("covers the type in the stylesheet", () => {
    // Guards the guard: a renamed property would leave the rule above passing
    // on an empty list.
    expect(sizes().length).toBeGreaterThan(300);
  });

  it("keeps the floor at the step the scale defines", () => {
    // Han strokes lose their counters before Latin does, and AA's large-text
    // exemption starts at 18.66px — nothing in the interface has ever been
    // inside it. The smallest step is the floor, so no rule may undercut it
    // by reaching for a raw value.
    const under = sizes().flatMap(({ slot }) =>
      [...slot.matchAll(RAW)]
        .filter((m) => Number(m[1]) < 11)
        .filter(() => typedByHand(slot))
        .map((m) => m[0]),
    );
    expect([...new Set(under)], "a size below the scale's floor").toEqual([]);
  });
});
