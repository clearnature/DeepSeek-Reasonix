package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
	// Writable opts this teammate's fork executions into a writable gate
	// (P6.1 enhancement 1). Default false keeps the P5 read-only fork.
	Writable bool
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
// TeamTask is one tracked assignment in the P6.1 dependency tree. Status is
// derived lazily from the underlying job; DependsOn gates the start of
// dependents (completion gate — a dependent assignment is refused until every
// job it depends on reaches a terminal state).
type TeamTask struct {
	ID        string
	Owner     string
	DependsOn []string
	Status    string
}

type TeammateStore struct {
	mu        sync.Mutex
	teammates map[string]*Teammate
	task      *TaskTool
	jm        *jobs.Manager
	// inboxRoot persists teammate mail on disk (P6.1 enhancement 2); empty
	// disables persistence (mail stays ephemeral in the P3 job queue).
	inboxRoot string
	// tasks is the dependency tree (jobID → task); completed gates live here.
	tasks map[string]*TeamTask
}

// NewTeammateStore wires a registry to the task tool that executes assignments.
// inboxRoot (optional) persists teammate mail on disk; empty keeps it ephemeral.
func NewTeammateStore(task *TaskTool, jm *jobs.Manager, inboxRoot ...string) *TeammateStore {
	ts := &TeammateStore{
		teammates: make(map[string]*Teammate),
		tasks:     make(map[string]*TeamTask),
		task:      task,
		jm:        jm,
	}
	if len(inboxRoot) > 0 {
		ts.inboxRoot = inboxRoot[0]
	}
	return ts
}

// Create registers a teammate identity. Duplicate names are rejected.
// Writable opts the member into a writable execution gate (P6.1); the default
// false keeps teammates read-only.
func (ts *TeammateStore) Create(name, role string, writable ...bool) error {
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
		Name:     name,
		Role:     role,
		State:    TeammateIdle,
		ToolSet:  []string{},
		Writable: len(writable) > 0 && writable[0],
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
// rejected (steer them mid-run with send_message instead). Optional dependsOn
// lists job ids that must be terminal before this assignment starts (P6.1
// completion gate / dependency tree).
func (ts *TeammateStore) Assign(ctx context.Context, name, prompt string, dependsOn ...string) (string, error) {
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
	// P6.1 completion gate checked before claiming the running slot, so a
	// refusal leaves the teammate idle for a later retry. Caller holds ts.mu.
	if len(dependsOn) > 0 {
		if reason := ts.pendingDependenciesLocked(dependsOn); reason != "" {
			ts.mu.Unlock()
			return "", fmt.Errorf("teammate %q blocked by unfinished dependency: %s", name, reason)
		}
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
		// Writable teammates get a writable execution gate (P6.1).
		spec.Context = ContextRequest{Fork: true, Silent: false, Writable: tm.Writable}
	} else {
		// Later assignments: continue the same transcript (prefix-stable).
		spec.Context = ContextRequest{ContinueFrom: ref, Writable: tm.Writable}
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
	if jobID != "" {
		// P6.1: replay persisted mail into the fresh job before its first turn.
		_ = ts.flushMailbox(name, jobs.SessionFromContext(ctx), jobID)
		ts.recordTask(jobID, name, dependsOn)
	}
	return jobID, nil
}

// recordTask adds the assignment to the dependency tree.
func (ts *TeammateStore) recordTask(jobID, owner string, dependsOn []string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tasks[jobID] = &TeamTask{ID: jobID, Owner: owner, DependsOn: append([]string(nil), dependsOn...)}
}

// pendingDependenciesLocked returns a non-empty reason naming the first
// dependency that is still running (or unknown), or "" when all are terminal.
// Caller must hold ts.mu.
func (ts *TeammateStore) pendingDependenciesLocked(dependsOn []string) string {
	for _, dep := range dependsOn {
		st := ""
		if tm, ok := ts.tasks[dep]; ok {
			st = tm.Status
		}
		if ts.jm != nil {
			if _, jobStatus, ok := ts.jm.Output(dep); ok {
				st = string(jobStatus)
			}
		}
		switch st {
		case "done", "failed", "killed", "interrupted", "cancelled":
			continue
		default:
			return dep + " (" + st + ")"
		}
	}
	return ""
}

// Tasks returns the tracked dependency tree, newest first, with derived status.
func (ts *TeammateStore) Tasks() []TeamTask {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]TeamTask, 0, len(ts.tasks))
	for id, t := range ts.tasks {
		st := ""
		if ts.jm != nil {
			if _, jobStatus, ok := ts.jm.Output(id); ok {
				st = string(jobStatus)
			}
		}
		if st == "" {
			st = t.Status
		}
		t.Status = st
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
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

// PostMail delivers a message to a teammate's persistent inbox (P6.1
// enhancement 2). It lands on disk when inboxRoot is set, so a message sent
// while the teammate is idle survives a restart; the next assignment flushes
// the backlog into the job's P3 steer queue before the first turn.
func (ts *TeammateStore) PostMail(name, text string) error {
	name = strings.TrimSpace(name)
	text = strings.TrimSpace(text)
	if name == "" || text == "" {
		return fmt.Errorf("mail needs a teammate name and text")
	}
	ts.mu.Lock()
	_, ok := ts.teammates[name]
	root := ts.inboxRoot
	ts.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown teammate %q", name)
	}
	if root == "" {
		return nil // ephemeral: the P3 queue is the mailbox while running
	}
	dir := filepath.Join(root, sanitizeMailName(name), "inbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	payload, _ := json.Marshal(mailItem{Name: name, Text: text, At: time.Now().Unix()})
	return os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", time.Now().UnixNano())), payload, 0o644)
}

// mailItem is one persisted inbox entry.
type mailItem struct {
	Name string
	Text string
	At   int64
}

func sanitizeMailName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '\x00' {
			return '_'
		}
		return r
	}, name)
}

// flushMailbox moves persisted mail into the running job's P3 steer queue and
// removes the delivered files. Called after a successful assignment.
func (ts *TeammateStore) flushMailbox(name, parentSession, jobID string) error {
	if ts.inboxRoot == "" || jobID == "" || ts.jm == nil {
		return nil
	}
	dir := filepath.Join(ts.inboxRoot, sanitizeMailName(name), "inbox")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var item mailItem
		if json.Unmarshal(data, &item) == nil && item.Text != "" {
			_ = ts.jm.SendMessageForSession(parentSession, jobID, item.Text)
		}
		_ = os.Remove(filepath.Join(dir, e.Name()))
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
