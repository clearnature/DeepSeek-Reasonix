package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// signal primitive

func TestBackgroundizeSignalRequestIdempotentOnce(t *testing.T) {
	sig := NewBackgroundizeSignal()
	if !sig.Request() {
		t.Fatal("first Request must report first=true")
	}
	if sig.Request() {
		t.Fatal("second Request must be an idempotent no-op")
	}
	if sig.Request() {
		t.Fatal("third Request must be an idempotent no-op")
	}
	if !sig.Requested() {
		t.Fatal("Requested must stay true after a request")
	}
	if NewBackgroundizeSignal().Requested() {
		t.Fatal("a fresh signal must not be requested")
	}
}

func TestBackgroundizeSignalFromContextNilSafe(t *testing.T) {
	ctx := context.Background()
	if sig := BackgroundizeSignalFromContext(ctx); sig != nil {
		t.Fatalf("no signal injected, got %p", sig)
	}
	sig := NewBackgroundizeSignal()
	withSig := WithBackgroundizeSignal(ctx, sig)
	if got := BackgroundizeSignalFromContext(withSig); got != sig {
		t.Fatalf("injected signal = %p, want %p", got, sig)
	}
	if got := BackgroundizeSignalFromContext(WithoutBackgroundizeSignal(withSig)); got != nil {
		t.Fatalf("cleared signal = %p, want nil", got)
	}
	var nilSig *BackgroundizeSignal
	if nilSig.Request() || nilSig.Requested() {
		t.Fatal("a nil signal must be inert")
	}
}

// iteration-boundary checkpoint

// TestBackgroundizeCheckpointReturnsSentinelBeforeSampling: a requested signal
// makes runToolLoop return the sentinel at the first iteration boundary
// without sampling the provider; the turn prompt is already committed.
func TestBackgroundizeCheckpointReturnsSentinelBeforeSampling(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "should not run"},
		{Type: provider.ChunkDone},
	}}
	sess := NewSession("sys")
	ag := New(sub, tool.NewRegistry(), sess, Options{}, event.Discard)
	sig := NewBackgroundizeSignal()
	sig.Request()
	err := ag.Run(WithBackgroundizeSignal(context.Background(), sig), "foreground task")
	if !errors.Is(err, errBackgroundizeRequested) {
		t.Fatalf("Run err = %v, want errBackgroundizeRequested", err)
	}
	if n := len(sub.requests); n != 0 {
		t.Fatalf("provider called %d times, want 0 (checkpoint fires before sampling)", n)
	}
	// beginRunTurn committed the user turn; no provider round ran yet.
	if want := 2; len(sess.Messages) != want {
		t.Fatalf("session has %d messages, want %d (system + prompt)", len(sess.Messages), want)
	}
}

func TestBackgroundizeCheckpointNoopWithoutSignal(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "done"},
		{Type: provider.ChunkDone},
	}}
	ag := New(sub, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	if err := ag.Run(context.Background(), "plain task"); err != nil {
		t.Fatalf("Run without signal: %v", err)
	}
	if n := len(sub.requests); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
}

// resume mode

// TestResumeRunSkipsPromptAdd: a resume run continues an in-memory session and
// must not re-append the task prompt as a fresh user turn.
func TestResumeRunSkipsPromptAdd(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "resumed answer"},
		{Type: provider.ChunkDone},
	}}
	sess := NewSession("sys")
	ag := New(sub, tool.NewRegistry(), sess, Options{}, event.Discard)
	if err := ag.Run(WithResumeSession(context.Background()), "already-prompted task"); err != nil {
		t.Fatalf("resume Run: %v", err)
	}
	if n := len(sub.requests); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	for _, m := range sess.Messages {
		if m.Role == provider.RoleUser {
			t.Fatalf("resume run appended a user turn: %+v", m)
		}
	}
	if len(sess.Messages) == 0 || sess.Messages[0].Role != provider.RoleSystem {
		t.Fatalf("resume session must start from the original system message, got %d messages", len(sess.Messages))
	}
}

