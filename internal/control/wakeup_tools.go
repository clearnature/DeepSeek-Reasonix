package control

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/tool"
)

// Scheduled wake-ups: qwen loop_wakeup + cron_create/cron_list/cron_delete.
// A wake-up does not run an unattended model turn (our qwen-mode default); it
// writes into the leader inbox, which compose injects on the next turn.

type wakeupEntry struct {
	ID       string    `json:"id"`
	Prompt   string    `json:"prompt"`
	Delay    string    `json:"delay,omitempty"`    // one-shot: e.g. "5m"
	Schedule string    `json:"schedule,omitempty"` // recurring: e.g. "10m"
	Created  time.Time `json:"created"`
}

type wakeupScheduler struct {
	mu      sync.Mutex
	entries map[string]*wakeupEntry
	timers  map[string]*time.Timer
	post    func(prompt string) error
}

func newWakeupScheduler(post func(string) error) *wakeupScheduler {
	return &wakeupScheduler{entries: map[string]*wakeupEntry{}, timers: map[string]*time.Timer{}, post: post}
}

func (s *wakeupScheduler) add(entry *wakeupEntry) error {
	var every time.Duration
	if entry.Delay != "" {
		d, err := time.ParseDuration(entry.Delay)
		if err != nil || d <= 0 {
			return fmt.Errorf("loop_wakeup: invalid delay %q", entry.Delay)
		}
		every = d
	} else if entry.Schedule != "" {
		d, err := time.ParseDuration(entry.Schedule)
		if err != nil || d <= 0 {
			return fmt.Errorf("cron: invalid schedule %q", entry.Schedule)
		}
		every = d
	} else {
		return fmt.Errorf("wakeup: delay or schedule is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[entry.ID] = entry
	recurring := entry.Schedule != ""
	s.timers[entry.ID] = s.arm(entry, every, recurring)
	return nil
}

// arm schedules one firing; recurring entries re-arm after each fire.
func (s *wakeupScheduler) arm(entry *wakeupEntry, every time.Duration, recurring bool) *time.Timer {
	return time.AfterFunc(every, func() {
		_ = s.post(entry.Prompt)
		if recurring {
			s.mu.Lock()
			if _, ok := s.entries[entry.ID]; ok {
				s.timers[entry.ID] = s.arm(entry, every, recurring)
			}
			s.mu.Unlock()
		}
	})
}

func (s *wakeupScheduler) remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[id]; ok {
		t.Stop()
		delete(s.timers, id)
	}
	if _, ok := s.entries[id]; ok {
		delete(s.entries, id)
		return true
	}
	return false
}

func (s *wakeupScheduler) list() []wakeupEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]wakeupEntry, 0, len(s.entries))
	for _, v := range s.entries {
		out = append(out, *v)
	}
	return out
}

// loopWakeupTool schedules a one-shot delayed wake-up.
type loopWakeupTool struct{ sched *wakeupScheduler }

