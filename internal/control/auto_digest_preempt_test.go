package control

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// preemptProvider replies immediately unless block is set (the digest round's
// preemption window): a blocked Stream holds the turn slot open until its ctx
// is cancelled, so the test can verify a user submission preempts it.
type preemptProvider struct {
	started chan struct{}
	block   atomic.Bool
	once    sync.Once
	calls   atomic.Int32
}

func (p *preemptProvider) Name() string { return "preempt" }

func (p *preemptProvider) callCount() int { return int(p.calls.Load()) }

func (p *preemptProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	p.calls.Add(1)
	if p.block.Load() {
		p.once.Do(func() { close(p.started) })
		<-ctx.Done()
	}
	ch := make(chan provider.Chunk, 2)
	if !p.block.Load() {
		ch <- provider.Chunk{Type: provider.ChunkText, Text: "ok."}
	}
	ch <- provider.Chunk{Type: provider.ChunkDone}
	close(ch)
	return ch, nil
}

func TestUserTurnPreemptsDigestRound(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	systemPrompt := "You are a terse coding agent."
	sink, _, events := collectSink()
	pv := &preemptProvider{started: make(chan struct{})}
	executor := agent.New(pv, tool.NewRegistry(), agent.NewSession(systemPrompt), agent.Options{Temperature: 0}, sink)
	jm := jobs.NewManager(sink)
	defer jm.Close()
	roReg := tool.NewRegistry()
	roReg.Add(fakeControlTool{})
	task := agent.NewTaskTool(pv, nil, roReg, 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(agent.NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)
	ts.SetSink(sink)
	sessionDir := t.TempDir()
	c := New(Options{
		Executor:     executor,
		Runner:       executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: systemPrompt,
		SessionDir:   sessionDir,
		SessionPath:  sessionDir + "/session.jsonl",
	})
	defer c.Close()

	if err := c.RunTurn(context.Background(), "warmup"); err != nil {
		t.Fatalf("warmup turn failed: %v", err)
	}
	time.Sleep(300 * time.Millisecond)

	pv.block.Store(true)
	digestDone := make(chan struct{})
	go func() {
		c.runDigestRound("worker-preempt")
		close(digestDone)
	}()
	select {
	case <-pv.started:
	case <-digestDone:
		var notices []string
		for _, e := range *events {
			if e.Kind == event.Notice {
				notices = append(notices, e.Text)
			}
		}
		t.Fatalf("digest round exited before reaching provider; notices=%q", notices)
	case <-time.After(5 * time.Second):
		t.Fatal("digest round neither reached provider nor exited")
	}

	// A user submission must preempt the digest, not fail: Submit is the real
	// user entry (submitCommandOrTurnReady → cancelRunningDigest).
	c.Submit("user message")
	select {
	case <-digestDone:
	case <-time.After(5 * time.Second):
		t.Fatal("digest round not released by user preemption")
	}
	// The preempting user turn must then run to completion (provider call 2).
	deadline := time.Now().Add(5 * time.Second)
	for {
		if pv.callCount() >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("user turn after preemption never ran")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
