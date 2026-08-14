/** Live-turn markers that a lagging history snapshot must not replace. */
export type HydrateLiveState = {
  running?: boolean;
  turnActive?: boolean;
  live?: unknown;
  currentAssistant?: unknown;
  pendingUser?: unknown;
  historyRevision?: number;
  historyDigest?: string;
  items: ReadonlyArray<{ kind: string; streaming?: boolean; status?: string }>;
};

export function hasCachedLiveTurn(state: HydrateLiveState | undefined): boolean {
  if (!state?.running && !state?.turnActive) return false;
  if (state.live || state.currentAssistant || state.pendingUser !== undefined) return true;
  return state.items.some((item) =>
    (item.kind === "assistant" && item.streaming) ||
    (item.kind === "tool" && item.status === "running"),
  );
}

// Skip replace when a live transcript is already on screen; an empty
// surface still has to apply history or switch-back shows Welcome.
export function shouldApplyHydratedHistory(
  skipHistory: boolean,
  hasProjection: boolean,
  foregroundTurnActive: boolean,
  state: HydrateLiveState | undefined,
): boolean {
  if (skipHistory || !hasProjection) return false;
  if (!foregroundTurnActive) return true;
  return (state?.items.length ?? 0) === 0 && !hasCachedLiveTurn(state);
}

export function sameSessionPlaceholderItems<T>(
  targetSessionPath: string | undefined,
  prev: { meta?: { sessionPath?: string }; items?: T[] } | undefined,
): T[] | undefined {
  const target = (targetSessionPath ?? "").trim();
  const current = (prev?.meta?.sessionPath ?? "").trim();
  if (!target || !current || target !== current) return undefined;
  return prev?.items;
}

// A hydrated page candidate for the resident transcript.
export type HydrateProjection = {
  items: ReadonlyArray<unknown>;
  revision?: number;
  digest?: string;
};

function sameHydrateFingerprint(state: HydrateLiveState | undefined, projection: HydrateProjection | undefined): boolean {
  if (!state || !projection) return false;
  const revision = projection.revision ?? 0;
  const digest = (projection.digest ?? "").trim();
  if (revision > 0 && state.historyRevision === revision) return true;
  if (digest !== "" && (state.historyDigest ?? "") === digest) return true;
  return false;
}

// Reject a replace when the resident transcript is longer and the candidate
// page carries the same fingerprint: the backend served a shorter
// same-fingerprint page (e.g. after a failed clear / retry), and applying it
// would roll the on-screen history back. #8731 — dev had only a load-side
// gate (hasReusableCachedTranscript), this closes the apply-side race.
export function isStaleResidentProjection(
  state: HydrateLiveState | undefined,
  projection: HydrateProjection | undefined,
): boolean {
  if (!state || !projection || state.items.length === 0) return false;
  if (projection.items.length >= state.items.length) return false;
  return sameHydrateFingerprint(state, projection);
}
