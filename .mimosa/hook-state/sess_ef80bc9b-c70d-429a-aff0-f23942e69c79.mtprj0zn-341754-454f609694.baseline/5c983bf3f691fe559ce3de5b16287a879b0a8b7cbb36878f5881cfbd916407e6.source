package agent

import (
	"fmt"
	"reasonix/internal/event"
	"reasonix/internal/tool"
	"strings"
	"testing"
)

func TestDebugFold2(t *testing.T) {
	const window = 20_000
	reg := tool.NewRegistry()
	reg.Add(largeSchemaTool{description: strings.Repeat("large deterministic schema. ", 1200)})
	sess := foldableSessionOverForce(20)
	a := New(&overflowSummaryProvider{}, reg, sess, Options{
		ContextWindow:   window,
		CompactRatio:    defaultCompactRatio,
		MaxOutputTokens: 8192,
	}, event.Discard)
	msgs := a.modelVisibleMessages()
	head, start, ok := a.planCompaction(msgs, 2, false)
	if !ok {
		t.Fatal("no compaction")
	}
	est1 := a.estimatedRequestTokens(a.summaryFoldEstimate(msgs, head, start, ""))
	est2 := a.estimatedRequestTokens(a.summaryRequest(msgs[head:start], nil, ""))
	est3 := a.estimatedRequestTokens(a.summaryRequest(msgs, nil, ""))
	maxSafe := a.summaryMaxPromptTokens()
	safeEnd := a.maximumSafeSummaryPrefixEnd(msgs, head, start, "")
	fmt.Printf("fold=%d safeEnd=%d\n", start-head, safeEnd-head)
	fmt.Printf("estFold=%d estFoldOnly=%d estAllView=%d maxSafe=%d\n", est1, est2, est3, maxSafe)
}
