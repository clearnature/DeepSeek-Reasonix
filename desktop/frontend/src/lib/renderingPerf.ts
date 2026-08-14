// RenderingPerf — longtask aggregation for jank/flicker diagnostics.
// The frontend samples PerformanceObserver("longtask") per 60s window and
// reports one aggregate sample (count / max / avg) to the stats file, so a
// "janky" complaint is diagnosable after the fact instead of subjective.

import { app } from "./bridge";

const REPORT_INTERVAL_MS = 60_000;

let samples = 0;
let maxMs = 0;
let totalMs = 0;

function flush(): void {
  if (samples === 0) return;
  const avg = Math.round(totalMs / samples);
  void app.ReportRenderingPerf(samples, Math.round(maxMs), avg);
  samples = 0;
  maxMs = 0;
  totalMs = 0;
}

export function startRenderingPerf(): void {
  if (typeof PerformanceObserver === "undefined") return;
  let observer: PerformanceObserver;
  try {
    observer = new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) {
        samples += 1;
        const ms = entry.duration;
        if (ms > maxMs) maxMs = ms;
        totalMs += ms;
      }
    });
    observer.observe({ entryTypes: ["longtask"] });
  } catch {
    return;
  }
  window.setInterval(flush, REPORT_INTERVAL_MS);
  // Flush on pagehide so a short-lived window still leaves a sample.
  window.addEventListener("pagehide", flush, { once: true });
}
