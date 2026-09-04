// team-test1 drives the P6 TeammateStore through the controller host commands
// to extend /home/yanli/work/test1 (SimpleCalc) with UI-layer unit tests.
//
// Three teammates run in parallel, each writing one zero-dependency node:test
// file under test1/test/ (node:vm stubs for the browser-only calc-ui.js).
// Grant mode is "path": test1 has no git, so worktree isolation is unusable.
//
// Usage: go run ./cmd/team-test1 [-pop 3]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"reasonix/internal/boot"
	"reasonix/internal/event"
)

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

// taskFor names one UI test file and its coverage scope. Each teammate gets a
// disjoint file so parallel writes to the shared test dir never collide.
func taskFor(n, target, scope string) string {
	return fmt.Sprintf(`在 %s 项目（SimpleCalc 计算器）完成 calc-ui 层单元测试开发。
背景：项目已实现（calc-core.js 407 行 / calc-ui.js 188 行 / index.html / style.css），
现有 test/calc-core.test.js 81 断言全过（node --test test/ 零 npm 依赖，Node>=18）。
calc-ui.js 是浏览器专属 IIFE（仅 window.CalcUI，无 module.exports），内部有
KEY_TO_ACTION 键盘映射表、actionFromKey、initTheme（prefers-color-scheme +
localStorage，A-6 降级）、render（is-error class）。SPEC.md（FR-8 键盘、NFR-3
无障碍、NFR-4 深色）与 ARCHITECTURE.md（A-1~A-6）是需求依据，先读这两个文件
和 calc-ui.js 再写。

你的任务：%s
要求：
1. 在 %s 目录创建测试文件 test/calc-ui-%s.test.js，用 Node 内置 node:test +
   node:assert/strict（零 npm 依赖）；用 node:vm 加载 calc-ui.js（模拟
   window.CalcCore、window 的 matchMedia/addEventListener、container 最小
   DOM 桩：querySelector/getElementById 返回带 classList/addEventListener/
   textContent 的桩元素），不要引入 jsdom 或任何第三方包。
2. 测试必须真实可运行：用 `+"`node --test test/calc-ui-%s.test.js`"+` 自测通过后
   才算完成（文件必须在 %s 根下能通过 node --test 全量运行）。
3. 遇到无安全默认的实现决策时，用 ask 工具向 leader 提问并按其回答继续。`,
		target, scope, target, n, n, target)
}

