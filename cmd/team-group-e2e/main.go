// team-group-e2e drives the full qwen-aligned group lifecycle through the
// controller host commands with a real model: /team-group create -> /team-create
// members -> /team-add real tasks -> wait for the roster to return to idle ->
// /team-group delete (members cleared, group name reset). Usage:
//
//	go run ./cmd/team-group-e2e
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"reasonix/internal/boot"
	"reasonix/internal/event"
	_ "reasonix/internal/provider/anthropic"
)

type captureSink struct {
	msgs []string
	ask  chan event.Ask
	mu   chan struct{}
}

func (s *captureSink) Emit(e event.Event) {
	if e.Kind == event.AskRequest {
		select {
		case s.ask <- e.Ask:
		default:
		}
		return
	}
	if e.Kind != event.Notice {
		return
	}
	<-s.mu
	if e.Detail != "" {
		s.msgs = append(s.msgs, e.Text+" || "+e.Detail)
	} else {
		s.msgs = append(s.msgs, e.Text)
	}
	s.mu <- struct{}{}
}

func jsonAnswers(answers []event.AskAnswer) string {
	b, _ := json.Marshal(answers)
	return string(b)
}

func main() {
	pop := flag.Int("pop", 2, "members per group")
	task := flag.String("task", "用一句中文简述什么是前缀缓存（不超过 20 字）。不要写文件。", "task to assign each member")
	flag.Parse()

	dir, err := os.MkdirTemp("", "team-group-e2e-")
	if err != nil {
		fmt.Println("tempdir:", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	sink := &captureSink{ask: make(chan event.Ask, 8), mu: make(chan struct{}, 1)}
	sink.mu <- struct{}{}
	ctrl, err := boot.Build(context.Background(), boot.Options{Sink: sink, SessionDir: dir, StatsSource: "harness"})
	if err != nil {
		fmt.Println("boot.Build:", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	ctrl.EnableHeadlessAsker()
	go func() {
		for ask := range sink.ask {
			var answers []event.AskAnswer
			for _, q := range ask.Questions {
				if len(q.Options) > 0 {
					answers = append(answers, event.AskAnswer{QuestionID: q.ID, Selected: []string{q.Options[0].Label}})
				}
			}
			fmt.Printf("AUTO-APPROVE %s\n", jsonAnswers(answers))
			ctrl.AnswerQuestion(ask.ID, answers)
		}
	}()

	// 1. Create the group (singleton).
	ctrl.Submit("/team-group create e2e-writers")
	time.Sleep(500 * time.Millisecond)

	// 2. Create members.
	names := make([]string, *pop)
	for i := range *pop {
		names[i] = fmt.Sprintf("m%d", i+1)
		ctrl.Submit("/team-create " + names[i] + " researcher")
	}
	time.Sleep(500 * time.Millisecond)

	// 3. Assign a real task to each member (real model round-trips).
	for _, n := range names {
		ctrl.Submit(fmt.Sprintf("/team-add %s %s", n, *task))
	}
	fmt.Printf("=== dispatched %d members: %v ===\n", *pop, names)

	// 4. Poll the roster until every member is idle again.
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Second)
		ctrl.Submit("/team-status")
		<-sink.mu
		var last string
		for _, m := range sink.msgs {
			if strings.HasPrefix(m, "team roster") || strings.Contains(m, "idle") || strings.Contains(m, "running") {
				last = m
			}
		}
		sink.mu <- struct{}{}
		fmt.Printf("[%s] roster: %q\n", time.Now().Format("15:04:05"), last)
		allIdle := true
		for _, n := range names {
			if !strings.Contains(last, n+"  idle") && !strings.Contains(last, n+" idle") {
				allIdle = false
			}
		}
		if allIdle {
			fmt.Println("=== all members idle — dissolving group ===")
			break
		}
	}
	ctrl.Submit("/team-group delete")
	time.Sleep(1 * time.Second)
	ctrl.Submit("/team-status")
	<-sink.mu
	status := strings.Join(sink.msgs, " | ")
	sink.mu <- struct{}{}
	fmt.Println("=== final:", status)
	if strings.Contains(status, "no teammates") {
		fmt.Println("GROUP_E2E_OK: members cleared after delete")
		return
	}
	fmt.Println("GROUP_E2E_FAIL: members survived group delete")
	os.Exit(1)
}
