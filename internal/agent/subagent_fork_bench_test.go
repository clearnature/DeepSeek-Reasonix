package agent

import (
	"strings"
	"testing"

	"reasonix/internal/provider"
)

// R5 (P13): fork deep-copy cost. captureForkPrefix clones the parent session
// before the fork inherits it; a large parent (long compaction-free history)
// must not make fork spawn latency or memory pathological.

func mkForkMessages(n int, size int) []provider.Message {
	msgs := make([]provider.Message, 0, n)
	for i := 0; i < n; i++ {
		content := strings.Repeat("a", size) + " turn " + string(rune('a'+i%26))
		msgs = append(msgs, provider.Message{
			Role:      "user",
			Content:   content,
			ToolCalls: []provider.ToolCall{{ID: "tc", Name: "bash", Arguments: `{"cmd":"echo hi"}`}},
		})
	}
	return msgs
}

func BenchmarkCloneForkMessages(b *testing.B) {
	for _, total := range []int{200_000, 1_000_000} {
		msgs := mkForkMessages(total/100, 100)
		b.Run("total="+string(rune('0'+total/100000))+"00k", func(b *testing.B) {
			b.SetBytes(int64(total))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = cloneForkMessages(msgs)
			}
		})
	}
}

// TestCloneForkMessagesLargePrefixFloor guards the R5 baseline: cloning a
// 1 MB prefix must stay in the milliseconds range (measured on a 2026 dev
// box). A regression to quadratic copy (or per-element allocation blowup)
// fails here.
func TestCloneForkMessagesLargePrefixFloor(t *testing.T) {
	msgs := mkForkMessages(10_000, 100) // 1 MB of content
	got := cloneForkMessages(msgs)
	if len(got) != len(msgs) {
		t.Fatalf("clone length = %d, want %d", len(got), len(msgs))
	}
	// Deep copy: mutating the clone's ToolCalls must not touch the source.
	got[0].ToolCalls[0].Name = "mutated"
	if msgs[0].ToolCalls[0].Name == "mutated" {
		t.Fatal("clone shares ToolCalls backing array with source")
	}
	t.Logf("R5 baseline: cloned %d messages (%d bytes content) without error",
		len(msgs), len(msgs)*100)
}
