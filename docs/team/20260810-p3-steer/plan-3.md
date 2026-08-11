# Plan 3: P3 steer channel - inject messages into a running background job / sub-agent

> Transaction: docs/team/20260810-p3-steer/ - independent planner view (one of 3 independent plans)
> Goal: community #7962 - no way to steer a running background sub-agent
> References: Qwen `send_message(task_id)` turn-boundary delivery + MAX_PENDING_MESSAGES backpressure rejection (docs/MULTIAGENT_QWEN_COMPARISON.md §1.3); CCB `queuePendingMessage -> queued_command` prefix injection
> Baseline facts (verified line by line): `internal/jobs/jobs.go`, `internal/agent/task.go`, `internal/agent/run_loop.go:275-361`, `internal/agent/agent.go:869-1029` (existing steer mechanism), `internal/control/input.go:188-195` (P1 landing point), `internal/control/controller.go:1398-1440` (slash dispatch), `internal/event/event.go:25-100` (Kind iota)
>
> NOTE: this file was re-rendered in ASCII English because the write_file tool chain
> deterministically mojibakes CJK content for this sub-agent (verified: grep for the
> mojibake form matches, grep for the correct UTF-8 form does not). Content is
> identical to the intended Chinese plan; only the language differs.

---

## 1. Topology scan

### File dependency map

```
internal/jobs/jobs.go   Job struct (+ pendingMessages) -- queue implementation
                        Manager (+ SendMessageForSession) -- backpressure write
                        jobCtxKey (L2064) / injected at StartForSession L460 -- consumer gets job via ctx
                        recordCompletion (L838) -- completion fallback counter
internal/agent/run_loop.go:282  -- consume job-message after consumeSteer (turn boundary)
internal/agent/agent.go -- add JobMessagePrefix + JobMessageText (mirror MidTurnSteerPrefix/SteerText)
internal/agent/preview.go:288   -- IsUserAuthoredTurn adds JobMessageText exclusion (title/turn-count hygiene)
internal/agent/task.go:80-86    -- subagentAlwaysHiddenTools adds send_message (children do not inherit)
internal/tool/builtin/          -- new send_message tool (registration/permission pattern of bgjobs.go)
internal/event/event.go:28-100  -- Kind iota: append JobMessage at the END (never insert mid-list)
internal/control/controller.go:1399 -- slash switch adds /task-message route
```

### Cascade risks

| Risk | Level | Notes |
|------|-------|-------|
| **event.Kind iota value drift** | HIGH | Comments insist "Appended last to keep Kind values wire-stable" - new Kind MUST be appended after ExtensionSurface; inserting mid-list renumbers existing values and breaks frontend/persistence serialization |
| **Sub-agent ctx has no Manager** | MED | `RunSubAgentWithSession` runs `jobs.WithoutManager(ctx)` when `opts.Jobs == nil` (task.go:1803-1805); the sub-agent run loop cannot reach the Manager via `FromContext` -> the consumer MUST go through `jobCtxKey` (preserved), not the manager (verified: WithoutManager only shadows ctxKey, never jobCtxKey) |
| **Nested sub-agent ctx inherits jobCtxKey** | MED | A nested foreground task keeps the top jobCtxKey through the whole ctx chain, so a nested run loop can also drain the same queue -> race window (see adversarial check B1) |
| **P1 envelope format red line** | MED | The "unread messages at completion" hint MUST NOT alter the `<background-job-result>` envelope structure (preview_test.go:220 XML-escaping test locks it); only a Notice Detail addition is allowed |
| **Toolset change = one-time cache miss** | LOW | A new send_message tool changes the parent tool schema (one-time deployment change, stable afterwards); each sub-agent turn-boundary injection costs one cache miss (same semantics as existing steer, endorsed by run_loop.go:280-281 comment "unavoidable") |
| **Message prefix spoof misclassification** | LOW | JobMessageText/SteerText prefixes differ; round-trip must not cross-match (each uses exact CutPrefix + transient-language-block skip) |

---

## 2. Multi-path reasoning

### Path A: job-level pendingMessages queue + generic run-loop consumption (ADOPTED)

**Queue lives on Job; consumption at run-loop turn boundary; triggers via tool + slash.**

- Write: `Manager.SendMessageForSession(parentSession, id, message)` -> `m.get` (short m.mu critical section, existing pattern) -> `j.mu.Lock` validate Running/kind/limits -> append `j.pendingMessages` -> Unlock
- Consume: `jobs.DrainJobMessageFromContext(ctx)` (get `*Job` from `jobCtxKey`, no Manager needed) -> inject a user message after consumeSteer at run_loop.go:282
- Triggers: `send_message` builtin tool (parent model) + `/task-message <job_id> <text>` slash (user)
- Completion fallback: recordCompletion reads the unconsumed count inside j.mu -> Notice Detail hint (no envelope change)

