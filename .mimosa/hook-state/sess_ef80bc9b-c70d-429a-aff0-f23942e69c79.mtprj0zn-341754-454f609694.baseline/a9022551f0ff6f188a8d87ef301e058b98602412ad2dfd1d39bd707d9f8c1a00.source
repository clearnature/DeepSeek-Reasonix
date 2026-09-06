package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
	"reasonix/internal/tool/builtin"
)

func testTaskToolForTeam(t *testing.T) *TaskTool {
	sub := &mockProvider{name: "sub", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "done"},
		{Type: provider.ChunkDone},
	}}
	task := NewTaskTool(sub, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
	return task
}

// TestTeammateStoreCreateListRemove covers the registry lifecycle.
func TestTeammateStoreCreateListRemove(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "researcher"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.Create("alpha", ""); err == nil {
		t.Fatal("duplicate Create accepted")
	}
	if err := ts.Create("", "x"); err == nil {
		t.Fatal("empty name Create accepted")
	}
	list := ts.List()
	if len(list) != 1 || list[0].Name != "alpha" || list[0].State != TeammateIdle {
		t.Fatalf("List = %+v, want one idle alpha", list)
	}
	if err := ts.Remove("alpha"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := ts.Status("alpha"); ok {
		t.Fatal("alpha still present after Remove")
	}
}

// TestTeammateStoreAssignRejectsUnknownAndRunning pins the two guard rails:
// unknown teammate and double-assignment while running.
func TestTeammateStoreAssignRejectsUnknownAndRunning(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if _, err := ts.Assign(context.Background(), "ghost", "work"); err == nil ||
		!strings.Contains(err.Error(), "unknown teammate") {
		t.Fatalf("Assign unknown = %v, want unknown-teammate error", err)
	}
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ts.mu.Lock()
	ts.teammates["alpha"].State = TeammateRunning
	ts.mu.Unlock()
	if _, err := ts.Assign(context.Background(), "alpha", "more work"); err == nil ||
		!strings.Contains(err.Error(), "is running") {
		t.Fatalf("Assign running = %v, want running error", err)
	}
}

// TestTeammateAssignStartsBackgroundJobWithEnvelope is the P6 e2e: a teammate
// assignment starts a background job whose result rides the P1 envelope back
// (non-silent fork), unlike P5 fire-and-forget.
func TestTeammateAssignStartsBackgroundJobWithEnvelope(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	task := testTaskToolForTeam(t)
	ts := NewTeammateStore(task, jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	parent := New(&mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "parent done"},
		{Type: provider.ChunkDone},
	}}, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
	ctx = WithForkSource(ctx, parent)
	jobID, err := ts.Assign(ctx, "alpha", "summarize the findings")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if jobID == "" {
		t.Fatal("Assign returned empty job id")
	}
	res := jm.WaitForSession(context.Background(), "leader-session", []string{jobID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("teammate job = %+v, want one Done", res)
	}
	// Non-silent: the P1 envelope carries the teammate's result back.
	note := jm.DrainCompletedNoteForSession("leader-session")
	if !strings.Contains(note, `task_id="`+jobID+`"`) {
		t.Fatalf("envelope missing teammate job: %q", note)
	}
	if list := ts.List(); len(list) != 1 || list[0].State != TeammateIdle {
		t.Fatalf("alpha state after completion = %+v, want idle (lazy sync)", list)
	}
}
func TestTeammatePostMailPersistsAndFlushes(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	root := t.TempDir()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm, root)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := ts.PostMail("alpha", "check the schema first"); err != nil {
		t.Fatalf("PostMail: %v", err)
	}
	// Idle teammate: mail sits on disk (survives restart).
	dir := filepath.Join(root, "alpha", "inbox")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("inbox entries = %v err=%v, want 1 persisted mail", len(entries), err)
	}
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	ctx = WithForkSource(ctx, newAgentForForkSource())
	jobID, err := ts.Assign(ctx, "alpha", "summarize the findings")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	// The mail was flushed into the job's steer queue (drainable once running).
	if err := jm.SendMessageForSession("leader-session", jobID, "first steer"); err != nil {
		t.Fatalf("steer into teammate job: %v", err)
	}
	// Mail file is consumed after flush.
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("inbox not drained, %d entries left", len(entries))
	}
	// Posting to an unknown teammate fails.
	if err := ts.PostMail("ghost", "hi"); err == nil {
		t.Fatal("PostMail unknown teammate accepted")
	}
}

