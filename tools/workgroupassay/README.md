# workgroup assay

Answers one question about real use: **is there enough in a turn's execution to
be worth folding?** Semantics and live/cold parity are settled elsewhere
(`internal/workgroup`, `internal/serve/workgroup_semantics_test.go`,
`internal/serve/trace_parity_test.go`); this only measures natural granularity.

```
go run ./tools/workgroupassay            # reads the default session root
go run ./tools/workgroupassay -root DIR
```

## The protocol, frozen before the sample existed

It is in `protocol.go` so changing it is a reviewable diff.

- **Sample**: the first 20 eligible authored turns produced *after* the freeze
  timestamp, in the order they happened. Not the last 20, not a chosen 20.
- **Eligible**: the turn carries a named `authoredTurn` (which only the current
  schema emits, and which a synthetic continuation never gets), and its
  session's trajectory reads `complete` — a truncated log is a prefix and
  cannot answer a question about a whole turn.
- **A turn with no tool calls is in the sample.** The question is how much of
  ordinary use is worth folding; dropping the quiet turns would answer a
  different one and flatter the feature.
- **No route quotas.** Whatever mix of executor-only and plan-and-execute turns
  ordinary use produces is the sample; the mix is reported as a description,
  never selected for.
- **No verdict before the sample is whole.** Under 20 the tool prints the count
  and stops. Reading a partial result and acting on it is the sequential
  decision this exists to prevent.

## The verdict

NO-GO requires **both**: more than half of groups are a single call, **and**
fewer than 40% of authored turns contain a group of three or more. Anything
else is not an automatic go — it hands the decision to a human looking at the
distribution and the actual group shapes.
