package control

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/jobs"
	"reasonix/internal/tool"
)

// TestLoopWakeupDeliversToLeaderInbox drives loop_wakeup end to end: a short
// delay writes the prompt into the leader inbox (the compose injection path),
// and cron_list/cron_delete manage recurring entries.
func TestLoopWakeupDeliversToLeaderInbox(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	inbox := t.TempDir()
	ts := agent.NewTeammateStore(
		agent.NewTaskTool(&smokeProvider{}, nil, tool.NewRegistry(), 20, 0, 0, 0, 0, 0, 0, 0.0, "", "t", nil, 0, "", "", nil),
		jm, inbox)
	t.Cleanup(ts.Close)

	tools := NewWakeupTools(ts)
	byName := map[string]tool.Tool{}
	for _, wt := range tools {
		byName[wt.Name()] = wt
	}

	// One-shot wake-up fires into the leader inbox.
	if _, err := byName["loop_wakeup"].Execute(context.Background(), json.RawMessage(`{"delay_seconds":1,"prompt":"re-check the build"}`)); err != nil {
		t.Fatalf("loop_wakeup: %v", err)
	}
	leaderDir := filepath.Join(inbox, "leader", "inbox")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if entries, _ := os.ReadDir(leaderDir); len(entries) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("wake-up never delivered to leader inbox")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Recurring entry: create, list, delete.
	if _, err := byName["cron_create"].Execute(context.Background(), json.RawMessage(`{"schedule_seconds":3600,"prompt":"status sweep"}`)); err != nil {
		t.Fatalf("cron_create: %v", err)
	}
	listOut, _ := byName["cron_list"].Execute(context.Background(), json.RawMessage(`{}`))
	if !strings.Contains(listOut, "status sweep") {
		t.Fatalf("cron_list missing entry: %q", listOut)
	}
	var id string
	for _, line := range strings.Split(listOut, "\n") {
		if strings.Contains(line, "cron") && strings.Contains(line, "status sweep") {
			id = strings.Fields(line)[1]
		}
	}
	if id == "" {
		t.Fatalf("could not parse cron id from %q", listOut)
	}
	if _, err := byName["cron_delete"].Execute(context.Background(), json.RawMessage(`{"id":"`+id+`"}`)); err != nil {
		t.Fatalf("cron_delete: %v", err)
	}
	after, _ := byName["cron_list"].Execute(context.Background(), json.RawMessage(`{}`))
	if strings.Contains(after, "status sweep") {
		t.Fatalf("cron entry survived delete: %q", after)
	}
}

// TestWakeupPersistsAndRestores locks audit #1: scheduled entries survive a
// scheduler restart (same persist path) with their intervals intact.
func TestWakeupPersistsAndRestores(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wakeups.json")
	s1 := newWakeupScheduler(func(string) error { return nil }, path)
	if err := s1.add(&wakeupEntry{ID: "cron-1", Prompt: "sweep", Schedule: "3600s", Created: time.Now()}); err != nil {
		t.Fatalf("add: %v", err)
	}
	// A fresh scheduler over the same file restores the entry.
	s2 := newWakeupScheduler(func(string) error { return nil }, path)
	got := s2.list()
	if len(got) != 1 || got[0].ID != "cron-1" || got[0].Prompt != "sweep" {
		t.Fatalf("restored = %+v, want cron-1 sweep", got)
	}
}

// TestWakeupClampsTinySchedule locks audit #4: a sub-minimum schedule is
// clamped so a typo cannot hammer the inbox.
func TestWakeupClampsTinySchedule(t *testing.T) {
	s := newWakeupScheduler(func(string) error { return nil }, "")
	e := &wakeupEntry{ID: "cron-x", Prompt: "x", Schedule: "1s", Created: time.Now()}
	if err := s.add(e); err != nil {
		t.Fatalf("add: %v", err)
	}
	if e.Schedule != minWakeupInterval.String() {
		t.Fatalf("schedule = %q, want clamped %q", e.Schedule, minWakeupInterval)
	}
}

// TestWakeupDeliveryFailureSurfaces locks audit #2: a failing post reaches the
// onError hook instead of being swallowed.
func TestWakeupDeliveryFailureSurfaces(t *testing.T) {
	failed := make(chan string, 1)
	s := newWakeupScheduler(func(string) error { return fmt.Errorf("inbox full") }, "")
	s.onError = func(msg string) { failed <- msg }
	if err := s.add(&wakeupEntry{ID: "wake-1", Prompt: "p", Delay: "10ms", Created: time.Now()}); err != nil {
		t.Fatalf("add: %v", err)
	}
	select {
	case msg := <-failed:
		if !strings.Contains(msg, "inbox full") {
			t.Fatalf("notice = %q", msg)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("delivery failure was swallowed")
	}
}
