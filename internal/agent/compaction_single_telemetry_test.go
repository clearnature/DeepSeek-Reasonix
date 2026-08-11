package agent

import (
	"context"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestCompactToProjectionEmitsSingleTelemetry pins the 2026-08-10 duplicate-row
// bug: the merged success path called emit(tele) and then
// emitCompactionTelemetry(tele) again, so every successful /compact wrote two
// identical compaction rows 20µs apart. A successful install must emit exactly
// one telemetry row.
func TestCompactToProjectionEmitsSingleTelemetry(t *testing.T) {
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "system stays"},
		{Role: provider.RoleUser, Content: "old request alpha"},
		{Role: provider.RoleAssistant, Content: strings.Repeat("analysis ", 160)},
		{Role: provider.RoleTool, ToolCallID: "read-1", Name: "read_file", Content: strings.Repeat("old tool output ", 160)},
		{Role: provider.RoleUser, Content: "unique boundary request"},
		{Role: provider.RoleAssistant, Content: "tail stays byte-for-byte"},
	}}
	prov := &fakeProvider{reply: "old work summarized"}
	sink := &noticeCaptureSink{}
	a := New(prov, tool.NewRegistry(), sess, Options{ArchiveDir: t.TempDir()}, event.Discard)
	a.sink = sink

	outcome, err := a.compactToProjection(context.Background(), CompactionTriggerManual, "", true, false)
	if err != nil {
		t.Fatalf("compactToProjection: %v", err)
	}
	if outcome != CompactionInstalled {
		t.Fatalf("outcome=%v, want Installed", outcome)
	}
	var telemetry []string
	for _, d := range sink.notices {
		if strings.Contains(d, "trigger=manual") && strings.Contains(d, "status=installed") {
			telemetry = append(telemetry, d)
		}
	}
	if len(telemetry) != 1 {
		t.Fatalf("emitted %d compaction telemetry rows, want exactly 1:\n%s", len(telemetry), strings.Join(telemetry, "\n"))
	}
}
