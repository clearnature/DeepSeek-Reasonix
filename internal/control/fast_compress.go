package control

import (
	"context"
	"fmt"
	"strings"

	"reasonix/internal/tool/builtin"
)

// applyFastCompress is the /compress-fast host command: elide stale tool
// results into a prune projection with no model round-trip. It lives in its
// own file (absent upstream) so a controller.go convergence cannot silently
// drop the implementation; only the routing case in controller.go is exposed.
func (c *Controller) applyFastCompress(trimmed string) {
	c.mu.Lock()
	running := c.running
	c.mu.Unlock()
	if running {
		c.notice("fast compress failed: a turn is running")
		return
	}
	exec := c.executor
	if exec == nil {
		c.notice("fast compress failed: no executor")
		return
	}
	stats, err := exec.FastCompressManual()
	if err != nil {
		c.notice("fast compress failed: " + err.Error())
		return
	}
	if stats.Results == 0 {
		c.notice("no stale tool results to compress")
		return
	}
	text := fmt.Sprintf("fast-compressed: %d stale tool result(s) elided", stats.Results)
	c.notice(text)
	c.emitAssistantText("fast compression complete: " + text)
}

// applyRetrieveInfo is the /retrieve_info host command: run the retrieval
// pipeline on the typed query and surface the rendered answer, with no model
// round-trip.
func (c *Controller) applyRetrieveInfo(trimmed string) {
	q := strings.TrimSpace(strings.TrimPrefix(trimmed, "/retrieve_info"))
	if q == "" {
		c.notice("retrieve_info: query is required")
		return
	}
	answer, _, err := builtin.RetrieveSystem(context.Background(), q)
	if err != nil {
		c.notice("retrieve_info failed: " + err.Error())
		return
	}
	c.noticeDetail("retrieve_info: "+q, answer)
}
