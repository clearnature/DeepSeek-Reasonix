package responses

import (
	"strings"

	"reasonix/internal/provider"
)

// responsesEffortVocabulary is the union of effort values the Responses
// endpoints accept (deepseek: minimal/low/high/max + medium/xhigh aliases;
// mimo: none/low/medium/high; off/disabled/auto are normalized downstream).
// An EffortOverride outside this set falls back to the configured effort,
// mirroring openai/effort.go's vocabulary check.
var responsesEffortVocabulary = map[string]bool{
	"none": true, "off": true, "disabled": true, "auto": true,
	"minimal": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true,
}

// requestEffort resolves one request's reasoning depth: a vocabulary-approved
// EffortOverride (e.g. "none" on compaction summaries) wins, otherwise the
// client-level configured effort applies. Empty means "leave unset".
func requestEffort(req provider.Request, configured string) string {
	want := strings.ToLower(strings.TrimSpace(req.EffortOverride))
	if want != "" && responsesEffortVocabulary[want] {
		return want
	}
	return strings.ToLower(strings.TrimSpace(configured))
}

// normalizeEffort applies vendor aliasing/normalization to a resolved effort
// (deepseek medium/xhigh → high; auto/disabled/off → none).
func normalizeEffort(effort, model string) string {
	if strings.EqualFold(strings.TrimSpace(model), "deepseek-v4-flash") || strings.EqualFold(strings.TrimSpace(model), "deepseek-v4-pro") {
		if effort == "medium" || effort == "xhigh" {
			effort = "high"
		}
	}
	switch effort {
	case "auto":
		return ""
	case "disabled", "off":
		return "none"
	}
	return effort
}

// reasoningBody assembles the optional reasoning request object from the
// resolved effort and the vendor-declared summary mode; nil omits the field.
func reasoningBody(effort, summaryMode string) map[string]any {
	reasoning := map[string]any{}
	if effort != "" {
		reasoning["effort"] = effort
	}
	if summaryMode != "" {
		reasoning["summary"] = summaryMode
	}
	if len(reasoning) == 0 {
		return nil
	}
	return reasoning
}
