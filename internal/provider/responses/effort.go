package responses

import (
	"strings"

	"reasonix/internal/provider"
)

// requestEffort resolves one request's reasoning depth: a non-empty
// EffortOverride (e.g. "none" on compaction summaries) wins, otherwise the
// client-level configured effort applies. Empty means "leave unset".
func requestEffort(req provider.Request, configured string) string {
	if e := strings.ToLower(strings.TrimSpace(req.EffortOverride)); e != "" {
		return e
	}
	return strings.ToLower(strings.TrimSpace(configured))
}
