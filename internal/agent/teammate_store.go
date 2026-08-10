package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"reasonix/internal/jobs"
)

// Teammate is one member of a P6 team: a persistent identity whose work is one
// background job per assignment. The first assignment forks the leader's
// prefix (cache hit on first request); later assignments continue the same
// transcript (prefix-stable). Results ride the P1 envelope back to the leader.
type Teammate struct {
	Name      string
	Role      string
	Model     string // pinned at create; Continue's validateMeta enforces it
	Effort    string // pinned at create; must stay byte-stable across rounds
	ToolSet   []string
	Ref       string // transcript ref after the first assignment
	LastJobID string
	State     TeammateState
}

// TeammateState is the assignment lifecycle of a teammate.
type TeammateState string

const (
	TeammateIdle    TeammateState = "idle"
	TeammateRunning TeammateState = "running"
)

// TeammateStore owns the team registry (in-memory; persistence is P6.1). It
// routes assignments through the TaskTool so fork/continue/steer/envelope all
// inherit the P1-P5 machinery unchanged.
type TeammateStore struct {
	mu        sync.Mutex
	teammates map[string]*Teammate
	task      *TaskTool
	jm        *jobs.Manager
}

// NewTeammateStore wires a registry to the task tool that executes assignments.
func NewTeammateStore(task *TaskTool, jm *jobs.Manager) *TeammateStore {
	return &TeammateStore{
		teammates: make(map[string]*Teammate),
		task:      task,
		jm:        jm,
	}
}

// Create registers a teammate identity. Duplicate names are rejected.
func (ts *TeammateStore) Create(name, role string) error {
	name = strings.TrimSpace(name)
	role = strings.TrimSpace(role)
	if name == "" {
		return fmt.Errorf("teammate name is required")
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if _, ok := ts.teammates[name]; ok {
		return fmt.Errorf("teammate %q already exists", name)
	}
	ts.teammates[name] = &Teammate{
		Name:    name,
		Role:    role,
		State:   TeammateIdle,
		ToolSet: []string{},
	}
	return nil
}

// List returns teammates sorted by name (deterministic for /team-status).
// Running members whose last job reached a terminal state are flipped to idle
// (lazy sync — no completion callback needed for the MVP).
func (ts *TeammateStore) List() []Teammate {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]Teammate, 0, len(ts.teammates))
	for _, tm := range ts.teammates {
		ts.syncStateLocked(tm)
		out = append(out, *tm)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// syncStateLocked flips a running teammate to idle once its last job is done.
func (ts *TeammateStore) syncStateLocked(tm *Teammate) {
	if tm.State != TeammateRunning || tm.LastJobID == "" || ts.jm == nil {
		return
	}
	if _, st, ok := ts.jm.Output(tm.LastJobID); ok && st != jobs.Running {
		tm.State = TeammateIdle
	}
}

// Status returns a teammate by name.
func (ts *TeammateStore) Status(name string) (Teammate, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tm, ok := ts.teammates[name]
	if !ok {
		return Teammate{}, false
	}
	return *tm, true
}

// Assign dispatches one job to a teammate. The first assignment forks the
// leader's prefix (non-silent: the result rides the P1 envelope back);
// later ones continue the teammate's own transcript. Running teammates are
// rejected (steer them mid-run with send_message instead).
func (ts *TeammateStore) Assign(ctx context.Context, name, prompt string) (string, error) {
	name = strings.TrimSpace(name)
	ts.mu.Lock()
	tm, ok := ts.teammates[name]
	if !ok {
		ts.mu.Unlock()
		return "", fmt.Errorf("unknown teammate %q (create it first with /team-create)", name)
	}
	if tm.State == TeammateRunning {
		ts.mu.Unlock()
		return "", fmt.Errorf("teammate %q is running; steer it with send_message or wait for the job to finish", name)
	}
	tm.State = TeammateRunning
	ref, toolset := tm.Ref, append([]string(nil), tm.ToolSet...)
	ts.mu.Unlock()

	spec := ProfileExecSpec{
		Task:   TaskSpec{Objective: prompt, Description: "teammate: " + name},
		Worker: WorkerSpec{Kind: "task", Name: "task", SystemPrompt: ts.task.sysPrompt},
		Grant:  CapabilityGrant{CallTools: toolset},
		Sched:  SchedulerPolicy{MaxSteps: 0, RunInBackground: true, Nested: SubagentDepth(ctx) > 0},
	}
	if ref == "" {
		// First assignment: fork the leader prefix, non-silent (envelope back).
		spec.Context = ContextRequest{Fork: true, Silent: false}
	} else {
		// Later assignments: continue the same transcript (prefix-stable).
		spec.Context = ContextRequest{ContinueFrom: ref}
	}
	out, err := ts.task.RunProfileSpec(ctx, spec)
	if err != nil {
		ts.mu.Lock()
		tm.State = TeammateIdle
		ts.mu.Unlock()
		return "", err
	}

	jobID := teammateJobID(out)
	ts.mu.Lock()
	tm.LastJobID = jobID
	tm.Ref = ref
	if jobID == "" {
		tm.State = TeammateIdle
	}
	ts.mu.Unlock()
	return jobID, nil
}

// teammateJobID extracts the quoted job id from a "Started background task
// \"id\" (...)" run result (the same shape extractJobID parses in tests).
func teammateJobID(out string) string {
	q := strings.Index(out, `"`)
	if q < 0 {
		return ""
	}
	end := strings.Index(out[q+1:], `"`)
	if end < 0 {
		return ""
	}
	return out[q+1 : q+1+end]
}

// Complete marks a teammate idle after its job reaches a terminal state. The
// job's P1 envelope has already delivered the result to the leader.
func (ts *TeammateStore) Complete(name, jobID, ref string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tm, ok := ts.teammates[name]
	if !ok {
		return
	}
	if tm.LastJobID != "" && tm.LastJobID != jobID {
		return // a newer assignment owns the slot
	}
	tm.State = TeammateIdle
	if ref != "" {
		tm.Ref = ref
	}
}

// Remove kills any running job and drops the member. The transcript is left
// in place (session destroy cleans it via DeleteSubagentsByParent); a re-create
// with the same name starts fresh.
func (ts *TeammateStore) Remove(name string) error {
	ts.mu.Lock()
	tm, ok := ts.teammates[name]
	if !ok {
		ts.mu.Unlock()
		return fmt.Errorf("unknown teammate %q", name)
	}
	jobID := tm.LastJobID
	delete(ts.teammates, name)
	ts.mu.Unlock()

	if jobID != "" && ts.jm != nil {
		ts.jm.Kill(jobID)
	}
	return nil
}

// DestroyAll tears down every teammate (session close path).
func (ts *TeammateStore) DestroyAll() {
	ts.mu.Lock()
	names := make([]string, 0, len(ts.teammates))
	for name := range ts.teammates {
		names = append(names, name)
	}
	ts.mu.Unlock()
	for _, name := range names {
		_ = ts.Remove(name)
	}
}
