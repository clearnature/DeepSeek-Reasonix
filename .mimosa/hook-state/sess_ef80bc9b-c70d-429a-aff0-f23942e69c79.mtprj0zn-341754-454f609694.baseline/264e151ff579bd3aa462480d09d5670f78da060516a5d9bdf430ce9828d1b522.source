package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/event"
)

// TestRecorderWritesEstimateAnomaly covers the estimate telemetry notice path:
// an unreliable admission-time prompt estimate must land in the daily stats
// file so an inflated desktop context % can be diagnosed post-hoc.
func TestRecorderWritesEstimateAnomaly(t *testing.T) {
	dir := t.TempDir()
	r := NewRecorder(&spySink{}, dir, "desktop")
	r.Emit(event.Event{Kind: event.Notice, Text: "estimate telemetry",
		Detail: "reason=inflated est=8900000 window=1048576 obs=1293567 chars=24000000 cchars=21000000 cjk=1000 cjkb=3000 msgs=42 top_role=assistant top_chars=12000000 cal=false"})
	flushRecorder(t, r)
	data, err := os.ReadFile(filepath.Join(dir, dailyJSONLFiles(t, dir)[0].Name()))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got struct {
		Estimate *EstimateAnomalyRecord `json:"estimate"`
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.Contains(line, "estimate") {
			if err := json.Unmarshal([]byte(line), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			break
		}
	}
	if got.Estimate == nil {
		t.Fatalf("no estimate row in daily file: %s", data)
	}
	e := got.Estimate
	if e.Reason != "inflated" || e.EstTok != 8900000 || e.WindowTok != 1048576 || e.ObsTok != 1293567 ||
		e.Chars != 24000000 || e.CompactChars != 21000000 || e.CJKRunes != 1000 || e.CJKBytes != 3000 ||
		e.Messages != 42 || e.TopRole != "assistant" || e.TopChars != 12000000 || e.Calibrated {
		t.Fatalf("estimate record = %+v", e)
	}
}
