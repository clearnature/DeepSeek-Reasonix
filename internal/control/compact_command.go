package control

import (
	"context"
	"fmt"
	"log/slog"

	"reasonix/internal/agent"
	"reasonix/internal/event"
)

// submitCompact runs /compact off the dispatch path and surfaces the outcome
// as both a notice and a visible assistant message: manual compaction must
// never be silent, since the async compaction card alone renders on no
// transcript surface in some agent sessions.
func (c *Controller) submitCompact(focus string) {
	go func() {
		before := c.contextTokens()
		if err := c.Compact(context.Background(), focus); err != nil {
			c.notice("compaction failed: " + err.Error())
			c.emitAssistantText("compaction failed: " + err.Error())
			return
		}
		src, result := c.compactionTokens()
		if src <= 0 || result <= 0 {
			// Older sidecars carry no receipt sizes; fall back to the prompt
			// gauges, which are only populated after a main request.
			src, result = before, c.contextTokens()
		}
		c.notice("compacted")
		c.emitAssistantText(compactionResultText(src, result))
		if err := c.SnapshotRewrite(); err != nil {
			slog.Warn("controller: snapshot after compact", "err", err)
		}
	}()
}

// emitAssistantText surfaces a controller-initiated result as a visible
// assistant message (same Text+Message pair the skill runner emits). Emitted
// outside a turn it never reaches the turn-event ledger, so it renders in the
// transcript without entering the model-visible context.
func (c *Controller) emitAssistantText(text string) {
	text = agent.DisplayAssistantText(text)
	c.sink.Emit(event.Event{Kind: event.Text, Text: text})
	c.sink.Emit(event.Event{Kind: event.Message, Text: text})
}

// contextTokens samples the current model-visible prompt size; -1 means the
// executor is unavailable and no before/after comparison can be shown.
func (c *Controller) contextTokens() int {
	if c.executor == nil {
		return -1
	}
	return c.executor.ContextReport().LatestPrompt
}

// compactionTokens reads the last applied compaction's own source/result sizes.
// A resumed session compacts before any main request, so the prompt gauges are
// still zero there while the receipt already carries the real numbers.
func (c *Controller) compactionTokens() (int, int) {
	if c.executor == nil {
		return -1, -1
	}
	rep := c.executor.ContextReport()
	return rep.LastSource, rep.LastResult
}

func compactionResultText(before, after int) string {
	if before >= 0 && after >= 0 {
		return fmt.Sprintf("compaction complete: manual compression, context %d→%d tokens", before, after)
	}
	return "compaction complete: manual compression finished"
}
