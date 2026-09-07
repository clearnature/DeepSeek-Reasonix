package control

import (
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestFastCompressCommandElidesAndInjectsVisibleResult locks in the
// /compress-fast route guard: the host command must reach the manual no-AI
// elision path and surface a visible assistant result message, so a future
// controller.go convergence dropping the case fails loudly instead of
// silently.
func TestFastCompressCommandElidesAndInjectsVisibleResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	bigTool := strings.Repeat("界", 9000)
	sess := agent.NewSession("sys")
	sess.Add(provider.Message{Role: provider.RoleUser, Content: "task"})
	sess.Add(provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "t1", Name: "read_file", Arguments: "{}"}}})
	sess.Add(provider.Message{Role: provider.RoleTool, ToolCallID: "t1", Name: "read_file", Content: bigTool})
	sess.Add(provider.Message{Role: provider.RoleAssistant, Content: "work"})
	sess.Add(provider.Message{Role: provider.RoleUser, Content: "tail"})

	exec := agent.New(nil, tool.NewRegistry(), sess, agent.Options{ContextWindow: 60_000, SessionPath: path}, event.Discard)
	sink := &recordingSink{}
	c := New(Options{Executor: exec, SessionDir: dir, SessionPath: path, Label: "test", Sink: sink})

	c.applyFastCompress("/compress-fast")

	var injected string
	for _, ev := range sink.all() {
		if ev.Kind == event.Message && strings.HasPrefix(ev.Text, "fast compression complete") {
			injected = ev.Text
		}
	}
	if !strings.Contains(injected, "1 stale tool result") {
		t.Fatalf("injected message = %q, want elision result with count", injected)
	}
}