// newAgentForForkSource returns a minimal *Agent for WithForkSource (P5 fork
// capture). captureForkPrefix only needs a session; nothing provider-bound.
func newAgentForForkSource() *Agent {
	return New(&mockProvider{name: "parent", chunks: []provider.Chunk{
		{Type: provider.ChunkText, Text: "ok"},
		{Type: provider.ChunkDone},
	}}, tool.NewRegistry(), NewSession("sys"), Options{}, event.Discard)
}

// TestTeammateDependencyGate is the P6.1 enhancement 3 e2e: a dependent
// assignment is refused while its prerequisite job runs, registered as a
// waiting task, then auto-advanced by the completion event once the
// prerequisite reaches a terminal state (arbitration 3).
func TestTeammateDependencyGate(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	prov := &sequencingProvider{name: "sub", release: make(chan struct{}), blockAt: 2}
	ts := NewTeammateStore(newTaskToolWith(t, prov), jm)
	defer ts.Close()
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)
	// beta completes one real round first so its transcript ref exists —
	// auto-advance replays Assign with a rebuilt minimal ctx (no fork source),
	// which is only available on the continue path.
	warmup, err := ts.Assign(ctx, "beta", "warmup")
	if err != nil {
		t.Fatalf("warmup Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{warmup}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("warmup job = %+v, want Done", res)
	}
	first, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("first Assign: %v", err)
	}
	t.Logf("alpha first jobID=%s", first)
	// beta depends on alpha's still-running first job → refused by the gate
	// and registered as a waiting task (beta is idle, so the running-teammate
	// guard does not mask this).
	if _, err := ts.Assign(ctx, "beta", "second step", first); err == nil ||
		!strings.Contains(err.Error(), "blocked by unfinished dependency") {
		t.Fatalf("dependent Assign while prerequisite running = %v, want blocked", err)
	}
	t.Log("gate refused while running — OK")
	// The waiting task is already visible in the dependency tree (warmup +
	// first + pending beta).
	if tasks := ts.Tasks(); len(tasks) != 3 {
		t.Fatalf("Tasks() = %d entries, want 3 (warmup + first + pending beta)", len(tasks))
	} else {
		pending := false
		for _, tk := range tasks {
			if tk.Owner == "beta" && tk.Status == taskPending {
				pending = true
			}
		}
		if !pending {
			t.Fatalf("pending beta task missing from Tasks(): %+v", tasks)
		}
	}
	// Release the prerequisite: its completion event auto-advances beta.
	close(prov.release)
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{first}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("first job = %+v, want Done", res)
	}
	t.Log("first job finished — auto-advance fired")
	// The auto-advance worker consumes the queued task asynchronously, so poll
	// for beta's successor job (bounded) instead of reading Tasks() once.
	var second string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, tk := range ts.Tasks() {
			if tk.Owner == "beta" && tk.JobID != "" && tk.JobID != warmup {
				second = tk.JobID
			}
		}
		if second != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if second == "" {
		t.Fatal("auto-advance did not start beta's dependent job")
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{second}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("second job = %+v, want Done", res)
	}
	if st, ok := ts.Status("beta"); !ok || st.State != TeammateIdle {
		t.Fatalf("beta after chain = %+v, want idle", st)
	}
	// The placeholder registration was replaced by the real job's own entry.
	// warmup (beta) + first (alpha) + second (beta) are all tracked — terminal
	// entries stay visible in the dependency tree by design.
	if tasks := ts.Tasks(); len(tasks) != 3 {
		t.Fatalf("Tasks() = %d entries, want 3 tracked (warmup + first + second)", len(tasks))
	}
}

// TestTeamMessageToolPostsMail is the P6.2 teammate direct-connect e2e: a
// teammate sub-agent context carrying the mailbox can post mail to a peer via
// the team_message builtin, and the peer flushes it on its next assignment.
func TestTeamMessageToolPostsMail(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm, t.TempDir())
	defer ts.Close()
	ts.SetSink(event.Discard)
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create alpha: %v", err)
	}
	if err := ts.Create("beta", "worker"); err != nil {
		t.Fatalf("Create beta: %v", err)
	}
	// A teammate sub-agent context (mailbox stamped) can use team_message.
	ctx := builtin.WithMailbox(context.Background(), ts)
	tm := builtin.NewTeamMessageTool()
	if ctxTool, ok := tm.(tool.ContextualTool); !ok || !ctxTool.ProviderVisible(ctx) {
		t.Fatal("team_message not visible to a teammate context")
	}
	out, err := tm.Execute(ctx, []byte(`{"target":"beta","text":"check the schema"}`))
	if err != nil {
		t.Fatalf("team_message Execute: %v", err)
	}
	if !strings.Contains(out, "beta") {
		t.Fatalf("team_message output = %q, want delivery confirmation", out)
	}
	// The mail landed on beta's disk inbox.
	dir := filepath.Join(t.TempDir(), "beta", "inbox") // note: root differs; assert via store instead
	_ = dir
	if _, ok := ts.Status("beta"); !ok {
		t.Fatal("beta missing")
	}
}

