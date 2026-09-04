package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// forkPrefixTestAgent 构造一个持有给定消息历史的父 Agent（NewSession 自动加
// RoleSystem 首条）。
func forkPrefixTestAgent(t *testing.T, msgs []provider.Message) *Agent {
	t.Helper()
	sess := NewSession("sys")
	for _, m := range msgs {
		sess.Add(m)
	}
	return New(&scriptedProvider{name: "p"}, tool.NewRegistry(), sess, Options{}, event.Discard)
}

func marshalMessages(t *testing.T, msgs []provider.Message) string {
	t.Helper()
	b, err := json.Marshal(msgs)
	if err != nil {
		t.Fatalf("marshal messages: %v", err)
	}
	return string(b)
}

func forkToolMsg(id, name, content string) provider.Message {
	return provider.Message{Role: provider.RoleTool, ToolCallID: id, Name: name, Content: content}
}

func forkAssistantCalls(calls ...provider.ToolCall) provider.Message {
	return provider.Message{Role: provider.RoleAssistant, ToolCalls: calls}
}

// TestCaptureForkPrefixByteIdentical 验证前缀与父已发送字节 byte-identical：
// 父历史末尾是当前未完成的 assistant 轮次（task 工具调用已发出但结果未配对），
// captureForkPrefix 必须精确剔除它，输出与父最后一次请求的完整消息逐字节一致
// —— 这是子代理首请求命中父已建缓存的硬前提。
func TestCaptureForkPrefixByteIdentical(t *testing.T) {
	parent := forkPrefixTestAgent(t, []provider.Message{
		{Role: provider.RoleUser, Content: "fix the widget"},
		forkAssistantCalls(provider.ToolCall{ID: "c1", Name: "ls", Arguments: `{}`}),
		forkToolMsg("c1", "ls", "ok"),
		// 当前轮次：task 工具调用已发出，结果尚未追加 —— 未配对，必须剔除。
		forkAssistantCalls(provider.ToolCall{ID: "c2", Name: "task", Arguments: `{"prompt":"x"}`}),
	})

	got := captureForkPrefix(parent, context.Background())

	// 父已发送 = system + 完整配对的已提交轮次。
	want := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "fix the widget"},
		forkAssistantCalls(provider.ToolCall{ID: "c1", Name: "ls", Arguments: `{}`}),
		forkToolMsg("c1", "ls", "ok"),
	}
	if len(got) != len(want) {
		t.Fatalf("prefix length = %d, want %d\ngot:  %s\nwant: %s", len(got), len(want), marshalMessages(t, got), marshalMessages(t, want))
	}
	if got := marshalMessages(t, got); got != marshalMessages(t, want) {
		t.Fatalf("prefix bytes differ from parent-sent bytes\n got: %s\nwant: %s", got, marshalMessages(t, want))
	}
	// 交叉验证：两者经 wire 过滤（ModelMessages+NormalizeMessages）后仍一致，
	// 保证真实请求路径（T5 e2e）上缓存前缀不变。
	if gotW := marshalMessages(t, provider.ModelMessages(provider.NormalizeMessages(got))); gotW != marshalMessages(t, provider.ModelMessages(provider.NormalizeMessages(want))) {
		t.Fatalf("wire-filtered prefix bytes differ\n got: %s", gotW)
	}
}

