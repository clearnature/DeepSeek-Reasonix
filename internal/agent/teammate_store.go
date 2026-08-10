package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool/builtin"
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
// the snapshot maintained by completion events: recordTask writes the initial
// Running value, HandleJobDone writes the immutable terminal value. DependsOn
// gates the start of dependents (completion gate — a dependent assignment is
// refused until every job it depends on reaches a terminal state). A waiting
// task (a dependency-gated assignment that could not start yet) has
// JobID == "" and Status == taskPending; the auto-advance path promotes it to
// a real assignment once every dependency is terminal.
type TeamTask struct {
	ID            string // task key; waiting tasks use a placeholder id
	JobID         string // actual job id once assigned; "" while waiting
	Owner         string
	Prompt        string // snapshot of the original assignment text (auto-advance replays it byte-identical)
	SessionID     string // leader session; auto-advance rebuilds a minimal ctx from it (never caches a ctx object)
	CreatedAt     time.Time
	DependsOn     []string
	Status        jobs.Status // "running" on assignment; "pending"/"blocked" while waiting; terminal after HandleJobDone
	BlockedReason string      // set when Status == taskBlocked
}

// Waiting-task statuses (team-local values carried by the jobs.Status type).
const (
	taskPending jobs.Status = "pending"
	taskBlocked jobs.Status = "blocked"
)

type TeammateStore struct {
	mu        sync.Mutex
	teammates map[string]*Teammate
	task      *TaskTool
	jm        *jobs.Manager
	// inboxRoot persists teammate mail on disk (P6.1 enhancement 2); empty
	// disables persistence (mail stays ephemeral in the P3 job queue).
	inboxRoot string
	// sink delivers mailbox-wakeup notices to the leader (P6.2); nil keeps
	// PostMail silent beyond the disk write.
	sink event.Sink
	// tasks is the dependency tree (jobID → task); completed gates live here.
	tasks map[string]*TeamTask
	// leader is the first fork source observed from an Assign context. It is
	// the minimal template auto-advance needs to fork a first-round teammate
	// (its rebuilt ctx carries no turn context). Set once, never the ctx object
	// itself — only the *Agent the ctx pointed at (lifecycle-design 主题 5).
	leader *Agent
	// autoCh receives waiting task IDs to auto-advance; a single worker goroutine
	// consumes it serially so assignments never run concurrently for one store
	// and the completion handler never blocks on jobs I/O (discipline G1).
	autoCh     chan string
	doneCh     chan struct{}
	workerOnce sync.Once
}

// NewTeammateStore wires a registry to the task tool that executes assignments.
// inboxRoot (optional) persists teammate mail on disk; empty keeps it ephemeral.
// The store registers its completion handler (HandleJobDone) on the job
// manager at construction (E1's SetJobDoneObserver), so teammate lifecycle is
// driven by terminal job events rather than lazy List() scans.
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
	ts.autoCh = make(chan string, 16)
	ts.doneCh = make(chan struct{})
	go ts.autoWorker()
	if jm != nil {
		jm.SetJobDoneObserver(ts.HandleJobDone)
	}
	return ts
}

// Close stops the auto-advance worker. Safe to call multiple times; jobs stay
// untouched (the manager owns them). Tests must defer Close to keep goleak clean.
func (ts *TeammateStore) Close() {
	ts.workerOnce.Do(func() { close(ts.doneCh) })
}

// autoWorker serially consumes ready-to-advance task IDs. Serial execution
// keeps auto-assignments of the same owner ordered and never blocks the job
// completion handler on fork/start I/O (discipline G1).
func (ts *TeammateStore) autoWorker() {
	for {
		select {
		case <-ts.doneCh:
			return
		case id := <-ts.autoCh:
			ts.autoAssignByID(id)
		}
	}
}

// enqueueAuto non-blockingly hands a waiting task to the auto-advance worker.
// A full queue marks the task blocked instead of blocking the completion
// handler; the next completion event re-scans blocked candidates (recovery).
func (ts *TeammateStore) enqueueAuto(id string) {
	select {
	case ts.autoCh <- id:
	default:
		ts.mu.Lock()
		if cur := ts.tasks[id]; cur != nil && cur.JobID == "" && cur.Status == taskPending {
			cur.Status = taskBlocked
			cur.BlockedReason = "auto-advance queue full"
		}
		ts.mu.Unlock()
		slog.Warn("team auto-advance queue full", "task", id)
	}
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
	slog.Info("team teammate created", "name", name, "role", role, "writable", ts.teammates[name].Writable)
	return nil
}

