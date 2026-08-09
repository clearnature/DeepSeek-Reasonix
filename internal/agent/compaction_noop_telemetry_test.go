package agent

import (
	"context"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
)

type noticeCaptureSink struct {
	notices []string
}

func (s *noticeCaptureSink) Emit(e event.Event) {
	if e.Kind == event.Notice {
		s.notices = append(s.notices, e.Detail)
	}
}

// TestCompactionNoopEmitsTelemetry pins the 2026-08-09 blind spot: a manual
// compact whose fold region is empty (planFold !ok) returned CompactionNoop
// with zero records, so /compact reported "compacted" while stats showed
// nothing. The deferred emit must land one status=noop row per silent exit.
func TestCompactionNoopEmitsTelemetry(t *testing.T) {
	cap := &capturingBudgetProvider{}
	cap.fakeProvider = &fakeProvider{reply: "SUMMARY"}
	cap.budget = 128 * 1024
	sink := &noticeCaptureSink{}
	a := &Agent{
		prov:              cap,
		contextWindow:     1_000_000,
		outputBudget:      128 * 1024,
		compactRatio:      0.8,
		compactForceRatio: 0.9,
		sink:              sink,
	}
	// A session so small the fold region is empty: everything fits in head+tail.
	a.session = &Session{Messages: []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "hi"},
	}}
	outcome, err := a.compactToProjection(context.Background(), "manual", "", true)
	if err != nil {
		t.Fatalf("compactToProjection: %v", err)
	}
	if outcome != CompactionNoop {
		t.Fatalf("outcome=%v, want Noop for tiny session", outcome)
	}
	found := false
	for _, d := range sink.notices {
		if strings.Contains(d, "status=noop") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no status=noop telemetry emitted; notices: %v", sink.notices)
	}
}
