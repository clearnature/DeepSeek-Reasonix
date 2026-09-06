package agent

// T4 组合/端到端测试：完成事件驱动的完整链路
// (Assign → background job → recordCompletion → SetJobDoneObserver →
//  TeammateStore.HandleJobDone → 置 idle / 依赖自动推进 / mailbox 唤醒)。
//
// 契约（E1/E2 已合入）：
//   - E1：jobs.Manager.SetJobDoneObserver(func(id string, st jobs.Status, err error))
//     （recordCompletion 两处无锁点同步触发，per-observer recover，test-strategy D4）
//   - E2：TeammateStore.HandleJobDone + NewTeammateStore 内自动注册
//     （tm.SetJobDoneObserver(ts.HandleJobDone)，teammate_store.go:111）
//
// 测试全部走真实 job 全链：不直接调用 HandleJobDone，而是通过真实后台 job 的
// 完成事件驱动 —— 同时验证「NewTeammateStore 自动注册」这条生产接线。
// 确定性等待点：recordCompletion → fireJobDoneObservers → close(j.done) 同一
// goroutine 串行 ⇒ WaitForSession 返回时 handler 必已执行完（test-strategy D4）。
//
// 基建复用 teammate_store_test.go：teamAssignCtx / newTaskToolWith /
// releaseBlockingProvider（保持 job 运行到测试释放的慢 provider）/ testTaskToolForTeam
// （mockProvider 立即 done 的最快变体）。
//
// 组合测试与 teammate_store_test.go 的确定性测试互补：
// 后者直接注入 HandleJobDone 覆盖业务分支，前者用真实完成事件验证接线。

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"reasonix/internal/event"
	"reasonix/internal/jobs"
)

// TestTeammateDoneIdleAllowsReassign（任务① = C1 组合版 + 最大用户价值）：
// teammate 完成第一个任务后 State 直接为 Idle（不调 List/Tasks 也 idle —— 证明
// 事件驱动而非懒同步），且第二次 Assign 不再被 running 守卫拒绝。
func TestTeammateDoneIdleAllowsReassign(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)

	first, err := ts.Assign(ctx, "alpha", "first task")
	if err != nil {
		t.Fatalf("first Assign: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{first}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("first job = %+v, want Done", res)
	}

	// Status() 不触发 syncStateLocked 懒同步 —— State 若为 Idle 只能是完成事件
	// 驱动（NewTeammateStore 自动注册的 HandleJobDone）置的。
	tm, ok := ts.Status("alpha")
	if !ok || tm.State != TeammateIdle {
		t.Fatalf("alpha State = %+v (ok=%v), want Idle driven by completion event", tm, ok)
	}

	// 第二次 Assign 不再被 running 守卫拒绝（完成事件驱动最大用户价值）。
	second, err := ts.Assign(ctx, "alpha", "second task")
	if err != nil {
		t.Fatalf("second Assign after completion rejected: %v", err)
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{second}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("second job = %+v, want Done", res)
	}
}

// TestTeammateDoneDependencyChainAutoAdvance（任务② = C2 组合版 + 仲裁 3 自动推进）：
// beta dependsOn=[alpha 的 job]。alpha 运行中 beta 的 Assign 被依赖门拒绝并登记
// 为等待任务（pending）；alpha 的完成事件驱动 HandleJobDone 自动把 beta 派活并
// 跑完 —— 断言顺序（alpha 先终态）与终态（都 Done、beta 回 Idle、tasks 补存终态）。
func TestTeammateDoneDependencyChainAutoAdvance(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	sub := &releaseBlockingProvider{name: "slow", release: make(chan struct{})}
	ts := NewTeammateStore(newTaskToolWith(t, sub), jm)
	defer ts.Close()
	for _, n := range []string{"alpha", "beta"} {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)

	jobA, err := ts.Assign(ctx, "alpha", "first step")
	if err != nil {
		t.Fatalf("alpha Assign: %v", err)
	}
	// beta 依赖 jobA，而 jobA 仍在运行：依赖门拒绝立即派活，并将 beta 登记为
	// 等待任务（pending）—— alpha 的完成事件将自动推进它（仲裁 3）。
	if _, err := ts.Assign(ctx, "beta", "second step", jobA); err == nil ||
		!strings.Contains(err.Error(), "blocked by unfinished dependency") {
		t.Fatalf("beta dependent Assign = %v, want blocked by dependency", err)
	}
	if tm, ok := ts.Status("beta"); !ok || tm.LastJobID != "" {
		t.Fatalf("beta got a job before alpha completed: %+v", tm)
	}

	// 释放 alpha 的 job：完成事件驱动 HandleJobDone → 自动推进 → beta 被派活。
	close(sub.release)
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobA}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("alpha job = %+v, want Done", res)
	}
	jobB := waitForTeammateJob(t, ts, "beta", 5*time.Second)
	if jobB == "" {
		t.Fatal("beta was not auto-assigned a job after alpha completed")
	}
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobB}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("beta job = %+v, want Done", res)
	}

	// 顺序：jobA 在 jobB 之前终态（结构保证：jobA 未 done 时 beta 不会启动）。
	// 终态：两个被跟踪任务都补存为 done；beta 回 Idle。
	for _, task := range ts.Tasks() {
		if task.ID != jobA && task.ID != jobB {
			continue
		}
		if task.Status != jobs.Done {
			t.Errorf("task %s Status = %q, want done", task.ID, task.Status)
		}
	}
	if tm, _ := ts.Status("beta"); tm.State != TeammateIdle {
		t.Fatalf("beta State = %q after its auto-assigned job, want Idle", tm.State)
	}
}