// List returns teammates sorted by name (deterministic for /team-status).
// Running members whose last job reached a terminal state are flipped to idle
// by HandleJobDone (event-driven, the primary path); the lazy sync below is a
// fallback for windows where no completion event arrived (e.g. before the
// observer was wired, or a destroy window swallowed the event).
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
// Lazy fallback only — the event-driven HandleJobDone is the primary path.
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

// SetSink wires mailbox-wakeup notices (P6.2). Must be called before mail is
// posted for the notices to surface; assignments are unaffected.
func (ts *TeammateStore) SetSink(sink event.Sink) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.sink = sink
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
	// Capture the leader fork source once (first Assign ctx that carries one):
	// auto-advance needs it to fork a first-round teammate, since its rebuilt
	// ctx has no turn context. Only the *Agent is stored, never a ctx object.
	if ts.leader == nil {
		if parent, ok := ForkSourceFromContext(ctx); ok {
			ts.leader = parent
		}
	}
	// P6.1 completion gate checked before claiming the running slot, so a
	// refusal leaves the teammate idle for a later retry. Caller holds ts.mu.
	// Arbitration 3: a dependency-gated refusal also registers a waiting task
	// (pending), which HandleJobDone auto-advances once every dependency is
	// terminal — the assignment is not lost, it just starts later.
	if len(dependsOn) > 0 {
		if reason := ts.pendingDependenciesLocked(dependsOn); reason != "" {
			ts.recordPendingTaskLocked(name, prompt, jobs.SessionFromContext(ctx), dependsOn)
			ts.mu.Unlock()
			slog.Debug("team dependency gate refused; registered pending", "teammate", name, "blocked_by", reason)
			return "", fmt.Errorf("teammate %q blocked by unfinished dependency: %s (registered as pending; auto-starts when deps finish)", name, reason)
		}
	}
	tm.State = TeammateRunning
	ref, toolset := tm.Ref, append([]string(nil), tm.ToolSet...)
	ts.mu.Unlock()

	// P6.2 teammate direct-connect: stamp the mailbox so the teammate sub-agent
	// can post mail to its peers via the team_message tool (builtin.Mailbox).
	ctx = builtin.WithMailbox(ctx, ts)
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
	// datamodel §4.3: the first fork produced a NEW transcript ref; write it
	// back so later assignments continue the same transcript (auto-advance
	// depends on a teammate having a ref — its rebuilt ctx has no fork source).
	if newRef := teammateRef(out); newRef != "" {
		ref = newRef
	}
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
		ts.recordTask(jobID, name, prompt, jobs.SessionFromContext(ctx), dependsOn)
	}
	slog.Info("team teammate assigned", "name", name, "job", jobID,
		"fork_first", ref == "", "depends_on", dependsOn)
	return jobID, nil
}

// recordTask adds a started assignment to the dependency tree. Status starts
// at Running; HandleJobDone overwrites it with the immutable terminal value.
// The prompt and session id are snapshotted here so auto-advance can replay
// the exact assignment (original owner + original prompt text) later.
func (ts *TeammateStore) recordTask(jobID, owner, prompt, sessionID string, dependsOn []string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tasks[jobID] = &TeamTask{
		ID:        jobID,
		JobID:     jobID,
		Owner:     owner,
		Prompt:    prompt,
		SessionID: sessionID,
		CreatedAt: time.Now(),
		DependsOn: append([]string(nil), dependsOn...),
		Status:    jobs.Running,
	}
}

// recordPendingTaskLocked registers a waiting task: a dependency-gated
// assignment that could not start yet (Assign's dependency gate). Caller must
// hold ts.mu. The placeholder id is a task key only and never becomes a job
// id; when HandleJobDone auto-advances the task, the placeholder entry is
// dropped and the started job gets its own registration — exactly one entry
// per task, never a duplicate pair.
func (ts *TeammateStore) recordPendingTaskLocked(owner, prompt, sessionID string, dependsOn []string) string {
	id := fmt.Sprintf("pending:%s:%d", owner, time.Now().UnixNano())
	ts.tasks[id] = &TeamTask{
		ID:        id,
		Owner:     owner,
		Prompt:    prompt,
		SessionID: sessionID,
		CreatedAt: time.Now(),
		DependsOn: append([]string(nil), dependsOn...),
		Status:    taskPending,
	}
	return id
}

