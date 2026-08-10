package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/provider/openai"
	"reasonix/internal/tool"
)

// echoTool is a trivial read-only tool used to drive a multi-step tool loop:
// each call appends an assistant(tool_call) + tool(result) pair to the history,
// growing the request prefix the way a real multi-turn session does.
type echoTool struct{}

func (echoTool) Name() string        { return "echo" }
func (echoTool) Description() string { return "echo back the given text" }
func (echoTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`)
}
func (echoTool) ReadOnly() bool { return true }
func (echoTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(args, &a)
	return "echoed: " + a.Text, nil
}

// collectSink captures the per-turn Usage events plus any compaction notices the
// agent emits, so the test can replay exactly what the status line would show.
type collectSink struct {
	usages  []*provider.Usage
	notices []string
	blocked bool
}

func (s *collectSink) Emit(e event.Event) {
	switch e.Kind {
	case event.Usage:
		if e.Usage != nil {
			s.usages = append(s.usages, e.Usage)
		}
	case event.Notice:
		s.notices = append(s.notices, e.Text)
	case event.ContextMaintenanceEvent:
		if e.Maintenance != nil && e.Maintenance.Status == "blocked" {
			s.blocked = true
		}
	}
}

// mockDeepSeek derives cache hits from the byte-identical prefix shared with
// the previous conversation request, directly measuring prefix stability.

type mockDeepSeek struct {
	t            *testing.T
	prevMessages []json.RawMessage   // last conversation request's messages
	reqChars     []int               // total prompt chars per conversation request
	hitChars     []int               // cached prefix chars per conversation request
	reqMsgs      [][]json.RawMessage // every conversation request's messages (P5 fork e2e)
	withTools    bool                // advertise the echo tool (and emit tool calls)
	reasoning    string              // chain-of-thought echoed every turn (round-tripped)
	toolRounds   int                 // remaining tool-call rounds before a final answer
}

func (m *mockDeepSeek) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	// Compaction issues a tool-less summarize request whose system prompt is the
	// summarizer prompt — answer it with a short summary and DON'T let it pollute
	// the conversation-prefix bookkeeping.
	if isSummarizeRequest(body) {
		writeSSE(w, m.t,
			streamChunk(deltaText("- goal: keep going\n- decisions: none\n- pending: continue")),
			finishChunk("stop"),
			usageChunk(100, 40, 0, 100),
		)
		return
	}

	msgs := decodeMessages(body)
	m.reqMsgs = append(m.reqMsgs, msgs)
	common := commonPrefixMsgs(m.prevMessages, msgs)
	hitChars := charsOf(msgs[:common])
	totalChars := charsOf(msgs)
	m.prevMessages = msgs
	m.reqChars = append(m.reqChars, totalChars)
	m.hitChars = append(m.hitChars, hitChars)

	promptTok := totalChars / 4
	hitTok := hitChars / 4
	missTok := promptTok - hitTok

	emitTool := m.withTools && m.toolRounds > 0
	if emitTool {
		m.toolRounds--
	}

	chunks := []sseResp{streamChunk(deltaReasoning(m.reasoning))}
	if emitTool {
		idx := len(m.reqChars)
		chunks = append(chunks,
			streamChunk(deltaToolCall(idx, "echo", fmt.Sprintf(`{"text":"round-%d"}`, idx))),
			finishChunk("tool_calls"))
	} else {
		chunks = append(chunks,
			streamChunk(deltaText("Done.")),
			finishChunk("stop"))
	}
	chunks = append(chunks, usageChunk(promptTok, 50, hitTok, missTok))
	writeSSE(w, m.t, chunks...)
}

func (m *mockDeepSeek) tools() *tool.Registry {
	reg := tool.NewRegistry()
	if m.withTools {
		reg.Add(echoTool{})
	}
	return reg
}

// hitRate is the status-line formula: hit / (hit+miss), falling back to prompt.
func hitRate(u *provider.Usage) int {
	denom := u.CacheHitTokens + u.CacheMissTokens
	if denom == 0 {
		denom = u.PromptTokens
	}
	if denom == 0 {
		return 0
	}
	return u.CacheHitTokens * 100 / denom
}

const systemPrompt = "You are reasonix, a coding agent. Be concise and follow project conventions. " +
	"This system prompt is the cacheable head of every request and must never change between turns."

// longReasoning stands in for a deepseek-reasoner chain-of-thought that the agent
// round-trips onto the assistant turn (agent.go round-trips ReasoningContent).
const longReasoning = "Let me reason about this carefully. I will weigh the constraints, " +
	"enumerate the candidate approaches, reject the ones that violate a requirement, and then " +
	"commit to the most defensible option, double-checking it against the original goal before answering."

// TestCacheHitPrefixStable proves the standard path keeps a byte-stable prefix:
// every request re-sends the full prior history untouched, and the displayed
// hit% equals hit/prompt%. This rules out "something is breaking the cache" and
// "the display math is wrong" for the no-compaction path.
func TestCacheHitPrefixStable(t *testing.T) {
	mock := &mockDeepSeek{t: t, withTools: true, reasoning: longReasoning, toolRounds: 2}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, sink := newAgent(t, srv.URL, mock.tools(), 0 /*no compaction*/, 0)
	if err := a.Run(context.Background(), "echo a couple things then finish"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Reconstruct the requests to check prefix stability. Replay equality is
	// already encoded in hitChars==full-previous-prefix, but assert it directly.
	for i := 1; i < len(mock.reqChars); i++ {
		// On request i the cached prefix should be the ENTIRE request i-1.
		if mock.hitChars[i] != mock.reqChars[i-1] {
			t.Errorf("PREFIX BROKEN at req %d: cached %d chars but the full prior request was %d chars",
				i, mock.hitChars[i], mock.reqChars[i-1])
		}
	}
	t.Logf("prefix STABLE across %d requests — nothing in the client breaks the cache", len(mock.reqChars))

	t.Logf("==== reported usage (what the status line renders) ====")
	for i, u := range sink.usages {
		want := -1
		if u.PromptTokens > 0 {
			want = 100 * u.CacheHitTokens / u.PromptTokens
		}
		t.Logf("turn %d: prompt=%d hit=%d miss=%d → 'cache %d%%' (hit/prompt=%d%%) | %s",
			i, u.PromptTokens, u.CacheHitTokens, u.CacheMissTokens, hitRate(u), want,
			strings.TrimSpace(FormatUsageLine(u, nil, nil)))
		if u.CacheHitTokens+u.CacheMissTokens != u.PromptTokens {
			t.Errorf("display denominator mismatch: hit+miss=%d != prompt=%d (status%% would read wrong)",
				u.CacheHitTokens+u.CacheMissTokens, u.PromptTokens)
		}
	}
}

// TestCacheHitClimbsWithoutCompaction runs a long multi-turn conversation with
// compaction DISABLED and prints the hit-rate curve. With a stable prefix the
// rate should climb past 90% as history dwarfs each turn's fresh tail.
func TestCacheHitClimbsWithoutCompaction(t *testing.T) {
	mock := &mockDeepSeek{t: t, reasoning: longReasoning}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, sink := newAgent(t, srv.URL, mock.tools(), 0 /*no compaction*/, 0)

	const turns = 14
	for i := range turns {
		userMsg := "Turn " + fmt.Sprint(i) + ": " + strings.Repeat("please consider this requirement. ", 6)
		if err := a.Run(context.Background(), userMsg); err != nil {
			t.Fatalf("Run %d: %v", i, err)
		}
	}

	t.Logf("==== hit-rate curve, NO compaction (%d turns) ====", turns)
	peak := 0
	for i, u := range sink.usages {
		r := hitRate(u)
		if r > peak {
			peak = r
		}
		t.Logf("turn %2d: prompt=%5d hit=%5d miss=%4d → cache %d%%", i, u.PromptTokens, u.CacheHitTokens, u.CacheMissTokens, r)
	}
	t.Logf("peak hit rate without compaction: %d%%", peak)
	if peak < 90 {
		t.Logf("NOTE: even with a perfectly stable prefix the rate plateaus below 90%% — "+
			"each turn's fresh tail (incl. %d-char round-tripped reasoning) is too large a share", len(longReasoning))
	}
}

// TestCacheHitSurvivesTooSmallWindow covers a window too small to summarize one
// turn. Maintenance must stop rewriting the same prefix and let cache hits
// recover instead of collapsing after every tool result.
func TestCacheHitSurvivesTooSmallWindow(t *testing.T) {
	mock := &mockDeepSeek{t: t, withTools: true, reasoning: longReasoning, toolRounds: 30}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, sink := newAgent(t, srv.URL, mock.tools(), 900 /*window tok*/, 4 /*recentKeep*/)

	if err := a.Run(context.Background(), strings.Repeat("please consider this requirement. ", 6)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	t.Logf("==== hit-rate curve, too-small window (900 tok) ====")
	collapses := 0
	for i, u := range sink.usages {
		r := hitRate(u)
		marker := ""
		if i > 0 && r+20 < hitRate(sink.usages[i-1]) {
			marker = "   <<< collapse"
			collapses++
		}
		t.Logf("step %2d: prompt=%5d hit=%5d miss=%4d → cache %3d%%%s", i, u.PromptTokens, u.CacheHitTokens, u.CacheMissTokens, r, marker)
	}

	for _, n := range sink.notices {
		t.Logf("notice: %s", n)
	}
	if sink.blocked {
		t.Log("context maintenance entered a durable blocked state")
	}

	// The guard caps the damage: a couple of compactions at most, not one per step.
	if collapses > 2 {
		t.Errorf("compaction cratered the cache %d times; the stuck guard should cap it at ≤2", collapses)
	}
	// With or without a blocked receipt, the same prefix must not be rewritten
	// after every following tool result, so the tail cache rate recovers.
	if n := len(sink.usages); n >= 6 {
		if tail := tailAverage(usageRates(sink.usages), 5); tail < 85 {
			t.Errorf("tail hit rate after the guard kicked in = %d%%, want ≥85%%", tail)
		}
	}
}

// TestReasoningRoundTripCost contrasts the hit-rate curve WITH vs WITHOUT the
// reasoning_content round-trip (agent.go re-sends the assistant chain-of-thought
// every turn). It quantifies how much that round-tripped CoT — assuming DeepSeek
// counts it as uncached prompt — drags the hit rate down at each turn.
func TestReasoningRoundTripCost(t *testing.T) {
	curve := func(reasoning string) []int {
		mock := &mockDeepSeek{t: t, reasoning: reasoning}
		srv := httptest.NewServer(http.HandlerFunc(mock.handler))
		defer srv.Close()
		a, sink := newAgent(t, srv.URL, mock.tools(), 0, 0)
		const turns = 12
		for i := range turns {
			if err := a.Run(context.Background(), strings.Repeat("please consider this requirement. ", 6)); err != nil {
				t.Fatalf("Run %d: %v", i, err)
			}
		}
		out := make([]int, len(sink.usages))
		for i, u := range sink.usages {
			out[i] = hitRate(u)
		}
		return out
	}

	withCoT := curve(longReasoning)
	without := curve("")

	t.Logf("==== reasoning round-trip: hit-rate cost per turn ====")
	t.Logf("turn | with reasoning round-trip | without (stripped) | delta")
	firstCross := func(c []int) int {
		for i, r := range c {
			if r >= 90 {
				return i
			}
		}
		return -1
	}
	for i := range withCoT {
		t.Logf("  %2d |          %3d%%             |       %3d%%          | +%d pts",
			i, withCoT[i], without[i], without[i]-withCoT[i])
	}
	t.Logf("turns needed to reach 90%%: with round-trip = %d, stripped = %d", firstCross(withCoT), firstCross(without))
}

// TestSessionAggregateCacheRate verifies the session-aggregate hit-rate the
// status line now shows: Agent.SessionCache() accumulates every turn's hit/miss
// (so it equals the sum of the per-turn usages), and the aggregate rate is the
// steadier, higher number compared to the volatile single-turn rate.
func TestSessionAggregateCacheRate(t *testing.T) {
	mock := &mockDeepSeek{t: t, reasoning: longReasoning}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, sink := newAgent(t, srv.URL, mock.tools(), 0, 0)
	const turns = 8
	for i := range turns {
		if err := a.Run(context.Background(), strings.Repeat("please consider this requirement. ", 6)); err != nil {
			t.Fatalf("Run %d: %v", i, err)
		}
	}

	// The agent's cumulative counters must equal the sum of the per-turn usages.
	var sumHit, sumMiss int
	for _, u := range sink.usages {
		sumHit += u.CacheHitTokens
		sumMiss += u.CacheMissTokens
	}
	hit, miss := a.SessionCache()
	if hit != sumHit || miss != sumMiss {
		t.Errorf("SessionCache()=%d/%d but per-turn sums are %d/%d", hit, miss, sumHit, sumMiss)
	}

	agg := 100 * hit / (hit + miss)
	last := sink.usages[len(sink.usages)-1]
	single := 100 * last.CacheHitTokens / last.PromptTokens
	t.Logf("after %d turns: aggregate(session) = %d%%  vs  single(last turn) = %d%%", turns, agg, single)
	if agg <= 0 || agg > 100 {
		t.Errorf("aggregate rate out of range: %d%%", agg)
	}
}

func TestSetSessionResetsSessionCache(t *testing.T) {
	mock := &mockDeepSeek{t: t, reasoning: longReasoning}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, _ := newAgent(t, srv.URL, mock.tools(), 0, 0)
	if err := a.Run(context.Background(), strings.Repeat("please consider this requirement. ", 6)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	hit, miss := a.SessionCache()
	if hit+miss == 0 {
		t.Fatalf("SessionCache()=%d/%d before reset, want telemetry to record the turn", hit, miss)
	}
	a.SetSession(NewSession("system"))
	hit, miss = a.SessionCache()
	if hit != 0 || miss != 0 {
		t.Fatalf("SessionCache()=%d/%d after SetSession, want reset", hit, miss)
	}
}

func TestReleaseCacheHitGuard(t *testing.T) {
	if os.Getenv("REASONIX_RELEASE_CACHE_GUARD") == "" {
		t.Skip("set REASONIX_RELEASE_CACHE_GUARD=1 to run the release cache guard")
	}

	threshold := envInt("REASONIX_CACHE_GUARD_THRESHOLD", 90)
	maxLowCases := envInt("REASONIX_CACHE_GUARD_MAX_LOW_CASES", 1)

	cases := []struct {
		name string
		run  func(*testing.T) []int
	}{
		{
			name: "plain-dialogue",
			run: func(t *testing.T) []int {
				return cacheCurve(t, &mockDeepSeek{t: t, reasoning: longReasoning}, 14)
			},
		},
		{
			name: "plain-dialogue-no-reasoning",
			run: func(t *testing.T) []int {
				return cacheCurve(t, &mockDeepSeek{t: t}, 14)
			},
		},
		{
			name: "long-dialogue",
			run: func(t *testing.T) []int {
				return cacheCurveWithMessages(t, &mockDeepSeek{t: t, reasoning: longReasoning}, repeatedMessages(18, 18))
			},
		},
		{
			name: "mixed-message-sizes",
			run: func(t *testing.T) []int {
				msgs := make([]string, 0, 20)
				for i := range 20 {
					repeats := 4
					if i%3 == 2 {
						repeats = 20
					}
					msgs = append(msgs, fmt.Sprintf("Turn %d: ", i)+strings.Repeat("preserve the request prefix while handling varied input. ", repeats))
				}
				return cacheCurveWithMessages(t, &mockDeepSeek{t: t, reasoning: longReasoning}, msgs)
			},
		},
		{
			name: "tool-loop",
			run: func(t *testing.T) []int {
				return toolLoopCurve(t, &mockDeepSeek{t: t, withTools: true, reasoning: longReasoning, toolRounds: 14})
			},
		},
		{
			name: "tool-loop-no-reasoning",
			run: func(t *testing.T) []int {
				return toolLoopCurve(t, &mockDeepSeek{t: t, withTools: true, toolRounds: 14})
			},
		},
		{
			name: "long-tool-loop",
			run: func(t *testing.T) []int {
				return toolLoopCurve(t, &mockDeepSeek{t: t, withTools: true, reasoning: longReasoning, toolRounds: 24})
			},
		},
		{
			name: "long-tool-loop-no-reasoning",
			run: func(t *testing.T) []int {
				return toolLoopCurve(t, &mockDeepSeek{t: t, withTools: true, toolRounds: 24})
			},
		},
	}

	type result struct {
		name string
		rate int
		all  []int
	}
	var lows []result
	for _, c := range cases {
		rates := c.run(t)
		rate := tailAverage(rates, 3)
		status := "pass"
		if rate < threshold {
			status = "low"
			lows = append(lows, result{name: c.name, rate: rate, all: rates})
		}
		t.Logf("CACHE_GUARD_RESULT: case=%s tail_avg=%d threshold=%d status=%s rates=%v",
			c.name, rate, threshold, status, rates)
	}

	if len(lows) > maxLowCases {
		var parts []string
		for _, low := range lows {
			parts = append(parts, fmt.Sprintf("%s=%d%%", low.name, low.rate))
		}
		msg := fmt.Sprintf("%d cache guard cases are below %d%%: %s", len(lows), threshold, strings.Join(parts, ", "))
		t.Logf("CACHE_GUARD_WARNING: %s", msg)
		if os.Getenv("REASONIX_CACHE_GUARD_STRICT") != "" {
			t.Fatal(msg)
		}
	}
}

func cacheCurve(t *testing.T, mock *mockDeepSeek, turns int) []int {
	return cacheCurveWithMessages(t, mock, repeatedMessages(turns, 6))
}

func repeatedMessages(turns, repeats int) []string {
	msgs := make([]string, 0, turns)
	for i := range turns {
		msgs = append(msgs, "Turn "+fmt.Sprint(i)+": "+strings.Repeat("please consider this requirement. ", repeats))
	}
	return msgs
}

func cacheCurveWithMessages(t *testing.T, mock *mockDeepSeek, messages []string) []int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, sink := newAgent(t, srv.URL, mock.tools(), 0, 0)
	for i, userMsg := range messages {
		if err := a.Run(context.Background(), userMsg); err != nil {
			t.Fatalf("Run %d: %v", i, err)
		}
	}
	return usageRates(sink.usages)
}

func toolLoopCurve(t *testing.T, mock *mockDeepSeek) []int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	a, sink := newAgent(t, srv.URL, mock.tools(), 0, 0)
	if err := a.Run(context.Background(), strings.Repeat("please consider this requirement. ", 6)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return usageRates(sink.usages)
}

func usageRates(usages []*provider.Usage) []int {
	out := make([]int, len(usages))
	for i, u := range usages {
		out[i] = hitRate(u)
	}
	return out
}

func tailAverage(xs []int, n int) int {
	if len(xs) == 0 {
		return 0
	}
	if n > len(xs) {
		n = len(xs)
	}
	sum := 0
	for _, x := range xs[len(xs)-n:] {
		sum += x
	}
	return sum / n
}

func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// newAgent wires a real openai.Provider at url into a real Agent.
func newAgent(t *testing.T, url string, reg *tool.Registry, contextWindow, recentKeep int) (*Agent, *collectSink) {
	t.Helper()
	prov, err := openai.New(provider.Config{
		Name:    "deepseek",
		BaseURL: url,
		Model:   "deepseek-reasoner",
		APIKey:  "test",
		Extra:   map[string]any{"api_key_env": "DEEPSEEK_API_KEY"},
	})
	if err != nil {
		t.Fatalf("provider New: %v", err)
	}
	sink := &collectSink{}
	a := New(prov, reg, NewSession(systemPrompt), Options{
		Temperature:   0,
		ContextWindow: contextWindow,
		RecentKeep:    recentKeep,
	}, sink)
	return a, sink
}

// request inspection helpers

func decodeMessages(body []byte) []json.RawMessage {
	var req struct {
		Messages []json.RawMessage `json:"messages"`
	}
	_ = json.Unmarshal(body, &req)
	return req.Messages
}

func isSummarizeRequest(body []byte) bool {
	msgs := decodeMessages(body)
	if len(msgs) == 0 {
		return false
	}
	var m struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal(msgs[0], &m)
	return m.Role == "system" && strings.Contains(m.Content, "compacting the earlier part")
}

func commonPrefixMsgs(a, b []json.RawMessage) int {
	n := 0
	for n < len(a) && n < len(b) && bytes.Equal(a[n], b[n]) {
		n++
	}
	return n
}

func charsOf(msgs []json.RawMessage) int {
	total := 0
	for _, m := range msgs {
		total += len(m)
	}
	return total
}

// SSE chunk builders matching the streamResponse shape the provider parses

type sseDelta struct {
	Content          string        `json:"content,omitempty"`
	ReasoningContent string        `json:"reasoning_content,omitempty"`
	ToolCalls        []sseToolCall `json:"tool_calls,omitempty"`
}

type sseToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type sseChoice struct {
	Delta        sseDelta `json:"delta"`
	FinishReason *string  `json:"finish_reason"`
}

type sseResp struct {
	Choices []sseChoice `json:"choices"`
	Usage   *sseUsage   `json:"usage,omitempty"`
}

type sseUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}

func deltaReasoning(s string) sseDelta { return sseDelta{ReasoningContent: s} }
func deltaText(s string) sseDelta      { return sseDelta{Content: s} }
func deltaToolCall(idx int, name, args string) sseDelta {
	tc := sseToolCall{Index: idx, ID: fmt.Sprintf("call_%d", idx), Type: "function"}
	tc.Function.Name = name
	tc.Function.Arguments = args
	return sseDelta{ToolCalls: []sseToolCall{tc}}
}

func streamChunk(d sseDelta) sseResp { return sseResp{Choices: []sseChoice{{Delta: d}}} }
func finishChunk(reason string) sseResp {
	return sseResp{Choices: []sseChoice{{FinishReason: &reason}}}
}
func usageChunk(prompt, completion, hit, miss int) sseResp {
	return sseResp{Usage: &sseUsage{
		PromptTokens:          prompt,
		CompletionTokens:      completion,
		TotalTokens:           prompt + completion,
		PromptCacheHitTokens:  hit,
		PromptCacheMissTokens: miss,
	}}
}

func writeSSE(w http.ResponseWriter, t *testing.T, chunks ...sseResp) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	f, ok := w.(http.Flusher)
	if !ok {
		t.Fatal("ResponseWriter is not a Flusher")
	}
	for _, c := range chunks {
		b, _ := json.Marshal(c)
		fmt.Fprintf(w, "data: %s\n\n", b)
		f.Flush()
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	f.Flush()
}

// TestForkChildFirstRequestReusesParentCachePrefix is the P5 fork cache e2e
// (T5): a fork child must inherit the parent's committed history byte-for-byte
// (T5-1), score a provider cache hit on its FIRST request (T5-2), and leave the
// parent session and prefix shape untouched (T5-3).
// Scenario: the parent really talks to the mock provider for two committed
// turns, so the provider-side cache covers the whole parent history. The fork
// is then spawned through the real TaskTool path (fire-and-forget silent
// background job, same provider). The mock derives cache hits from the
// byte-identical shared prefix, so the child's first request re-sending the
// parent's last-request bytes MUST report usage.cache_hit_tokens > 0.
func TestForkChildFirstRequestReusesParentCachePrefix(t *testing.T) {
	mock := &mockDeepSeek{t: t, reasoning: longReasoning}
	srv := httptest.NewServer(http.HandlerFunc(mock.handler))
	defer srv.Close()

	// The provider's cached prefix now covers the parent's full history, and
	// the parent's last request carries turn-1's assistant response (the exact
	// bytes a fork child must re-send).
	parent, _ := newAgent(t, srv.URL, mock.tools(), 0 /*no compaction*/, 0)
	for i, msg := range []string{
		"turn one: establish a committed history",
		"turn two: commit another round",
	} {
		if err := parent.Run(context.Background(), msg); err != nil {
			t.Fatalf("parent run %d: %v", i+1, err)
		}
	}
	if len(mock.reqMsgs) != 2 {
		t.Fatalf("parent sent %d requests, want 2", len(mock.reqMsgs))
	}

	// 2. The inherited prefix is the parent's committed history — the same
	//    bytes the parent last sent, plus the not-yet-sent assistant tail.
	prefix := captureForkPrefix(parent, context.Background())
	if len(prefix) != len(parent.session.Snapshot()) {
		t.Fatalf("fork prefix len %d != parent session len %d", len(prefix), len(parent.session.Snapshot()))
	}

	// 3. Freeze the parent-untouched baselines (T5-3).
	parentBefore := marshalMessages(t, parent.session.Snapshot())
	schemas := parent.tools.Schemas()
	shapeBefore := parent.capturePrefixShape(schemas)

	// 4. Spawn the fork through the real task path: fire-and-forget silent
	//    background job whose child uses the same provider (same server-side
	//    cache as the parent).
	subProv, err := openai.New(provider.Config{
		Name:    "deepseek",
		BaseURL: srv.URL,
		Model:   "deepseek-reasoner",
		APIKey:  "test",
		Extra:   map[string]any{"api_key_env": "DEEPSEEK_API_KEY"},
	})
	if err != nil {
		t.Fatalf("subagent provider New: %v", err)
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

	out, err := task.Execute(fctx, []byte(`{"prompt":"investigate the committed history and report back","fork":true}`))
	if err != nil {
		t.Fatalf("fork Execute: %v", err)
	}
	if !strings.Contains(out, "Started background fork") {
		t.Fatalf("fork output = %q, want 'Started background fork'", out)
	}
	jobID := extractJobID(out)
	if jobID == "" {
		t.Fatalf("no job id in fork output:\n%s", out)
	}
	res := jm.WaitForSession(context.Background(), "parent-session", []string{jobID}, 10)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("fork job result = %+v, want one done job", res)
	}

	// 5. The child's first request is the third request the mock saw.
	if len(mock.reqMsgs) != 3 {
		t.Fatalf("mock saw %d requests, want 3 (parent×2 + fork child first)", len(mock.reqMsgs))
	}
	parentLast := mock.reqMsgs[1]
	childFirst := mock.reqMsgs[2]

	// T5-1 byte-identical: the child's first request re-sends the parent's
	// last-request bytes verbatim as its prefix — the parent's cache key.
	if common := commonPrefixMsgs(parentLast, childFirst); common != len(parentLast) {
		t.Fatalf("fork child first request shares only %d/%d messages with the parent's last request — prefix not byte-identical",
			common, len(parentLast))
	}
	// …and it is exactly the inherited prefix plus exactly one new user message.
	if len(childFirst) != len(prefix)+1 {
		t.Fatalf("child first request has %d messages, want prefix %d + 1 user", len(childFirst), len(prefix))
	}

	// T5-2 cache hit: the mock derives hit chars from the byte-identical
	// prefix, so the child's first request must report cache_hit_tokens > 0.
	if mock.hitChars[2] <= 0 {
		t.Fatalf("child first request cache hit = %d chars, want > 0", mock.hitChars[2])
	}
	if len(childSink.usages) == 0 {
		t.Fatalf("no subagent usage was reported for the fork child")
	}
	if u := childSink.usages[0]; u.CacheHitTokens <= 0 {
		t.Fatalf("child first usage = %+v, want CacheHitTokens > 0", u)
	}

	// T5-3 parent untouched: session bytes and prefix shape are stable.
	if after := marshalMessages(t, parent.session.Snapshot()); after != parentBefore {
		t.Fatalf("parent session mutated by fork\n before: %s\n after: %s", parentBefore, after)
	}
	if shapeAfter := parent.capturePrefixShape(schemas); shapeAfter != shapeBefore {
		t.Fatalf("parent prefix shape changed by fork: %v -> %v", shapeBefore, shapeAfter)
	}

	t.Logf("==== fork cache reuse (T5) ====")
	t.Logf("parent last request: %d msgs / %d chars", len(parentLast), mock.reqChars[1])
	t.Logf("fork child first request: %d msgs / %d chars, cached prefix %d chars → cache_hit_tokens > 0",
		len(childFirst), mock.reqChars[2], mock.hitChars[2])
	t.Logf("parent session + prefix shape unchanged by the fork")
}

// TestIsFreshSubagentSessionForkLock locks the P5 discipline point: a fork
// child's session is prefilled with the parent prefix, so
// isFreshSubagentSession MUST return false and the subagentStartContext
// prepend must never fire for it (an inserted block would break the prefix).
func TestIsFreshSubagentSessionForkLock(t *testing.T) {
	// Control: a plain fresh sub-agent session (system only) is fresh.
	if !isFreshSubagentSession(NewSession(systemPrompt)) {
		t.Fatal("plain fresh sub-agent session must be fresh")
	}
	// A fork session is NewSession("") + the parent prefix added one by one
	// (the exact PrepareParentFork shape): system + history ⇒ not fresh.
	prefix := captureForkPrefix(forkPrefixTestAgent(t, []provider.Message{
		{Role: provider.RoleUser, Content: "committed turn"},
		{Role: provider.RoleAssistant, Content: "answer"},
	}), context.Background())
	forkSess := NewSession("")
	for _, m := range prefix {
		forkSess.Add(m)
	}
	if isFreshSubagentSession(forkSess) {
		t.Fatal("fork session (prefilled with the parent prefix) must NOT be fresh")
	}
	// The store path that actually builds fork children produces the same shape.
	store := NewSubagentStore(t.TempDir())
	run, err := store.PrepareParentFork(prefix, SubagentSpec{
		ParentSession: "parent-session",
		Registry:      tool.NewRegistry(),
	})
	if err != nil {
		t.Fatalf("PrepareParentFork: %v", err)
	}
	defer run.Release()
	if isFreshSubagentSession(run.Session) {
		t.Fatal("PrepareParentFork session must NOT be fresh")
	}
}
