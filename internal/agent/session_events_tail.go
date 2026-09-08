package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// tailScanChunk bounds each backwards read while locating the last replace
// record; the accumulated window grows only until it holds that one line.
const tailScanChunk = int64(64 << 10)

// replaySessionEventLogTail replays an oversized event log from its last
// replace record instead of refusing it outright. Replace records carry the
// whole transcript, so the rebuilt state is identical to a full replay while
// the bytes before the record (which tripped the budget) are never decoded.
// A log with no replace record keeps the original refusal: an append-only log
// has no snapshot to anchor on, and guessing would drop turns.
func replaySessionEventLogTail(ctx context.Context, f *os.File, path string, size int64, limits sessionReplayLimits, hasher *sessionTranscriptHasher, replay sessionEventReplay) (sessionEventReplay, error) {
	offset, err := tailReplaceOffset(f, size)
	if err != nil {
		return replay, err
	}
	if offset < 0 {
		return replay, sessionReplayLimitError(path, "encoded_bytes", size, limits.maxBytes)
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return replay, err
	}
	return replaySessionEventRecords(ctx, f, path, offset, limits, hasher, replay)
}

// tailReplaceOffset scans backwards for the start offset of the last complete
// replace record, or -1 when the log holds none. Lines are newline-delimited
// (appendSessionEvent appends one marshalled record plus '\n'), so a record
// never has to be decoded to find its boundary.
func tailReplaceOffset(f *os.File, size int64) (int64, error) {
	var window []byte
	off := size
	for off > 0 {
		start := max(off-tailScanChunk, 0)
		part := make([]byte, off-start)
		if _, err := f.ReadAt(part, start); err != nil && err != io.EOF {
			return -1, err
		}
		window = append(part, window...)
		off = start
		lines := bytes.Split(window, []byte{'\n'})
		first := 0
		if off > 0 {
			// The window's first line continues into the unscanned prefix.
			first = 1
		}
		for i := len(lines) - 1; i >= first; i-- {
			line := bytes.TrimSpace(lines[i])
			if len(line) == 0 || line[0] != '{' {
				continue
			}
			var hdr struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(line, &hdr) != nil {
				continue
			}
			if hdr.Type != sessionEventTypeReplace {
				continue
			}
			pos := off
			for k := 0; k < i; k++ {
				pos += int64(len(lines[k])) + 1
			}
			return pos, nil
		}
	}
	return -1, nil
}

// replaySessionEventRecords decodes the wire records starting at r, whose first
// byte sits at baseOffset in the file. Offsets reported back to callers (used
// to truncate a torn tail) stay file-absolute.
func replaySessionEventRecords(ctx context.Context, r io.Reader, path string, baseOffset int64, limits sessionReplayLimits, hasher *sessionTranscriptHasher, replay sessionEventReplay) (sessionEventReplay, error) {
	// Stat and read are not atomic across processes. LimitReader keeps a log
	// that grows after Stat inside the same byte budget.
	limited := &io.LimitedReader{R: &contextReader{ctx: ctx, reader: r}, N: limits.maxBytes + 1}
	dec := json.NewDecoder(limited)
	for {
		if err := ctx.Err(); err != nil {
			return replay, err
		}
		var rec sessionEventWireRecord
		if err := dec.Decode(&rec); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return replay, ctxErr
			}
			if limited.N == 0 {
				return replay, sessionReplayLimitError(path, "encoded_bytes", limits.maxBytes+1, limits.maxBytes)
			}
			if errors.Is(err, io.EOF) {
				return replay, nil
			}
			replay.damaged = true
			return replay, nil
		}
		if rec.SchemaVersion != sessionEventSchemaVersion {
			return replay, fmt.Errorf("decode session event log %s: unsupported schema version %d", path, rec.SchemaVersion)
		}
		if replay.records >= limits.maxRecords {
			return replay, sessionReplayLimitError(path, "event_records", int64(replay.records+1), int64(limits.maxRecords))
		}
		switch rec.Type {
		case sessionEventTypeReplace:
			msgs, collectionItems, err := decodeSessionEventMessages(ctx, path, rec.Messages, 0, 0, limits)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return replay, ctxErr
				}
				if errors.Is(err, ErrSessionReplayLimitExceeded) {
					return replay, err
				}
				replay.damaged = true
				return replay, nil
			}
			replay.msgs = msgs
			replay.collectionItems = collectionItems
			replay.times = make([]time.Time, len(replay.msgs))
			hasher.rehash(msgs)
		case sessionEventTypeAppend:
			if rec.MessageIndex != len(replay.msgs) {
				replay.damaged = true
				return replay, nil
			}
			msgs, collectionItems, err := decodeSessionEventMessages(ctx, path, rec.Messages, len(replay.msgs), replay.collectionItems, limits)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return replay, ctxErr
				}
				if errors.Is(err, ErrSessionReplayLimitExceeded) {
					return replay, err
				}
				replay.damaged = true
				return replay, nil
			}
			replay.msgs = append(replay.msgs, msgs...)
			replay.collectionItems = collectionItems
			for range msgs {
				replay.times = append(replay.times, rec.CreatedAt)
			}
			hasher.addAll(msgs)
		default:
			return replay, fmt.Errorf("decode session event log %s: unsupported event type %q", path, rec.Type)
		}
		replay.records++
		replay.lastGoodEnd = baseOffset + dec.InputOffset()
	}
}
