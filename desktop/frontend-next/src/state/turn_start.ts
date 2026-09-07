import type { WireEvent } from "../port/wire";
import type { SessionState } from "./session_types";

// A client mints its own user row and nothing on the wire echoes it back, so
// what joins the two is causation: the send that caused this turn is the row
// the kernel's turn_started is about. Names are taken in the order they were
// sent, and one that no longer names a row — taken back, or replaced by a
// rebuild — is dropped rather than handed to the wrong turn.
export function nameTurnStart(s: SessionState, ev: WireEvent): SessionState {
  if (ev.authoredTurn === undefined || ev.msgIndex === undefined) return s;
  const at = s.awaitingTurnStart.findIndex((id) => s.items.some((i) => i.t === "user" && i.id === id));
  if (at < 0) return { ...s, awaitingTurnStart: [] };
  const named = s.awaitingTurnStart[at];
  return {
    ...s,
    awaitingTurnStart: s.awaitingTurnStart.slice(at + 1),
    items: s.items.map((i) =>
      i.t === "user" && i.id === named ? { ...i, authoredTurn: ev.authoredTurn, msgIndex: ev.msgIndex } : i,
    ),
  };
}
