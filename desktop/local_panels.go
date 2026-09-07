package main

// Local desktop panel bindings re-injected after the v1.38 topology merge
// (56a3a9db4): upstream refactor 6ae8cc8a8 split app.go, so local-only job/
// team panel extensions live here instead of forking upstream files.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/config"
	"reasonix/internal/control"
	"reasonix/internal/jobs"
)

func (a *App) ReportRenderingPerf(count int, maxMs int64, avgMs int64) {
	// count=0 with zero aggregates is the startup ping (link proof); any
	// other count<=0 is a no-op.
	if count < 0 || (count == 0 && (maxMs != 0 || avgMs != 0)) {
		return
	}
	row := fmt.Sprintf("{\"ts\":%q,\"source\":\"desktop\",\"rendering\":{\"count\":%d,\"max_ms\":%d,\"avg_ms\":%d}}\n",
		time.Now().Format(time.RFC3339Nano), count, maxMs, avgMs)
	dir := config.StatsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(row)
	_ = f.Close()
}

func controllerBootTurns(ctrl control.SessionAPI) int {
	if ctrl == nil {
		return 0
	}
	return ctrl.Turn()
}

type JobPanelView struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Stalled bool   `json:"stalled"`
	Tail    string `json:"tail"`
}
type JobOutputView struct {
	ID     string `json:"id"`
	Output string `json:"output"`
}

// taskMonitorTargetForTab snapshots the workspace and session identity owned by
// tabID. Wails dispatches bound calls concurrently, so resolving the active tab
// inside a task operation would allow a later tab switch to retarget it.

func (a *App) jobPanelTargetForTab(tabID string) (control.SessionAPI, error) {
	tabID = strings.TrimSpace(tabID)
	if tabID == "" {
		return nil, fmt.Errorf("job panel tab id is required")
	}

	a.mu.RLock()
	tab := a.tabByIDLocked(tabID)
	if tab == nil {
		a.mu.RUnlock()
		return nil, fmt.Errorf("task monitor tab %q is unavailable", tabID)
	}
	ctrl := tab.Ctrl
	a.mu.RUnlock()
	return ctrl, nil
}

func jobSnapshotsFor(ctrl control.SessionAPI) ([]jobs.JobSnapshot, bool) {
	src, ok := ctrl.(interface{ JobSnapshots() []jobs.JobSnapshot })
	if !ok {
		return nil, false
	}
	return src.JobSnapshots(), true
}

func (a *App) JobPanelJobsForTab(tabID string) ([]JobPanelView, error) {
	ctrl, err := a.jobPanelTargetForTab(tabID)
	if err != nil {
		return nil, err
	}
	if ctrl == nil {
		return nil, nil
	}
	snaps, ok := jobSnapshotsFor(ctrl)
	if !ok {
		return nil, fmt.Errorf("this runtime cannot report job snapshots")
	}
	views := make([]JobPanelView, 0, len(snaps))
	for _, s := range snaps {
		views = append(views, JobPanelView{
			ID:      s.ID,
			Kind:    s.Kind,
			Label:   s.Label,
			Status:  s.Status,
			Stalled: s.Stalled,
			Tail:    s.Tail,
		})
	}
	return views, nil
}

func (a *App) JobOutputForTab(tabID, jobID string) (JobOutputView, error) {
	ctrl, err := a.jobPanelTargetForTab(tabID)
	if err != nil {
		return JobOutputView{}, err
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return JobOutputView{}, fmt.Errorf("job id is required")
	}
	if ctrl == nil {
		return JobOutputView{}, fmt.Errorf("job %q is unavailable", jobID)
	}
	snaps, ok := jobSnapshotsFor(ctrl)
	if !ok {
		return JobOutputView{}, fmt.Errorf("this runtime cannot report job output")
	}
	for _, s := range snaps {
		if s.ID == jobID {
			return JobOutputView{ID: s.ID, Output: s.Tail}, nil
		}
	}
	return JobOutputView{}, fmt.Errorf("job %q is unavailable", jobID)
}

type TeamPanelView struct {
	Roster    []agent.RosterView      `json:"roster"`
	Approvals []agent.ApprovalRequest `json:"approvals"`
}

// TeamPanelViewForTab returns the P12 team panel projection for the tab's
// controller (nil when the tab has no team). Safe to poll: both views are
// non-consuming snapshots.

func (a *App) TeamPanelViewForTab(tabID string) (*TeamPanelView, error) {
	ctrl, err := a.jobPanelTargetForTab(tabID)
	if err != nil {
		return nil, err
	}
	if ctrl == nil {
		return nil, nil
	}
	if tc, ok := ctrl.(interface {
		TeamRosterView() []agent.RosterView
		TeamApprovalsView() []agent.ApprovalRequest
	}); ok {
		return &TeamPanelView{
			Roster:    tc.TeamRosterView(),
			Approvals: tc.TeamApprovalsView(),
		}, nil
	}
	return nil, fmt.Errorf("this runtime cannot report a team panel")
}
