package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
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
	mu       sync.Mutex
	msgs     []string
	ask      chan event.Ask
	askCount int
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
	grantMode := flag.String("grant", "worktree", "teammate grant mode: worktree (git worktree isolation) | path (WritePathSet path grant, shared workspace)")
	flag.Parse()

	ws := "/home/yanli/work/DeepSeek-Reasonix"
	sessionDir, _ := os.MkdirTemp("", "team-ga2-*")
	fmt.Println("sessionDir:", sessionDir, "grant:", *grantMode)
	start := time.Now()

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

	// Generation 4: grant-mode comparison (worktree vs path) × natural ask.
	// Same sorting task for all three individuals; only the isolation mode
	// differs per run (-grant). Fitness + cache stability (prefix_hash churn,
	// hit rate) drive the evaluation: path-grant shares the leader workspace,
	// so tool outputs (ls/glob/grep) may jitter and pollute the cache prefix.
	grantDir := func(n string) string {
		if *grantMode == "path" {
			d := filepath.Join(ws, ".reasonix", "ga-g4", n)
			_ = os.MkdirAll(d, 0o755)
			return d
		}
		return filepath.Join(ws, ".reasonix", "worktrees", n)
	}
	ctrl.Submit("/team-create g1 coder")
	ctrl.Submit("/team-create g2 coder")
	ctrl.Submit("/team-create g3 coder")
	time.Sleep(1 * time.Second)
	for _, n := range []string{"g1", "g2", "g3"} {
		if *grantMode == "path" {
			ctrl.Submit("/team-grant " + n + " " + grantDir(n))
		} else {
			ctrl.Submit("/team-grant " + n + " worktree")
		}
	}
	time.Sleep(1 * time.Second)

	const task = "在 %s 完成一个冒泡排序小包：① 创建 order.go：实现 BubbleSort（Go 语言 package sortx，返回排序后的新切片，不修改输入）；② 创建 order_test.go：为 BubbleSort 写至少 3 个单元测试（含空切片、单元素、乱序）；③ 创建 README.md：一行说明该包用途与用法。遇到无安全默认的实现决策时，用 ask 工具向 leader 提问并按其回答继续。"
	for _, n := range []string{"g1", "g2", "g3"} {
		ctrl.Submit(fmt.Sprintf("/team-add "+n+" "+task, grantDir(n)))
	}
	fmt.Printf("=== G5 种群 3 个体并行评估（grant=%s × 多文件任务 × natural ask）===\n", *grantMode)

	lastRoster := func() string {
		for _, v := range slices.Backward(sink.msgs) {
			if strings.HasPrefix(v, "team roster") {
				return v
			}
		}
		return ""
	}

	deadline := start.Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(15 * time.Second)
		ctrl.Submit("/team-status")
		r := lastRoster()
		if strings.Contains(r, "g1  idle") && strings.Contains(r, "g2  idle") && strings.Contains(r, "g3  idle") {
			fmt.Println("=== 种群评估完成 ===")
			fmt.Println(r)
			evaluate(ws, sink, start)
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

func evaluate(ws string, sink *captureSink, start time.Time) {
	// Fitness: produced order.go in own worktree = 1.0. AskRequest count
	// measures autonomy; cache stats measure prefix stability (path-grant
	// shares the workspace, so tool outputs may jitter the prefix).
	fmt.Println("\n=== G5 适应度评估（多文件 fitness × ask 次数）===")
	genes := map[string]string{"g1": "natural", "g2": "natural", "g3": "natural"}
	for _, n := range []string{"g1", "g2", "g3"} {
		dirs := []string{
			filepath.Join(ws, ".reasonix", "worktrees", n),
			filepath.Join(ws, ".reasonix", "ga-g4", n),
		}
		dir := ""
		for _, d := range dirs {
			if fi, err := os.Stat(filepath.Join(d, "order.go")); err == nil && fi.Size() > 0 {
				dir = d
				break
			}
		}
		if dir == "" {
			fmt.Printf("g%s (%-7s) fitness=0.0 (no order.go)\n", n, genes[n])
			continue
		}
		// Multi-file fitness: order.go + order_test.go + README.md each count.
		score := 0.0
		var sizes []string
		for _, f := range []string{"order.go", "order_test.go", "README.md"} {
			if fi, err := os.Stat(filepath.Join(dir, f)); err == nil && fi.Size() > 0 {
				score += 1.0 / 3
				sizes = append(sizes, f+":"+fmt.Sprint(fi.Size()))
			}
		}
		fmt.Printf("g%s (%-7s) fitness=%.2f [%s]\n", n, genes[n], score, strings.Join(sizes, " "))
	}
	fmt.Printf("AskRequest 总数: %d\n", sink.AskRequests())
	cacheStats(start, time.Now())
	fmt.Println("=== G5 结论：多文件任务下 path-grant 是否仍无缓存污染（prefix 抖动/命中率）===")
}

// cacheStats summarizes prefix stability and hit rate for the experiment
// window from the daily stats file — the cache-pollution signal for path-grant.
func cacheStats(start, end time.Time) {
	statsHome := os.Getenv("REASONIX_STATE_HOME")
	if statsHome == "" {
		home, _ := os.UserHomeDir()
		statsHome = filepath.Join(home, ".reasonix")
	}
	day := start.Format("2006-01-02")
	f, err := os.Open(filepath.Join(statsHome, "stats", day+".jsonl"))
	if err != nil {
		fmt.Println("cacheStats: 无 stats 文件:", err)
		return
	}
	defer f.Close()
	var reqs, hits, misses int
	phs := map[string]struct{}{}
	lo := start.Format("2006-01-02T15:04")
	hi := end.Format("2006-01-02T15:04")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r struct {
			Ts         string `json:"ts"`
			CacheHit   int    `json:"cache_hit"`
			CacheMiss  int    `json:"cache_miss"`
			PrefixHash string `json:"prefix_hash"`
		}
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil || len(r.Ts) < 16 {
			continue
		}
		// ISO timestamps sort lexicographically; the minute window is enough
		// to isolate the experiment (sessionDir start → evaluation end).
		if r.Ts[:16] < lo || r.Ts[:16] > hi {
			continue
		}
		reqs++
		hits += r.CacheHit
		misses += r.CacheMiss
		if r.PrefixHash != "" {
			phs[r.PrefixHash] = struct{}{}
		}
	}
	total := hits + misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	fmt.Printf("缓存指标（实验窗口 %s→%s）: 请求=%d prefix_hash唯一=%d 命中率=%.2f%% miss=%d\n",
		start.Format("15:04:05"), end.Format("15:04:05"), reqs, len(phs), hitRate, misses)
}