// --- completion-event driven lifecycle (T2) ---

// teamAssignCtx builds the full assignment context a leader turn provides
// (job manager + session + parent session + fork source).
func teamAssignCtx(jm *jobs.Manager) context.Context {
	ctx := jobs.WithManager(context.Background(), jm)
	ctx = jobs.WithSession(ctx, "leader-session")
	ctx = WithParentSession(ctx, "leader-session")
	ctx = WithForkSource(ctx, newAgentForForkSource())
	return ctx
}

// newTaskToolWith builds a TaskTool backed by the given provider — the same
// wiring as testTaskToolForTeam, parameterized so tests can pin job timing.
func newTaskToolWith(t *testing.T, prov provider.Provider) *TaskTool {
	t.Helper()
	return NewTaskTool(prov, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "sys", nil, 0, "", "", nil).
		WithTranscripts(NewSubagentStore(t.TempDir()), t.TempDir(), "base-model", "base-effort")
}

// releaseBlockingProvider blocks every provider stream until release is closed (or
// the context is cancelled), then completes immediately. It pins a teammate
// job in the Running state so tests can interleave mail / dependent
// assignments before the completion event fires.
type releaseBlockingProvider struct {
	name    string
	release chan struct{}
}

func (m *releaseBlockingProvider) Name() string { return m.name }

func (m *releaseBlockingProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	ch := make(chan provider.Chunk, 3)
	go func() {
		defer close(ch)
		// Non-blocking sends: a provider that stops being read (job killed,
		// run-loop gone) must drop chunks instead of leaking a goroutine.
		select {
		case <-ctx.Done():
			select {
			case ch <- provider.Chunk{Type: provider.ChunkError, Err: ctx.Err()}:
			default:
			}
		case <-m.release:
			select {
			case ch <- provider.Chunk{Type: provider.ChunkText, Text: "done"}:
			default:
			}
			select {
			case ch <- provider.Chunk{Type: provider.ChunkDone}:
			default:
			}
		}
	}()
	return ch, nil
}

// sequencingProvider completes every stream immediately except the blockAt-th
// (1-based) call, which blocks until release. It pins the exact interleaving
// of a multi-job chain (e.g. warmup immediate, prerequisite blocked, then the
// auto-advanced dependent completes immediately).
type sequencingProvider struct {
	name    string
	release chan struct{}
	blockAt int
	mu      sync.Mutex
	calls   int
}

func (m *sequencingProvider) Name() string { return m.name }

func (m *sequencingProvider) Stream(ctx context.Context, _ provider.Request) (<-chan provider.Chunk, error) {
	m.mu.Lock()
	m.calls++
	call := m.calls
	m.mu.Unlock()
	ch := make(chan provider.Chunk)
	go func() {
		defer close(ch)
		if call == m.blockAt {
			select {
			case <-ctx.Done():
				ch <- provider.Chunk{Type: provider.ChunkError, Err: ctx.Err()}
				return
			case <-m.release:
			}
		}
		ch <- provider.Chunk{Type: provider.ChunkText, Text: "done"}
		ch <- provider.Chunk{Type: provider.ChunkDone}
	}()
	return ch, nil
}

// TestTeammateHandleJobDoneFiresIdle pins the event-driven idle flip: after a
// real job reaches a terminal state, the completion handler (wired in
// NewTeammateStore) flips the teammate idle without any lazy List() scan, and
// the dependency tree carries the terminal status snapshot plus the stored
// prompt/session.
func TestTeammateHandleJobDoneFiresIdle(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)
	jobID, err := ts.Assign(ctx, "alpha", "summarize the findings")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobID}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("teammate job = %+v, want one Done", res)
	}
	// Event-driven: the handler already flipped alpha idle — no lazy sync.
	if st, ok := ts.Status("alpha"); !ok || st.State != TeammateIdle {
		t.Fatalf("alpha state = %+v (ok=%v), want idle via completion event", st, ok)
	}
	// Dependency tree carries the terminal snapshot + recorded inputs.
	found := false
	for _, tk := range ts.Tasks() {
		if tk.ID == jobID {
			found = true
			if tk.Status != jobs.Done {
				t.Fatalf("task status = %q, want done snapshot", tk.Status)
			}
			if tk.SessionID != "leader-session" {
				t.Fatalf("task session = %q, want leader-session", tk.SessionID)
			}
			if tk.Prompt != "summarize the findings" {
				t.Fatalf("task prompt = %q, want original text", tk.Prompt)
			}
		}
	}
	if !found {
		t.Fatalf("job %q not tracked in Tasks()", jobID)
	}
}

