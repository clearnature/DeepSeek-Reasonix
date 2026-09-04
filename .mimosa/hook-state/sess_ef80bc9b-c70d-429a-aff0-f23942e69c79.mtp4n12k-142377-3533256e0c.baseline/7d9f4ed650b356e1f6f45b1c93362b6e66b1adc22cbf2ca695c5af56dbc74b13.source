package main

import (
	"context"
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

// Research-domain pipeline v2: researcher -> validator -> reporter.
// Fix: verify() copies outputs to /tmp BEFORE ctrl.Close() wipes worktrees.

type captureSink struct {
	mu   sync.Mutex
	msgs []string
}

func (s *captureSink) Emit(e event.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.Kind == event.Notice {
		s.msgs = append(s.msgs, e.Text)
	}
}
func (s *captureSink) Close() {}

const outDir = "/tmp/team-research-final"

func main() {
	ws := "/home/yanli/work/DeepSeek-Reasonix"
	sessionDir, _ := os.MkdirTemp("", "team-res2-*")
	fmt.Println("sessionDir:", sessionDir)

	sink := &captureSink{}
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

	ctrl.Submit("/new")
	time.Sleep(3 * time.Second)

	ctrl.Submit("/team-create res researcher")
	ctrl.Submit("/team-create val validator")
	ctrl.Submit("/team-create rep reporter")
	time.Sleep(1 * time.Second)
	for _, n := range []string{"res", "val", "rep"} {
		ctrl.Submit("/team-grant " + n + " worktree")
	}
	time.Sleep(1 * time.Second)

	ctrl.Submit("/team-add res 你是研究者（元认知坐标 [2,2,1,0,1,1] 研究域/探究智能）。研究问题：LLM 前缀缓存（prompt caching）的机制与成本模型。产出 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/res/findings.md：①缓存机制（前缀字节匹配原理/TTL）②成本模型（hit 价 vs miss 价，官方定价依据）③业界实践（DeepSeek/Anthropic/OpenAI 的缓存定价策略对比）④对 agent 应用的启示（前缀稳定性原则）。基于公开已知事实，标注不确定处。")
	fmt.Println("=== [1/3] researcher 研究中 ===")

	lastRoster := func() string {
		for _, v := range slices.Backward(sink.msgs) {
			if strings.HasPrefix(v, "team roster") {
				return v
			}
		}
		return ""
	}
	waitIdle := func(names ...string) bool {
		deadline := time.Now().Add(9 * time.Minute)
		for time.Now().Before(deadline) {
			time.Sleep(15 * time.Second)
			ctrl.Submit("/team-status")
			r := lastRoster()
			all := true
			for _, n := range names {
				if !strings.Contains(r, "  "+n+"  idle") {
					all = false
				}
			}
			if all {
				return true
			}
		}
		return false
	}

	if !waitIdle("res") {
		fmt.Println("!!! researcher 超时")
		os.Exit(1)
	}
	fmt.Println("=== [2/3] validator 验证中（depends:task-1）===")

	ctrl.Submit("/team-add val 你是验证者（元认知坐标 [2,1,2,0,1,1] 研究域/推理智能）。验证 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/res/findings.md 的研究：①事实核查（缓存机制描述是否准确、定价数字是否有依据）②逻辑一致性（成本模型推导是否成立）③不确定性标注是否诚实。在 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/val/ 写 validation.md：每项结论的验证结果（✅证实/⚠️存疑/❌有误）+ 修正建议。 depends:task-1")
	if !waitIdle("val") {
		fmt.Println("!!! validator 超时")
		os.Exit(1)
	}
	fmt.Println("=== [3/3] reporter 报告中（depends:task-2）===")

	ctrl.Submit("/team-add rep 你是报告者（元认知坐标 [2,0,0,1,0,2] 研究域/表达智能）。读 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/res/findings.md 和 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/val/validation.md，产出最终研究报告 /home/yanli/work/DeepSeek-Reasonix/.reasonix/worktrees/rep/report.md：结论（有证实的）→ 证据 → 存疑项（标注）→ 对 agent 应用的行动建议。结构清晰可执行。 depends:task-2")
	if !waitIdle("rep") {
		fmt.Println("!!! reporter 超时")
		os.Exit(1)
	}
	fmt.Println("=== 研究域流水线全部完成 ===")
	verifyAndCopy(ws)
}

func verifyAndCopy(ws string) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Printf("WARN mkdir %s: %v\n", outDir, err)
	}
	srcs := []struct{ name, path string }{
		{"res", filepath.Join(ws, ".reasonix", "worktrees", "res", "findings.md")},
		{"val", filepath.Join(ws, ".reasonix", "worktrees", "val", "validation.md")},
		{"rep", filepath.Join(ws, ".reasonix", "worktrees", "rep", "report.md")},
	}
	fmt.Println("\n=== 研究域流水线产出验证 + 复制（Close 前）===")
	for _, s := range srcs {
		if fi, err := os.Stat(s.path); err == nil {
			dst := filepath.Join(outDir, filepath.Base(s.path))
			data, _ := os.ReadFile(s.path)
			if err := os.WriteFile(dst, data, 0o644); err != nil {
				fmt.Printf("WARN write %s: %v\n", dst, err)
			}
			fmt.Printf("OK %s -> %s (%d bytes)\n", s.name, dst, fi.Size())
		} else {
			fmt.Printf("MISS %s: %v\n", s.name, err)
		}
	}
	if b, err := os.ReadFile(filepath.Join(outDir, "report.md")); err == nil {
		fmt.Printf("\n=== 最终报告（reporter）头部 ===\n%s\n", truncate(string(b), 600))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n...（截断）"
}
