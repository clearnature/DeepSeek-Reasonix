package main

import "time"

// The P2-2d protocol, frozen before any of its sample existed. It lives in the
// source so that moving it is a diff someone has to justify, rather than a flag
// chosen once the numbers are visible.
const (
	// frozenAt is when the protocol was fixed. Turns before it are not sample:
	// they were produced while the rules were still being written.
	frozenAt = "2026-09-08T02:39:15Z"
	// sampleSize is the whole sample. Not a minimum: collecting past it because
	// the answer sits near a threshold is the optional stopping this guards.
	sampleSize = 20
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
