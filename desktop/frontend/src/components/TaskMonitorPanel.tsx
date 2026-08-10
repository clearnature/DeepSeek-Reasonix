import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  AlertCircle,
  ChevronDown,
  ChevronRight,
  Clock,
  List,
  Loader2,
  RotateCw,
  X,
  XCircle,
} from "lucide-react";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import type { JobPanelView, TaskEvent, TaskSnapshot } from "../lib/types";

// --- helpers ---

// The panel renders a job tail up to this many UTF-8 bytes and marks the rest
// as truncated. The backend snapshot tail is already bounded at 4KiB; the 512B
// render cap keeps a large detail list cheap without stealing the model's
// wait/bash_output stream (P2 red line: panel reads are non-consuming).
const TAIL_RENDER_LIMIT_BYTES = 512;

const STATE_CONFIG: Record<
  string,
  { key: "queued" | "running" | "waiting" | "succeeded" | "failed" | "cancelled" | "stale" | "stalled"; color: string; dot: string }
> = {
  queued: { key: "queued", color: "#6b7280", dot: "⚪" },
  running: { key: "running", color: "#3b82f6", dot: "🔵" },
  waiting: { key: "waiting", color: "#f59e0b", dot: "🟡" },
  succeeded: { key: "succeeded", color: "#22c55e", dot: "🟢" },
  failed: { key: "failed", color: "#ef4444", dot: "🔴" },
  cancelled: { key: "cancelled", color: "#9ca3af", dot: "⏹️" },
  stale: { key: "stale", color: "#d4d4d8", dot: "⬜" },
  // stalled decorates a running job whose progress has not advanced; the color
  // is a deeper amber than the waiting badge so the two states stay distinct.
  stalled: { key: "stalled", color: "#d97706", dot: "⚠️" },
};

function stateConfig(state: string, t: ReturnType<typeof useT>) {
  const config = STATE_CONFIG[state];
  return config
    ? { ...config, label: t(`task.state.${config.key}` as never) }
    : { label: state, color: "#6b7280", dot: "❓" };
}

function runtimeConfig(state: string | undefined, t: ReturnType<typeof useT>) {
  switch (state) {
    case "alive":
      return { label: t("task.runtime.live"), color: "#22c55e" };
    case "exited":
      return { label: t("task.runtime.exited"), color: "#9ca3af" };
    default:
      return { label: t("task.runtime.unknown"), color: "#6b7280" };
  }
}

function safeStateClass(state: string): string {
  // Sanitize state for use in CSS class names — only allow word chars.
  return state.replace(/[^a-zA-Z0-9_-]/g, "_");
}

function elapsed(iso: string): string {
  if (!iso) return "—";
  const ms = Date.now() - new Date(iso).getTime();
  if (isNaN(ms) || ms < 0) return "—";
  const s = Math.floor(ms / 1000);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  return `${h}h`;
}

function shortID(id: string): string {
  return id.length > 8 ? id.slice(0, 8) : id;
}

function eventSummary(ev: TaskEvent, t: ReturnType<typeof useT>): string {
  if (ev.error_code) return t("task.event.error", { code: ev.error_code });
  switch (ev.event_type) {
    case "state_change":
      return t("task.event.stateChange", { state: ev.state, runtime: runtimeConfig(ev.runtime_state, t).label });
    case "error":
      return ev.error_summary || t("task.error");
    default:
      return ev.event_type;
  }
}

// isTerminalStatus matches both the task lifecycle states and the job lifecycle
// statuses. Terminal wins over any stalled decoration (P2: stalled only ever
// decorates a running job).
function isTerminalStatus(status: string): boolean {
  return (
    status === "succeeded" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "stale" ||
    status === "completed"
  );
}

function utf8ByteLength(s: string): number {
  let bytes = 0;
  for (let i = 0; i < s.length; i++) {
    const code = s.charCodeAt(i);
    if (code < 0x80) bytes += 1;
    else if (code < 0x800) bytes += 2;
    else if (code >= 0xd800 && code <= 0xdbff && i + 1 < s.length) {
      const low = s.charCodeAt(i + 1);
      if (low >= 0xdc00 && low <= 0xdfff) {
        bytes += 4;
        i += 1;
      } else bytes += 3;
    } else bytes += 3;
  }
  return bytes;
}

