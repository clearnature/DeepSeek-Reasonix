package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/event"
)

// Resume telemetry persistence guard: the controller emits one "resume
// telemetry" notice per C1 gate decision; the recorder must parse it into a
// Resume row so a reopen's replay cost is diagnosable after the fact.

func TestRecorderWritesResumeTelemetry(t *testing.T) {
	dir := t.TempDir()
	r := NewRecorder(&spySink{}, dir, "desktop")
	r.Emit(event.Event{Kind: event.Notice, Text: "resume telemetry", Detail: "path=/tmp/sess.jsonl state=cold idle_min=1530 decision=replay proj_valid=true proj_covered=6497 view_fp=abc123 covered_match=true"})
	flushRecorder(t, r)
	files := dailyJSONLFiles(t, dir)
	if len(files) != 1 {
		t.Fatalf("want 1 daily file, got %d", len(files))
	}
	data, err := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got struct {
		Resume *ResumeRecord `json:"resume"`
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.Contains(line, "resume") {
			if err := json.Unmarshal([]byte(line), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			break
		}
	}
	if got.Resume == nil {
		t.Fatalf("no resume row in daily file: %s", data)
	}
	if got.Resume.State != "cold" || got.Resume.IdleMin != 1530 || got.Resume.Decision != "replay" ||
		got.Resume.Path != "/tmp/sess.jsonl" || got.Resume.ViewFP != "abc123" || !got.Resume.Covered {
		t.Fatalf("resume record = %+v", got.Resume)
	}
	if !got.Resume.ProjValid || got.Resume.ProjCovered != 6497 {
		t.Fatalf("projection coverage fields must survive the round-trip: %+v", got.Resume)
	}
	// Non-resume notices are never persisted as resume rows.
	r.Emit(event.Event{Kind: event.Notice, Text: "compaction telemetry", Detail: "trigger=auto mode=summarized cache=warm src=100"})
	flushRecorder(t, r)
	files = dailyJSONLFiles(t, dir)
	data, err = os.ReadFile(filepath.Join(dir, files[0].Name()))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var all []struct {
		Resume *ResumeRecord `json:"resume"`
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var row struct {
			Resume *ResumeRecord `json:"resume"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		all = append(all, row)
	}
	resumeRows := 0
	for _, row := range all {
		if row.Resume != nil {
			resumeRows++
		}
	}
	if resumeRows != 1 {
		t.Fatalf("compaction notice must not create a resume row: got %d resume rows", resumeRows)
	}
}