func main() {
	pop := flag.Int("pop", 3, "number of parallel teammates")
	flag.Parse()

	// Cross-project test: workspace is Reasonix, but the grant points at
	// test1 via NormalizeWritePathsExternal (user-level explicit grant).
	ws := "/home/yanli/work/test1"
	target := "/home/yanli/work/test1"
	sessionDir, _ := os.MkdirTemp("", "team-test1-*")
	fmt.Println("sessionDir:", sessionDir, "target:", target, "pop:", *pop)
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

	ctrl.EnableHeadlessAsker()
	go func() {
		for ask := range sink.ask {
			var answers []event.AskAnswer
			for _, q := range ask.Questions {
				if len(q.Options) > 0 {
					answers = append(answers, event.AskAnswer{QuestionID: q.ID, Selected: []string{q.Options[0].Label}})
				}
			}
			fmt.Printf("AUTO-APPROVE %s -> %s\n", ask.ID, jsonAnswers(answers))
			ctrl.AnswerQuestion(ask.ID, answers)
		}
	}()

	ctrl.Submit("/new")
	time.Sleep(3 * time.Second)

	// Three teammates: keyboard map, theme, a11y+error-state render. Each owns
	// one test file under test1/test/ (shared dir, disjoint filenames).
	tasks := []struct {
		name  string
		scope string
	}{
		{"kbd", "为键盘映射写单测（FR-8 全表：0-9 . + - * / Enter = Backspace Escape Delete → 正确 action；未映射键 Tab/方向键/字母 → 无 action 无副作用；PREVENT_KEYS 命中键 preventDefault 被调用、未命中不调用）。用 vm 桩捕获 window keydown 回调并手动触发，断言 handleAction 经 CalcCore 得到正确结果。"},
		{"theme", "为主题逻辑写单测（NFR-4：默认跟随 prefers-color-scheme dark/light；toggle 点击切换主题类；localStorage 持久化（再次 init 恢复）；A-6：localStorage.getItem/setItem 抛异常时降级内存保持不崩溃；深色类名与浅色类名互斥）。vm 桩：matchMedia 返回可控对象、localStorage 用内存桩。"},
		{"a11y", "为无障碍与错误态渲染写单测（NFR-3：所有按钮元素可读名称——DOM 桩收集按钮并断言 aria-label/文本非空；错误态：core 进入 PHASES.ERROR 时 render 给显示区加 is-error class、显示「错误」；AC/数字键退出错误态后 is-error 移除）。vm 桩：CalcCore 用真 calc-core.js（require）包一层，container 桩暴露按钮列表。"},
	}
	names := make([]string, *pop)
	for i := range *pop {
		names[i] = tasks[i%len(tasks)].name + fmt.Sprintf("%d", i/len(tasks))
	}
	for _, n := range names {
		ctrl.Submit("/team-create " + n + " coder")
	}
	time.Sleep(1 * time.Second)
	for _, n := range names {
		ctrl.Submit("/team-grant " + n + " /tmp/team-external-test")
	}
	time.Sleep(1 * time.Second)

	for i, n := range names {
		t := tasks[i%len(tasks)]
		ctrl.Submit(fmt.Sprintf("/team-add "+n+" %s", taskFor(t.name, target, t.scope)))
	}
	fmt.Printf("=== team-test1：%d 个 teammate 并行开发 calc-ui 单测（%s）===\n", *pop, target)

	lastRoster := func() string {
		for _, v := range slices.Backward(sink.msgs) {
			if strings.HasPrefix(v, "team roster") {
				return v
			}
		}
		return ""
	}

	deadline := start.Add(12 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(15 * time.Second)
		ctrl.Submit("/team-status")
		r := lastRoster()
		fmt.Printf("[%s] roster: %q msgs=%d\n", time.Now().Format("15:04:05"), r, len(sink.msgs))
		sink.mu.Lock()
		for i, m := range sink.msgs {
			if i >= len(sink.msgs)-12 {
				fmt.Printf("    msg: %.120s\n", m)
			}
		}
		sink.mu.Unlock()
		allIdle := true
		for _, n := range names {
			if !strings.Contains(r, n+"  idle") {
				allIdle = false
				break
			}
		}
		if allIdle {
			fmt.Println("=== teammate 全部完成 ===")
			fmt.Println(r)
			verify(target, names)
			return
		}
	}
	fmt.Println("!!! 超时（12 分钟）")
	fmt.Println(lastRoster())
	verify(target, names)
	os.Exit(1)
}

// verify lists the produced test files and runs the full node --test suite.
func verify(target string, names []string) {
	fmt.Println("\n=== 产出清单 ===")
	for _, f := range []string{"calc-ui-kbd.test.js", "calc-ui-theme.test.js", "calc-ui-a11y.test.js"} {
		p := filepath.Join(target, "test", f)
		if fi, err := os.Stat(p); err == nil {
			fmt.Printf("  %s (%d bytes)\n", f, fi.Size())
		} else {
			fmt.Printf("  %s — 缺失\n", f)
		}
	}
	fmt.Println("\n=== node --test test/ 全量验证 ===")
	cmd := exec.Command("node", "--test",
		"test/calc-core.test.js", "test/calc-ui-kbd.test.js",
		"test/calc-ui-theme.test.js", "test/calc-ui-a11y.test.js")
	cmd.Dir = target
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		fmt.Println("验证失败:", err)
		os.Exit(1)
	}
	fmt.Println("=== 全量测试通过 ===")
}

func jsonAnswers(a []event.AskAnswer) string {
	b, _ := json.Marshal(a)
	return string(b)
}
