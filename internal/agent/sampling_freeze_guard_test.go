package agent

import (
	"context"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// TestPrepareSamplingRequestFreezesSentUnit is the guard for the f1cce4a6b
// freeze call sites: a real prepare must capture the exact messages+tools it
// is about to send, otherwise the summarizer silently falls back to the live
// registry and every summary request forks at the tools seam (system-only
// cache hits). The v1.38 convergence dropped both call sites and no test
// caught it because the summary tests froze units by hand.
func TestPrepareSamplingRequestFreezesSentUnit(t *testing.T) {
	prov := &countingProvider{reply: "digest"}
	reg := tool.NewRegistry()
	reg.Add(echoTool{})
	a := New(prov, reg, &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
	}}, Options{ContextWindow: 100_000, MaxOutputTokens: 1024}, event.Discard)

	if _, err := a.prepareSamplingRequest(context.Background()); err != nil {
		t.Fatalf("prepareSamplingRequest: %v", err)
	}
	saved := a.savedMainRequest()
	if saved == nil || len(saved.messages) == 0 {
		t.Fatal("prepareSamplingRequest did not freeze the sent unit — saveMainRequest call sites are missing")
	}
	if len(saved.tools) != 1 || saved.tools[0].Name != "echo" {
		t.Fatalf("frozen tools = %+v, want the single echo schema", saved.tools)
	}
}
