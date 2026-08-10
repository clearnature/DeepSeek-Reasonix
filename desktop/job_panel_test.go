package desktop

import (
	"sync"
	"testing"

	"reasonix/internal/control"
	"reasonix/internal/jobs"
)

// jobSnapshotController stubs the controller surface the job panel needs: it
// embeds control.SessionAPI so it satisfies WorkspaceTab.Ctrl, and overrides
// JobSnapshots to return canned panel data. The embedded API stays nil — the
// panel bridge only asserts JobSnapshots, so no other method is reached.
type jobSnapshotController struct {
	control.SessionAPI
	mu    sync.Mutex
	snaps []jobs.JobSnapshot
}

func (c *jobSnapshotController) JobSnapshots() []jobs.JobSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]jobs.JobSnapshot(nil), c.snaps...)
}

func (c *jobSnapshotController) setSnaps(snaps []jobs.JobSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snaps = append([]jobs.JobSnapshot(nil), snaps...)
}

func TestJobPanelJobsForTabRoutesByTabOwnedController(t *testing.T) {
	ctrlA := &jobSnapshotController{}
	ctrlA.setSnaps([]jobs.JobSnapshot{
		{ID: "a-1", Kind: "bash", Label: "tests A", Status: "running", Tail: "a-tail", Stalled: true},
	})
	ctrlB := &jobSnapshotController{}
	ctrlB.setSnaps([]jobs.JobSnapshot{
		{ID: "b-1", Kind: "test", Label: "tests B", Status: "completed", Tail: "b-tail"},
	})
	app := &App{tabs: map[string]*WorkspaceTab{
		"tab-a": {ID: "tab-a", Ctrl: ctrlA},
		"tab-b": {ID: "tab-b", Ctrl: ctrlB},
	}}

	gotA, err := app.JobPanelJobsForTab("tab-a")
	if err != nil {
		t.Fatalf("JobPanelJobsForTab(tab-a): %v", err)
	}
	if len(gotA) != 1 || gotA[0].ID != "a-1" || gotA[0].Kind != "bash" || gotA[0].Label != "tests A" ||
		gotA[0].Status != "running" || !gotA[0].Stalled || gotA[0].Tail != "a-tail" {
		t.Fatalf("JobPanelJobsForTab(tab-a) = %#v, want the a-1 projection", gotA)
	}

	gotB, err := app.JobPanelJobsForTab("tab-b")
	if err != nil {
		t.Fatalf("JobPanelJobsForTab(tab-b): %v", err)
	}
	if len(gotB) != 1 || gotB[0].ID != "b-1" || gotB[0].Tail != "b-tail" {
		t.Fatalf("JobPanelJobsForTab(tab-b) = %#v, want only b-1", gotB)
	}
}

func TestJobPanelJobsForTabValidatesTabID(t *testing.T) {
	app := &App{tabs: map[string]*WorkspaceTab{
		"known": {ID: "known", Ctrl: &jobSnapshotController{}},
	}}

	if _, err := app.JobPanelJobsForTab(""); err == nil {
		t.Fatal("empty tab id must be rejected")
	}
	if _, err := app.JobPanelJobsForTab("   "); err == nil {
		t.Fatal("blank tab id must be rejected")
	}
	if _, err := app.JobPanelJobsForTab("missing"); err == nil {
		t.Fatal("unknown tab id must be rejected")
	}

	// A tab whose controller is still booting has no jobs to show: empty, no error.
	app.tabs["boot"] = &WorkspaceTab{ID: "boot"}
	got, err := app.JobPanelJobsForTab("boot")
	if err != nil {
		t.Fatalf("booting tab must not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("booting tab jobs = %#v, want empty", got)
	}
}

func TestJobOutputForTabReturnsBoundedOutput(t *testing.T) {
	ctrl := &jobSnapshotController{}
	ctrl.setSnaps([]jobs.JobSnapshot{
		{ID: "j-1", Kind: "bash", Label: "build", Status: "running", Tail: "bounded-output"},
	})
	app := &App{tabs: map[string]*WorkspaceTab{"tab-a": {ID: "tab-a", Ctrl: ctrl}}}

	got, err := app.JobOutputForTab("tab-a", "j-1")
	if err != nil {
		t.Fatalf("JobOutputForTab: %v", err)
	}
	if got.ID != "j-1" || got.Output != "bounded-output" {
		t.Fatalf("JobOutputForTab = %#v, want j-1/bounded-output", got)
	}
}

func TestJobOutputForTabRejectsUnknownOrBlankJob(t *testing.T) {
	ctrl := &jobSnapshotController{}
	ctrl.setSnaps([]jobs.JobSnapshot{{ID: "j-1", Tail: "t"}})
	app := &App{tabs: map[string]*WorkspaceTab{"tab-a": {ID: "tab-a", Ctrl: ctrl}}}

	if _, err := app.JobOutputForTab("tab-a", ""); err == nil {
		t.Fatal("blank job id must be rejected")
	}
	if _, err := app.JobOutputForTab("tab-a", "nope"); err == nil {
		t.Fatal("unknown job id must be rejected")
	}
	if _, err := app.JobOutputForTab("missing", "j-1"); err == nil {
		t.Fatal("unknown tab id must be rejected")
	}
}
