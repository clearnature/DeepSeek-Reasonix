package agent

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"time"

	"reasonix/internal/event"
)

// P10 reliability: stalled-teammate abort and crash-recovery snapshots.

// SetStallAbort configures the stalled-teammate abort threshold (0 disables
// abort, keeping the jobs-layer warning only).
func (ts *TeammateStore) SetStallAbort(d time.Duration) {
	ts.mu.Lock()
	ts.stallAbort = d
	ts.mu.Unlock()
}

// SetSnapshotPath enables crash recovery: the team state (teammates, grants,
// approvals) is snapshotted atomically on every mutation and loaded on
// construction. Empty path disables persistence. Restoring happens here so
// the load sees the configured path.
func (ts *TeammateStore) SetSnapshotPath(p string) {
	ts.mu.Lock()
	ts.snapshotPath = p
	ts.mu.Unlock()
	ts.loadSnapshot()
}

// checkStalled kills running teammates whose jobs are stalled (P10). Called
// periodically by the stall worker; an aborted teammate returns to idle so
// the leader can re-assign. Warning-only when stallAbort is zero.
func (ts *TeammateStore) checkStalled() {
	ts.mu.Lock()
	abort := ts.stallAbort
	jm := ts.jm
	session := ts.session
	type target struct{ name, jobID string }
	var targets []target
	for _, tm := range ts.teammates {
		if tm.State == TeammateRunning && tm.LastJobID != "" {
			targets = append(targets, target{name: tm.Name, jobID: tm.LastJobID})
		}
	}
	ts.mu.Unlock()
	if abort <= 0 || jm == nil || session == "" || len(targets) == 0 {
		return
	}
	snaps := jm.JobSnapshotsForSession(session)
	if len(snaps) == 0 {
		return
	}
	stalled := make(map[string]bool, len(snaps))
	for _, s := range snaps {
		stalled[s.ID] = s.Stalled
	}
	for _, t := range targets {
		if !stalled[t.jobID] {
			continue
		}
		ts.mu.Lock()
		if tm, ok := ts.teammates[t.name]; ok {
			tm.State = TeammateIdle
		}
		ts.mu.Unlock()
		jm.Kill(t.jobID)
		ts.mu.Lock()
		sink := ts.sink
		ts.mu.Unlock()
		if sink != nil {
			sink.Emit(event.Event{Kind: event.Notice, Text: "teammate " + t.name + " stalled and was aborted — reassign with /team-add"})
		}
	}
}

// saveSnapshot writes the team state atomically (tmp + rename) so a crash
// never leaves a half-written file. Runs asynchronously so mutation methods
// holding ts.mu can call it without self-deadlock; the goroutine's own Lock
// waits for the current holder to release, and the last write wins. Failures
// are logged, never fatal.
func (ts *TeammateStore) saveSnapshot() {
	go func() {
		ts.mu.Lock()
		path := ts.snapshotPath
		if path == "" {
			ts.mu.Unlock()
			return
		}
		snap := teamSnapshot{
			Teammates: make([]Teammate, 0, len(ts.teammates)),
			Grants:    make(map[string]WritePathSet, len(ts.grants)),
			Approvals: make(map[string]*approvalReq, len(ts.approvals)),
		}
		for _, tm := range ts.teammates {
			snap.Teammates = append(snap.Teammates, *tm)
		}
		maps.Copy(snap.Grants, ts.grants)
		maps.Copy(snap.Approvals, ts.approvals)
		ts.mu.Unlock()

		payload, err := json.Marshal(snap)
		if err != nil {
			return
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, payload, 0o644); err != nil {
			return
		}
		_ = os.Rename(tmp, path)
	}()
}

// loadSnapshot restores a persisted team state after a crash. Running jobs
// died with the process, so restored teammates come back idle; grants and
// pending approvals survive for re-assignment.
func (ts *TeammateStore) loadSnapshot() {
	ts.mu.Lock()
	path := ts.snapshotPath
	ts.mu.Unlock()
	if path == "" {
		return
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return // no snapshot (first run) — not an error
	}
	var snap teamSnapshot
	if err := json.Unmarshal(payload, &snap); err != nil {
		return
	}
	ts.mu.Lock()
	for i := range snap.Teammates {
		tm := &snap.Teammates[i]
		tm.State = TeammateIdle
		if _, exists := ts.teammates[tm.Name]; !exists {
			ts.teammates[tm.Name] = tm
		}
	}
	for k, v := range snap.Grants {
		if _, ok := ts.teammates[k]; ok {
			ts.grants[k] = v
		}
	}
	for k, v := range snap.Approvals {
		if _, ok := ts.teammates[v.Teammate]; ok {
			ts.approvals[k] = v
		}
	}
	ts.mu.Unlock()
}

type teamSnapshot struct {
	Teammates []Teammate              `json:"teammates"`
	Grants    map[string]WritePathSet `json:"grants"`
	Approvals map[string]*approvalReq `json:"approvals"`
}
