package main

import (
	"fmt"
	"net/http"
)

// hasRunningForTabs reports whether any live terminal belongs to the given
// tabs, gating worktree merge/finalize on quiet terminals (v1.38 contract).
func (m *terminalManager) hasRunningForTabs(tabIDs map[string]struct{}) bool {
	if m == nil || len(tabIDs) == 0 {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, session := range m.sessions {
		if session == nil || !session.view.Running {
			continue
		}
		if _, ok := tabIDs[session.tabID]; ok {
			return true
		}
	}
	return false
}

// workspaceRuntimeReservationErrLocked reports a human-readable blocker when
// cleanup or a merge-back reservation is in flight for the workspace.
func (a *App) workspaceRuntimeReservationErrLocked(workspaceKey string) error {
	if a.workspaceCleanupReservedLocked(workspaceKey) {
		return fmt.Errorf("workspace cleanup is in progress")
	}
	if a.workspaceMergeReservedLocked(workspaceKey) {
		return fmt.Errorf("workspace merge-back is in progress")
	}
	return nil
}

// newTakeoverMirror builds a takeover mirror with the standard lifecycle
// channels already allocated (v1.38 takeover contract).
func newTakeoverMirror(app *App, key, tabID, sessionPath string, sink *tabEventSink, record takeoverServeRecord, client *http.Client, grant takeoverGrant) *takeoverMirror {
	return &takeoverMirror{
		app: app, key: key, tabID: tabID, sessionPath: sessionPath, sink: sink,
		record: record, client: client, grant: grant, bindingRevision: 1,
		stop: make(chan struct{}), done: make(chan struct{}), wake: make(chan struct{}, 1),
	}
}
