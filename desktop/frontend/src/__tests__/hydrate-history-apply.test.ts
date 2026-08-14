import { test } from "node:test";
import assert from "node:assert/strict";
import { isStaleResidentProjection, type HydrateLiveState, type HydrateProjection } from "../lib/hydrateHistoryApply";

// #8731 apply-side gate: a shorter same-fingerprint page must not replace a
// longer resident transcript (rolls the on-screen history back after a
// failed clear / retry).
test("isStaleResidentProjection rejects a shorter same-fingerprint page", () => {
  const resident: HydrateLiveState = {
    historyRevision: 42,
    historyDigest: "d1",
    items: [{ kind: "user" }, { kind: "assistant" }, { kind: "tool" }],
  };
  const projection: HydrateProjection = { items: [{ kind: "user" }], revision: 42, digest: "d1" };
  assert.equal(isStaleResidentProjection(resident, projection), true);
});

test("isStaleResidentProjection allows a longer or equal page", () => {
  const resident: HydrateLiveState = { historyRevision: 7, historyDigest: "d", items: [{ kind: "user" }] };
  assert.equal(
    isStaleResidentProjection(resident, { items: [{ kind: "user" }, { kind: "assistant" }], revision: 8, digest: "d" }),
    false,
  );
  assert.equal(
    isStaleResidentProjection(resident, { items: [{ kind: "user" }], revision: 8, digest: "d" }),
    false,
    "equal length is not stale",
  );
});

test("isStaleResidentProjection allows a same-length different fingerprint", () => {
  const resident: HydrateLiveState = { historyRevision: 1, historyDigest: "d1", items: [{ kind: "user" }] };
  assert.equal(isStaleResidentProjection(resident, { items: [{ kind: "assistant" }], revision: 2, digest: "d2" }), false);
});

test("isStaleResidentProjection no-ops on empty resident or no projection", () => {
  assert.equal(isStaleResidentProjection(undefined, { items: [{ kind: "user" }] }), false);
  assert.equal(isStaleResidentProjection({ items: [] }, { items: [{ kind: "user" }] }), false);
  assert.equal(isStaleResidentProjection({ items: [{ kind: "user" }] }, undefined), false);
});
