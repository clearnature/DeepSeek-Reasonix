package agent

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCompactionDetailKeysAreParsed guards the four-way telemetry chain
// (struct ↔ emit detail ↔ stats parse ↔ JSON tag). The v1.38 convergence
// dropped local keys while leaving the struct fields in place, so the build
// stayed green and the values silently never reached stats — summary_input and
// pref_hash both regressed that way. Adding an emit key without a matching
// parse case now fails here instead.
func TestCompactionDetailKeysAreParsed(t *testing.T) {
	full := CompactionTelemetry{
		Trigger: "manual", Mode: "summarized", SummaryInputMode: "cache_prefix", CacheState: "warm",
		SourceTokens: 1, FoldTokens: 2, Spans: 3, ProjectionTokens: 4,
		InputTokens: 5, OutputTokens: 6, CacheHitTokens: 7, CacheMissTokens: 8, CacheWriteTokens: 9,
		RequestCount: 10, UserTurnsKept: 11, UserTurnsDropped: 12,
		ViewFP: "view", WireFP: "wire", ToolsCount: 13, ToolsFP: "tools", ToolsSource: "frozen",
		PrefixHash: "pref", PrefLen: 14, WireLen: 15, WireDiff: "i3:origin", ProviderRequestID: "req", Error: "boom",
	}
	detail := compactionDetailString(full) + " err_type=boom"
	emitted := compactionDetailKeys(detail)
	if len(emitted) == 0 {
		t.Fatal("compaction detail emitted no keys")
	}
	parsed := statsCompactionParseKeys(t)
	for k := range emitted {
		if !parsed[k] {
			t.Errorf("emit key %q has no case in internal/stats/recorder.go; stats would silently drop it", k)
		}
	}
	for _, must := range []string{"summary_input", "pref_hash", "pref_len", "wire_len", "wire_diff", "view_fp", "wire_fp", "tools_source"} {
		if !emitted[must] {
			t.Errorf("emit lost key %q — the stats record depends on it", must)
		}
	}
}

func compactionDetailKeys(detail string) map[string]bool {
	out := map[string]bool{}
	for _, tok := range strings.Fields(detail) {
		if k, _, ok := strings.Cut(tok, "="); ok {
			out[k] = true
		}
	}
	return out
}

// statsCompactionParseKeys reads the recorder's switch cases instead of
// importing stats, which would invert the dependency direction.
func statsCompactionParseKeys(t *testing.T) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("..", "stats", "recorder.go"))
	if err != nil {
		t.Fatalf("read stats recorder: %v", err)
	}
	out := map[string]bool{}
	for _, m := range regexp.MustCompile(`case "([a-z_]+)":`).FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = true
	}
	return out
}
