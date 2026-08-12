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
	mu        sync.Mutex
	msgs      []string
	ask       chan event.Ask
	askCount  int
}

func (s *captureSink) Emit(e event.Event) {
	if e.Kind == event.AskRequest {
		s.mu.Lock()
		s.askCount++
		s.mu.Unlock()
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

func (s *captureSink) AskRequests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.askCount
}

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

	// Generation 3: worktree-grant × ask-gene variation. Same sorting task for
	// all three individuals; only the ask gene differs:
	//   g1 no-ask   — no ask guidance (fully autonomous)
	//   g2 natural  — natural guidance (ask only when no safe default)
	//   g3 force    — forced ask first (G2 gene, kept as control)
	// Fitness + AskRequest count drive the inter-generation comparison.
	ctrl.Submit("/team-create g1 coder")
	ctrl.Submit("/team-create g2 coder")
	ctrl.Submit("/team-create g3 coder")
	time.Sleep(1 * time.Second)
	for _, n := range []string{"g1", "g2", "g3"} {
		ctrl.Submit("/team-grant " + n + " worktree")
	}
	time.Sleep(1 * time.Second)

	const task = "在 %s 创建 order.go：实现冒泡排序（BubbleSort），Go 语言 package sortx，返回排序后的新切片（不修改输入）。"
	ctrl.Submit(fmt.Sprintf("/team-add g1 "+task, "/home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/g1/"))
	ctrl.Submit(fmt.Sprintf("/team-add g2 "+task+" 遇到无安全默认的实现决策时（例如参数校验策略、命名风格），用 ask 工具向 leader 提问并按其回答继续。", "/home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/g2/"))
	ctrl.Submit(fmt.Sprintf("/team-add g3 "+task+" 开始编码前必须先向 leader 提问一个明确的实现决策（用 ask 工具），等待 leader 的回答后再继续，最后严格按回答完成实现。", "/home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/g3/"))
	fmt.Println("=== G3 种群 3 个体并行评估（worktree-grant × ask 基因：no/natural/force）===")

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
			evaluate(ws, sink)
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

func evaluate(ws string, sink *captureSink) {
	// Fitness: produced order.go in own worktree = 1.0. AskRequest count
	// measures autonomy (fewer asks = fewer interruptions); the inter-
	// generation comparison weighs fitness vs ask frequency.
	fmt.Println("\n=== G3 适应度评估（fitness × ask 次数）===")
	genes := map[string]string{"g1": "no-ask", "g2": "natural", "g3": "force"}
	for _, n := range []string{"g1", "g2", "g3"} {
		p := filepath.Join(ws, ".reasonix", "worktrees", n, "order.go")
		if fi, err := os.Stat(p); err == nil {
			fmt.Printf("g%s (%-7s) fitness=1.0 (%d bytes)\n", n, genes[n], fi.Size())
		} else {
			fmt.Printf("g%s (%-7s) fitness=0.0 (no output: %v)\n", n, genes[n], err)
		}
	}
	fmt.Printf("AskRequest 总数: %d（g1 no-ask 应≈0，g3 force 应≥1）\n", sink.AskRequests())
	fmt.Println("=== G3 结论：natural 是否以最少 ask 达成 fitness=1.0（自主性最优）===")
}
