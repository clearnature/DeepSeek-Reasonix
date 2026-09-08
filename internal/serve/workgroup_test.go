package serve

import (
	"fmt"
	"strings"

	"reasonix/internal/event"
	"reasonix/internal/eventwire"
)

// Call is one tool invocation as an atom: a barrier lands between a dispatch and
// its result, and a call outlives its round, and neither may materialize a
// second Call for the same id.
type Call struct {
	ID            string
	Source        string
	Issuer        event.ToolIssuer
	ParentID      string
	Rounds        []string
	SawDispatch   bool
	SawResult     bool
	Interruptions []string
}

func (c Call) TopLevel() bool { return c.ParentID == "" }

// WorkGroup is a run of the assistant's own top-level calls with one producer.
// It is a candidate under test, not a decided shape: what it does with a
// barrier, a host's bookkeeping and a line the user typed is exactly what the
// fixtures are there to falsify.
type WorkGroup struct {
	ID      string
	Source  string
	Members []string
	// Members still in flight when the group ended. Atomizing calls already stops
	// one landing in two groups, so a misplaced boundary now shows up here: the
	// group is sealed and its member's result arrives outside every group.
	OpenAtClose []string
	ClosedBy    string
}

// workGroupRules is the candidate, spelled out so a sabotage can flip one part
// and nothing else. Every field here is a claim a fixture has to survive.
type workGroupRules struct {
	// barrierClosesAfterCall: a blocking user decision does not split the call
	// it landed in; it ends the group once that call has finished.
	barrierClosesAfterCall bool
	// barrierSplitsInPlace is the shape already shown wrong on real logs: it
	// tears one call into two groups. Kept as the sabotage arm.
	barrierSplitsInPlace bool
	// hostIsTransparent: the host's own bookkeeping is not a step and does not
	// cut. Off, it fragments a turn every time the host advances a list.
	hostIsTransparent bool
	// hostCounts folds the host's bookkeeping in as the assistant's work.
	hostCounts bool
	// providerCounts: a provider-executed call is still the assistant working.
	providerCounts bool
	// userCuts: a line the user typed ends the group and nothing merges across.
	userCuts bool
	// sourceCuts: two producers never share a group.
	sourceCuts bool
	// promoteChildren lifts a sub-agent's calls to top-level members.
	promoteChildren bool
}

func workGroupV1() workGroupRules {
	return workGroupRules{
		barrierClosesAfterCall: true,
		hostIsTransparent:      true,
		providerCounts:         true,
		userCuts:               true,
		sourceCuts:             true,
	}
}

func (r workGroupRules) member(c Call) bool {
	if !c.TopLevel() && !r.promoteChildren {
		return false
	}
	switch c.Issuer {
	case event.IssuedByModel:
		return true
	case event.IssuedByProvider:
		return r.providerCounts
	case event.IssuedByHost:
		return r.hostCounts
	}
	return false
}

// foldWorkGroups atomizes calls and then groups them. Two passes in one walk:
// a call is settled by its own frames wherever they land, and the grouping only
// ever reads settled facts about it.
func foldWorkGroups(frames []eventwire.Event, r workGroupRules) ([]Call, []WorkGroup) {
	calls := map[string]*Call{}
	var order []string
	var groups []WorkGroup
	var cur *WorkGroup
	open := map[string]bool{}
	round := ""
	started := false

	close := func(why string) {
		if cur != nil && len(cur.Members) > 0 {
			cur.ClosedBy = why
			for _, id := range cur.Members {
				if open[id] {
					cur.OpenAtClose = append(cur.OpenAtClose, id)
				}
			}
			groups = append(groups, *cur)
		}
		cur = nil
	}
	for _, e := range frames {
		switch e.Kind {
		case "turn_started":
			if started {
				close("next turn")
			}
			started = true
		case "turn_done":
			close("turn done")
		case "stream_attempt":
			if e.StreamAttempt != nil && e.StreamAttempt.Action == "begin" {
				round = e.StreamAttempt.ID
			}
		case "approval_request", "ask_request":
			for id := range open {
				calls[id].Interruptions = append(calls[id].Interruptions, e.Kind)
			}
			if r.barrierSplitsInPlace {
				close("barrier")
			}
		case "tool_dispatch", "tool_result":
			t := e.Tool
			if t == nil || t.ID == "" {
				continue
			}
			c, seen := calls[t.ID]
			if !seen {
				c = &Call{ID: t.ID, Source: e.Source, Issuer: event.ToolIssuer(t.Issuer), ParentID: t.ParentID}
				calls[t.ID] = c
				order = append(order, t.ID)
				admit(&cur, &groups, r, *c, close)
			}
			if round != "" && (len(c.Rounds) == 0 || c.Rounds[len(c.Rounds)-1] != round) {
				c.Rounds = append(c.Rounds, round)
			}
			if e.Kind == "tool_dispatch" {
				c.SawDispatch = true
				open[t.ID] = true
				continue
			}
			c.SawResult = true
			delete(open, t.ID)
			if r.barrierClosesAfterCall && len(c.Interruptions) > 0 && r.member(*c) {
				close("barrier settled")
			}
		}
	}
	close("end of record")
	out := make([]Call, 0, len(order))
	for _, id := range order {
		out = append(out, *calls[id])
	}
	return out, groups
}

// admit decides what a newly seen call does to the open group: end it, join it,
// or pass through without touching it.
func admit(cur **WorkGroup, groups *[]WorkGroup, r workGroupRules, c Call, close func(string)) {
	if c.Issuer == event.IssuedByUser && r.userCuts {
		close("user intervention")
		return
	}
	if !r.member(c) {
		if c.Issuer == event.IssuedByHost && !r.hostIsTransparent {
			close("host bookkeeping")
		}
		return
	}
	if *cur != nil && r.sourceCuts && (*cur).Source != c.Source {
		close("producer change")
	}
	if *cur == nil {
		*cur = &WorkGroup{ID: c.ID, Source: c.Source}
	}
	(*cur).Members = append((*cur).Members, c.ID)
}

// duplicateMembership is the invariant a partition is only allowed to exist
// under: one tool call id materializes at most one top-level Call, and appears
// in at most one group. The 6-of-76 duplicates a real corpus showed came from a
// barrier splitting a call in place — which is what this refuses.
func duplicateMembership(groups []WorkGroup) string {
	seen := map[string]int{}
	for _, g := range groups {
		for _, id := range g.Members {
			seen[id]++
		}
	}
	var bad []string
	for id, n := range seen {
		if n > 1 {
			bad = append(bad, fmt.Sprintf("%s in %d groups", id, n))
		}
	}
	if len(bad) == 0 {
		return ""
	}
	return strings.Join(bad, "; ")
}

func groupShape(groups []WorkGroup) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		row := g.Source + ":" + strings.Join(g.Members, ",") + "|" + g.ClosedBy
		if len(g.OpenAtClose) > 0 {
			row += "|open:" + strings.Join(g.OpenAtClose, ",")
		}
		out = append(out, row)
	}
	return out
}

// sealedOverAnOpenCall reports groups closed while a member was still running.
// A partition may not do this: the member's result then belongs to no group,
// and a reader is left with a finished fold over an unfinished call.
func sealedOverAnOpenCall(groups []WorkGroup) string {
	var bad []string
	for _, g := range groups {
		if len(g.OpenAtClose) > 0 {
			bad = append(bad, fmt.Sprintf("%s sealed over %s", g.ID, strings.Join(g.OpenAtClose, ",")))
		}
	}
	return strings.Join(bad, "; ")
}
