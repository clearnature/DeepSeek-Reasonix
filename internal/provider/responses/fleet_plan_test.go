package responses

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildFleetRetrievalTasks(t *testing.T) {
	// 2 场景 × 2 语言 = 4 任务
	tasks := BuildFleetRetrievalTasks("AI 产业", DepthL2, []string{"zh", "en"}, []InfoDomain{DomainEconomic, DomainCode})
	if len(tasks) != 4 {
		t.Fatalf("tasks=%d want 4", len(tasks))
	}
	// 每个 prompt 含场景名 + 语言 + InfoFrame 输出要求
	for _, task := range tasks {
		if !strings.Contains(task.Prompt, "经济") && !strings.Contains(task.Prompt, "代码") {
			t.Fatalf("prompt missing domain: %q", task.Prompt)
		}
		if !strings.Contains(task.Prompt, `"domain"`) {
			t.Fatalf("prompt missing InfoFrame schema: %q", task.Prompt)
		}
		if !task.ReadOnly {
			t.Fatal("retrieval tasks must be read-only")
		}
		if len(task.Tools) == 0 {
			t.Fatal("tasks must whitelist web_search")
		}
	}
}

func TestBuildFleetRetrievalTasksDefaults(t *testing.T) {
	// 无参数默认：zh/en × general = 2 任务
	tasks := BuildFleetRetrievalTasks("主题", DepthL1, nil, nil)
	if len(tasks) != 2 {
		t.Fatalf("default tasks=%d want 2", len(tasks))
	}
}

func TestParseInfoFrame(t *testing.T) {
	raw := []byte(`{"domain":"economic","language":"en","topic":"AI","facts":["market $50B"],"sources":[{"title":"Reuters","url":"https://reuters.com/a"}],"confidence":0.8}`)
	f, err := ParseInfoFrame(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f.Domain != DomainEconomic || f.Language != "en" || len(f.Facts) != 1 || f.Confidence != 0.8 {
		t.Fatalf("frame wrong: %+v", f)
	}
	// 坏输入
	if _, err := ParseInfoFrame([]byte(`{"domain":"x"}`)); err == nil {
		t.Fatal("empty topic must error")
	}
	if _, err := ParseInfoFrame([]byte(`not json`)); err == nil {
		t.Fatal("bad json must error")
	}
}

func TestAssembleFrameView(t *testing.T) {
	replies := [][]byte{
		[]byte(`{"domain":"economic","language":"zh","topic":"AI","facts":["市场 500亿","增长 20%"],"confidence":0.8}`),
		[]byte(`{"domain":"economic","language":"en","topic":"AI","facts":["market $50B","增长 20%"],"confidence":0.7}`),
		[]byte(`garbage`), // 坏帧跳过
	}
	v := AssembleFrameView("AI 产业", replies)
	if len(v.Frames) != 2 {
		t.Fatalf("valid frames=%d want 2", len(v.Frames))
	}
	// 事实去重
	dup := 0
	for _, f := range v.MergedFacts {
		if strings.Contains(f, "增长 20%") {
			dup++
		}
	}
	if dup != 1 {
		t.Fatalf("facts must dedupe across frames: %v", v.MergedFacts)
	}
	if len(v.Languages) != 2 {
		t.Fatalf("languages=%v want zh/en", v.Languages)
	}
	_ = json.RawMessage{} // keep encoding/json import
}
