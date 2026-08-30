package agent

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

func TestProjectionValidRejectsEditedPrefix(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task-v1"},
		{Role: provider.RoleAssistant, Content: "done"},
		{Role: provider.RoleUser, Content: "next"},
	}
	st := CompactionState{
		TranscriptVersion: 2,
		PromptCacheKey:    "ws|sess|model",
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "sys"},
				{Role: provider.RoleUser, Content: "summary"},
			},
			TranscriptVersion: 2,
			CoveredCount:      3,
			CoveredPrefixHash: coveredPrefixHash(msgs, 3),
		},
	}
	if !projectionValid(st, msgs, "ws|sess|model") {
		t.Fatal("expected valid projection for matching prefix")
	}
	// Append-only growth still valid.
	grown := append(append([]provider.Message(nil), msgs...), provider.Message{Role: provider.RoleAssistant, Content: "more"})
	if !projectionValid(st, grown, "ws|sess|model") {
		t.Fatal("append-only growth should keep projection valid")
	}
	// Prefix edit invalidates.
	edited := append([]provider.Message(nil), msgs...)
	edited[1].Content = "task-EDITED"
	if projectionValid(st, edited, "ws|sess|model") {
		t.Fatal("edited covered prefix must invalidate projection")
	}
}

func TestProjectionValidRejectsCacheKeyMismatch(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
	}
	hash := coveredPrefixHash(msgs, 2)
	st := CompactionState{
		TranscriptVersion: 1,
		PromptCacheKey:    "ws|sess|model-a",
		Projection: ContextProjection{
			Messages:          []provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
			CoveredCount:      2,
			CoveredPrefixHash: hash,
			TranscriptVersion: 1,
		},
	}
	if projectionValid(st, msgs, "ws|sess|model-b") {
		t.Fatal("model/lineage key mismatch must invalidate projection")
	}
	if !projectionValid(st, msgs, "ws|sess|model-a") {
		t.Fatal("matching key should be valid")
	}
	// Fail closed: blank stored key is rejected when current key is known.
	st.PromptCacheKey = ""
	if projectionValid(st, msgs, "ws|sess|model-a") {
		t.Fatal("missing sidecar cache key must invalidate when lineage is known")
	}
	// Missing prefix hash is always rejected.
	st.PromptCacheKey = "ws|sess|model-a"
	st.Projection.CoveredPrefixHash = ""
	if projectionValid(st, msgs, "ws|sess|model-a") {
		t.Fatal("missing CoveredPrefixHash must invalidate projection")
	}
}

func TestCoveredPrefixHashIncludesProviderVisibleFields(t *testing.T) {
	base := []provider.Message{{
		Role:               provider.RoleAssistant,
		Content:            "answer",
		ReasoningContent:   "think",
		ReasoningID:        "rid-1",
		ReasoningStatus:    "completed",
		ReasoningSignature: "sig-1",
		Images:             []string{"data:image/png;base64,AAA"},
		ToolCalls: []provider.ToolCall{{
			ID: "c1", Name: "f", Arguments: `{}`, ThoughtSignature: "ts-1",
		}},
		ResponsesItems: []json.RawMessage{json.RawMessage(`{"type":"web_search_call"}`)},
	}}
	h1 := coveredPrefixHash(base, 1)
	if h1 == "" {
		t.Fatal("empty fingerprint")
	}
	// Each provider-visible field change must move the hash.
	cases := []struct {
		name string
		mut  func([]provider.Message)
	}{
		{"images", func(m []provider.Message) { m[0].Images = []string{"data:image/png;base64,BBB"} }},
		{"reasoning_id", func(m []provider.Message) { m[0].ReasoningID = "rid-2" }},
		{"reasoning_status", func(m []provider.Message) { m[0].ReasoningStatus = "in_progress" }},
		{"reasoning_signature", func(m []provider.Message) { m[0].ReasoningSignature = "sig-2" }},
		{"thought_signature", func(m []provider.Message) { m[0].ToolCalls[0].ThoughtSignature = "ts-2" }},
		{"responses_items", func(m []provider.Message) {
			m[0].ResponsesItems = []json.RawMessage{json.RawMessage(`{"type":"other"}`)}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := []provider.Message{base[0]}
			mutated[0].ToolCalls = append([]provider.ToolCall(nil), base[0].ToolCalls...)
			mutated[0].Images = append([]string(nil), base[0].Images...)
			mutated[0].ResponsesItems = append([]json.RawMessage(nil), base[0].ResponsesItems...)
			tc.mut(mutated)
			if coveredPrefixHash(mutated, 1) == h1 {
				t.Fatalf("%s change did not alter coveredPrefixHash", tc.name)
			}
		})
	}
}

