package responses

import (
	"context"
	"strings"
	"testing"
)

func TestParsePropositions(t *testing.T) {
	// 纯净 JSON
	reply := `[{"title":"化肥供给冲击","query":"霍尔木兹封锁化肥出口影响","aspects":["信号","影响"]},{"title":"小麦减产","query":"全球小麦产量预测","aspects":["后果"]}]`
	props, err := ParsePropositions(reply)
	if err != nil || len(props) != 2 {
		t.Fatalf("parse: %v props=%d", err, len(props))
	}
	if props[0].Title != "化肥供给冲击" || props[1].Query != "全球小麦产量预测" {
		t.Fatalf("props wrong: %+v", props)
	}
	// 散文包裹
	props2, err := ParsePropositions("以下是拆解结果：\n[{\"title\":\"A\",\"query\":\"q1\"}]\n希望对你有帮助")
	if err != nil || len(props2) != 1 {
		t.Fatalf("prose wrap parse failed: %v", err)
	}
	// 无效
	if _, err := ParsePropositions("不是 JSON"); err == nil {
		t.Fatal("garbage must error")
	}
	if _, err := ParsePropositions(`[{"title":"","query":""}]`); err == nil {
		t.Fatal("empty propositions must error")
	}
}

func TestDecomposePrompt(t *testing.T) {
	p := DecomposePrompt("复杂命题")
	for _, want := range []string{"拆解", "子命题", `"title"`, "独立"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
}

func TestExecuteInfoEventModel(t *testing.T) {
	ex2 := &ModelExecutor{
		Decompose: func(topic string, maxTokens int) (string, error) {
			return `[{"title":"A","query":"q1"},{"title":"B","query":"q2"}]`, nil
		},
		Fetch: fetchForTest,
	}
	m, err := ex2.Execute("测试命题")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(m.Propositions) != 2 {
		t.Fatalf("propositions=%d want 2", len(m.Propositions))
	}
	if len(m.Chains) != 2 || len(m.FleetTasks) != 2 {
		t.Fatalf("chains=%d tasks=%d want 2 each", len(m.Chains), len(m.FleetTasks))
	}
	if m.EventMain.UpdateCount != 2 {
		t.Fatalf("event updates=%d want 2", m.EventMain.UpdateCount)
	}
	// 报告渲染含五步
	rep := m.Report()
	for _, want := range []string{"命题拆解", "多因果线", "并行检索任务", "信息流", "信息帧拼图"} {
		if !strings.Contains(rep, want) {
			t.Fatalf("report missing %q", want)
		}
	}
}

// fetchForTest is a FetchFunc-compatible stub.
func fetchForTest(ctx context.Context, q string, tier RetrievalTier) (*KnowledgeEntry, error) {
	return &KnowledgeEntry{Query: q, KeyFacts: []string{"事实:" + q}}, nil
}