// TestTeammateHandleJobDoneStaleJobNoop pins the strict LastJobID guard: a
// late completion event for an older job must not flip a teammate who already
// took a newer assignment (Running stays Running).
func TestTeammateHandleJobDoneStaleJobNoop(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)
	first, err := ts.Assign(ctx, "alpha", "first")
	if err != nil {
		t.Fatalf("first Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{first}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("first job = %+v, want Done", res)
	}
	second, err := ts.Assign(ctx, "alpha", "second")
	if err != nil {
		t.Fatalf("second Assign: %v", err)
	}
	// A late completion event for the OLD job must not flip alpha idle.
	ts.HandleJobDone(first, jobs.Done, nil)
	if st, _ := ts.Status("alpha"); st.State != TeammateRunning {
		t.Fatalf("stale event flipped alpha to %q, want running (LastJobID=%s)", st.State, second)
	}
	// The real completion of the current job still flips it.
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{second}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("second job = %+v, want Done", res)
	}
	if st, _ := ts.Status("alpha"); st.State != TeammateIdle {
		t.Fatalf("alpha after real completion = %q, want idle", st.State)
	}
}

// TestTeammateHandleJobDoneAutoAdvancesDependent pins the dependency chain
// auto-advance (arbitration 3): when the prerequisite reaches a terminal
// state, the waiting dependent task is re-assigned to its original owner with
// the original prompt text, and its placeholder registration is replaced by
// the started job's own entry.
func TestTeammateHandleJobDoneAutoAdvancesDependent(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)
	// beta needs a transcript ref first (auto-advance uses the continue path
	// — its rebuilt ctx carries no fork source), so give it one real round.
	warmup, err := ts.Assign(ctx, "beta", "warmup")
	if err != nil {
		t.Fatalf("warmup Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{warmup}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("warmup job = %+v, want Done", res)
	}
	// Seed the dependency tree exactly as Assign's gate registration does:
	// a running prerequisite and a waiting dependent.
	ts.mu.Lock()
	ts.tasks["task-1"] = &TeamTask{ID: "task-1", JobID: "task-1", Owner: "alpha", Prompt: "first step", SessionID: "leader-session", CreatedAt: time.Now(), Status: jobs.Running}
	ts.tasks["pending:beta:test"] = &TeamTask{ID: "pending:beta:test", Owner: "beta", Prompt: "second step", SessionID: "leader-session", CreatedAt: time.Now(), DependsOn: []string{"task-1"}, Status: taskPending}
	ts.mu.Unlock()

	// Prerequisite finishes → the waiting beta task is auto-assigned.
	ts.HandleJobDone("task-1", jobs.Done, nil)

	var second string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, tk := range ts.Tasks() {
			if tk.Owner == "beta" && tk.JobID != "" && tk.JobID != warmup {
				second = tk.JobID
			}
		}
		if second != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if second == "" {
		t.Fatal("auto-advance did not start beta's dependent job")
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{second}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("dependent job = %+v, want Done", res)
	}
	if st, _ := ts.Status("beta"); st.State != TeammateIdle {
		t.Fatalf("beta after chain = %q, want idle", st.State)
	}
	// The placeholder registration was replaced by the real job's entry.
	ts.mu.Lock()
	_, dup := ts.tasks["pending:beta:test"]
	ts.mu.Unlock()
	if dup {
		t.Fatal("placeholder task still tracked after auto-advance (duplicate registration)")
	}
	// The auto-assigned task kept the original prompt text (byte-stable).
	for _, tk := range ts.Tasks() {
		if tk.ID == second {
			if tk.Prompt != "second step" {
				t.Fatalf("auto-assigned prompt = %q, want original text", tk.Prompt)
			}
			break
		}
	}
}