func TestLoadProjectionSidecarRebindsMatchingContentAcrossLineage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
	}
	hash := coveredPrefixHash(msgs, 2)
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:     compactionStateSchemaV1,
		PromptCacheKey:    "ws|s|other-model",
		TranscriptVersion: 1,
		Projection: ContextProjection{
			Messages:          []provider.Message{{Role: provider.RoleSystem, Content: "sys summary"}},
			CoveredCount:      2,
			CoveredPrefixHash: hash,
		},
	}); err != nil {
		t.Fatal(err)
	}
	sess := NewSession("sys")
	sess.Add(provider.Message{Role: provider.RoleUser, Content: "task"})
	a := New(nil, nil, sess, Options{
		SessionPath: path,
		WorkspaceID: "ws",
		ModelRef:    "this-model",
	}, event.Discard)
	// New() already called LoadProjectionSidecar; the projection body matches
	// the canonical covered prefix, so it must be rebound to the current key
	// instead of being dropped (upgrade / model-switch path).
	if len(a.sess.compactionState.Projection.Messages) == 0 {
		t.Fatal("matching projection body was dropped on lineage change")
	}
	wantKey := promptCacheKey("ws", BranchID(path), "this-model")
	if a.sess.compactionState.PromptCacheKey != wantKey {
		t.Fatalf("PromptCacheKey = %q, want %q", a.sess.compactionState.PromptCacheKey, wantKey)
	}
	if a.sess.checkpointState != "restored" {
		t.Fatalf("checkpointState = %q, want restored", a.sess.checkpointState)
	}
	// The rebind must be persisted so the next launch does not re-downgrade.
	disk, ok, err := LoadCompactionState(path)
	if err != nil || !ok {
		t.Fatalf("sidecar should remain on disk: ok=%v err=%v", ok, err)
	}
	if disk.PromptCacheKey != wantKey {
		t.Fatalf("persisted PromptCacheKey = %q, want %q", disk.PromptCacheKey, wantKey)
	}
}

func TestLoadProjectionSidecarDropsForeignCacheKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "sys"}}
	// Content validation must fail despite a model-only key change: lineage
	// rebinding cannot resurrect a projection whose canonical prefix differs.
	foreign := []provider.Message{{Role: provider.RoleSystem, Content: "sys-old"}}
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:  compactionStateSchemaV1,
		PromptCacheKey: "ws|s|other-model",
		Projection: ContextProjection{
			Messages:          msgs,
			CoveredCount:      1,
			CoveredPrefixHash: coveredPrefixHash(foreign, 1),
		},
	}); err != nil {
		t.Fatal(err)
	}
	a := New(nil, nil, NewSession("sys"), Options{
		SessionPath: path,
		WorkspaceID: "ws",
		ModelRef:    "this-model",
	}, event.Discard)
	// New() already called LoadProjectionSidecar; mismatched content must drop
	// the projection body and keep the sidecar file for the other model.
	if len(a.sess.compactionState.Projection.Messages) != 0 {
		t.Fatalf("foreign projection loaded: %+v", a.sess.compactionState.Projection)
	}
	if _, ok, err := LoadCompactionState(path); err != nil || !ok {
		t.Fatalf("sidecar should remain on disk: ok=%v err=%v", ok, err)
	}
}

func TestForceThresholdNoopReturnsCompactionRequired(t *testing.T) {
	// Huge tool result is entirely in the recent tail → no fold region, but
	// estimate exceeds force; preflight must refuse (not mid-turn).
	huge := strings.Repeat("word ", 5000)
	sess := &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "read", Arguments: "{}"}}},
		{Role: provider.RoleTool, ToolCallID: "1", Name: "read", Content: huge},
	}}
	a := New(&fakeProvider{reply: "unused"}, tool.NewRegistry(), sess, Options{
		ContextWindow:     200,
		CompactRatio:      0.5,
		CompactForceRatio: 0.6,
		RecentKeep:        2,
	}, event.Discard)

	_, err := a.contextManager().Prepare(context.Background(), ContextPreparePolicy{Trigger: CompactionTriggerPressure})
	if err == nil {
		t.Fatal("expected ErrCompactionRequired when force threshold has no fold region")
	}
	if !errors.Is(err, ErrCompactionRequired) {
		t.Fatalf("err = %v, want ErrCompactionRequired", err)
	}
}

