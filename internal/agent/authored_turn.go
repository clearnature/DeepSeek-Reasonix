package agent

import (
	"reasonix/internal/event"
	"reasonix/internal/provider"
)

// TurnClassification names the authored turn a message belongs to. StartsTurn
// marks the one that opens it; Steer and Synthetic say why a user message that
// opened none is still a user message.
type TurnClassification struct {
	AuthoredTurn int
	StartsTurn   bool
	Steer        bool
	Synthetic    bool
}

// ClassifyTurn answers which authored turn msg belongs to, given the turns
// closed before it. Both projections of that number derive it here: the
// display-index sidecar counts a whole transcript with it, and a live turn
// start names its own pending message with it.
func ClassifyTurn(msg provider.Message, priorTurn int) TurnClassification {
	class := TurnClassification{AuthoredTurn: priorTurn}
	if msg.Role != provider.RoleUser {
		return class
	}
	content := UserMessageText(msg)
	if IsUserAuthoredTurn(content) {
		class.AuthoredTurn = priorTurn + 1
		class.StartsTurn = true
		return class
	}
	if _, isSteer := SteerText(content); isSteer {
		class.Steer = true
	} else if IsSyntheticUserText(content) {
		class.Synthetic = true
	}
	return class
}

// PriorAuthoredTurn is the authored-turn count over msgs: the number the next
// message appended after them would follow. Re-derived from the transcript
// rather than tracked beside it, so no counter can disagree with the messages
// it claims to number.
func PriorAuthoredTurn(msgs []provider.Message) int {
	turn := 0
	for _, m := range msgs {
		turn = ClassifyTurn(m, turn).AuthoredTurn
	}
	return turn
}

// turnStartedEvent announces the turn and names the message it is about: the
// authored turn that message opens, and the session index it will take. Both
// come from ClassifyTurn against the live transcript, so this projection and
// the durable display index answer the same number for the same message. A
// turn that opens no authored message names none rather than minting one.
func (a *Agent) turnStartedEvent(pending provider.Message) event.Event {
	started := event.Event{Kind: event.TurnStarted}
	msgs := a.sess.conversation.Snapshot()
	class := ClassifyTurn(pending, PriorAuthoredTurn(msgs))
	if !class.StartsTurn {
		return started
	}
	index := len(msgs)
	started.AuthoredTurn = &class.AuthoredTurn
	started.MsgIndex = &index
	return started
}
