import { useCallback, useEffect, useRef, useState } from "react";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";

// P12 team panel: a live non-consuming projection of the tab's team —
// roster (identity/role/state/write posture) plus pending plan/tool
// approvals. Polled like the task monitor (best-effort; absent bridge or
// missing team renders an empty state).

export interface TeamPanelView {
  roster: Array<{
    name: string;
    role: string;
    state: string;
    writable: boolean;
    worktree: boolean;
    token: boolean;
    job_id?: string;
  }>;
  approvals: Array<{
    request_id: string;
    teammate: string;
    kind?: string;
    plan?: string;
    at?: number;
  }>;
}

export function TeamPanel({ tabID }: { tabID: string }) {
  const t = useT();
  const [view, setView] = useState<TeamPanelView | null>(null);
  const timerRef = useRef<number | null>(null);

  const fetchView = useCallback(async () => {
    if (typeof (app as any).TeamPanelViewForTab !== "function") {
      return;
    }
    try {
      const v = await (app as any).TeamPanelViewForTab(tabID);
      setView(v ?? { roster: [], approvals: [] });
    } catch {
      // best-effort: keep the previous view
    }
  }, [tabID]);

  useEffect(() => {
    void fetchView();
    timerRef.current = window.setInterval(() => void fetchView(), 3000);
    return () => {
      if (timerRef.current !== null) window.clearInterval(timerRef.current);
    };
  }, [fetchView]);

  const roster = view?.roster ?? [];
  const approvals = view?.approvals ?? [];

  return (
    <div className="teampanel" role="region" aria-label="Team panel">
      <div className="teampanel-header">
        <strong>{t("team.panelTitle")}</strong>
        <span className="teampanel-meta">
          {roster.length > 0 ? `${roster.length} ${t("team.members")}` : t("team.empty")}
        </span>
      </div>

      {roster.length > 0 && (
        <table className="teampanel-roster">
          <thead>
            <tr>
              <th>{t("team.name")}</th>
              <th>{t("team.role")}</th>
              <th>{t("team.state")}</th>
              <th>{t("team.posture")}</th>
              <th>{t("team.job")}</th>
            </tr>
          </thead>
          <tbody>
            {roster.map((m) => (
              <tr key={m.name}>
                <td>{m.name}</td>
                <td>{m.role || "—"}</td>
                <td>
                  <span className={`teampanel-state ${m.state}`}>{m.state}</span>
                </td>
                <td>
                  {m.worktree
                    ? t("team.worktree")
                    : m.writable
                      ? t("team.writable")
                      : m.token
                        ? t("team.token")
                        : t("team.readonly")}
                </td>
                <td>{m.job_id || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {approvals.length > 0 && (
        <div className="teampanel-approvals">
          <div className="teampanel-subhead">
            {t("team.pendingApprovals")} ({approvals.length})
          </div>
          {approvals.map((a) => (
            <div key={a.request_id} className="teampanel-approval">
              <code>{a.request_id}</code>
              <span className="teampanel-approval-from">{a.teammate}</span>
              <span className={`teampanel-approval-kind ${a.kind ?? "plan"}`}>{a.kind ?? "plan"}</span>
              <p className="teampanel-approval-plan">{a.plan ?? ""}</p>
            </div>
          ))}
          <p className="teampanel-hint">
            {t("team.approveHint")}
          </p>
        </div>
      )}
    </div>
  );
}
