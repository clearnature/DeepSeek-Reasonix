package agent

import (
	"fmt"
	"log/slog"
	"strings"
)

// Name returns the explicit group label, or "" in the legacy unnamed
// single-team mode (qwen TeamFile.name analog).
func (ts *TeammateStore) Name() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.name
}

// CreateGroup names this store's team, mirroring qwen's singleton team
// semantics: one active group, so a second create is refused until
// DeleteGroup clears it.
func (ts *TeammateStore) CreateGroup(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("group name is required")
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.name != "" {
		return fmt.Errorf("a team %q is already active — delete it before creating a new one", ts.name)
	}
	ts.name = name
	slog.Info("team group created", "name", name)
	ts.saveSnapshot()
	return nil
}

// DeleteGroup dissolves the team: stops and drops every member, clears the
// name, and resets to the unnamed single-member mode. It is the qwen
// team_delete counterpart (our store is the whole team; there is no separate
// team dir to remove).
func (ts *TeammateStore) DeleteGroup() error {
	ts.mu.Lock()
	name := ts.name
	ts.name = ""
	members := make([]string, 0, len(ts.teammates))
	for n := range ts.teammates {
		members = append(members, n)
	}
	ts.teammates = map[string]*Teammate{}
	ts.mu.Unlock()

	if len(members) > 0 {
		// Stop any running members first (their jobs may still hold slots).
		for _, n := range members {
			_ = ts.TeamStop(n)
		}
	}
	slog.Info("team group deleted", "name", name, "members", len(members))
	ts.saveSnapshot()
	return nil
}
