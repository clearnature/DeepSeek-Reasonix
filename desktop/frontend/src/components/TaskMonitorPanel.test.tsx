// Run: tsx src/components/TaskMonitorPanel.test.tsx

// The panel's locale is detected from the ambient navigator.language; pin it
// to en so assertions on English labels stay stable regardless of host locale.
Object.defineProperty(globalThis.navigator, "language", {
  value: "en-US",
  configurable: true,
});

import { JSDOM } from "jsdom";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { LocaleProvider } from "../lib/i18n";

type Task = Record<string, unknown>;
type Event = Record<string, unknown>;

let passed = 0;
let failed = 0;

function ok(value: boolean, label: string) {
  if (value) {
    process.stdout.write(`  PASS  ${label}\n`);
    passed += 1;
  } else {
    process.stdout.write(`  FAIL  ${label}\n`);
    failed += 1;
  }
}

function snap(overrides: Task = {}): Task {
  return {
    schema_version: 1,
    task_id: "task-0001",
    session_id: "sess-1",
    state: "running",
    runtime_state: "alive",
    version: 1,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T01:00:00Z",
    ...overrides,
  };
}

function taskEvent(overrides: Event = {}): Event {
  return {
    sequence: 1,
    timestamp: "2025-01-01T00:00:01Z",
    event_type: "state_change",
    task_id: "task-0001",
    session_id: "sess-1",
    state: "running",
    runtime_state: "alive",
    ...overrides,
  };
}

