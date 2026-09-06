package agent

import (
	"path/filepath"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// TestMaybePersistFreshMainRequestThrottled pins the fresh-wire sidecar
// refresh: the first main request persists the frozen bytes immediately, a
// refresh within the throttle interval is skipped, and a refresh after the
// interval rewrites the sidecar. Without this, a long-lived session resumes
// with a stale frozen prefix whose server-side cache was already evicted
// (2026-08-31 23:01: 0% hit after 15h of activity).
func TestMaybePersistFreshMainRequestThrottled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	a := New(&countingProvider{reply: "digest"}, nil, &Session{}, Options{
		ContextWindow: 200000,
		SessionPath:   path,
	}, event.Discard)

	first := []provider.Message{
		{Role: provider.RoleSystem, Content: "system"},
		{Role: provider.RoleUser, Content: "hello one"},
	}
	a.saveMainRequest(first, nil)

	st, ok, err := LoadCompactionState(path)
	if err != nil || !ok {
		t.Fatalf("sidecar not written on first save: ok=%v err=%v", ok, err)
	}
	if len(st.LastWireMessages) != 2 || st.LastWireMessages[1].Content != "hello one" {
		t.Fatalf("sidecar missing first wire: %d messages, last=%q", len(st.LastWireMessages), st.LastWireMessages[len(st.LastWireMessages)-1].Content)
	}

	// Within the throttle interval the sidecar must not be rewritten.
	second := []provider.Message{
		{Role: provider.RoleSystem, Content: "system"},
		{Role: provider.RoleUser, Content: "hello two"},
	}
	a.saveMainRequest(second, nil)
	st, _, _ = LoadCompactionState(path)
	if st.LastWireMessages[len(st.LastWireMessages)-1].Content != "hello one" {
		t.Fatalf("throttle failed: sidecar refreshed early to %q", st.LastWireMessages[len(st.LastWireMessages)-1].Content)
	}

	// Past the interval the refresh lands.
	a.sess.lastMainReqPersist.Store(0)
	a.saveMainRequest(second, nil)
	st, _, _ = LoadCompactionState(path)
	if st.LastWireMessages[len(st.LastWireMessages)-1].Content != "hello two" {
		t.Fatalf("interval refresh did not rewrite sidecar: %q", st.LastWireMessages[len(st.LastWireMessages)-1].Content)
	}
}

// TestSummaryToolsSourceTelemetry pins the tools_source attribution used by
// compaction telemetry: a frozen main-request tool set is reported as
// "frozen", an empty registry as "none", and the fingerprint is non-empty for
// any real set so a system-only summary hit can be pinned to the tool seam.
func TestSummaryToolsSourceTelemetry(t *testing.T) {
	a := newFoldAgent(t, 200000, &countingProvider{reply: "digest"})
	msgs := []provider.Message{{Role: provider.RoleSystem, Content: "system"}}

	// No frozen bytes and no live registry -> none.
	if schemas, source := a.summaryToolsSource(); source != "none" || len(schemas) != 0 {
		t.Fatalf("empty registry: source=%s schemas=%d, want none/0", source, len(schemas))
	}

	// Frozen main-request tool set -> frozen, with a fingerprint.
	tools := []provider.ToolSchema{{Name: "bash", Description: "run commands"}, {Name: "read_file"}}
	a.saveMainRequest(msgs, tools)
	schemas, source := a.summaryToolsSource()
	if source != "frozen" || len(schemas) != 2 {
		t.Fatalf("frozen set: source=%s schemas=%d, want frozen/2", source, len(schemas))
	}
	if fp := toolsFingerprint(schemas); len(fp) != 32 {
		t.Fatalf("tools fingerprint = %q, want 32 hex chars", fp)
	}
	if toolsFingerprint(nil) != "" {
		t.Fatal("empty tool set must have an empty fingerprint")
	}
}