// TestTeammateDoneMailboxBacklogWakeup（任务③ = C3 组合版 + 仲裁 4 聚合唤醒）：
// alpha 运行期间 PostMail 2 封（落盘积压、flushMailbox 不跑）→ alpha 完成 →
// leader sink 收到一次聚合 Notice（含 N=2）。运行中 PostMail + 完成唤醒正是
// mailbox 唤醒的价值场景（test-strategy C3）。
func TestTeammateDoneMailboxBacklogWakeup(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	var mu sync.Mutex
	var notices []event.Event
	sub := &releaseBlockingProvider{name: "slow", release: make(chan struct{})}
	ts := NewTeammateStore(newTaskToolWith(t, sub), jm, t.TempDir())
	defer ts.Close()
	ts.SetSink(event.FuncSink(func(e event.Event) {
		mu.Lock()
		notices = append(notices, e)
		mu.Unlock()
	}))
	if err := ts.Create("alpha", "worker"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	ctx := teamAssignCtx(jm)

	jobA, err := ts.Assign(ctx, "alpha", "first task")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	// alpha 运行中：mail 落盘积压（flushMailbox 只在 Assign 时跑，不消费）。
	if err := ts.PostMail("alpha", "mail one"); err != nil {
		t.Fatalf("PostMail 1: %v", err)
	}
	if err := ts.PostMail("alpha", "mail two"); err != nil {
		t.Fatalf("PostMail 2: %v", err)
	}

	close(sub.release)
	if res := jm.WaitForSession(context.Background(), "leader-session", []string{jobA}, 5); len(res) != 1 || res[0].Status != jobs.Done {
		t.Fatalf("alpha job = %+v, want Done", res)
	}

	// 聚合 Notice 恰好一条且携带 N=2。PostMail 的即时通知文案是
	// "teammate X received mail"（不含计数），不会误命中含 "2" 的断言；
	// 聚合文案 "teammate X has N unread mail" 含 N。
	var agg []string
	mu.Lock()
	for _, e := range notices {
		if strings.Contains(e.Text, "alpha") && strings.Contains(e.Text, "2") {
			agg = append(agg, e.Text)
		}
	}
	mu.Unlock()
	if len(agg) != 1 {
		t.Fatalf("aggregate backlog notices = %v, want exactly one carrying N=2", agg)
	}
	if !strings.Contains(agg[0], "unread mail") {
		t.Errorf("aggregate notice = %q, want the unread-backlog wording", agg[0])
	}
}

// TestTeammateDoneConcurrentAssignCompletion（任务④ = -race 组合）：
// 多个 teammate 并发 Assign + 真实完成事件交错：不 panic、不死锁、终态合法。
// 配合 `go test -race` 作为锁序/并发护栏（test-strategy R1 骨架 + 真实事件）。
func TestTeammateDoneConcurrentAssignCompletion(t *testing.T) {
	jm := jobs.NewManager(event.Discard)
	defer jm.Close()
	ts := NewTeammateStore(testTaskToolForTeam(t), jm)
	defer ts.Close()
	// Two teammates keep concurrent Assign below the background-task slot cap
	// (maxConcurrentBackgroundTasks), so Assign never fails on the session
	// limit while still exercising interleaved completion events under -race.
	names := []string{"alpha", "beta"}
	for _, n := range names {
		if err := ts.Create(n, "worker"); err != nil {
			t.Fatalf("Create %s: %v", n, err)
		}
	}
	ctx := teamAssignCtx(jm)

	var wg sync.WaitGroup
	for _, n := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			for i := range 2 {
				id, err := ts.Assign(ctx, name, fmt.Sprintf("task %d", i))
				if err != nil {
					t.Errorf("%s Assign round %d: %v", name, i, err)
					return
				}
				res := jm.WaitForSession(context.Background(), "leader-session", []string{id}, 5)
				if len(res) != 1 || res[0].Status != jobs.Done {
					t.Errorf("%s job %s = %+v, want Done", name, id, res)
					return
				}
			}
		}(n)
	}
	wg.Wait()

	// 每个 teammate 最后一个 job 已 WaitForSession（⇒ handler 已执行）→ 全 Idle。
	for _, n := range names {
		if tm, ok := ts.Status(n); !ok || tm.State != TeammateIdle {
			t.Errorf("%s final State = %+v, want Idle", n, tm)
		}
	}
}

// waitForTeammateJob 轮询 teammate 的 LastJobID：自动推进在完成事件里 Assign
// 后继任务后写入新 jobID，测试据此拿到自动派活的 job 并 WaitForSession。
// （自动推进可能由完成事件的同步回调或 worker 路径完成，轮询兼容两种时序。）
func waitForTeammateJob(t *testing.T, ts *TeammateStore, name string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if tm, ok := ts.Status(name); ok && tm.LastJobID != "" {
			return tm.LastJobID
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ""
}