const dom = new JSDOM("<!doctype html><html><body></body></html>", {
  pretendToBeVisual: true,
  url: "http://localhost/",
});
(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
globalThis.window = dom.window as unknown as Window & typeof globalThis;
globalThis.document = dom.window.document;
globalThis.Node = dom.window.Node;
globalThis.Element = dom.window.Element;
globalThis.HTMLElement = dom.window.HTMLElement;
globalThis.SVGElement = dom.window.SVGElement;
globalThis.Event = dom.window.Event;
globalThis.MouseEvent = dom.window.MouseEvent;
globalThis.requestAnimationFrame = dom.window.requestAnimationFrame.bind(dom.window);
globalThis.cancelAnimationFrame = dom.window.cancelAnimationFrame.bind(dom.window);

let listTasksImpl: () => Promise<Task[]> = async () => [];
let listEventsImpl: () => Promise<Event[]> = async () => [];
let listJobsImpl: () => Promise<Task[]> = async () => [];
let jobOutputImpl: (jobID: string) => Promise<Task> = async () => ({ id: "", output: "" });
const listTaskTabIDs: string[] = [];
const listEventCalls: unknown[][] = [];
const requeueCalls: unknown[][] = [];
const listJobTabIDs: string[] = [];
const jobOutputCalls: unknown[][] = [];
const mockApp = {
  ListTasks: () => listTasksImpl(),
  GetTask: async () => null,
  ListTaskEvents: () => listEventsImpl(),
  StopTask: async () => ({ schema_version: 1, command: "stop", task_id: "", accepted: true, idempotent: false }),
  CancelTask: async () => ({ schema_version: 1, command: "cancel", task_id: "", accepted: true, idempotent: false }),
  RequeueTask: async (...args: unknown[]) => {
    requeueCalls.push(args);
    return {
      schema_version: 1,
      command: "requeue",
      task_id: String(args[0] ?? ""),
      state: "queued",
      runtime_state: "exited",
      version: 2,
      accepted: true,
      idempotent: false,
    };
  },
  OpenTaskSession: async () => ({ schema_version: 1, command: "open_session", task_id: "", session_id: "sess-1", accepted: true, idempotent: false }),
  ListTasksForTab: async (tabID: string) => {
    listTaskTabIDs.push(tabID);
    return listTasksImpl();
  },
  ListTaskEventsForTab: async (...args: unknown[]) => {
    listEventCalls.push(args);
    return listEventsImpl();
  },
  JobPanelJobsForTab: async (tabID: string) => {
    listJobTabIDs.push(tabID);
    return listJobsImpl();
  },
  JobOutputForTab: async (...args: unknown[]) => {
    jobOutputCalls.push(args);
    return jobOutputImpl(String(args[1] ?? ""));
  },
  StopTaskForTab: async () => ({ schema_version: 1, command: "stop", task_id: "", accepted: true, idempotent: false }),
  CancelTaskForTab: async () => ({ schema_version: 1, command: "cancel", task_id: "", accepted: true, idempotent: false }),
  RequeueTaskForTab: async (...args: unknown[]) => {
    requeueCalls.push(args);
    return {
      schema_version: 1,
      command: "requeue",
      task_id: String(args[1] ?? ""),
      state: "queued",
      runtime_state: "exited",
      version: 2,
      accepted: true,
      idempotent: false,
    };
  },
  OpenTaskSessionForTab: async () => ({ schema_version: 1, command: "open_session", task_id: "", session_id: "sess-1", accepted: true, idempotent: false }),
};
(window as unknown as { go: { main: { App: typeof mockApp } } }).go = { main: { App: mockApp } };

const { TaskMonitorPanel } = await import("./TaskMonitorPanel");

let activeRoot: Root | null = null;
let activeHost: HTMLElement | null = null;

async function flush() {
  await new Promise((resolve) => setTimeout(resolve, 25));
}

async function renderPanel(
  onClose?: () => void,
  onOpenSession?: (tabID: string, taskID: string) => Promise<boolean> | boolean,
  tabID = "tab-a",
  onJobAttention?: (job: unknown) => void,
) {
  activeHost = document.createElement("div");
  document.body.appendChild(activeHost);
  activeRoot = createRoot(activeHost);
  await act(async () => {
    activeRoot?.render(
      <LocaleProvider>
        <TaskMonitorPanel
          tabID={tabID}
          onClose={onClose}
          onOpenSession={onOpenSession}
          onJobAttention={onJobAttention}
        />
      </LocaleProvider>,
    );
    await flush();
  });
}

async function cleanup() {
  if (activeRoot) {
    await act(async () => activeRoot?.unmount());
  }
  activeHost?.remove();
  activeRoot = null;
  activeHost = null;
  listTasksImpl = async () => [];
  listEventsImpl = async () => [];
  listJobsImpl = async () => [];
  jobOutputImpl = async () => ({ id: "", output: "" });
  listTaskTabIDs.length = 0;
  listEventCalls.length = 0;
  requeueCalls.length = 0;
  listJobTabIDs.length = 0;
  jobOutputCalls.length = 0;
}

function buttonByLabel(label: string): HTMLButtonElement {
  const button = Array.from(document.querySelectorAll<HTMLButtonElement>("button"))
    .find((candidate) => candidate.getAttribute("aria-label") === label);
  if (!button) { process.stderr.write(`MISSING [${label}]\nBODY: ${document.body.textContent?.slice(0, 800)}\n`); throw new Error(`missing button: ${label}`); }
  return button;
}

function buttonByText(label: string): HTMLButtonElement {
  const button = Array.from(document.querySelectorAll<HTMLButtonElement>("button"))
    .find((candidate) => candidate.textContent?.trim() === label);
  if (!button) throw new Error(`missing button text: ${label}`);
  return button;
}

async function click(button: HTMLButtonElement) {
  await act(async () => {
    button.click();
    await flush();
  });
}

async function openPanel() {
  await click(buttonByLabel("Expand tasks"));
}

async function check(label: string, run: () => Promise<boolean>) {
  try {
    ok(await run(), label);
  } catch (error) {
    process.stderr.write(`  ERROR ${label}: ${String(error)}\n`);
    ok(false, label);
  } finally {
    await cleanup();
  }
}

console.log("\nTask Monitor panel");

await check("renders the panel header", async () => {
  await renderPanel();
  return document.body.textContent?.includes("Tasks") === true;
});

await check("shows the empty state", async () => {
  await renderPanel();
  await openPanel();
  return document.body.textContent?.includes("No background tasks") === true;
});

await check("shows task-fetch errors", async () => {
  listTasksImpl = async () => { throw new Error("Network error"); };
  await renderPanel();
  await openPanel();
  return document.body.textContent?.includes("Network error") === true;
});

await check("binds task reads to the source tab", async () => {
  listTasksImpl = async () => [snap({ session_id: "sess-current" })];
  await renderPanel(undefined, undefined, "tab-source");
  return listTaskTabIDs.length === 1 && listTaskTabIDs[0] === "tab-source";
});

await check("renders lifecycle badges", async () => {
  listTasksImpl = async () => [snap({ task_id: "a1" }), snap({ task_id: "b2", state: "failed" })];
  await renderPanel();
  await openPanel();
  const text = document.body.textContent ?? "";
  return text.includes("Running") && text.includes("Failed");
});

await check("separates lifecycle state from runtime liveness", async () => {
  listTasksImpl = async () => [
    snap({ task_id: "failed-1", state: "failed", runtime_state: "exited" }),
    snap({ task_id: "legacy-1", runtime_state: undefined }),
  ];
  await renderPanel();
  await openPanel();
  const text = document.body.textContent ?? "";
  return text.includes("Exited") && text.includes("Runtime unknown");
});

await check("requeues failed exited tasks", async () => {
  listTasksImpl = async () => [snap({ task_id: "failed-1", state: "failed", runtime_state: "exited", version: 7 })];
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Task failed-1 — Failed"));
  await click(buttonByText("Requeue"));
  return JSON.stringify(requeueCalls[0]) === JSON.stringify(["tab-a", "failed-1", 7, "desktop-requeue-failed-1-7"]);
});

await check("expands and collapses task details", async () => {
  listTasksImpl = async () => [snap({ state: "succeeded" })];
  await renderPanel();
  await openPanel();
  const row = buttonByLabel("Task task-000 — Succeeded");
  await click(row);
  const expanded = document.body.textContent?.includes("Task ID") === true;
  await click(row);
  return expanded && document.body.textContent?.includes("Task ID") !== true;
});

await check("loads recent task events", async () => {
  listTasksImpl = async () => [snap({ state: "failed" })];
  listEventsImpl = async () => [taskEvent({ event_type: "error", error_code: "CRASH" })];
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Task task-000 — Failed"));
  return document.body.textContent?.includes("CRASH") === true
    && JSON.stringify(listEventCalls[0]) === JSON.stringify(["tab-a", "task-0001", 0]);
});

await check("shows task-event errors", async () => {
  listTasksImpl = async () => [snap()];
  listEventsImpl = async () => { throw new Error("Event failure"); };
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Task task-000 — Running"));
  return document.body.textContent?.includes("Event failure") === true;
});

await check("calls the close callback", async () => {
  let closeCalls = 0;
  await renderPanel(() => { closeCalls += 1; });
  await click(buttonByLabel("Close session summary"));
  return closeCalls === 1;
});

await check("opens the task session through the navigation callback", async () => {
  listTasksImpl = async () => [snap()];
  let openedTarget: string[] = [];
  await renderPanel(undefined, async (tabID, taskID) => {
    openedTarget = [tabID, taskID];
    return true;
  });
  await openPanel();
  await click(buttonByLabel("Task task-000 — Running"));
  await click(buttonByText("Open session"));
  return JSON.stringify(openedTarget) === JSON.stringify(["tab-a", "task-0001"]);
});

await check("does not close the panel for a stale open completion", async () => {
  listTasksImpl = async () => [snap()];
  let closeCalls = 0;
  await renderPanel(() => { closeCalls += 1; }, async () => false);
  await openPanel();
  await click(buttonByLabel("Task task-000 — Running"));
  await click(buttonByText("Open session"));
  return closeCalls === 0;
});

await check("refreshes tasks on request", async () => {
  let calls = 0;
  listTasksImpl = async () => (++calls === 1 ? [] : [snap({ task_id: "ok" })]);
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Refresh"));
  return document.body.textContent?.includes("ok") === true;
});

await check("shows the task count", async () => {
  listTasksImpl = async () => [snap({ task_id: "a" }), snap({ task_id: "b" })];
  await renderPanel();
  return document.querySelector(".taskmonitor__count")?.textContent === "2";
});

await check("marks only terminal tasks", async () => {
  listTasksImpl = async () => [snap({ task_id: "t1", state: "succeeded" }), snap({ task_id: "t2" })];
  await renderPanel();
  await openPanel();
  return document.querySelectorAll(".taskmonitor__terminal").length === 1;
});

// ── P2 T4: job panel rows (kind badge / label / tail) ──

function job(overrides: Task = {}): Task {
  return {
    id: "job-1",
    kind: "bash",
    label: "tests",
    status: "running",
    stalled: false,
    tail: "line1",
    ...overrides,
  };
}

await check("binds job reads to the source tab", async () => {
  listJobsImpl = async () => [job()];
  await renderPanel(undefined, undefined, "tab-source");
  return listJobTabIDs.length === 1 && listJobTabIDs[0] === "tab-source";
});

await check("renders job rows with kind badge, label, and tail", async () => {
  listJobsImpl = async () => [job({ id: "job-1", kind: "bash", label: "tests", status: "running", tail: "line1" })];
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Job job-1 — Running"));
  const text = document.body.textContent ?? "";
  return text.includes("Kind") && text.includes("bash") && text.includes("tests")
    && text.includes("Tail") && text.includes("line1");
});

await check("truncates long job tails at 512 bytes with a marker", async () => {
  listJobsImpl = async () => [job({ tail: "x".repeat(600) })];
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Job job-1 — Running"));
  const pre = document.querySelector(".taskmonitor__tail pre");
  const marker = document.querySelector(".taskmonitor__tail--truncated");
  return pre?.textContent?.length === 512
    && Boolean(marker)
    && (document.body.textContent ?? "").includes("Tail truncated to 512 bytes");
});

await check("refreshes the detail tail from JobOutputForTab on expand", async () => {
  listJobsImpl = async () => [job({ tail: "initial" })];
  jobOutputImpl = async () => ({ id: "job-1", output: "refreshed-output" });
  await renderPanel();
  await openPanel();
  await click(buttonByLabel("Job job-1 — Running"));
  await act(async () => { await new Promise((r) => setTimeout(r, 60)); });
  return (document.body.textContent ?? "").includes("refreshed-output")
    && JSON.stringify(jobOutputCalls[0]) === JSON.stringify(["tab-a", "job-1"]);
});

await check("marks stalled running jobs only, terminal wins", async () => {
  listJobsImpl = async () => [
    job({ id: "j-run", kind: "bash", label: "stalled", status: "running", stalled: true, tail: "" }),
    job({ id: "j-done", kind: "test", label: "done", status: "completed", stalled: true, tail: "" }),
  ];
  await renderPanel();
  await openPanel();
  const badges = document.querySelectorAll(".taskmonitor__badge--stalled");
  return badges.length === 1 && badges[0]?.textContent?.includes("Stalled") === true;
});

await check("fires job attention once per stalled job across polls", async () => {
  const attention: unknown[] = [];
  listJobsImpl = async () => [job({ id: "job-1", status: "running", stalled: true, tail: "" })];
  await renderPanel(undefined, undefined, "tab-a", (j) => attention.push(j));
  await openPanel();
  await click(buttonByLabel("Refresh"));
  return attention.length === 1
    && (attention[0] as Record<string, unknown>)?.id === "job-1";
});

await check("fires job attention for failed jobs", async () => {
  const attention: unknown[] = [];
  listJobsImpl = async () => [job({ id: "job-2", status: "failed", stalled: false, tail: "" })];
  await renderPanel(undefined, undefined, "tab-a", (j) => attention.push(j));
  await openPanel();
  return attention.length === 1 && (attention[0] as Record<string, unknown>)?.id === "job-2";
});

dom.window.close();
console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
