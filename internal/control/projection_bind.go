package control

import (
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

// maybeColdResumePrune records warm/cold/unknown only; it never rewrites history.
func (c *Controller) maybeColdResumePrune(path string) {
	if c.executor == nil || path == "" {
		return
	}
	// Sidecar path is rebound in Resume; only refresh cache state here.
	m, ok, err := agent.LoadBranchMeta(path)
	if err != nil || !ok || m.UpdatedAt.IsZero() {
		c.executor.SetCacheState(agent.CacheStateUnknown)
		slog.Info("controller: resume cache state", "path", path, "cache_state", agent.CacheStateUnknown)
		c.emitResumeTelemetry(path, "unknown", 0, false, 0)
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
	// A valid projection means resume sends projection + tail, not the full
	// transcript — don't claim "full history kept" or users pay a cold summarize.
	projValid, projCovered := c.resumeProjectionStatus(path)
	c.emitResumeTelemetry(path, string(state), idleMin, projValid, projCovered)
	if c.disableColdResumePrune || state != agent.CacheStateCold {
		return
	}
	if projValid {
		c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
			"resumed after %s idle (provider cache expired) — context projection restored (%d messages covered); compaction deferred until context pressure",
			time.Since(last).Round(time.Minute), projCovered)})
		return
	}
	c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: fmt.Sprintf(
		"resumed after %s idle (provider cache likely expired) — full history kept; compaction deferred until context pressure",
		time.Since(last).Round(time.Minute))})
}

// resumeProjectionStatus reports whether the session sidecar carries a usable
// projection body, and how many canonical messages it covers. A covered count
// with a non-empty body means the first send after resume transmits
// projection + tail instead of the full transcript.
func (c *Controller) resumeProjectionStatus(path string) (valid bool, covered int) {
	st, ok, err := agent.LoadCompactionState(path)
	if err != nil || !ok {
		return false, 0
	}
	if st.Projection.CoveredCount > 0 && len(st.Projection.Messages) > 0 {
		return true, st.Projection.CoveredCount
	}
	return false, 0
}

// emitResumeTelemetry lands the C1 gate decision in the stats file so a
// historical reopen (and its cache warmth) is diagnosable after the fact.
func (c *Controller) emitResumeTelemetry(path, state string, idleMin int, projValid bool, projCovered int) {
	viewFP := ""
	coveredMatch := false
	if c.executor != nil {
		viewFP = c.executor.ModelVisibleFingerprint()
		coveredMatch, _ = c.executor.ProjectionCoveredMatch()
	}
	c.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: "resume telemetry",
		Detail: fmt.Sprintf("path=%s state=%s idle_min=%d decision=replay proj_valid=%t proj_covered=%d view_fp=%s covered_match=%t",
			path, state, idleMin, projValid, projCovered, viewFP, coveredMatch)})
}
