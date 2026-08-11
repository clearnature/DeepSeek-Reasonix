//go:build realapi

package agent

import (
	"context"
	"os"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/provider/openai"
	"reasonix/internal/tool"
)

// TestForkCacheHitRealDeepSeek is the P5 release verification against the real
// DeepSeek API (REASONIX_REAL_API=1 + DEEPSEEK_API_KEY). The fork child's
// first request must report cache_hit_tokens > 0 because it re-sends the
// parent's committed prefix bytes — the provider's server-side cache key.
func TestForkCacheHitRealDeepSeek(t *testing.T) {
	if os.Getenv("REASONIX_REAL_API") == "" {
		t.Skip("set REASONIX_REAL_API=1 to run against the real DeepSeek API")
	}
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Fatal("DEEPSEEK_API_KEY not set")
	}
	systemPrompt := "You are a terse coding agent. Answer in one short line."
	parentSink := &collectSink{}
	parent, parentProv, err := newRealAgent(t, "https://api.deepseek.com", "deepseek-v4-flash", key, systemPrompt, parentSink)
	if err != nil {
		t.Fatalf("parent provider: %v", err)
	}
	defer func() {
		if c, ok := parentProv.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}()
	for i, msg := range []string{
		// Each turn must accumulate real prefix length: DeepSeek v4-flash's
		// prefix-cache floor is ~256 tokens (vllm#42948), so a two-line
		// history never builds a cacheable prefix. The assistant replies add
		// to the committed history on every turn.
		"turn one: write a 3-sentence paragraph about prefix caching, then reply with the word: alpha",
		"turn two: write a 3-sentence paragraph about KV cache reuse, then reply with the word: beta",
		"turn three: write a 3-sentence paragraph about prompt byte stability, then reply with the word: gamma",
		"turn four: write a 3-sentence paragraph about server-side caching, then reply with the word: delta",
	} {
		if err := parent.Run(context.Background(), msg); err != nil {
			t.Fatalf("parent run %d: %v", i+1, err)
		}
	}
	if len(parentSink.usages) != 4 {
		t.Fatalf("parent usages = %d, want 4", len(parentSink.usages))
	}
	t.Logf("parent request 4: prompt=%d cache_hit=%d", parentSink.usages[3].PromptTokens, parentSink.usages[3].CacheHitTokens)
	if parentSink.usages[3].PromptTokens < 256 {
		t.Fatalf("parent prefix %d tokens < DeepSeek v4-flash cache floor (~256); lengthen the fixture", parentSink.usages[3].PromptTokens)
	}
	if parentSink.usages[3].CacheHitTokens <= 0 {
		t.Logf("NOTE: parent's own 4th request did not hit the cache (hit=%d) — the server may need a moment or the floor is higher; the fork assertion below is the decisive one", parentSink.usages[3].CacheHitTokens)
	}

	// Spawn the fork through the real task path; the child reuses the same
	// provider/model so it shares the parent's server-side prefix cache.
	subProv, err := openai.New(provider.Config{
		Name:    "deepseek",
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-v4-flash",
		APIKey:  key,
	})
	if err != nil {
		t.Fatalf("subagent provider: %v", err)
	}
	task := NewTaskTool(subProv, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")

	childSink := &collectSink{}
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	fctx := withCallContext(context.Background(), "fork-call", childSink, nil, false)
	fctx = WithForkSource(fctx, parent)
	fctx = jobs.WithSession(fctx, "parent-session")
	fctx = jobs.WithManager(fctx, jm)
	fctx = WithParentSession(fctx, "parent-session")

	out, err := task.Execute(fctx, []byte(`{"prompt":"Report the two words you were asked to remember.","fork":true}`))
	if err != nil {
		t.Fatalf("fork Execute: %v", err)
	}
	jobID := extractJobID(out)
	if jobID == "" {
		t.Fatalf("no job id in fork output:\n%s", out)
	}
	res := jm.WaitForSession(context.Background(), "parent-session", []string{jobID}, 90)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("fork job result = %+v, want one done job", res)
	}
	if len(childSink.usages) == 0 {
		t.Fatalf("no usage reported for the fork child")
	}
	child := childSink.usages[0]
	t.Logf("fork child first request: prompt=%d cache_hit=%d", child.PromptTokens, child.CacheHitTokens)
	if child.CacheHitTokens <= 0 {
		t.Fatalf("fork child first request cache_hit=%d, want > 0 (prefix must hit the parent's server cache)", child.CacheHitTokens)
	}
	// Close the real providers' HTTP/2 keep-alive connections so goleak does
	// not report their transport goroutines as leaks at package teardown.
	if c, ok := subProv.(interface{ Close() error }); ok {
		_ = c.Close()
	}
}

// newRealAgent builds an Agent wired to a real OpenAI-compatible provider;
// usage flows through the collectSink's Emit (event.Usage).
func newRealAgent(t *testing.T, baseURL, model, key, systemPrompt string, sink *collectSink) (*Agent, provider.Provider, error) {
	t.Helper()
	prov, err := openai.New(provider.Config{
		Name:    "deepseek",
		BaseURL: baseURL,
		Model:   model,
		APIKey:  key,
	})
	if err != nil {
		return nil, nil, err
	}
	return New(prov, tool.NewRegistry(), NewSession(systemPrompt), Options{Temperature: 0}, sink), prov, nil
}
