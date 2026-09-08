package main

import "time"

// The P2-2d protocol, in the source so that moving it is a diff someone has to
// justify rather than a flag chosen once the numbers are visible. It was first
// frozen for a natural sample of twenty; that was given up on purpose, to buy
// an answer now instead of waiting days. What the change costs is in
// designedSample below and rides every number this tool prints.
const (
	// frozenAt is when the protocol was fixed, before any of its sample existed.
	frozenAt = "2026-09-08T02:39:15Z"
	// sampleSize is the whole sample. Not a minimum: collecting past it because
	// the answer sits near a threshold is the optional stopping this guards.
	sampleSize = 12
	// designedSample records what this sample is not. The turns are authored by
	// whoever wrote the task list, so the group sizes carry that person's idea
	// of what an agent does. A natural sample was the alternative and would not
	// carry it. Every number below inherits this and none of them should be
	// reported as "what ordinary use looks like".
	designedSample = true
	// The NO-GO gate, both halves required. One of them failing alone does not
	// decide anything — a partition can be coarse and still rare, or common and
	// still thin, and those are different products.
	singletonShareGate = 0.50
	richTurnShareGate  = 0.40
	// richTurn is a group worth offering as one fold.
	richTurnCalls = 3
)

func frozenTime() time.Time {
	t, err := time.Parse(time.RFC3339, frozenAt)
	if err != nil {
		panic("workgroupassay: frozenAt is not a timestamp: " + err.Error())
	}
	return t
}
