package control

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
)

// bindExecutorProjection rebinds the agent's projection sidecar to path.
// loadSidecar=true loads an existing sidecar (resume/switch); false clears
// in-memory projection without deleting another session's .context.json.
func (c *Controller) bindExecutorProjection(path string, loadSidecar bool) {
	if c == nil || c.executor == nil {
		return
	}
	c.executor.BindSessionPath(path, loadSidecar)
}

// maybeColdResumePrune records warm/cold/unknown and compacts on cold resume.
func (c *Controller) maybeColdResumePrune(path string) {
	if c.disableColdResumePrune || c.executor == nil || path == "" {
		return
	}
	// Idle time comes from branch meta only — every session the controller has
	// ever snapshotted carries one. A meta-less transcript (e.g. a legacy import
	// not yet saved) skips the prune until its first snapshot creates the meta.
	m, ok, err := agent.LoadBranchMeta(path)
	if err != nil || !ok || m.UpdatedAt.IsZero() {
		c.executor.SetCacheState(agent.CacheStateUnknown)
		slog.Info("controller: resume cache state", "path", path, "cache_state", agent.CacheStateUnknown)
		c.emitResumeTelemetry(path, "unknown", 0)
		return
	}
	last := m.UpdatedAt
	state := agent.CacheStateWarm
	if time.Since(last) >= c.cacheColdAfter() {
		state = agent.CacheStateCold
	}
	c.executor.SetCacheState(state)
	slog.Info("controller: resume cache state", "path", path, "cache_state", state, "idle", time.Since(last).Round(time.Minute).String())
	idleMin := int(time.Since(last).Minutes())
	c.emitResumeTelemetry(path, string(state), idleMin)
	if state != agent.CacheStateCold {
		return
	}
	// Outside the cache window: the prefix is cold. Prune stale tool results
	// first (cheap, no network); then compact the whole session so the first
	// send after resume is a small digest+tail prefix, not a full-price replay
	// of the entire history.
	st, err := c.executor.PruneStaleToolResults()
	if err != nil {
		slog.Warn("controller: cold-resume prune", "err", err)
	} else if st.Results > 0 {
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
			"resumed after %s idle (provider cache expired) — elided %d stale tool results to cheapen the cold restart",
			time.Since(last).Round(time.Minute), st.Results)})
	} else {
		slog.Warn("controller: cold-resume prune: no results", "results", st.Results, "contextWindow", c.executor.ContextWindow())
	}
	// Full compaction: A1 merges the digest chain into one rolling digest, B2
	// folds small user turns beyond the position-fixed window. The session
	// becomes a small, byte-stable prefix — the deterministic cheap replay.
	if err := c.Compact(context.Background(), "cold-resume replay gate"); err != nil {
		slog.Warn("controller: cold-resume compact", "err", err)
	}
	if err := c.SnapshotRewrite(); err != nil {
		slog.Warn("controller: post-prune snapshot", "err", err)
	}
}

// emitResumeTelemetry lands the C1 gate decision in the stats file so a
// historical reopen (and its cache warmth) is diagnosable after the fact.
func (c *Controller) emitResumeTelemetry(path, state string, idleMin int) {
	c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: "resume telemetry",
		Detail: fmt.Sprintf("path=%s state=%s idle_min=%d decision=replay", path, state, idleMin)})
}
