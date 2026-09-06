//go:build realapi

package control

import (
	"os"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/provider/openai"
	"reasonix/internal/tool"
)

// TestTeamChainRealDeepSeek is the P6 release verification against the real
// DeepSeek API: /team-create → /team-add → the teammate's fork job runs with
// the real provider → its result rides the P1 envelope back → the roster shows
// idle. Run with REASONIX_REAL_API=1 + DEEPSEEK_API_KEY.
func TestTeamChainRealDeepSeek(t *testing.T) {
	if os.Getenv("REASONIX_REAL_API") == "" {
		t.Skip("set REASONIX_REAL_API=1 to run against the real DeepSeek API")
	}
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Fatal("DEEPSEEK_API_KEY not set")
	}
	systemPrompt := "You are a terse coding agent. Answer in one short sentence."

	prov, err := openai.New(provider.Config{
		Name:    "deepseek",
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-v4-flash",
		APIKey:  key,
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	defer func() {
		if c, ok := prov.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}()

	sink, _, _ := collectSink()
	executor := agent.New(prov, tool.NewRegistry(), agent.NewSession(systemPrompt), agent.Options{Temperature: 0}, sink)
	jm := jobs.NewManager(sink)
	defer jm.Close()

	subProv, err := openai.New(provider.Config{
		Name:    "deepseek",
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-v4-flash",
		APIKey:  key,
	})
	if err != nil {
		t.Fatalf("subagent provider: %v", err)
	}
	defer func() {
		if c, ok := subProv.(interface{ Close() error }); ok {
			_ = c.Close()
		}
	}()
	task := agent.NewTaskTool(subProv, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", systemPrompt, nil, 0, "", "", nil).
		WithTranscripts(agent.NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	ts := agent.NewTeammateStore(task, jm, t.TempDir())
	t.Cleanup(ts.Close)
	ts.SetSink(sink)

	c := New(Options{
		Executor:     executor,
		Jobs:         jm,
		Teammates:    ts,
		Sink:         sink,
		SystemPrompt: systemPrompt,
		SessionDir:   t.TempDir(),
		SessionPath:  t.TempDir(),
	})

	// 1. /team-create alpha researcher
	c.Submit("/team-create alpha researcher")
	if _, ok := ts.Status("alpha"); !ok {
		t.Fatalf("alpha not registered after /team-create")
	}

	// 2. /team-add alpha <task> — starts a real fork job on the background
	c.Submit("/team-add alpha 简述前缀缓存如何降低推理成本")

	// 3. Wait for the teammate's fork job to reach a terminal state (real
	//    provider round-trips take a few seconds).
	deadline := time.Now().Add(90 * time.Second)
	st := agent.TeammateRunning
	for {
		// List lazily syncs a finished job back to idle.
		for _, tm := range ts.List() {
			if tm.Name == "alpha" {
				st = tm.State
			}
		}
		if st == agent.TeammateIdle {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("teammate alpha still %s after 90s", st)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 4. The P1 envelope carried the teammate's result back to the leader.
	note := jm.DrainCompletedNoteForSession(c.parentSessionID())
	if !strings.Contains(note, "<background-job-result") {
		t.Fatalf("no P1 envelope for teammate result; note=%q", note)
	}
	t.Logf("teammate result envelope received: %s", note)
}