func TestSummarizeOnceDoesNotRetry(t *testing.T) {
	fp := &retryUsageProvider{
		failOnce: errors.New("transient"),
		reply:    "digest body",
		usage1:   &provider.Usage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12, RequestCount: 1},
		usage2:   &provider.Usage{PromptTokens: 11, CompletionTokens: 3, TotalTokens: 14, RequestCount: 1},
	}
	a := New(fp, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	_, _, err := a.summarizeOnce(context.Background(), nil, []provider.Message{
		{Role: provider.RoleUser, Content: "fold me"},
	}, "")
	if err == nil {
		t.Fatal("expected first-attempt failure to surface without retry")
	}
	if fp.calls != 1 {
		t.Fatalf("provider calls = %d, want exactly 1", fp.calls)
	}
}

// retryUsageProvider fails the first Stream, then returns reply + usage2.
type retryUsageProvider struct {
	calls    int
	failOnce error
	reply    string
	usage1   *provider.Usage
	usage2   *provider.Usage
}

func (p *retryUsageProvider) Name() string { return "retry-usage" }
func (p *retryUsageProvider) Stream(_ context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	p.calls++
	ch := make(chan provider.Chunk, 4)
	if p.calls == 1 && p.failOnce != nil {
		if p.usage1 != nil {
			ch <- provider.Chunk{Type: provider.ChunkUsage, Usage: p.usage1}
		}
		ch <- provider.Chunk{Type: provider.ChunkError, Err: p.failOnce}
		close(ch)
		return ch, nil
	}
	ch <- provider.Chunk{Type: provider.ChunkText, Text: p.reply}
	if p.usage2 != nil {
		ch <- provider.Chunk{Type: provider.ChunkUsage, Usage: p.usage2}
	}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func TestCompactInstallsCoveredPrefixHash(t *testing.T) {
	fp := &fakeProvider{reply: "digest"}
	sess := NewSession("sys")
	for range 8 {
		sess.Add(provider.Message{Role: provider.RoleUser, Content: strings.Repeat("u", 80)})
		sess.Add(provider.Message{Role: provider.RoleAssistant, Content: strings.Repeat("a", 120)})
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	a := New(fp, tool.NewRegistry(), sess, Options{
		ContextWindow: 2000,
		RecentKeep:    2,
		ArchiveDir:    dir,
		SessionPath:   path,
		WorkspaceID:   "ws",
		ModelRef:      "m",
	}, event.Discard)
	if err := a.CompactNow(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	st := a.sess.compactionState
	if st.Projection.CoveredPrefixHash == "" {
		t.Fatal("CoveredPrefixHash not set")
	}
	if st.PromptCacheKey != promptCacheKey("ws", BranchID(path), "m") {
		t.Fatalf("PromptCacheKey = %q", st.PromptCacheKey)
	}
	msgs, _ := sess.snapshotMessagesVersion()
	if !projectionValid(st, msgs, st.PromptCacheKey) {
		t.Fatal("fresh projection should validate")
	}
}

func TestLoadProjectionSidecarDegradedKeepsBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	orig := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task-v1"},
		{Role: provider.RoleAssistant, Content: "done"},
		{Role: provider.RoleUser, Content: "next"},
	}
	// Sidecar records the unedited prefix hash. NonToolContentHash marks it
	// as a trustworthy compaction artifact (full metadata).
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:     compactionStateSchemaV1,
		PromptCacheKey:    promptCacheKey("ws", BranchID(path), "model"),
		TranscriptVersion: 1,
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "sys summary"},
				{Role: provider.RoleUser, Content: "task summary"},
			},
			CoveredCount:       3,
			CoveredPrefixHash:  coveredPrefixHash(orig, 3),
			NonToolContentHash: nonToolContentHash(orig, 3),
		},
	}); err != nil {
		t.Fatal(err)
	}
	// Conversation has an edited covered prefix (hash mismatch, same lineage).
	edited := append([]provider.Message(nil), orig...)
	edited[1].Content = "task-EDITED"
	sess := NewSession("sys")
	sess.Add(edited[1])
	sess.Add(edited[2])
	sess.Add(edited[3])
	a := New(nil, nil, sess, Options{
		SessionPath: path,
		WorkspaceID: "ws",
		ModelRef:    "model",
	}, event.Discard)
	// Same-lineage hash mismatch must keep the projection body for the
	// degraded tail-only send (12:49 case) — not drop it and replay the full
	// disk transcript.
	if len(a.sess.compactionState.Projection.Messages) == 0 {
		t.Fatal("same-lineage degraded projection body was dropped")
	}
	if a.sess.checkpointState != "none" {
		t.Fatalf("checkpointState = %q, want none (degraded, not restored)", a.sess.checkpointState)
	}
	// modelVisibleMessages must send projection+tail, not the full transcript.
	visible := a.modelVisibleMessages()
	if len(visible) != 3 {
		t.Fatalf("model-visible = %d messages, want 3 (2 projection + 1 tail)", len(visible))
	}
	if visible[0].Content != "sys summary" {
		t.Fatalf("visible[0] = %q, want projection summary head", visible[0].Content)
	}
	if visible[len(visible)-1].Content != "next" {
		t.Fatalf("visible tail = %q, want latest message", visible[len(visible)-1].Content)
	}
}