// boundedTail renders a job tail up to TAIL_RENDER_LIMIT_BYTES (byte-safe, so a
// multi-byte rune is never split) and reports whether it had to truncate.
function boundedTail(tail: string): { text: string; truncated: boolean } {
  if (!tail) return { text: "", truncated: false };
  if (utf8ByteLength(tail) <= TAIL_RENDER_LIMIT_BYTES) {
    return { text: tail, truncated: false };
  }
  let chars = tail.length;
  while (chars > 0 && utf8ByteLength(tail.slice(0, chars)) > TAIL_RENDER_LIMIT_BYTES) {
    chars -= 1;
  }
  return { text: tail.slice(0, chars), truncated: true };
}

// A panel row merges the jobs-first source with the store fallback: when a job
// snapshot matches a task's job_id the job decorates that task row; standalone
// jobs render as their own row. Jobs are authoritative when both exist.
interface PanelRow {
  key: string;
  task?: TaskSnapshot;
  job?: JobPanelView;
}

// --- component ---

const POLL_INTERVAL_MS = 5000;

export function TaskMonitorPanel({
	tabID,
  onClose,
  onOpenSession,
  initialOpen = false,
  popover = false,
  summaryMode = false,
  onJobAttention,
}: {
  tabID: string;
  onClose?: () => void;
  onOpenSession?: (tabID: string, taskID: string) => Promise<boolean> | boolean;
  initialOpen?: boolean;
  popover?: boolean;
  summaryMode?: boolean;
  // Wired by App to useController's notice: fired once per new stalled/failed
  // job seen by the 5s poll so the transcript stays informed (P2 T4).
  onJobAttention?: (job: JobPanelView) => void;
}) {
  const t = useT();
  const [tasks, setTasks] = useState<TaskSnapshot[]>([]);
  const [jobs, setJobs] = useState<JobPanelView[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const expandedRef = useRef<Set<string>>(new Set());
  const [open, setOpen] = useState(initialOpen);
  const [actionTask, setActionTask] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<{ task: TaskSnapshot; action: "stop" | "cancel" } | null>(null);

  // Per-task event state
  const [taskEvents, setTaskEvents] = useState<Map<string, TaskEvent[]>>(
    () => new Map(),
  );
  const [eventsLoading, setEventsLoading] = useState<Set<string>>(new Set());
  const [eventsError, setEventsError] = useState<Map<string, string>>(
    () => new Map(),
  );
  const eventCursors = useRef<Map<string, number>>(new Map());
  // Job ids already surfaced through onJobAttention, keyed by id:attention so a
  // job that unstalls and stalls again can notify once more, but a stable state
  // is not re-notified every 5s poll.
  const attentionNotified = useRef<Set<string>>(new Set());
  // Tracks whether an expanded row is a task (events polling) or a job (tail
  // refresh), so the 5s poll refreshes the right source for each expanded row.
  const expandedRowKind = useRef<Map<string, "task" | "job">>(new Map());

  const fetchTasks = useCallback(async () => {
    try {
      setError(null);
      const list = await app.ListTasksForTab(tabID);
      setTasks(list ?? []);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, [tabID]);

  const fetchJobs = useCallback(async () => {
    // The jobs surface is a best-effort P2 addition; browsers/mocks without the
    // bridge method keep rendering task rows from the store.
    if (typeof app.JobPanelJobsForTab !== "function") return;
    try {
      const list = await app.JobPanelJobsForTab(tabID);
      const next = list ?? [];
      setJobs(next);
      if (onJobAttention) {
        for (const job of next) {
          const attention =
            job.stalled === true && job.status === "running"
              ? "stalled"
              : job.status === "failed"
                ? "failed"
                : "";
          if (!attention) continue;
          const key = `${job.id}:${attention}`;
          if (attentionNotified.current.has(key)) continue;
          attentionNotified.current.add(key);
          onJobAttention(job);
        }
      }
    } catch {
      // Best-effort: the task rows still render from the store.
    }
  }, [tabID, onJobAttention]);

  const fetchJobOutput = useCallback(async (jobID: string) => {
    // Refreshes a job's detail tail from the bounded output bridge method. Also
    // best-effort: on failure the list-provided tail is kept.
    if (typeof app.JobOutputForTab !== "function") return;
    try {
      const out = await app.JobOutputForTab(tabID, jobID);
      if (out && out.id && out.output) {
        setJobs((prev) => prev.map((j) => (j.id === jobID ? { ...j, tail: out.output } : j)));
      }
    } catch {
      // keep the list tail
    }
  }, [tabID]);

  // Fetch events for a single task, using afterSequence for incremental load.
  const fetchEvents = useCallback(async (taskID: string) => {
    setEventsLoading((prev) => new Set(prev).add(taskID));
    setEventsError((prev) => {
      const next = new Map(prev);
      next.delete(taskID);
      return next;
    });
    try {
      const cursor = eventCursors.current.get(taskID) ?? 0;
      const events = await app.ListTaskEventsForTab(tabID, taskID, cursor);
      if (events.length > 0) {
        setTaskEvents((prev) => {
          const next = new Map(prev);
          const existing = next.get(taskID) ?? [];
          // Merge, deduplicate by sequence
          const seen = new Set(existing.map((e) => e.sequence));
          const merged = [...existing, ...events.filter((e) => !seen.has(e.sequence))];
          merged.sort((a, b) => a.sequence - b.sequence);
          next.set(taskID, merged);
          return next;
        });
        // Update cursor to the max sequence
        const maxSeq = events.reduce(
          (max, e) => Math.max(max, e.sequence),
          cursor,
        );
        eventCursors.current.set(taskID, maxSeq);
      }
    } catch (e) {
      setEventsError((prev) => {
        const next = new Map(prev);
        next.set(taskID, String(e));
        return next;
      });
    } finally {
      setEventsLoading((prev) => {
        const next = new Set(prev);
        next.delete(taskID);
        return next;
      });
    }
  }, [tabID]);

  // Initial fetch + periodic polling. While the panel is open both the task
  // store and the jobs snapshot poll every 5s; expanded rows refresh their own
  // source (events for tasks, bounded tail for jobs). `expanded` is read via a
  // ref so expanding a row does not re-run this effect: a re-run would refetch
  // the jobs list and overwrite the just-refreshed detail tail (P2 race).
  useEffect(() => {
    fetchTasks();
    fetchJobs();
    const interval = setInterval(() => {
      fetchTasks();
      fetchJobs();
      expandedRef.current.forEach((id) => {
        const kind = expandedRowKind.current.get(id);
        if (kind === "task") fetchEvents(id);
        else if (kind === "job") fetchJobOutput(id);
      });
    }, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [fetchTasks, fetchJobs, fetchEvents, fetchJobOutput]);

  const toggleRow = (row: PanelRow) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(row.key)) {
        next.delete(row.key);
        expandedRowKind.current.delete(row.key);
      } else {
        next.add(row.key);
        expandedRowKind.current.set(row.key, row.task ? "task" : "job");
        // Load events on first expand; refresh the job detail tail on open.
        if (row.task && !taskEvents.has(row.task.task_id)) {
          fetchEvents(row.task.task_id);
        }
        if (row.job) fetchJobOutput(row.job.id);
      }
      expandedRef.current = next;
      return next;
    });
  };

  const controlTask = async (task: TaskSnapshot, action: "stop" | "cancel" | "requeue" | "open") => {
    if ((action === "stop" || action === "cancel") && (!pendingAction || pendingAction.task.task_id !== task.task_id || pendingAction.action !== action)) {
      setPendingAction({ task, action });
      return;
    }
    setPendingAction(null);
    setActionTask(task.task_id);
    setActionError(null);
    setActionMessage(null);
    try {
      if (action === "open" && onOpenSession) {
        const opened = await onOpenSession(tabID, task.task_id);
        if (opened) onClose?.();
        return;
      }
      const result = action === "stop"
        ? await app.StopTaskForTab(tabID, task.task_id, task.version, "desktop request", `desktop-${action}-${task.task_id}-${task.version}`)
        : action === "cancel"
          ? await app.CancelTaskForTab(tabID, task.task_id, task.version, "desktop request", `desktop-${action}-${task.task_id}-${task.version}`)
          : action === "requeue"
            ? await app.RequeueTaskForTab(tabID, task.task_id, task.version, `desktop-${action}-${task.task_id}-${task.version}`)
            : await app.OpenTaskSessionForTab(tabID, task.task_id);
      if (result.error) {
        setActionError(`${result.error.code}: ${result.error.message}`);
      } else if (action === "open") {
        const sessionID = result.session_id?.trim();
        if (!sessionID) throw new Error("Task session is unavailable");
        setActionMessage(`Session: ${sessionID}`);
      } else {
        setActionMessage(result.idempotent ? "Already applied" : "Task updated");
        await fetchTasks();
      }
    } catch (e) {
      setActionError(String(e));
    } finally {
      setActionTask(null);
    }
  };

  // Merge the jobs snapshot (authoritative when it matches) with the task store
  // (fallback). Standalone jobs render as their own rows.
  const jobByID = useMemo(() => {
    const m = new Map<string, JobPanelView>();
    for (const j of jobs) m.set(j.id, j);
    return m;
  }, [jobs]);

  const rows = useMemo<PanelRow[]>(() => {
    const taskRows: PanelRow[] = tasks.map((task) => ({
      key: task.task_id,
      task,
      job: task.job_id ? jobByID.get(task.job_id) : undefined,
    }));
    const matchedJobIDs = new Set<string>();
    for (const r of taskRows) if (r.job) matchedJobIDs.add(r.job.id);
    const jobRows: PanelRow[] = [];
    for (const j of jobs) {
      if (!matchedJobIDs.has(j.id)) jobRows.push({ key: j.id, job: j });
    }
    taskRows.sort(
      (a, b) =>
        new Date(b.task!.updated_at).getTime() - new Date(a.task!.updated_at).getTime(),
    );
    return [...taskRows, ...jobRows];
  }, [tasks, jobByID]);

  return (
    <div className={`taskmonitor${popover ? " taskmonitor--popover" : ""}`}>
      <div className="taskmonitor__head">
        <button
          className="taskmonitor__toggle"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-label={open ? "Collapse tasks" : "Expand tasks"}
        >
          {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        </button>
        <span className="taskmonitor__title">{summaryMode ? t("summary.session") : t("summary.tasks")}</span>
        <span className="taskmonitor__count">{rows.length}</span>
        <button
          className="taskmonitor__refresh"
          onClick={() => {
            setLoading(true);
            fetchTasks();
            fetchJobs();
          }}
          title={t("summary.refresh")}
          aria-label={t("summary.refresh")}
        >
          <RotateCw size={12} />
        </button>
        {onClose && (
          <button
            className="taskmonitor__close"
            onClick={onClose}
            title={t("common.close")}
            aria-label={t("summary.close")}
          >
            <X size={14} />
          </button>
        )}
      </div>

      {open && (
        <div className="taskmonitor__body">
          {summaryMode && <div className="taskmonitor__category-title">{t("summary.tasks")}</div>}
          {actionError && <div className="taskmonitor__state taskmonitor__state--error">{actionError}</div>}
          {actionMessage && <div className="taskmonitor__state">{actionMessage}</div>}
          {loading && (
            <div className="taskmonitor__state">
              <Loader2 size={16} className="taskmonitor__spinner" />
              <span>{t("common.loading")}</span>
            </div>
          )}

          {error && (
            <div className="taskmonitor__state taskmonitor__state--error">
              <AlertCircle size={16} />
              <span>{error}</span>
            </div>
          )}

          {!loading && !error && rows.length === 0 && (
            <div className="taskmonitor__state taskmonitor__state--empty">
              <Clock size={16} />
              <span>{t("summary.noTasks")}</span>
            </div>
          )}

          {!loading &&
            rows.map((row) => {
              const task = row.task;
              const job = row.job;
              const isJobOnly = !task;
              const state = task ? task.state : job?.status ?? "";
              const cfg = stateConfig(state, t);
              const runtime = runtimeConfig(task?.runtime_state, t);
              const isOpen = expanded.has(row.key);
              const terminal = task ? isTerminalStatus(task.state) : isTerminalStatus(job?.status ?? "");
              // stalled decorates a running job only; terminal always wins.
              const showStalled = job?.stalled === true && job.status === "running" && !isTerminalStatus(task?.state ?? "");
              const evs = task ? taskEvents.get(task.task_id) ?? [] : [];
              const evLoading = task ? eventsLoading.has(task.task_id) : false;
              const evError = task ? eventsError.get(task.task_id) : undefined;
              const tailInfo = boundedTail(job?.tail ?? "");

              return (
                <div
                  key={row.key}
                  className={`taskmonitor__task taskmonitor__task--${safeStateClass(state)}`}
                >
                  <div className="taskmonitor__task-head">
                    <button
                      className="taskmonitor__expand"
                      onClick={() => toggleRow(row)}
                      aria-expanded={isOpen}
                      aria-label={
                        isJobOnly
                          ? t("summary.jobLabelRow", { id: shortID(job!.id), state: cfg.label })
                          : t("summary.taskLabel", { id: shortID(task!.task_id), state: cfg.label })
                      }
                    >
                      <span
                        className="taskmonitor__dot"
                        style={{ color: cfg.color }}
                      >
                        {cfg.dot}
                      </span>
                      <span className="taskmonitor__id">
                        {shortID(row.key)}
                      </span>
                      <span
                        className="taskmonitor__badge"
                        style={{
                          backgroundColor: cfg.color + "18",
                          color: cfg.color,
                        }}
                      >
                        {cfg.label}
                      </span>
                      {showStalled && (
                        <span
                          className="taskmonitor__badge taskmonitor__badge--stalled"
                          style={{
                            backgroundColor: STATE_CONFIG.stalled.color + "18",
                            color: STATE_CONFIG.stalled.color,
                          }}
                          title={t("summary.jobStalled")}
                        >
                          {STATE_CONFIG.stalled.dot} {t("summary.jobStalled")}
                        </span>
                      )}
                      {!isJobOnly && (
                        <span
                          className="taskmonitor__runtime"
                          style={{ color: runtime.color }}
                          title="Runtime process state"
                        >
                          <span aria-hidden="true">{task!.runtime_state === "alive" ? "●" : "○"}</span>
                          {runtime.label}
                        </span>
                      )}
                      {terminal && (
                        <XCircle size={12} className="taskmonitor__terminal" />
                      )}
                      <span className="taskmonitor__time">
                        {isJobOnly ? "—" : elapsed(task!.updated_at)}
                      </span>
                      {isOpen ? (
                        <ChevronDown size={12} />
                      ) : (
                        <ChevronRight size={12} />
                      )}
                    </button>
                  </div>

                  {isOpen && (
                    <div className="taskmonitor__detail">
                      <dl>
                        {!isJobOnly && task && (
                          <>
                            <dt>{t("summary.taskId")}</dt>
                            <dd>{task.task_id}</dd>
                            <dt>{t("summary.sessionId")}</dt>
                            <dd>{task.session_id || "—"}</dd>
                            <dt>{t("summary.state")}</dt>
                            <dd>{task.state}</dd>
                            <dt>{t("summary.runtime")}</dt>
                            <dd>{runtime.label}</dd>
                            <dt>{t("summary.updated")}</dt>
                            <dd>{new Date(task.updated_at).toLocaleString()}</dd>
                            {task.error_code && (
                              <>
                                <dt>{t("summary.errorCode")}</dt>
                                <dd className="taskmonitor__err">{task.error_code}</dd>
                              </>
                            )}
                            {task.error_summary && (
                              <>
                                <dt>{t("summary.detail")}</dt>
                                <dd className="taskmonitor__err-summary">
                                  {task.error_summary}
                                </dd>
                              </>
                            )}
                          </>
                        )}
                        {job && (
                          <>
                            {isJobOnly && (
                              <>
                                <dt>{t("summary.jobId")}</dt>
                                <dd>{job.id}</dd>
                                <dt>{t("summary.state")}</dt>
                                <dd>{job.status}</dd>
                              </>
                            )}
                            <dt>{t("summary.jobKind")}</dt>
                            <dd>
                              <span className="taskmonitor__kind-badge">{job.kind || "job"}</span>
                            </dd>
                            {job.label && (
                              <>
                                <dt>{t("summary.jobLabel")}</dt>
                                <dd>{job.label}</dd>
                              </>
                            )}
                            <dt>{t("summary.tail")}</dt>
                            <dd
                              className={`taskmonitor__tail${tailInfo.truncated ? " taskmonitor__tail--truncated" : ""}`}
                              title={tailInfo.truncated ? t("summary.tailTruncated") : undefined}
                            >
                              {tailInfo.text ? <pre>{tailInfo.text}</pre> : "—"}
                              {tailInfo.truncated && (
                                <span className="taskmonitor__tail-trunc">{t("summary.tailTruncated")}</span>
                              )}
                            </dd>
                          </>
                        )}
                      </dl>

                      {!isJobOnly && task && (
                        <>
                          {/* Events section */}
                          <div className="taskmonitor__events">
                            <div className="taskmonitor__events-head">
                              <List size={12} />
                              <span>{t("summary.recentEvents")}</span>
                              {evs.length > 0 && (
                                <span className="taskmonitor__events-count">
                                  {evs.length}
                                </span>
                              )}
                            </div>

                            {evLoading && evs.length === 0 && (
                              <div className="taskmonitor__state">
                                <Loader2
                                  size={12}
                                  className="taskmonitor__spinner"
                                />
                                <span>{t("summary.loadingEvents")}</span>
                              </div>
                            )}

                            {evError && (
                              <div className="taskmonitor__state taskmonitor__state--error">
                                <AlertCircle size={12} />
                                <span>{evError}</span>
                              </div>
                            )}

                            {!evLoading && !evError && evs.length === 0 && (
                              <div className="taskmonitor__state taskmonitor__state--empty">
                                <span>{t("summary.noEvents")}</span>
                              </div>
                            )}

                            {evs.length > 0 && (
                              <ul className="taskmonitor__event-list">
                                {evs.map((ev) => (
                                  <li
                                    key={ev.sequence}
                                    className="taskmonitor__event"
                                  >
                                    <span className="taskmonitor__event-seq">
                                      #{ev.sequence}
                                    </span>
                                    <span className="taskmonitor__event-type">
                                      {eventSummary(ev, t)}
                                    </span>
                                    <span className="taskmonitor__event-time">
                                      {new Date(ev.timestamp).toLocaleTimeString()}
                                    </span>
                                  </li>
                                ))}
                              </ul>
                            )}
                          </div>
                          <div className="taskmonitor__actions">
                            {(task.state === "queued" || task.state === "running" || task.state === "waiting") && (
                              <>
                                <button disabled={actionTask === task.task_id} onClick={() => void controlTask(task, "stop")}>{t("summary.stop")}</button>
                                <button disabled={actionTask === task.task_id} onClick={() => void controlTask(task, "cancel")}>{t("summary.cancel")}</button>
                              </>
                            )}
                            {(task.state === "failed" || task.state === "stale") && (
                              <button disabled={actionTask === task.task_id || task.runtime_state === "alive"} onClick={() => void controlTask(task, "requeue")}>{t("summary.requeue")}</button>
                            )}
                            <button disabled={actionTask === task.task_id} onClick={() => void controlTask(task, "open")}>{t("summary.openSession")}</button>
                          </div>
                          {pendingAction?.task.task_id === task.task_id && (
                            <div className="taskmonitor__confirm">
                              <span>{t(pendingAction.action === "stop" ? "summary.confirmStop" : "summary.confirmCancel")}</span>
                              <button type="button" onClick={() => void controlTask(task, pendingAction.action)}>{t("common.confirm")}</button>
                              <button type="button" onClick={() => setPendingAction(null)}>{t("summary.keep")}</button>
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
        </div>
      )}
    </div>
  );
}