// TestBackgroundizeCheckpointNoopOnResumeRun: a resume run must never
// re-trigger the sentinel even when the (already consumed) signal is still
// requested — the handoff already happened.
func TestBackgroundizeCheckpointNoopOnResumeRun(t *testing.T) {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "resumed answer"},
		{Type: provider.ChunkDone},
	}}
	ag := New(sub, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	sig := NewBackgroundizeSignal()
	sig.Request()
	ctx := WithBackgroundizeSignal(WithResumeSession(context.Background()), sig)
	if err := ag.Run(ctx, "resumed task"); err != nil {
		t.Fatalf("resume Run with consumed signal: %v (want no sentinel)", err)
	}
	if n := len(sub.requests); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
}

// foreground→background handoff

// backgroundizeTriggerTool requests the signal the moment it executes, so a
// task that calls it backgrounds itself at the next iteration boundary.
type backgroundizeTriggerTool struct {
	sig *BackgroundizeSignal
}

func (t backgroundizeTriggerTool) Name() string        { return "trigger_backgroundize" }
func (t backgroundizeTriggerTool) Description() string { return "" }
func (t backgroundizeTriggerTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}
func (t backgroundizeTriggerTool) ReadOnly() bool { return true }
func (t backgroundizeTriggerTool) Execute(context.Context, json.RawMessage) (string, error) {
	t.sig.Request()
	return "backgroundize requested", nil
}

// handoffProvider is a thread-safe stand-in for mockProvider: the foreground
// sub-agent and the resumed background job stream concurrently (foreground run
// loop returning while the job starts), so plain mockProvider's unsynchronized
// request append would race under -race.
type handoffProvider struct {
	mu       sync.Mutex
	streams  [][]provider.Chunk
	calls    int
	requests []provider.Request
}

func (p *handoffProvider) Name() string { return "handoff" }

func (p *handoffProvider) Stream(_ context.Context, req provider.Request) (<-chan provider.Chunk, error) {
	p.mu.Lock()
	call := p.calls
	p.calls++
	p.requests = append(p.requests, req)
	chunks := p.streams[call]
	if call >= len(p.streams) {
		chunks = p.streams[len(p.streams)-1]
	}
	p.mu.Unlock()
	ch := make(chan provider.Chunk, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func (p *handoffProvider) requestSnapshot() []provider.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]provider.Request(nil), p.requests...)
}