func TestModelVisibleDegradedCoveredExceedsTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "s.jsonl")
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task"},
	}
	// Sidecar claims to cover 5 messages while the disk transcript has only 2
	// (post-compaction transcript shrink). The projection body still covers
	// the folded history — send it alone rather than replaying the transcript.
	if err := SaveCompactionState(path, CompactionState{
		SchemaVersion:     compactionStateSchemaV1,
		PromptCacheKey:    promptCacheKey("ws", BranchID(path), "model"),
		TranscriptVersion: 1,
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "folded summary"},
			},
			CoveredCount:      5,
			CoveredPrefixHash: "stale-hash",
		},
	}); err != nil {
		t.Fatal(err)
	}
	sess := NewSession(msgs[0].Content)
	sess.Add(msgs[1])
	a := New(nil, nil, sess, Options{
		SessionPath: path,
		WorkspaceID: "ws",
		ModelRef:    "model",
	}, event.Discard)
	if len(a.sess.compactionState.Projection.Messages) == 0 {
		t.Fatal("covered-exceeds projection body was dropped")
	}
	visible := a.modelVisibleMessages()
	if len(visible) != 1 {
		t.Fatalf("model-visible = %d messages, want 1 (projection body only)", len(visible))
	}
	if visible[0].Content != "folded summary" {
		t.Fatalf("visible[0] = %q, want folded summary", visible[0].Content)
	}
}

func TestVisibleInputForFoldMatchesSamplingViewOnProjectionLoss(t *testing.T) {
	// Projection loss with a usable body must degrade both sampling and compaction
	// to the same projection+tail view, or summary requests lose the prefix bytes (2026-08-30: 3.6% hit).
	canonical := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "task-v1"},
		{Role: provider.RoleAssistant, Content: "done"},
		{Role: provider.RoleUser, Content: "mid-1"},
		{Role: provider.RoleAssistant, Content: "mid-2"},
		{Role: provider.RoleUser, Content: "tail"},
	}
	st := CompactionState{
		TranscriptVersion: 1,
		PromptCacheKey:    "ws|sess|model",
		Projection: ContextProjection{
			Messages: []provider.Message{
				{Role: provider.RoleSystem, Content: "sys"},
				{Role: provider.RoleUser, Content: "SUMMARY"},
			},
			TranscriptVersion:  1,
			CoveredCount:       3,
			CoveredPrefixHash:  "stale-hash", // fingerprint mismatch → invalid
			NonToolContentHash: "usable-body",
		},
	}
	a := &Agent{}
	foldView, onProjection := a.visibleInputForFold(st, canonical, 1)
	if !onProjection {
		t.Fatal("projection loss with usable body must degrade to projection+tail, not canonical")
	}
	want := append([]provider.Message{}, st.Projection.Messages...)
	want = append(want, canonical[st.Projection.CoveredCount:]...)
	if len(foldView) != len(want) {
		t.Fatalf("fold view = %d messages, want %d", len(foldView), len(want))
	}
	for i := range want {
		if foldView[i].Role != want[i].Role || foldView[i].Content != want[i].Content {
			t.Fatalf("fold view[%d] = (%s %q), want (%s %q)", i, foldView[i].Role, foldView[i].Content, want[i].Role, want[i].Content)
		}
	}
	// Sampling and compaction must resolve the identical view.
	samplingView, _ := a.visibleMessagesWithFlag(st, canonical, "ws|sess|model")
	if len(samplingView) != len(foldView) {
		t.Fatalf("sampling view = %d messages, fold view = %d", len(samplingView), len(foldView))
	}
	for i := range foldView {
		if samplingView[i].Role != foldView[i].Role || samplingView[i].Content != foldView[i].Content {
			t.Fatalf("view divergence at %d: sampling (%s %q) vs fold (%s %q)", i, samplingView[i].Role, samplingView[i].Content, foldView[i].Role, foldView[i].Content)
		}
	}
}