// TestTruncateUnfinishedTurn 表驱动验证截断正确性：LocalOnly 剔除、未配对
// assistant 剔除、批量部分配对整轮剔除、配对完整/纯文本回复保留。
func TestTruncateUnfinishedTurn(t *testing.T) {
	c1 := provider.ToolCall{ID: "c1", Name: "ls", Arguments: `{}`}
	c2 := provider.ToolCall{ID: "c2", Name: "task", Arguments: `{}`}
	cases := []struct {
		name string
		in   []provider.Message
		want []provider.Message
	}{
		{
			name: "tail_unpaired_assistant_calls",
			in: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				forkAssistantCalls(c1), forkToolMsg("c1", "ls", "ok"),
				forkAssistantCalls(c2), // 未配对
			},
			want: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				forkAssistantCalls(c1), forkToolMsg("c1", "ls", "ok"),
			},
		},
		{
			name: "tail_local_only_stream",
			in: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				forkAssistantCalls(c1), forkToolMsg("c1", "ls", "ok"),
				{Role: provider.RoleAssistant, Content: "partial", LocalOnly: true}, // 流式输出未完成
			},
			want: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				forkAssistantCalls(c1), forkToolMsg("c1", "ls", "ok"),
			},
		},
		{
			name: "batch_partially_paired",
			in: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{c1, c2}},
				forkToolMsg("c1", "ls", "ok"), // c2 结果缺失 → 整轮未完成
			},
			want: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
			},
		},
		{
			name: "batch_fully_paired",
			in: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{c1, c2}},
				forkToolMsg("c1", "ls", "ok"),
				forkToolMsg("c2", "task", "done"),
			},
			want: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{c1, c2}},
				forkToolMsg("c1", "ls", "ok"),
				forkToolMsg("c2", "task", "done"),
			},
		},
		{
			name: "plain_text_assistant_kept",
			in: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				{Role: provider.RoleAssistant, Content: "done"},
			},
			want: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				{Role: provider.RoleAssistant, Content: "done"},
			},
		},
		{
			name: "local_only_then_unpaired",
			in: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				forkAssistantCalls(c1), forkToolMsg("c1", "ls", "ok"),
				{Role: provider.RoleAssistant, LocalOnly: true, ToolCalls: []provider.ToolCall{c2}},
				forkAssistantCalls(c2), // 未配对
			},
			want: []provider.Message{
				{Role: provider.RoleUser, Content: "u"},
				forkAssistantCalls(c1), forkToolMsg("c1", "ls", "ok"),
			},
		},
		{
			name: "empty_and_no_assistant_unchanged",
			in:   []provider.Message{{Role: provider.RoleUser, Content: "u"}, forkToolMsg("c1", "ls", "ok")},
			want: []provider.Message{{Role: provider.RoleUser, Content: "u"}, forkToolMsg("c1", "ls", "ok")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateUnfinishedTurn(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d\ngot:  %s\nwant: %s", len(got), len(tc.want), marshalMessages(t, got), marshalMessages(t, tc.want))
			}
			if g, w := marshalMessages(t, got), marshalMessages(t, tc.want); g != w {
				t.Fatalf("truncation mismatch\n got: %s\nwant: %s", g, w)
			}
		})
	}
}

// TestCaptureForkPrefixParentSessionUntouched 验证父 Session 零改动：捕获前后
// 父会话的序列化字节完全一致（红线：fork 捕获零发送、父零改动）。
func TestCaptureForkPrefixUsesProjectedView(t *testing.T) {
	// 父已压缩（有效投影）时，fork 前缀必须是投影视图而非原始全量——
	// 共享大上下文否则每个 fork 子代理都继承未压缩历史、反复触发
	// overflow 压缩（8/12 凌晨 24 次反复压缩根因之一）。
	msgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "a1"},
		{Role: provider.RoleUser, Content: "u2"},
		{Role: provider.RoleAssistant, Content: "a2"},
		{Role: provider.RoleUser, Content: "u3"},
	}
	a := forkPrefixTestAgent(t, msgs)
	canonical, version := a.Session().snapshotMessagesVersion() // 含 NewSession 自动加的 system 首条
	n := len(canonical)                                         // 投影全量覆盖 → modelVisibleFromProjection 返回纯投影
	projMsgs := []provider.Message{
		{Role: provider.RoleSystem, Content: "sys"},
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "summary of a1 and a2"},
		{Role: provider.RoleUser, Content: "u3"},
	}
	a.sess.compactionMu.Lock()
	a.sess.compactionState.PromptCacheKey = promptCacheKey("", "", "")
	a.sess.compactionState.Projection = ContextProjection{
		Messages:          projMsgs,
		TranscriptVersion: version,
		CoveredCount:      n,
		CoveredPrefixHash: coveredPrefixHash(canonical, n),
	}
	a.sess.compactionMu.Unlock()

	prefix := captureForkPrefix(a, context.Background())
	if len(prefix) != len(projMsgs) {
		t.Fatalf("fork prefix len %d != projected len %d (raw canonical would be %d)",
			len(prefix), len(projMsgs), len(msgs))
	}
	// 前缀必须带摘要（投影特征），不得含被折叠的原始 a1/a2。
	var sb strings.Builder
	for _, m := range prefix {
		sb.WriteString(m.Content)
	}
	joined := sb.String()
	if !strings.Contains(joined, "summary of a1 and a2") {
		t.Fatalf("fork prefix lacks projection summary: %q", joined)
	}
	if !strings.Contains(joined, "a1") || !strings.Contains(joined, "a2") {
		t.Fatalf("fork prefix dropped content: %q", joined)
	}
}

