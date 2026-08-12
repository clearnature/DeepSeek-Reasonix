package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"reasonix/internal/boot"
	"reasonix/internal/event"

	_ "reasonix/internal/provider/anthropic"
	_ "reasonix/internal/provider/openai"
)

// GA engine with AUTO-APPROVAL: the population includes ask-gene individuals
// ("ask the leader when a decision is ambiguous"). The engine's sink watches
// for AskRequest events and answers them automatically (first option), so ask
// individuals complete their evaluation instead of hanging forever waiting for
// a human.
//
// KNOWN GAP (2026-08-12, verified by experiment): sub-agents (teammates) are
// spawned with a nil Asker, so AskTool.Call returns a headless model-assumption
// fallback and NEVER emits AskRequest — the auto-approver below is currently
// unreachable for teammate asks. The ask-gene evaluation therefore does NOT
// exercise the approval chain until "sub-agent asker injection" lands
// (teammate inherits the leader's Asker). See docs/team/20260812-subagent-asker/.
// Product behavior untouched: Controller.Ask still waits for a real user; the
// engine drives it programmatically via AnswerQuestion.

type captureSink struct {
	mu   sync.Mutex
	msgs []string
	ask  chan event.Ask
}

func (s *captureSink) Emit(e event.Event) {
	if e.Kind == event.AskRequest {
		select {
		case s.ask <- e.Ask:
		default:
		}
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.Kind == event.Notice {
		s.msgs = append(s.msgs, e.Text)
	}
}
func (s *captureSink) Close() {}

func main() {
	ws := "/home/yanli/work/DeepSeek-Reasonix"
	sessionDir, _ := os.MkdirTemp("", "team-ga2-*")
	fmt.Println("sessionDir:", sessionDir)

	sink := &captureSink{ask: make(chan event.Ask, 8)}
	ctrl, err := boot.Build(context.Background(), boot.Options{
		Sink:          sink,
		WorkspaceRoot: ws,
		SessionDir:    sessionDir,
		StatsSource:   "desktop",
	})
	if err != nil {
		fmt.Println("boot.Build:", err)
		os.Exit(1)
	}
	defer ctrl.Close()

	// Headless run: wire the controller as the executor's Asker (silent gate
	// kept) so teammate `ask` reaches the approval chain instead of the
	// nil-asker fallback — the auto-approver below then answers it.
	ctrl.EnableHeadlessAsker()

	// Auto-approver: answer every AskRequest with the first option.
	go func() {
		for ask := range sink.ask {
			var answers []event.AskAnswer
			for _, q := range ask.Questions {
				if len(q.Options) > 0 {
					answers = append(answers, event.AskAnswer{QuestionID: q.ID, Selected: []string{q.Options[0].Label}})
				}
			}
			fmt.Printf("AUTO-APPROVE request=%s questions=%d -> %s\n", ask.ID, len(ask.Questions), jsonAnswers(answers))
			ctrl.AnswerQuestion(ask.ID, answers)
		}
	}()

	ctrl.Submit("/new")
	time.Sleep(3 * time.Second)

	// Population of 3: two plain variants + one ASK-GENE variant.
	ctrl.Submit("/team-create g1 coder")
	ctrl.Submit("/team-create g2 coder")
	ctrl.Submit("/team-create g3 coder")
	time.Sleep(1 * time.Second)
	for _, n := range []string{"g1", "g2", "g3"} {
		ctrl.Submit("/team-grant " + n + " worktree")
	}
	time.Sleep(1 * time.Second)

	ctrl.Submit("/team-add g1 在 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/g1/ 创建 order.go：实现插入排序（InsertionSort），Go 语言 package sortx。")
	ctrl.Submit("/team-add g2 在 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/g2/ 创建 order.go：实现选择排序（SelectionSort），Go 语言 package sortx。")
	ctrl.Submit("/team-add g3 在 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/g3/ 创建 order.go：实现冒泡排序（BubbleSort），Go 语言 package sortx。开始编码前必须先向 leader 提问一个明确的实现决策（用 ask 工具：例如排序是否原地修改输入切片、是否添加单元测试、错误处理策略三者选一），等待 leader 的回答后再继续，最后严格按回答完成实现。")
	fmt.Println("=== 种群 3 个体并行评估（g3 = ask 基因）===")

	lastRoster := func() string {
		for _, v := range slices.Backward(sink.msgs) {
			if strings.HasPrefix(v, "team roster") {
				return v
			}
		}
		return ""
	}

	start := time.Now()
	deadline := start.Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(15 * time.Second)
		ctrl.Submit("/team-status")
		r := lastRoster()
		if strings.Contains(r, "g1  idle") && strings.Contains(r, "g2  idle") && strings.Contains(r, "g3  idle") {
			fmt.Println("=== 种群评估完成 ===")
			fmt.Println(r)
			evaluate(ws)
			return
		}
	}
	fmt.Println("!!! 超时")
	os.Exit(1)
}

func jsonAnswers(a []event.AskAnswer) string {
	b, _ := json.Marshal(a)
	return string(b)
}

func evaluate(ws string) {
	// Fitness: produced order.go in own worktree = 1.0. Ask-gene individual
	// completing after auto-approval gets a FULL evaluation (no -0.3 penalty
	// guesswork — the ask was actually answered).
	fmt.Println("\n=== GA 适应度评估 ===")
	for _, n := range []string{"g1", "g2", "g3"} {
		p := filepath.Join(ws, ".reasonix", "worktrees", n, "order.go")
		if fi, err := os.Stat(p); err == nil {
			fmt.Printf("g%s fitness=1.0 (%s, %d bytes)\n", n, strings.TrimSuffix(n, ""), fi.Size())
		} else {
			fmt.Printf("g%s fitness=0.0 (no output: %v)\n", n, err)
		}
	}
}
