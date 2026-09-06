package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
)

// Compaction telemetry persistence guards: each compaction pass emits one
// Notice (success or failure); the recorder parses it into a row that
// QueryCompactions returns and usage aggregation never sees.

func TestRecorderPersistsCompactionTelemetry(t *testing.T) {
	dir := t.TempDir()
	inner := &spySink{}
	r := NewRecorder(inner, dir, "desktop")
	// Compaction notices must reach the frontend unchanged.
	r.Emit(event.Event{Kind: event.Notice, Text: "compaction failed", Detail: "trigger=manual mode=summarized cache=warm src=5108937 proj=0 in=0 out=0 hit=0 miss=0 write=0 reqs=2 provider_request_id=req-abc err_type=deepseek-responses: status 400: over window"})
	r.Emit(event.Event{Kind: event.Notice, Text: "compaction telemetry", Detail: "trigger=auto mode=summarized cache=warm src=2666458 proj=297000 in=10 out=20 hit=30 miss=40 write=50 reqs=1 tpc=0.250 view_fp=abc123 wire_fp=def456 tools_count=23 tools_fp=feedface tools_source=frozen"})
	flushRecorder(t, r)

	w := NewWriter(dir)
	rows := w.QueryCompactions("desktop", time.Now().Add(-time.Hour), time.Now())
	if len(rows) != 2 {
		t.Fatalf("want 2 compaction rows, got %d", len(rows))
	}
	// Newest first: the auto row landed after the failed row.
	failed, auto := rows[1], rows[0]
	if failed.Trigger != "manual" || failed.Mode != "summarized" || failed.Cache != "warm" {
		t.Fatalf("failed row: trigger=%s mode=%s cache=%s", failed.Trigger, failed.Mode, failed.Cache)
	}
	if failed.Status != "failed" {
		t.Fatalf("failed row must carry status=failed, got %q", failed.Status)
	}
	if failed.SourceTok != 5108937 || failed.Reqs != 2 {
		t.Fatalf("failed row numbers: src=%d reqs=%d", failed.SourceTok, failed.Reqs)
	}
	if failed.RequestID != "req-abc" {
		t.Fatalf("failed row request id: %q", failed.RequestID)
	}
	if !strings.Contains(failed.Error, "status 400") || !strings.Contains(failed.Error, "over window") {
		t.Fatalf("failed row error should keep the full multi-word message: %q", failed.Error)
	}
	if auto.Trigger != "auto" || auto.SourceTok != 2666458 || auto.ProjTok != 297000 || auto.ViewFP != "abc123" {
		t.Fatalf("auto row: trigger=%s src=%d proj=%d view_fp=%s", auto.Trigger, auto.SourceTok, auto.ProjTok, auto.ViewFP)
	}
	if auto.WireFP != "def456" || auto.ToolsFP != "feedface" || auto.ToolsSource != "frozen" || auto.ToolsCount != 23 {
		t.Fatalf("auto row wire attribution: wire_fp=%s tools_count=%d tools_fp=%s tools_source=%s", auto.WireFP, auto.ToolsCount, auto.ToolsFP, auto.ToolsSource)
	}
	if auto.TokPerChar != 0.25 {
		t.Fatalf("auto row tpc: %v, want 0.250 (calibrated factor must survive the stats round-trip)", auto.TokPerChar)
	}
	if auto.InputTok != 10 || auto.OutTok != 20 || auto.HitTok != 30 || auto.MissTok != 40 || auto.WriteTok != 50 {
		t.Fatalf("auto row tokens: in=%d out=%d hit=%d miss=%d write=%d", auto.InputTok, auto.OutTok, auto.HitTok, auto.MissTok, auto.WriteTok)
	}
	// Compaction rows must not leak into billed usage aggregation.
	got, err := w.Query(SourceFilter{From: time.Now().Add(-time.Hour), To: time.Now()})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if got.Tokens != 0 || got.Requests != 0 {
		t.Fatalf("compaction rows leaked into usage: tokens=%d requests=%d", got.Tokens, got.Requests)
	}
	// Non-compaction notices (e.g. context-warning) are not persisted.
	before := len(rows)
	r.Emit(event.Event{Kind: event.Notice, Text: "Context is getting large; preserving cache until cleanup is needed.", Detail: "context reached 50% of window"})
	flushRecorder(t, r)
	if got := len(w.QueryCompactions("desktop", time.Now().Add(-time.Hour), time.Now())); got != before {
		t.Fatalf("unrelated notice persisted as compaction: %d rows", got)
	}
}

func TestRecorderCompactionForwardsNoticeToFrontend(t *testing.T) {
	dir := t.TempDir()
	inner := &spySink{}
	r := NewRecorder(inner, dir, "desktop")
	r.Emit(event.Event{Kind: event.Notice, Text: "compaction telemetry", Detail: "trigger=overflow mode=summarized cache=cold src=100 proj=50 in=1 out=2 hit=3 miss=4 write=5 reqs=1"})
	flushRecorder(t, r)
	if len(inner.events) != 1 {
		t.Fatalf("frontend must receive the compaction notice, got %d events", len(inner.events))
	}
}

func TestRecorderPersistsCompactionNoop(t *testing.T) {
	dir := t.TempDir()
	inner := &spySink{}
	r := NewRecorder(inner, dir, "desktop")
	// A compaction that folds nothing must still land one row with
	// status=noop — the blind spot where /compact reported "compacted"
	// while the stats file showed nothing.
	r.Emit(event.Event{Kind: event.Notice, Text: "compaction telemetry", Detail: "trigger=manual mode=summarized status=noop cache=warm src=2180242 proj=0 in=0 out=0 hit=0 miss=0 write=0 reqs=0"})
	flushRecorder(t, r)

	w := NewWriter(dir)
	rows := w.QueryCompactions("desktop", time.Now().Add(-time.Hour), time.Now())
	if len(rows) != 1 {
		t.Fatalf("want 1 compaction row, got %d", len(rows))
	}
	rec := rows[0]
	if rec.Status != "noop" {
		t.Fatalf("status=%q, want noop", rec.Status)
	}
	if rec.Trigger != "manual" || rec.Mode != "summarized" || rec.SourceTok != 2180242 {
		t.Fatalf("record fields wrong: %+v", rec.CompactionRecord)
	}
}

func TestRecorderWritesCompactionPrefHash(t *testing.T) {
	dir := t.TempDir()
	r := NewRecorder(&spySink{}, dir, "desktop")
	r.Emit(event.Event{Kind: event.Notice, Text: "compaction telemetry",
		Detail: "trigger=pressure mode=summarized status=installed cache=warm src=1000 fold=200 spans=1 proj=300 in=400 out=50 hit=10 miss=390 write=0 reqs=1 tpc=0.25 reason=fold user_kept=2 user_dropped=1 pref_hash=abc123def456"})
	flushRecorder(t, r)
	files := dailyJSONLFiles(t, dir)
	data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got struct {
		Compaction *CompactionRecord `json:"compaction"`
	}
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Compaction == nil || got.Compaction.PrefHash != "abc123def456" {
		t.Fatalf("pref_hash not parsed: %+v", got.Compaction)
	}
	if got.Compaction.UserKept != 2 || got.Compaction.UserDrop != 1 {
		t.Fatalf("user turn fields: %+v", got.Compaction)
	}
}