func TestCaptureForkInheritance(t *testing.T) {
	parent := forkPrefixTestAgent(t, []provider.Message{{Role: provider.RoleUser, Content: "u1"}})
	parent.modelRef = "deepseek/deepseek-v4-flash"
	parent.sess.output.lastUsage.Store(&provider.Usage{PromptTokens: 858_000})
	parent.setPromptTokenCalibration(858_000, requestCalibrationShapeOf(provider.Request{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: strings.Repeat("x", 300_000)}},
	}))

	// 同 model：继承 usage + calibration（值拷贝，改父不影响子）。
	usage, cal, ok := captureForkInheritance(parent, "deepseek/deepseek-v4-flash")
	if !ok || usage == nil || usage.PromptTokens != 858_000 || cal == nil {
		t.Fatalf("same-model inheritance = ok:%v usage:%v cal:%v, want values", ok, usage, cal)
	}
	parent.sess.output.lastUsage.Store(&provider.Usage{PromptTokens: 1})
	parent.setPromptTokenCalibration(1, parent.requestCalibrationShape(provider.Request{
		Messages: []provider.Message{{Role: provider.RoleUser, Content: "x"}},
	}))
	if usage.PromptTokens != 858_000 {
		t.Fatalf("inherited usage mutated with parent: %d", usage.PromptTokens)
	}

	// 跨 model：不继承（tokenizer 属性不可移植）。
	if _, _, ok := captureForkInheritance(parent, "qwen/qwen3"); ok {
		t.Fatal("cross-model inheritance must be rejected")
	}

	// 父无值：ok=false。
	parent.modelRef = "deepseek/deepseek-v4-flash"
	parent.sess.output.lastUsage.Store(nil)
	parent.sess.output.promptCalibration.Store(nil)
	if _, _, ok := captureForkInheritance(parent, "deepseek/deepseek-v4-flash"); ok {
		t.Fatal("inheritance with empty parent values must report ok=false")
	}
}

func TestCaptureForkPrefixParentSessionUntouched(t *testing.T) {
	parent := forkPrefixTestAgent(t, []provider.Message{
		{Role: provider.RoleUser, Content: "u", Images: []string{"data:image/png;base64,AAAA"}},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "c1", Name: "ls", Arguments: `{}`}},
			MemoryCitations: []provider.MemoryCitation{{ID: "m1", Source: "mem"}}},
		forkToolMsg("c1", "ls", "ok"),
		{Role: provider.RoleAssistant, LocalOnly: true, Content: "partial"}, // 未完成轮次
	})

	before := marshalMessages(t, parent.Session().Snapshot())
	_ = captureForkPrefix(parent, context.Background())
	after := marshalMessages(t, parent.Session().Snapshot())
	if before != after {
		t.Fatalf("parent session mutated by captureForkPrefix\n before: %s\n after: %s", before, after)
	}
}

