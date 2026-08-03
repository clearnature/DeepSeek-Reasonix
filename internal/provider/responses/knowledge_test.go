package responses

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNgramSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
		desc string
	}{
		{"今天北京天气怎么样", "北京今天天气如何", 0.7, "近义句式"},
		{"2026年8月3日北京天气", "北京今天天气如何", 0.5, "部分重叠"},
		{"图灵奖得主是谁", "北京天气如何", 0.05, "无关"},
		{"same query", "same query", 1.0, "完全相同"},
		{"", "任意", 0.0, "空串"},
	}
	for _, c := range cases {
		got := NgramSimilarity(c.a, c.b)
		t.Logf("%s: %q vs %q = %.3f (want ~%.2f)", c.desc, c.a, c.b, got, c.want)
		if c.desc == "完全相同" && got != 1.0 {
			t.Errorf("identical strings must be 1.0, got %.3f", got)
		}
		if c.desc == "空串" && got != 0.0 {
			t.Errorf("empty must be 0.0, got %.3f", got)
		}
		if c.desc == "无关" && got >= DefaultSemanticThreshold {
			t.Errorf("unrelated queries must not pass threshold %v, got %.3f", DefaultSemanticThreshold, got)
		}
	}
}

func TestLoadKnowledgeSemanticHitAndMiss(t *testing.T) {
	dir := mustKnowledgeDir(t)
	// 清理测试可能残留
	q1 := "2026年8月3日北京天气怎么样？"
	SaveKnowledge(&KnowledgeEntry{Query: q1, AnswerSummary: "晴朗 25-32℃"})
	defer os.Remove(filepath.Join(dir, KnowledgeHash(q1)+".json"))

	// 近义改写 → 应命中
	entry, sim, hit := LoadKnowledgeSemantic("北京今天天气如何？", DefaultSemanticThreshold)
	if !hit {
		t.Fatalf("semantic variant should hit, got sim=%.3f", sim)
	}
	if entry.AnswerSummary != "晴朗 25-32℃" {
		t.Fatalf("wrong entry: %#v", entry)
	}

	// 无关问题 → 不应命中
	if _, _, hit := LoadKnowledgeSemantic("图灵奖得主是谁", DefaultSemanticThreshold); hit {
		t.Fatal("unrelated query must miss")
	}
}

func TestLoadKnowledgeSemanticSkipsExpired(t *testing.T) {
	dir := mustKnowledgeDir(t)
	q := "过期测试问题"
	e := &KnowledgeEntry{Query: q, AnswerSummary: "x", ExpiresAt: time.Now().Add(-time.Hour)}
	SaveKnowledge(e)
	defer os.Remove(filepath.Join(dir, KnowledgeHash(q)+".json"))

	if _, _, hit := LoadKnowledgeSemantic(q, DefaultSemanticThreshold); hit {
		t.Fatal("expired entry must miss")
	}
}