// TestTaskToolForegroundToBackgroundHandoff runs a foreground task that asks
// to be backgrounded mid-run and verifies the serialized handoff: the same
// in-memory session continues as a job with exactly one copy of the task
// prompt (the resume run skips re-adding it) and the job delivers the resumed
// final answer.
func TestTaskToolForegroundToBackgroundHandoff(t *testing.T) {
	sig := NewBackgroundizeSignal()
	sub := &handoffProvider{streams: [][]provider.Chunk{
		{
			toolCallChunk("c1", "trigger_backgroundize", `{}`),
			{Type: provider.ChunkDone},
		},
		{
			{Type: provider.ChunkText, Text: "final answer after handoff"},
			{Type: provider.ChunkDone},
		},
	}}
	reg := tool.NewRegistry()
	reg.Add(backgroundizeTriggerTool{sig: sig})
	task := NewTaskTool(sub, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ctx := WithBackgroundizeSignal(testTaskContext(), sig)
	ctx = jobs.WithSession(ctx, "parent-session")
	ctx = jobs.WithManager(ctx, jm)

	out, err := task.Execute(ctx, []byte(`{"prompt":"do the work"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Started background task") {
		t.Fatalf("handoff output = %q, want 'Started background task'", out)
	}
	jobID := extractJobID(out)
	if jobID == "" {
		t.Fatalf("no background job id in output:\n%s", out)
	}
	res := jm.WaitForSession(context.Background(), "parent-session", []string{jobID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("handoff job result = %+v, want one done job", res)
	}
	if !strings.Contains(res[0].Output, "final answer after handoff") {
		t.Fatalf("job output = %q, want the resumed final answer", res[0].Output)
	}

	// The provider saw two requests: the foreground first round (tool call)
	// and the resumed first round. Both must carry exactly one user turn — the
	// resume must not re-append the task prompt, or the prefix would break.
	reqs := sub.requestSnapshot()
	if len(reqs) != 2 {
		t.Fatalf("provider saw %d requests, want 2 (foreground + resumed)", len(reqs))
	}
	if n := countUserMessages(reqs[0].Messages); n != 1 {
		t.Fatalf("foreground first request has %d user messages, want 1", n)
	}
	if n := countUserMessages(reqs[1].Messages); n != 1 {
		t.Fatalf("resumed first request has %d user messages, want 1 (no duplicated prompt)", n)
	}
	// The resumed request also carries the committed foreground round
	// (assistant tool call + tool result), proving the checkpoint returned
	// only after the current round was fully committed.
	if !hasMessageRole(reqs[1].Messages, provider.RoleTool) {
		t.Fatalf("resumed request lost the committed foreground tool round: %+v", reqs[1].Messages)
	}
}

func countUserMessages(msgs []provider.Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == provider.RoleUser {
			n++
		}
	}
	return n
}

func hasMessageRole(msgs []provider.Message, role provider.Role) bool {
	for _, m := range msgs {
		if m.Role == role {
			return true
		}
	}
	return false
}

// cancelProbeProvider returns a tool-call turn first, then blocks on ctx until
// it is cancelled — used to prove the parent-turn cancel propagates into the
// resumed job instead of being short-circuited by the sentinel handoff.
type cancelProbeProvider struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	once    sync.Once
}

func (p *cancelProbeProvider) Name() string { return "cancel-probe" }

func (p *cancelProbeProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	p.mu.Lock()
	call := p.calls
	p.calls++
	p.mu.Unlock()
	if call == 0 {
		ch := make(chan provider.Chunk, 2)
		ch <- toolCallChunk("c1", "trigger_backgroundize", `{}`)
		ch <- provider.Chunk{Type: provider.ChunkDone}
		close(ch)
		return ch, nil
	}
	p.once.Do(func() { close(p.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestBackgroundizeParentCancelPropagatesToHandoffJob: after the foreground
// task hands off to a background job, cancelling the parent turn must stop the
// resumed run — the sentinel branch must not short-circuit cancellation.
func TestBackgroundizeParentCancelPropagatesToHandoffJob(t *testing.T) {
	prov := &cancelProbeProvider{started: make(chan struct{})}
	sig := NewBackgroundizeSignal()
	reg := tool.NewRegistry()
	reg.Add(backgroundizeTriggerTool{sig: sig})
	task := NewTaskTool(prov, nil, reg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ctx, cancel := context.WithCancel(testTaskContext())
	defer cancel()
	ctx = WithBackgroundizeSignal(ctx, sig)
	ctx = jobs.WithSession(ctx, "parent-session")
	ctx = jobs.WithManager(ctx, jm)

	out, err := task.Execute(ctx, []byte(`{"prompt":"long running work"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	jobID := extractJobID(out)
	if jobID == "" {
		t.Fatalf("no background job id in output:\n%s", out)
	}
	select {
	case <-prov.started:
	case <-time.After(2 * time.Second):
		t.Fatal("resumed job never reached the provider")
	}
	// Cancel the parent turn mid-resume: the job must stop, not keep running.
	cancel()
	res := jm.WaitForSession(context.Background(), "parent-session", []string{jobID}, 5)
	if len(res) != 1 {
		t.Fatalf("handoff job result = %+v, want one terminal job", res)
	}
	if res[0].Status == jobs.Done {
		t.Fatalf("job finished Done despite the parent turn being cancelled; the cancel was short-circuited")
	}
}
