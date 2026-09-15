package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// summaryCostProbe records every provider request so both summary strategies
// can be priced on the same session.
type summaryCostProbe struct {
	requests []provider.Request
}

func (p *summaryCostProbe) Name() string { return "cost-probe" }

func (p *summaryCostProbe) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.requests = append(p.requests, req)
	prompt := denseTokens(req)
	return chunks(
		provider.Chunk{Type: provider.ChunkText, Text: "## Goal\nsummarised"},
		provider.Chunk{Type: provider.ChunkUsage, Usage: &provider.Usage{
			PromptTokens: prompt, CompletionTokens: 100, TotalTokens: prompt + 100, RequestCount: 1,
		}},
		provider.Chunk{Type: provider.ChunkDone},
	), nil
}

// prefixHitTokens reports how many prompt tokens of req reuse the frozen main
// request's leading bytes. The provider's cache matches a leading run of
// identical bytes, so the measure is the longest shared message prefix.
func (p *summaryCostProbe) prefixHitTokens(req provider.Request, main []provider.Message) int {
	shared := 0
	for shared < len(req.Messages) && shared < len(main) {
		if providerVisibleFingerprint([]provider.Message{req.Messages[shared]}) !=
			providerVisibleFingerprint([]provider.Message{main[shared]}) {
			break
		}
		shared++
	}
	if shared == 0 {
		return 0
	}
	return denseTokens(provider.Request{Messages: req.Messages[:shared]})
}

// TestSummaryStrategyCostComparison prices both summary strategies on the same
// over-length session. DeepSeek bills a cache hit at a fraction of the input
// rate, so the strategy replaying the frozen main-request prefix pays far less
// for the same digest than one re-sending the live view at full price.
func TestSummaryStrategyCostComparison(t *testing.T) {
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "system prompt"}}
	for i := range 400 {
		body := strings.Repeat(fmt.Sprintf("tool output line %d with identifier PATH_%03d. ", i, i), 40)
		msgs = append(msgs,
			provider.Message{Role: provider.RoleUser, Content: fmt.Sprintf("turn %d: read it", i)},
			provider.Message{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: fmt.Sprintf("c%d", i), Name: "read_file", Arguments: "{}"}}},
			provider.Message{Role: provider.RoleTool, ToolCallID: fmt.Sprintf("c%d", i), Name: "read_file", Content: body},
		)
	}

	// DeepSeek pay-as-you-go: ¥0.5/M cached input, ¥4/M uncached input.
	const hitPrice, missPrice = 0.5, 4.0

	measure := func(name string, window int, reuseFrozenPrefix bool) {
		probe := &summaryCostProbe{}
		reg := tool.NewRegistry()
		a := New(probe, reg, &Session{Messages: msgs}, Options{
			ContextWindow: window,
			CompactRatio:  0.8,
			ArchiveDir:    t.TempDir(),
		}, event.Discard)
		if reuseFrozenPrefix {
			wire := freezeProviderRequest(provider.Request{Messages: append([]provider.Message(nil), msgs...)})
			a.saveMainRequest(wire.Messages, wire.Tools)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := a.CompactNow(ctx, ""); err != nil {
			t.Logf("%s: CompactNow returned %v", name, err)
		}

		var hitTok, missTok, calls int
		for _, req := range probe.requests {
			prompt := denseTokens(req)
			hit := 0
			if reuseFrozenPrefix {
				hit = probe.prefixHitTokens(req, msgs)
			}
			hitTok += hit
			missTok += prompt - hit
			calls++
		}
		cost := float64(hitTok)/1e6*hitPrice + float64(missTok)/1e6*missPrice
		full := float64(hitTok+missTok) / 1e6 * missPrice
		t.Logf("%-14s calls=%-3d hitTok=%-8d missTok=%-8d cost=¥%.6f  all-miss=¥%.6f  saved=%.1f%%",
			name, calls, hitTok, missTok, cost, full, (1-cost/full)*100)
	}

	t.Logf("=== summary strategy cost, 400 tool turns, main prompt %d tokens ===", denseTokens(provider.Request{Messages: msgs}))
	t.Log("--- single-request path (200k window: fold fits one summary call) ---")
	measure("aligned-200k", 200_000, true)
	measure("live-200k", 200_000, false)
	t.Log("--- chunked path (60k window forces the fragment fallback) ---")
	measure("aligned-60k", 60_000, true)
	measure("live-60k", 60_000, false)
}