### Path B: drainer func injected via ctx (REJECTED)

`runSubSession` does `ctx = jobs.WithMessageDrainer(ctx, func()...)`; run loop consumes via `MessageDrainerFromContext`.

- **Rejection**: a drainer bound to a job cannot distinguish top-level vs nested on the ctx chain - a nested foreground sub-agent inherits the same drainer, requiring an extra "clear" mechanism (`WithoutMessageDrainer`) and per-layer override logic, bloating the interface. Path A's existing `jobCtxKey` isolates per-job naturally (nested background jobs create a new Job and isolate automatically; nested foreground sharing the same job is the intended semantics) with zero new ctx keys.

### Path C: reuse the parent-agent steer queue (REJECTED)

Write job messages directly into the sub-agent `Agent.Steer()` queue.

- **Rejection**: the sub-agent `Agent` instance is created inside the `RunProfileSpec` job goroutine (task.go:924-958) and is unreachable from outside; `Steer` only accepts while `steerRunActive`, so it cannot queue outside the run window. Forcing a reference would expose the sub-agent Agent to the jobs package - breaking the jobs->agent dependency direction.

### Decision

**Adopt Path A.** Complexity concentrates in the jobs package (queue + one Manager method + two ctx helpers); the agent package adds only a 5-line injection plus message wrapping; trigger paths follow existing tool/command patterns; lock ordering reuses the P1 established pattern.

---

## 3. Final design

### 3.1 Trigger paths (requirement 1)

**Path 1 - tool (parent model)**: new builtin tool `send_message` (new file `internal/tool/builtin/job_message.go`, registration/permission pattern of bgjobs.go):

```go
func init() { tool.RegisterBuiltin(jobMessage{}) }
type jobMessage struct{}
// Name() "send_message"
// Description(): "Queue a message for a running background task (task-...). The
//   sub-agent consumes it at its next tool-round boundary as guidance, not a
//   new task. Fails when the job is not running or its queue is full."
// Schema: {"job_id": string(required), "message": string(required)}
// ReadOnly(): false        <- steering action, not read-only
// ProviderVisible(ctx): jobs.FromContext ok (parent visible)
// Execute: jm.SendMessageForSession(jobs.SessionFromContext(ctx), job_id, message)
//          -> success "Message queued for job <id>." / failure returns concrete error
```

- Sub-agent isolation: add `send_message` to `subagentAlwaysHiddenTools` (task.go:80-86) so child registries strip it unconditionally; ReadOnly=false is also filtered by `plannerExecutionRegistry`/`strictReadOnlyExecutionRegistry` (task.go:1910-1938, 1954-1973) as defense in depth.

**Path 2 - slash (user)**: add a case to the `controller.go:1399` switch:

```
/task-message <job_id> <message...>
```

- Parse: `fields := strings.Fields(trimmed)`; job_id=fields[1], message=join of the rest (spaces allowed).
- Execute: `c.jobs.SendMessageForSession(c.parentSessionID(), jobID, msg)` -> `c.notice(...)` feedback (success / unknown job / already terminal / queue full / missing args).
- Guard: `c.jobs != nil` (same guard as controller.go:2979).

**Path 3 (reserved, not built)**: desktop task panel buttons belong to P2; this plan only exposes the underlying API for the UI.

### 3.2 jobs queue implementation (requirement 2)

New fields on `Job` (guarded by the existing `j.mu`; no new lock):

```go
pendingMessages []string
droppedMessages int // unconsumed count at completion (fallback hint)
```

Constants (mirroring Qwen MAX_PENDING_MESSAGES backpressure):

```go
maxPendingJobMessages = 16         // queue length cap
maxPendingJobBytes    = 8 * 1024   // total queue byte cap
maxJobMessageBytes    = 2 * 1024   // per-message cap (rune-safe truncate or reject)
```

New `Manager` method:

```go
// SendMessageForSession delivers one instruction to a running task job. Backpressure:
// reject with a concrete error when the queue is full/over limit (never silently drop -
// an instruction, unlike a result, cannot be degraded).
// Lock order: m.get (short m.mu critical section, existing pattern) releases before
// j.mu is taken; the two locks never nest.
func (m *Manager) SendMessageForSession(parentSession, id, message string) error
```

Validation order (inside j.mu): job exists -> `status == Running` (terminal rejects) -> `Kind == "task"` (bash jobs have no consumer loop; explicit reject) -> single message <= maxJobMessageBytes -> queue count/total bytes not full -> append.