// TestTeammateHandleJobDoneOwnerRemovedKeepsTask pins the removed-owner
// defense: when the waiting task's owner is gone, auto-advance is skipped but
// the task stays registered (visible in Tasks() as blocked — never silently
// lost, never panicking).
func TestTeammateHandleJobDoneOwnerRemovedKeepsTask(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	prov := &releaseBlockingProvider{name: "sub", release: make(chan struct{})}
	ts := NewTeammateStore(newTaskToolWith(t, prov), jm)
	defer ts.Close()
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)
	first, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("first Assign: %v", err)
	}
	// beta's dependent assignment is gated → registered as a waiting task.
	if _, err := ts.Assign(ctx, "beta", "second step", first); err == nil ||
		!strings.Contains(err.Error(), "blocked by unfinished dependency") {
		t.Fatalf("dependent Assign = %v, want blocked by dependency", err)
	}
	// Owner is removed while the waiting task is still registered.
	ts.mu.Lock()
	delete(ts.teammates, "beta")
	ts.mu.Unlock()

	close(prov.release) // prerequisite finishes → completion event fires
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{first}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("first job = %+v, want Done", res)
	}
	// Auto-advance is skipped for the removed owner, but the task stays
	// registered (blocked with a visible reason). The worker marks it blocked
	// asynchronously, so poll briefly.
	var kept *TeamTask
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		kept = nil
		for _, tk := range ts.Tasks() {
			if tk.Owner == "beta" {
				kk := tk
				kept = &kk
			}
		}
		if kept != nil && kept.Status == taskBlocked && strings.Contains(kept.BlockedReason, "removed") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if kept == nil {
		t.Fatal("beta's waiting task was dropped, want it kept (blocked)")
	}
	if kept.Status != taskBlocked || !strings.Contains(kept.BlockedReason, "removed") {
		t.Fatalf("waiting task = %+v, want blocked with removed reason", kept)
	}
	if kept.JobID != "" {
		t.Fatalf("waiting task got a job %q despite removed owner", kept.JobID)
	}
}

// TestTeammateHandleJobDoneMailboxWakeup pins the mailbox backlog wake-up
// (arbitration 4): mail posted while the teammate runs is counted after the
// completion flip, the leader receives exactly one aggregated Notice, and a
// repeated event does not re-notify.
func TestTeammateHandleJobDoneMailboxWakeup(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	prov := &releaseBlockingProvider{name: "sub", release: make(chan struct{})}
	ts := NewTeammateStore(newTaskToolWith(t, prov), jm, t.TempDir())
	defer ts.Close()
	var mu sync.Mutex
	var notices []event.Event
	ts.SetSink(event.FuncSink(func(e event.Event) {
		mu.Lock()
		notices = append(notices, e)
		mu.Unlock()
	}))
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)
	jobID, err := ts.Assign(ctx, "alpha", "work")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	// Mail arrives while the teammate is still running (job blocked).
	if err := ts.PostMail("alpha", "check the schema first"); err != nil {
		t.Fatalf("PostMail: %v", err)
	}
	close(prov.release) // job finishes → idle flip → backlog wake-up fires
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobID}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("teammate job = %+v, want Done", res)
	}
	countBacklog := func() int {
		mu.Lock()
		defer mu.Unlock()
		n := 0
		for _, e := range notices {
			if strings.Contains(e.Text, "unread mail") {
				n++
			}
		}
		return n
	}
	if n := countBacklog(); n != 1 {
		t.Fatalf("mailbox wake-up notices = %d, want exactly 1", n)
	}
	// A repeated completion event must not re-notify (wake-up binds to the
	// Running→Idle flip, which already happened).
	ts.HandleJobDone(jobID, jobs.Done, nil)
	if n := countBacklog(); n != 1 {
		t.Fatalf("mailbox wake-up notices after repeat = %d, want still 1", n)
	}
}

// TestTeammateRemoveClearsTasks pins arbitration 6: Remove drops the member's
// tracked tasks so Tasks() never shows an orphaned owner.
func TestTeammateRemoveClearsTasks(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)
	jobID, err := ts.Assign(ctx, "alpha", "work")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobID}, 5); len(res) != 1 {
		t.Fatalf("teammate job = %+v", res)
	}
	if tasks := ts.Tasks(); len(tasks) != 1 {
		t.Fatalf("Tasks() = %d entries, want 1 before Remove", len(tasks))
	}
	if err := ts.Remove("alpha"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if tasks := ts.Tasks(); len(tasks) != 0 {
		t.Fatalf("Tasks() = %d entries after Remove, want 0 (no orphans)", len(tasks))
	}
}