// TestCloneForkMessagesDeepCopies 验证深拷贝：修改克隆结果的内嵌 slice 与指针
// 目标不会回流到源消息（Snapshot 深拷贝的硬要求）。
func TestCloneForkMessagesDeepCopies(t *testing.T) {
	ec := 7
	src := []provider.Message{
		{Role: provider.RoleUser, Content: "u", Images: []string{"img1", "img2"},
			ResponsesItems: []json.RawMessage{json.RawMessage(`{"k":1}`)}},
		{Role: provider.RoleAssistant, Content: "a",
			ToolCalls:        []provider.ToolCall{{ID: "c1", Name: "ls", Arguments: `{}`}},
			DecisionReceipts: []*provider.DecisionReceipt{{ID: "d1", Kind: "approve", Outcome: "ok"}},
			ToolExecution:    &provider.ToolExecution{Kind: "bash", ExitCode: &ec},
			InterruptedTurn:  &provider.InterruptedTurnRecovery{Pending: true, InterruptedTools: []string{"ls"}},
			MemoryCitations:  []provider.MemoryCitation{{ID: "m1", Source: "mem"}},
		},
	}
	before := marshalMessages(t, src)

	cloned := cloneForkMessages(src)
	if len(cloned) != len(src) {
		t.Fatalf("clone length = %d, want %d", len(cloned), len(src))
	}
	// 修改克隆的每一类内嵌可变字段。
	cloned[0].Images[0] = "mutated"
	cloned[0].ResponsesItems[0] = json.RawMessage(`{"k":2}`)
	cloned[1].ToolCalls[0].Name = "mutated"
	cloned[1].ToolCalls[0].Arguments = `{"x":1}`
	cloned[1].DecisionReceipts[0].Outcome = "mutated"
	cloned[1].ToolExecution.Kind = "mutated"
	*cloned[1].ToolExecution.ExitCode = 999
	cloned[1].InterruptedTurn.InterruptedTools[0] = "mutated"
	cloned[1].MemoryCitations[0].ID = "mutated"

	if after := marshalMessages(t, src); after != before {
		t.Fatalf("clone write-back mutated source\n before: %s\n after: %s", before, after)
	}
}

// TestForkSourceContext 验证 WithForkSource/ForkSourceFromContext 往返（T0-A
// ctx 通道，仿 evidence.WithSessionMessages 的注入模式）。
func TestForkSourceContext(t *testing.T) {
	parent := forkPrefixTestAgent(t, nil)
	ctx := WithForkSource(context.Background(), parent)
	got, ok := ForkSourceFromContext(ctx)
	if !ok || got != parent {
		t.Fatalf("ForkSourceFromContext = (%v, %v), want parent=%v", got, ok, parent)
	}
	if _, ok := ForkSourceFromContext(context.Background()); ok {
		t.Fatalf("ForkSourceFromContext on plain ctx unexpectedly ok")
	}
}

// TestCaptureSkillForkPrefixAppendsBodyToSystemTail 验证技能 fork 前缀：
// 父 system 字节保留在头部（命中父缓存），技能 body 追加在 system 尾部，
// 且返回切片与父会话零共享。
func TestCaptureSkillForkPrefixAppendsBodyToSystemTail(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "a1"},
	}
	parent := forkPrefixTestAgent(t, msgs)
	const body = "你是 team-executor。\n执行已批准计划。"
	prefix := CaptureSkillForkPrefix(parent, body)
	if len(prefix) == 0 {
		t.Fatal("CaptureSkillForkPrefix returned nil with a live parent")
	}
	if !strings.HasPrefix(prefix[0].Content, "sys") {
		t.Fatalf("system lost parent prefix: %q", prefix[0].Content[:min(20, len(prefix[0].Content))])
	}
	if !strings.HasSuffix(prefix[0].Content, body) {
		t.Fatalf("system tail missing skill body: %q", prefix[0].Content)
	}
	// 深拷贝：修改返回前缀不得回流父会话。
	before := marshalMessages(t, parent.Session().Snapshot())
	prefix[1].Content = "mutated"
	prefix[0].ToolCalls = []provider.ToolCall{{ID: "x", Name: "y", Arguments: `{}`}}
	if after := marshalMessages(t, parent.Session().Snapshot()); after != before {
		t.Fatalf("fork prefix write-back mutated parent\n before: %s\n after: %s", before, after)
	}
}

// TestPrepareSkillForkSessionPrefillsSession 验证 session 预填数量与内容。
func TestPrepareSkillForkSessionPrefillsSession(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "u1"},
		{Role: provider.RoleAssistant, Content: "a1"},
	}
	parent := forkPrefixTestAgent(t, msgs)
	sess := PrepareSkillForkSession(parent, "body")
	if sess == nil {
		t.Fatal("PrepareSkillForkSession returned nil with a live parent")
	}
	if got := sess.Len(); got != len(msgs)+1 {
		t.Fatalf("session messages = %d, want %d (system + history)", got, len(msgs)+1)
	}
	// 无会话父 → 回退冷启动（nil）。
	if got := PrepareSkillForkSession(nil, "body"); got != nil {
		t.Fatalf("PrepareSkillForkSession(nil) = non-nil, want nil")
	}
}