// pendingDependenciesLocked returns a non-empty reason naming the first
// dependency that is still running (or unknown), or "" when all are terminal.
// The dependency-tree snapshot (recordTask initial / HandleJobDone terminal)
// is the sole source — never consult the manager (a consuming read that would
// also see the stale Running status while a completion event is in flight).
// Caller must hold ts.mu.
func (ts *TeammateStore) pendingDependenciesLocked(dependsOn []string) string {
	// tasks snapshot is the source of truth: recordTask writes Running at
	// start, HandleJobDone writes the terminal state at completion. Never
	// consult jm.Output here — it reads the job status field, which is still
	// Running when the completion event fires (recordCompletion runs before
	// the status flip), and it is a consuming read that would steal the P1
	// envelope tail.
	for _, dep := range dependsOn {
		st := jobs.Status("")
		if tm, ok := ts.tasks[dep]; ok {
			st = tm.Status
		}
		switch st {
		case jobs.Done, jobs.Failed, jobs.Killed, jobs.Interrupted:
			continue
		default:
			return dep + " (" + string(st) + ")"
		}
	}
	return ""
}

// Tasks returns the tracked dependency tree, newest first. Status is the
// snapshot maintained by recordTask (Running) and HandleJobDone (terminal), so
// the tree stays accurate even after a manager purge — no live query needed.
// Waiting tasks are visible with Status == pending (or blocked with a reason).
func (ts *TeammateStore) Tasks() []TeamTask {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	out := make([]TeamTask, 0, len(ts.tasks))
	for _, t := range ts.tasks {
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

// teammateRef extracts the transcript reference from a run result (the
// "Subagent reference: <sa_...>" line FormatSubagentReference emits in
// task.go). Empty when the line is missing. This is the minimal back-fill
// needed so a teammate keeps the ref its first fork produced — auto-advance
// (arbitration 3) replays Assign with a rebuilt ctx that carries no fork
// source, so it can only continue an existing transcript. The full
// continue-semantics repair (fork prefix verification) remains a separately
// tracked item (arbitration 7).
func teammateRef(out string) string {
	const marker = "Subagent reference: "
	i := strings.Index(out, marker)
	if i < 0 {
		return ""
	}
	rest := out[i+len(marker):]
	if end := strings.IndexByte(rest, '\n'); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

// isTerminalStatus reports whether a job status is terminal. Completion events
// only fire for these; Running and empty values are never terminal.
func isTerminalStatus(st jobs.Status) bool {
	switch st {
	case jobs.Done, jobs.Failed, jobs.Killed, jobs.Interrupted:
		return true
	}
	return false
}

// HandleJobDone is the completion-event handler wired in NewTeammateStore via
// jobs.SetJobDoneObserver (E1). It runs synchronously on the job's finishing
// goroutine for EVERY terminal background job of the manager, so the first
// step filters to registered teammate tasks. It is invoked without a context
// and must not depend on any caller ctx. It never panics: internal errors are
// slog-recorded, and the jobs layer wraps this callback with recover as the
// outer safety net.
//
// Responsibilities (arbitration 1-6):
//   - persist the terminal status into the dependency tree (idempotent; a
//     terminal value is never overwritten),
//   - flip the owning teammate Running→Idle (strict LastJobID match; only
//     the Running→Idle transition flips),
//   - wake the leader once when the finished teammate has a mail backlog
//     (the only completion Notice, aggregated with N to prevent storms),
//   - auto-advance waiting tasks whose dependencies are all terminal by
//     re-issuing Assign with the originally recorded owner/prompt and a
//     freshly rebuilt ctx (arbitration 3; see assignContext — no ctx caching).
//
// Lock order: this handler runs from a jobs-layer lock-free point, takes
// ts.mu only for the atomic settle (never ts.jm.* inside), then does the
// mailbox IO and the auto-advance enqueue outside the lock. The actual
// auto-assign runs on the store's single worker goroutine (autoWorker), so
// the callback never blocks on fork/start I/O (discipline G1).
func (ts *TeammateStore) HandleJobDone(id string, st jobs.Status, err error) {
	if !isTerminalStatus(st) {
		return // terminal filter: Running and unknown statuses are ignored
	}
	ts.mu.Lock()
	if _, ok := ts.tasks[id]; !ok {
		ts.mu.Unlock()
		return // not a teammate job (the observer is a manager-wide singleton)
	}
	// Dependency settle: write the terminal status once (idempotent).
	if t := ts.tasks[id]; t != nil && !isTerminalStatus(t.Status) {
		t.Status = st
	}
	// Idle flip: strict LastJobID match, Running→Idle only.
	var flipped string
	for _, tm := range ts.teammates {
		if tm.LastJobID == id && tm.State == TeammateRunning {
			tm.State = TeammateIdle
			flipped = tm.Name
			break
		}
	}
	// Auto-advance candidates: only a clean Done advances dependent tasks
	// (Failed/Killed/Interrupted settle the gate so the leader can re-issue
	// manually, but never auto-start a successor — discipline review 2/6).
	// Blocked candidates are re-scanned too: each completion event is a
	// retry window for a task that failed transiently (queue full, owner
	// briefly busy, slot limit).
	var ready []string
	for _, t := range ts.tasks {
		if t == nil || t.JobID != "" {
			continue
		}
		if t.Status != taskPending && t.Status != taskBlocked {
			continue
		}
		if st != jobs.Done {
			continue
		}
		if !ts.depsTerminalLocked(t.DependsOn) {
			continue
		}
		ready = append(ready, t.ID)
	}
	ts.mu.Unlock()

	if flipped != "" && ts.inboxRoot != "" {
		if n := ts.countInbox(flipped); n > 0 {
			ts.notifyMailBacklog(flipped, n)
		}
	}
	for _, id := range ready {
		ts.enqueueAuto(id)
	}
}

// depsTerminalLocked reports whether every listed dependency has a terminal
// status in the dependency tree. Unknown dependencies never count as terminal
// (no event can settle them), so a chain with a hole stays waiting instead of
// advancing — safe, no infinite loop (arbitration 3).
func (ts *TeammateStore) depsTerminalLocked(deps []string) bool {
	for _, dep := range deps {
		t, ok := ts.tasks[dep]
		if !ok || !isTerminalStatus(t.Status) {
			return false
		}
	}
	return true
}

// assignContext rebuilds the minimal context needed to re-issue an automatic
// assignment from a stored session id. The stored value is a string, never a
// ctx object (a turn ctx is cancelled long before a background job finishes);
// a fresh ctx is built on every call — never cached. The leader fork source
// (captured once from the first Assign ctx) is re-attached so a first-round
// teammate can fork; it is the *Agent, not the turn ctx.
func (ts *TeammateStore) assignContext(sessionID string) context.Context {
	ctx := context.Background()
	if ts.jm != nil {
		ctx = jobs.WithManager(ctx, ts.jm)
	}
	ctx = jobs.WithSession(ctx, sessionID)
	ctx = WithParentSession(ctx, sessionID)
	ts.mu.Lock()
	leader := ts.leader
	ts.mu.Unlock()
	if leader != nil {
		ctx = WithForkSource(ctx, leader)
	}
	return ctx
}

// autoAssign re-issues one waiting task as a real assignment. It never
// preempts: if the owner is gone or busy, the task is marked blocked and stays
// visible in Tasks() (no silent dropout). On success the started job's own
// registration (recordTask inside Assign) becomes the single entry and the
// placeholder is dropped — one registration per task, never a duplicate pair.
// autoAssignByID re-issues one waiting task as a real assignment. It never
// preempts: if the owner is gone or busy, the task is marked blocked and stays
// visible in Tasks() (no silent dropout). On success the started job's own
// registration (recordTask inside Assign) becomes the single entry and the
// placeholder is dropped — one registration per task, never a duplicate pair.
func (ts *TeammateStore) autoAssignByID(id string) {
	ts.mu.Lock()
	cur := ts.tasks[id]
	if cur == nil || cur.JobID != "" {
		ts.mu.Unlock()
		return // already advanced or removed (idempotent)
	}
	if cur.Status != taskPending && cur.Status != taskBlocked {
		ts.mu.Unlock()
		return
	}
	owner, prompt, sessionID := cur.Owner, cur.Prompt, cur.SessionID
	deps := append([]string(nil), cur.DependsOn...)
	if _, ok := ts.teammates[owner]; !ok {
		cur.Status = taskBlocked
		cur.BlockedReason = "teammate removed"
		ts.mu.Unlock()
		slog.Warn("team auto-assign skipped", "task", id, "owner", owner, "reason", "teammate removed")
		return
	}
	ts.mu.Unlock()

	ctx := ts.assignContext(sessionID)
	jobID, err := ts.Assign(ctx, owner, prompt, deps...)
	if err != nil || jobID == "" {
		reason := "auto-assign produced no job"
		if err != nil {
			reason = err.Error()
		}
		ts.mu.Lock()
		if cur := ts.tasks[id]; cur != nil && cur.JobID == "" && (cur.Status == taskPending || cur.Status == taskBlocked) {
			cur.Status = taskBlocked
			cur.BlockedReason = reason
		}
		ts.mu.Unlock()
		slog.Warn("team auto-assign failed", "task", id, "owner", owner, "job", jobID, "err", err)
		return
	}
	ts.mu.Lock()
	delete(ts.tasks, id)
	ts.mu.Unlock()
	slog.Info("team auto-assigned", "task", id, "owner", owner, "job", jobID)
}

// Complete marks a teammate idle after its job reaches a terminal state. The
// job's P1 envelope has already delivered the result to the leader.
// Legacy API kept for compatibility; HandleJobDone is the event-driven path
// and does not call this.
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

// TeamStop stops a running teammate's job (task_stop, P6.2): kill the last
// job and flip the member back to idle so it can take a new assignment. Idle
// teammates are a no-op. The job's transcript survives for the next continue.
func (ts *TeammateStore) TeamStop(name string) error {
	ts.mu.Lock()
	tm, ok := ts.teammates[name]
	if !ok {
		ts.mu.Unlock()
		return fmt.Errorf("unknown teammate %q", name)
	}
	jobID := tm.LastJobID
	if tm.State == TeammateIdle {
		ts.mu.Unlock()
		return nil
	}
	tm.State = TeammateIdle
	ts.mu.Unlock()

	if jobID != "" && ts.jm != nil {
		ts.jm.KillForSession("", jobID)
	}
	slog.Info("team teammate stopped", "name", name, "job", jobID)
	return nil
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
	// Arbitration 6: drop this teammate's tracked tasks (started and waiting)
	// so Tasks() never shows an orphaned owner. A running orphan job keeps
	// running; its completion event finds no entry and is a no-op.
	for tid, t := range ts.tasks {
		if t.Owner == name {
			delete(ts.tasks, tid)
		}
	}
	ts.mu.Unlock()

	if jobID != "" && ts.jm != nil {
		ts.jm.Kill(jobID)
	}
	slog.Info("team teammate removed", "name", name, "killed_job", jobID)
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
		// Ephemeral: the P3 steer queue is the mailbox while running. Still
		// notify so the leader knows the mail was accepted (wake-up signal).
		ts.notifyMail(name)
		return nil
	}
	dir := filepath.Join(root, sanitizeMailName(name), "inbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	payload, _ := json.Marshal(mailItem{Name: name, Text: text, At: time.Now().Unix()})
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.json", time.Now().UnixNano())), payload, 0o644); err != nil {
		return err
	}
	ts.notifyMail(name)
	return nil
}

// notifyMail emits the mailbox-wakeup notice (P6.2): the leader learns a
// teammate received mail while idle, so it can decide to assign work that
// flushes the inbox.
func (ts *TeammateStore) notifyMail(name string) {
	ts.mu.Lock()
	sink := ts.sink
	ts.mu.Unlock()
	if sink != nil {
		sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: "teammate " + name + " received mail — /team-add <name> <task> to flush it"})
	}
}

