package event

// ToolIssuer says who initiated a tool invocation — a separate axis from
// Event.Source, which says who produced the frame. One executor frame can carry
// a call the model asked for, one the host minted, or one the provider ran on
// its own side, and only this tells them apart. Set it where the invocation is
// decided; nothing downstream can re-derive it.
type ToolIssuer string

const (
	// IssuedByModel: the call came out of a model response's tool_calls. Which
	// model is Event.Source's answer, not this one's.
	IssuedByModel ToolIssuer = "model"
	// IssuedByHost: the host invoked it itself — seeding an approved plan's task
	// list, advancing a step, closing a goal, expanding one delegation call.
	IssuedByHost ToolIssuer = "host"
	// IssuedByUser: a person invoked it directly, by slash command or inline
	// shell. Neither the model nor the host chose to run it.
	IssuedByUser ToolIssuer = "user"
	// IssuedByProvider: the provider ran it and reported only a result. There is
	// no dispatch to pair with, and that is the call's shape, not a gap.
	IssuedByProvider ToolIssuer = "provider"
)

// ModelWork reports whether the call is work the model asked for. A fold that
// counts a turn's steps excludes the rest: the host's bookkeeping and a line a
// person typed are not the model working. An unset issuer answers false rather
// than guess — guessing "model" is what counts a seeded task list as a step.
func (i ToolIssuer) ModelWork() bool { return i == IssuedByModel }