`Job` + ctx helpers:

```go
// DrainNextMessage pops the head (FIFO); only a short j.mu critical section.
// Consumers take one per loop iteration, same cadence as steers; multiple
// messages are injected across turns.
func (j *Job) DrainNextMessage() (string, bool)

// DrainJobMessageFromContext is for the run loop: get *Job via jobCtxKey then drain.
// Parent ctx has no jobCtxKey -> always false, behavior unchanged.
func DrainJobMessageFromContext(ctx context.Context) (string, bool)
```

**Completion fallback**: `recordCompletion` reads `len(j.pendingMessages)` inside its j.mu critical section; when non-zero, append `"N message(s) were not delivered before job finished"` to the closing Notice Detail. **No envelope change** (P1 red line). `droppedMessages` is recorded for a future P2 panel.

### 3.3 Sub-agent consumption injection point (requirement 3)

Append after consumeSteer at `run_loop.go:282` (same turn boundary):

```go
// Job messages (P3): a background task's own run loop consumes messages
// queued by the parent (send_message / /task-message). The parent agent has
// no job in its context, so this is a no-op there. One injection per loop
// iteration, same cadence as steers.
if text, ok := jobs.DrainJobMessageFromContext(ctx); ok {
    a.session.Add(provider.Message{
        Role: provider.RoleUser,
        Content: a.withTurnPreferences(jobMessageText(text)),
    })
    a.sink.Emit(event.Event{Kind: event.JobMessage, Text: text})
}
```

- **ctx chain (verified)**: the job goroutine's jobCtx carries jobCtxKey (StartForSession L460) -> runSession -> runSubSession -> `RunSubAgentWithSession` -> `sub.Run(jobCtx)` -> `runToolLoop(ctx = a.withAgentContext(jobCtx))`. `withAgentContext` (agent.go:146-161) only overwrites manager/memory/planmode values, **never jobCtxKey** -> consumption is reachable.
- **Does not break the sub-agent prefix**: injection appends via `a.session.Add` to the session tail (turn boundary, not history insertion, canonical untouched); the sub-agent stable head (system prompt + first task user message) stays byte-identical. Each injection costs one cache miss on the next sampling call (unavoidable, same semantics as steer).
- **Parent agent unaffected**: the parent ctx has no jobCtxKey -> constant no-op.

Message wrapping (agent.go, next to MidTurnSteerPrefix; the prefix deliberately differs from steer to prevent `SteerText` cross-matching):

```go
const JobMessagePrefix = "[Instruction from the parent agent. Do not treat this as a new task; use it only as additional guidance for the current task after completing the current step.]"
func jobMessageText(text string) string { return JobMessagePrefix + "\n" + text }
// JobMessageText mirrors SteerText: skip language-block wrappers, exact CutPrefix,
// trim the delivery marker.
func JobMessageText(content string) (string, bool)
```

`preview.go:288` `IsUserAuthoredTurn` adds the exclusion (so a job message never counts as a user turn for sub-agent titles / turn counts):

```go
if _, isJobMsg := JobMessageText(content); isJobMsg { return false }
```

`event.go` appends at the iota tail (after ExtensionSurface, wire-stable discipline):

```go
// JobMessage fires when a queued background-job message is consumed and
// injected into the job's run loop. Text carries the raw message (without the
// wrapper prefix). Appended last to keep the Kind values before it wire-stable.
JobMessage
```

### 3.4 P1 lock-ordering pattern reuse (requirement 4)

| Operation | Lock order | Verification |
|-----------|-----------|--------------|
| SendMessageForSession | `m.get` (m.mu) **released** then j.mu short section | Same pattern as P1 `recordCompletion` ("copy under j.mu, append under m.mu, the two critical sections never nest"), opposite direction but equally non-nested |
| DrainNextMessage | j.mu only | No m.mu involvement |
| recordCompletion fallback count | j.mu read (existing section) -> m.mu append (existing order) | No new lock order |
| Start initialization | No new lock (zero values suffice) | - |

`-race` verification: `go test ./internal/jobs/ -race` (covers the send/consume/completion three-way race).

---

## 4. Task breakdown (execution team)

