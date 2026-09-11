package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReplaySessionEventLogTailRecoversOversizedLog covers the deadlock the
// byte budget used to create: an oversized log tripped replaySessionEventLog
// before save.go could run compactSessionEventLog, so the log could never
// shrink and every save failed. Replace records are self-contained snapshots,
// so replaying from the last one rebuilds the same state without decoding the
// oversized prefix.
func TestReplaySessionEventLogTailRecoversOversizedLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "oversized.events.jsonl")
	var b strings.Builder
	for i := range 300 {
		fmt.Fprintf(&b, `{"schema_version":1,"type":"append","message_index":%d,"messages":[{"role":"user","content":"bulk filler that pushes the log past the byte budget"}]}`+"\n", i)
	}
	b.WriteString(`{"schema_version":1,"type":"replace","messages":[{"role":"system","content":"sys"},{"role":"user","content":"tail"}]}` + "\n")
	contents := b.String()
	if err := os.WriteFile(logPath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write event log: %v", err)
	}
	if int64(len(contents)) <= 512 {
		t.Fatalf("fixture too small (%d bytes) to exceed the budget", len(contents))
	}

	full, err := replaySessionEventLogWithLimits(logPath, sessionReplayLimits{
		maxBytes: int64(len(contents) + 1), maxRecords: 10_000, maxMessages: 10_000,
	}, nil)
	if err != nil {
		t.Fatalf("full replay: %v", err)
	}

	tail, err := replaySessionEventLogWithLimits(logPath, sessionReplayLimits{
		maxBytes: 512, maxRecords: 10_000, maxMessages: 10_000,
	}, nil)
	if err != nil {
		t.Fatalf("tail replay of oversized log: %v", err)
	}
	if len(tail.msgs) != len(full.msgs) || len(tail.msgs) != 2 {
		t.Fatalf("tail replay got %d messages, full replay %d, want 2", len(tail.msgs), len(full.msgs))
	}
	if tail.msgs[1].Content != "tail" {
		t.Fatalf("tail replay lost the replace snapshot: %q", tail.msgs[1].Content)
	}
	if tail.lastGoodEnd != full.lastGoodEnd {
		t.Fatalf("lastGoodEnd = %d, full replay %d — tail offsets must stay file-absolute", tail.lastGoodEnd, full.lastGoodEnd)
	}
	if tail.damaged {
		t.Fatal("tail replay marked the log damaged")
	}
}

// TestReplaySessionEventLogTailRefusesAppendOnlyLog keeps the safety property:
// an oversized log with no replace record has no snapshot to anchor on, so it
// must still be refused rather than guessed at.
func TestReplaySessionEventLogTailRefusesAppendOnlyLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "appendonly.events.jsonl")
	var b strings.Builder
	for range 300 {
		b.WriteString(`{"schema_version":1,"type":"append","message_index":0,"messages":[{"role":"user","content":"bulk filler that pushes the log past the byte budget"}]}` + "\n")
	}
	if err := os.WriteFile(logPath, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("write event log: %v", err)
	}
	_, err := replaySessionEventLogWithLimits(logPath, sessionReplayLimits{
		maxBytes: 512, maxRecords: 10_000, maxMessages: 10_000,
	}, nil)
	if err == nil {
		t.Fatal("append-only oversized log replayed without an anchor, want refusal")
	}
}