// countInbox returns the number of undelivered mail files in a teammate's
// inbox — the same consumption set as flushMailbox. Ephemeral mode (no
// inboxRoot) always reports zero. Read errors degrade to zero (the next
// assignment's flushMailbox remains the fallback drain).
func (ts *TeammateStore) countInbox(name string) int {
	if ts.inboxRoot == "" {
		return 0
	}
	dir := filepath.Join(ts.inboxRoot, sanitizeMailName(name), "inbox")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("team mailbox count failed", "name", name, "err", err)
		}
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			n++
		}
	}
	return n
}

// notifyMailBacklog emits the aggregated mailbox-wakeup notice — the only
// completion Notice (arbitration 4): once per finished teammate, only when
// mail is actually backlogged, with the count N to prevent storms. The
// optional segment is wrapped in recover so a sink hiccup cannot break the
// job pipeline; a lost notice is covered by the next assignment's flush.
func (ts *TeammateStore) notifyMailBacklog(name string, n int) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("team mailbox notice panicked", "name", name, "panic", r)
		}
	}()
	ts.mu.Lock()
	sink := ts.sink
	ts.mu.Unlock()
	if sink != nil {
		sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: fmt.Sprintf("teammate %s has %d unread mail — /team-add %s <task> to flush it", name, n, name)})
	}
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