| Task | Content | Files | Verification |
|------|---------|-------|--------------|
| **T1** | Job.pendingMessages + SendMessageForSession + DrainNextMessage + DrainJobMessageFromContext + constants + recordCompletion fallback count | `internal/jobs/jobs.go`, `internal/jobs/jobs_test.go` | `go test ./internal/jobs/ -race`; unit tests: delivery ok / queue-full reject (count+bytes dual backpressure) / terminal reject / unknown job / bash-kind reject / FIFO consume / unconsumed-count hint at completion |
| **T2** | run-loop injection + JobMessagePrefix/JobMessageText + IsUserAuthoredTurn exclusion + event.Kind=JobMessage (iota tail) | `internal/agent/run_loop.go`, `internal/agent/agent.go`, `internal/agent/preview.go`, `internal/event/event.go` | `go test ./internal/agent/ -run 'Steer|JobMessage'`; `go test ./internal/event/`; unit tests: round-trip (incl. language wrapper), parent no-op, IsUserAuthoredTurn exclusion, Kind value stability |
| **T3** | send_message tool registration (bgjobs.go pattern) + subagentAlwaysHiddenTools append | `internal/tool/builtin/job_message.go` (new), `internal/agent/task.go` | `go test ./internal/tool/...`; registry tests: visible in parent registry, invisible in child registry, ReadOnly()==false |
| **T4** | `/task-message <job_id> <text>` slash route + notice feedback | `internal/control/controller.go`, `internal/control/controller_test.go` | `go test ./internal/control/ -run 'TaskMessage'`; arg parsing / success / failure / unknown-job branches |
| **T5** | E2E: send_message while a background task runs -> sub-agent consumes next turn -> instruction takes effect | integration test (jobs + agent run loop combined) | targeted e2e passes: injected message appears in the sub-agent session and later rounds follow it |
| **T6** | Transaction status update | `docs/team/20260810-p3-steer/` | file exists, final plan cites this plan number |

Dependency order: T1 -> T2 -> (T3/T4 in parallel, both depend on T1) -> T5 (depends on T1-T3) -> T6.

---

## 5. Adversarial self-check (devil's advocate)

**B1 (highest risk) - nested foreground sub-agent consumption race**: while the top-level sub-agent waits for a nested task, a send_message may be consumed by the nested run loop (ctx chain inherits jobCtxKey). Mitigation: queue consumption is atomic (j.mu), each message is taken by exactly one run loop; the nested entity is the task's current worker, so the instruction still takes effect. Whether to add a "depth guard" (Job records top depth; consume only when depth matches) is left as an execution-time decision - jobs cannot import agent (dependency direction), the guard needs the agent package to pass the depth via a method; recommend T2 unit tests expose the race before deciding.
**B2 - message body not retained**: unconsumed messages at completion are only counted in the Notice; the body is lost and unrecoverable. Mitigation: differs from Qwen's "unread never dropped" philosophy, but this queue is in-process memory (same as P1); persistence is deferred to P6. Known limitation, flagged.
**B3 - backpressure reject vs drop**: reject with an error at queue-full - the parent model may ignore the error and continue, losing the message but the caller is informed. Identical to Qwen L570-577; adopted.
**B4 - JobMessageText round-trip through withTurnPreferences**: language-block wrappers + delivery marker require mirroring SteerText's stripping logic (trimLeadingTransientBlock + CutSuffix); T2 must include a wrapper-wrapped round-trip test, otherwise history replay misclassifies the message as an ordinary user turn.
**B5 - event.Kind values**: must append at the iota tail; T2 asserts JobMessage > ExtensionSurface (numerically) and existing Kind values unchanged.
**B6 - tool name collision**: verified by repo-wide grep; `send_message` exists only as private bot-package methods (not in the tool namespace); no conflict.
**B7 - read-only sub-agent exposure**: send_message ReadOnly()==false; double protection (AlwaysHidden + execution filter).

---

## 6. Cache / discipline checkpoints (Reasonix domain)

- **Outbound byte changes**: parent sees only a one-time toolset change (send_message schema; no runtime values in the schema, prefix stable); sub-agent turn-boundary job-message injection is a tail append (fixed position, no history insertion, canonical untouched, `ComposeSynthetic` never delivers - P1 discipline). OK
- **Prefix stability**: stable system/tool prefix unchanged; each job-message injection costs one cache miss (endorsed by the existing steer comment at run_loop.go:280-281 "unavoidable"). OK
- **Sub-agent prefix preserved**: injection appends at the run-loop turn boundary via `session.Add`, not history insertion. OK
- **Lock ordering**: m.mu and j.mu never nest (write side: m.get releases then j.mu; consume side: j.mu only; fallback side: P1's existing j.mu->m.mu order), `-race` verified. OK
- **Execution team does not overreach**: T1/T2 are disjoint (jobs queue vs agent injection); T3/T4 parallel but both depend on T1; T5 E2E depends on T1-T3. OK

<!-- P3 plan-3 ASCII rendering (CJK write path is broken in this sub-agent tool chain) -->
