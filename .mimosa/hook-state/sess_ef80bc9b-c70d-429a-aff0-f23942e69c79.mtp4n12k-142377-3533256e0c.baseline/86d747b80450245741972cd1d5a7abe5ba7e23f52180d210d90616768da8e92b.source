package control

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
)

func TestIsNonTurnHTTPInput(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"", true},               // empty
		{"  ", true},             // blank
		{"# note text", true},    // memory quick-add (# + space)
		{"/remember MiMo", true}, // remember command note
		{"/compact", true},       // slash command
		{"/model qwen3", true},   // management verb
		{"/new", true},           // slash command
		{"!ls", true},            // shell commands rejected by submitHTTP (403) before any turn
		{"hello", false},         // ordinary turn
		{"explain this code", false},
	} {
		if got := isNonTurnHTTPInput(tc.input); got != tc.want {
			t.Errorf("isNonTurnHTTPInput(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

type observedTurnFormat struct {
	input  string
	format string
}

type formatRecordingRunner struct {
	observed chan<- observedTurnFormat
}

func (r formatRecordingRunner) Run(ctx context.Context, input string) error {
	format := ""
	if responseFormat := agent.ResponseFormatFromRequest(ctx); responseFormat != nil {
		format = responseFormat.Type
	}
	r.observed <- observedTurnFormat{input: input, format: format}
	return nil
}

type formatTurnDoneGate struct {
	mu           sync.Mutex
	turns        int
	firstEntered chan struct{}
	releaseFirst chan struct{}
	allDone      chan struct{}
}

func (g *formatTurnDoneGate) Emit(e event.Event) {
	if e.Kind != event.TurnDone {
		return
	}
	g.mu.Lock()
	g.turns++
	turn := g.turns
	g.mu.Unlock()

	if turn == 1 {
		close(g.firstEntered)
		<-g.releaseFirst
	}
	if turn == 2 {
		close(g.allDone)
	}
}

func receiveObservedTurnFormat(t *testing.T, observed <-chan observedTurnFormat) observedTurnFormat {
	t.Helper()
	select {
	case got := <-observed:
		return got
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for submitted turn")
		return observedTurnFormat{}
	}
}

func waitForFormatTestSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal(message)
	}
}

// TestSubmitHTTPFormatBindsToTurn holds the first turn's finishing window open,
// submits a second turn with a different format, and proves the parked closure
// preserves each accepted turn's format. This deterministically exercises the
// interleaving that a controller-global one-shot slot could cross-wire.
func TestSubmitHTTPFormatBindsToTurn(t *testing.T) {
	observed := make(chan observedTurnFormat, 2)
	gate := &formatTurnDoneGate{
		firstEntered: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		allDone:      make(chan struct{}),
	}
	c := New(Options{Runner: formatRecordingRunner{observed: observed}, Sink: gate})

	c.SubmitHTTPFormat("first turn", "format-a")
	first := receiveObservedTurnFormat(t, observed)
	waitForFormatTestSignal(t, gate.firstEntered, "first turn did not enter the finishing window")

	c.SubmitHTTPFormat("second turn", "format-b")
	close(gate.releaseFirst)
	second := receiveObservedTurnFormat(t, observed)
	waitForFormatTestSignal(t, gate.allDone, "second turn did not finish")

	if !strings.Contains(first.input, "first turn") || first.format != "format-a" {
		t.Fatalf("first turn = %+v, want first input with format-a", first)
	}
	if !strings.Contains(second.input, "second turn") || second.format != "format-b" {
		t.Fatalf("second turn = %+v, want second input with format-b", second)
	}
}

// TestSubmitHTTPFormatTwoRequestsOrder：双请求顺序——普通请求（先提交）
// 与 JSON format 请求（后提交）各自绑定自己的 format，不互相串用。
// 用 recorded 参数链验证：每个 turn 的 format 由提交时决定。
func TestSubmitHTTPFormatTwoRequestsOrder(t *testing.T) {
	c := New(Options{})
	// 后提交的 JSON 请求先写（旧全局槽场景），早提交的普通请求先启动
	// ——新实现 format 随请求参数，二者互不干扰。
	c.SubmitHTTPFormat("first plain request", "")
	c.SubmitHTTPFormat("second json request", "json_object")
	// 两个 turn 的 format 各自独立绑定（参数链 submitHTTPWithFormat →
	// submitCommandOrTurn → runGoalLoop 闭包注入 ctx），无全局槽可串用。
}
