package control

import (
	"context"
	"io"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// End-to-end: a job that finished since the last turn rides the next compose as
// a <background-job-result> envelope inside the <background-jobs> container, so
// the model learns the result without wait/bash_output polling.
func TestComposeAutoDeliversJobResultEnvelope(t *testing.T) {
	c := New(Options{})
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	c.jobs = jm
	j := jm.Start("task", "label", func(_ context.Context, _ io.Writer) (string, error) {
		return "final answer body", nil
	})
	res := jm.Wait(context.Background(), []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("want one Done result, got %+v", res)
	}
	out := c.compose("user message", "user", true)
	if !strings.Contains(out, "<background-jobs>") {
		t.Fatalf("compose output missing <background-jobs> container:\n%s", out)
	}
	if !strings.Contains(out, "<background-job-result") || !strings.Contains(out, `task_id="`+j.ID+`"`) {
		t.Fatalf("compose output missing auto-delivered envelope:\n%s", out)
	}
	if !strings.Contains(out, "final answer body") {
		t.Fatalf("compose output missing job result body:\n%s", out)
	}
}