func (t *loopWakeupTool) Name() string { return "loop_wakeup" }
func (t *loopWakeupTool) Description() string {
	return "Schedule a one-shot wake-up: after delaySeconds the prompt is delivered to the leader inbox and arrives with the next turn. Use to come back to something later (re-check a build, follow up on a review). qwen loop_wakeup analog."
}
func (t *loopWakeupTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"delay_seconds":{"description":"Seconds to wait before delivering the prompt.","type":"integer","minimum":1},"prompt":{"description":"What to surface when the wake-up fires.","type":"string"}},"required":["delay_seconds","prompt"],"type":"object"}`)
}
func (t *loopWakeupTool) ReadOnly() bool { return false }
func (t *loopWakeupTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		DelaySeconds int    `json:"delay_seconds"`
		Prompt       string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("loop_wakeup: %w", err)
	}
	if p.DelaySeconds <= 0 || strings.TrimSpace(p.Prompt) == "" {
		return "", fmt.Errorf("loop_wakeup: delay_seconds and prompt are required")
	}
	e := &wakeupEntry{ID: fmt.Sprintf("wake-%d", time.Now().UnixNano()), Prompt: p.Prompt,
		Delay: (time.Duration(p.DelaySeconds) * time.Second).String(), Created: time.Now()}
	if err := t.sched.add(e); err != nil {
		return "", err
	}
	return fmt.Sprintf("wake-up %s armed for %ds", e.ID, p.DelaySeconds), nil
}

// cronCreateTool schedules a recurring wake-up.
type cronCreateTool struct{ sched *wakeupScheduler }

func (t *cronCreateTool) Name() string { return "cron_create" }
func (t *cronCreateTool) Description() string {
	return "Schedule a recurring wake-up: every scheduleSeconds the prompt is delivered to the leader inbox and arrives with the next turn. Use for periodic checks (status sweeps, re-validation). Delete with cron_delete. qwen cron_create analog."
}
func (t *cronCreateTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"schedule_seconds":{"description":"Repeat interval in seconds.","type":"integer","minimum":1},"prompt":{"description":"What to surface on each firing.","type":"string"}},"required":["schedule_seconds","prompt"],"type":"object"}`)
}
func (t *cronCreateTool) ReadOnly() bool { return false }
func (t *cronCreateTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ScheduleSeconds int    `json:"schedule_seconds"`
		Prompt          string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cron_create: %w", err)
	}
	if p.ScheduleSeconds <= 0 || strings.TrimSpace(p.Prompt) == "" {
		return "", fmt.Errorf("cron_create: schedule_seconds and prompt are required")
	}
	e := &wakeupEntry{ID: fmt.Sprintf("cron-%d", time.Now().UnixNano()), Prompt: p.Prompt,
		Schedule: (time.Duration(p.ScheduleSeconds) * time.Second).String(), Created: time.Now()}
	if err := t.sched.add(e); err != nil {
		return "", err
	}
	return fmt.Sprintf("cron %s created (every %ds)", e.ID, p.ScheduleSeconds), nil
}

// cronListTool lists scheduled wake-ups.
type cronListTool struct{ sched *wakeupScheduler }

// NewCronListTool registers the cron_list tool.
func NewCronListTool(sched *wakeupScheduler) tool.Tool { return &cronListTool{sched: sched} }

func (t *cronListTool) Name() string { return "cron_list" }
func (t *cronListTool) Description() string {
	return "List scheduled wake-ups (one-shot loop_wakeup and recurring cron entries) with their ids and intervals. qwen cron_list analog."
}
func (t *cronListTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (t *cronListTool) ReadOnly() bool          { return false }
func (t *cronListTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	items := t.sched.list()
	if len(items) == 0 {
		return "no scheduled wake-ups", nil
	}
	var b strings.Builder
	for _, v := range items {
		kind, every := "wake", v.Delay
		if v.Schedule != "" {
			kind, every = "cron", v.Schedule
		}
		fmt.Fprintf(&b, "- %s [%s every %s]: %s\n", v.ID, kind, every, v.Prompt)
	}
	return strings.TrimSuffix(b.String(), "\n"), nil
}

// cronDeleteTool cancels a scheduled wake-up.
type cronDeleteTool struct{ sched *wakeupScheduler }

// NewCronDeleteTool registers the cron_delete tool.
func NewCronDeleteTool(sched *wakeupScheduler) tool.Tool { return &cronDeleteTool{sched: sched} }

func (t *cronDeleteTool) Name() string { return "cron_delete" }
func (t *cronDeleteTool) Description() string {
	return "Cancel a scheduled wake-up by id (from cron_list). qwen cron_delete analog."
}
func (t *cronDeleteTool) Schema() json.RawMessage {
	return json.RawMessage(`{"properties":{"id":{"description":"Wake-up id from cron_list.","type":"string"}},"required":["id"],"type":"object"}`)
}
func (t *cronDeleteTool) ReadOnly() bool { return false }
func (t *cronDeleteTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("cron_delete: %w", err)
	}
	if !t.sched.remove(strings.TrimSpace(p.ID)) {
		return "", fmt.Errorf("cron_delete: unknown id %q", p.ID)
	}
	return fmt.Sprintf("wake-up %s cancelled", p.ID), nil
}

// NewWakeupTools returns the loop_wakeup + cron family sharing one scheduler.
func NewWakeupTools(ts *agent.TeammateStore) []tool.Tool {
	sched := newWakeupScheduler(func(prompt string) error {
		return ts.PostMailToLeader("wakeup", prompt)
	})
	return []tool.Tool{
		&loopWakeupTool{sched: sched},
		&cronCreateTool{sched: sched},
		&cronListTool{sched: sched},
		&cronDeleteTool{sched: sched},
	}
}
