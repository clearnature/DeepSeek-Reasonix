package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/provider"
)

// TestOfficialMessagesTokensIncludesServerSearch guards the CJK/official-estimate
// path against dropping server-search replay bytes (Anthropic counts replayed
// search results toward input tokens; officialMessagesTokens must match).
func TestOfficialMessagesTokensIncludesServerSearch(t *testing.T) {
	msg := provider.Message{
		Role:         provider.RoleUser,
		Content:      "search now",
		ServerSearch: []provider.ServerSearchCall{{ID: "ws_1", Query: "deepseek pricing", Raw: json.RawMessage(`{"page":"` + strings.Repeat("x", 4000) + `"}`)}},
	}
	base := officialMessagesTokens([]provider.Message{{Role: provider.RoleUser, Content: "search now"}})
	with := officialMessagesTokens([]provider.Message{msg})
	if with <= base {
		t.Fatalf("officialMessagesTokens did not account for server-search raw: base=%d with=%d", base, with)
	}
	if with < base+100 {
		t.Fatalf("server-search raw undercounted: base=%d with=%d", base, with)
	}
}