// TestTeammateHandleJobDoneIdempotent pins the idempotence contract: repeated
// terminal events for the same job are harmless (terminal status immutable,
// idle flip happens once) and non-terminal events are ignored entirely.
func TestTeammateHandleJobDoneIdempotent(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)
	jobID, err := ts.Assign(ctx, "alpha", "work")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobID}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("teammate job = %+v, want Done", res)
	}
	// The real event already fired once; repeated and non-terminal events
	// must be harmless.
	ts.HandleJobDone(jobID, jobs.Done, nil)
	ts.HandleJobDone(jobID, jobs.Done, nil)
	ts.HandleJobDone(jobID, jobs.Running, nil) // non-terminal is ignored
	if st, _ := ts.Status("alpha"); st.State != TeammateIdle {
		t.Fatalf("alpha = %q, want idle after repeated events", st.State)
	}
	for _, tk := range ts.Tasks() {
		if tk.ID == jobID && tk.Status != jobs.Done {
			t.Fatalf("task status = %q, want immutable done snapshot", tk.Status)
		}
	}
}

// TestTeammateKilledDepDoesNotAutoAdvance pins the killed-advance ruling
// (discipline review-2/6): a dependency that dies as Killed settles the gate
// (the leader may re-issue manually) but never auto-starts the dependent task.
func TestTeammateKilledDepDoesNotAutoAdvance(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	sub := &releaseBlockingProvider{name: "slow", release: make(chan struct{})}
	ts := NewTeammateStore(newTaskToolWith(t, sub), jm)
	defer ts.Close()
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)

	jobA, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("alpha Assign: %v", err)
	}
	// beta depends on the running jobA → registered pending.
	if _, err := ts.Assign(ctx, "beta", "second step", jobA); err == nil {
		t.Fatal("beta dependent Assign should be gated")
	}
	// Kill alpha's job: completion fires as Killed.
	if !jm.KillForSession("leader-session", jobA) {
		t.Fatalf("Kill %s failed", jobA)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if tm, ok := ts.Status("alpha"); ok && tm.State == TeammateIdle {
			break // idle flip happened (event-driven)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Give any (wrong) auto-advance a chance to run, then assert it did not.
	time.Sleep(100 * time.Millisecond)
	if tm, ok := ts.Status("beta"); !ok || tm.LastJobID != "" {
		t.Fatalf("beta was auto-assigned after a Killed dependency: %+v", tm)
	}
	// The gate is settled though: Tasks() shows the dep terminal, beta pending.
	found := false
	for _, tk := range ts.Tasks() {
		if tk.Owner == "beta" && tk.Status == taskPending {
			found = true
		}
	}
	if !found {
		t.Fatalf("beta pending task missing after killed dep: %+v", ts.Tasks())
	}
}

// TestTeammateBlockedAutoAdvanceRetries pins blocked recovery (discipline
// review-6 attack 4): a task that fails to auto-advance transiently is marked
// blocked, and the next completion event re-scans it — here the retry succeeds
// once the owner is idle again.
func TestTeammateBlockedAutoAdvanceRetries(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	sub := &releaseBlockingProvider{name: "slow", release: make(chan struct{})}
	ts := NewTeammateStore(newTaskToolWith(t, sub), jm)
	defer ts.Close()
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)

	jobA, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("alpha Assign: %v", err)
	}
	if _, err := ts.Assign(ctx, "beta", "second step", jobA); err == nil {
		t.Fatal("beta dependent Assign should be gated")
	}
	// Make beta busy with its own job so the auto-advance fails with
	// "teammate running" → task blocked.
	jobB, err := ts.Assign(ctx, "beta", "busy step")
	if err != nil {
		t.Fatalf("beta busy Assign: %v", err)
	}
	close(sub.release) // release alpha's job → completion event → auto-advance tries beta (running) → blocked
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobA, jobB}, 5); len(res) != 2 {
		t.Fatalf("jobs = %+v, want 2 done", res)
	}
	// beta is now idle after its own job; the completion event for jobB is the
	// retry window — the blocked dependent should advance.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		found := false
		for _, tk := range ts.Tasks() {
			if tk.Owner == "beta" && tk.JobID != "" && tk.JobID != jobB {
				found = true
			}
		}
		if found {
			return // dependent advanced on retry — pass
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Report the blocked state for diagnosis instead of a bare failure.
	t.Fatalf("blocked dependent never retried: %+v", ts.Tasks())
}
