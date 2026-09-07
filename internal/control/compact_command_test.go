package control

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// TestManualCompactEmitsVisibleResultMessage locks in the /compact UX contract:
// manual compaction must surface a visible assistant message (Text+Message
// pair) with the outcome, because the async compaction card alone renders on
// no transcript surface in some agent sessions.
func TestManualCompactEmitsVisibleResultMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	sess := agent.NewSession("sys")
	for i := range 30 {
		content := "question " + string(rune('a'+i%26)) + " " + strings.Repeat("words ", 60)
		sess.Add(provider.Message{Role: provider.RoleUser, Content: content})
		sess.Add(provider.Message{Role: provider.RoleAssistant, Content: "answer " + strings.Repeat("words ", 40)})
	}
	exec := agent.New(nil, nil, sess, agent.Options{}, event.Discard)
	sink := &recordingSink{}
	c := New(Options{Executor: exec, SessionDir: dir, SessionPath: path, Label: "test", Sink: sink})

	c.submitCompact("")

	deadline := time.Now().Add(10 * time.Second)
	var injected string
	for time.Now().Before(deadline) {
		for _, ev := range sink.all() {
			if ev.Kind == event.Message && strings.HasPrefix(ev.Text, "compaction") {
				injected = ev.Text
			}
		}
		if injected != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if injected == "" {
		t.Fatal("no visible compaction result message emitted after /compact")
	}
	if !strings.Contains(injected, "compaction complete") && !strings.Contains(injected, "compaction failed") {
		t.Fatalf("injected message = %q, want a compaction complete/failed outcome", injected)
	}
}
