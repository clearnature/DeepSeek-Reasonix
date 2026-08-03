package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"reasonix/internal/provider/responses"
)

func TestRetrieveInfoToolLocalHit(t *testing.T) {
	cleanCacheForTool(t)
	q := "2026年8月3日北京天气"
	responses.SaveKnowledge(&responses.KnowledgeEntry{Query: q, AnswerSummary: "晴朗 25-32℃"})
	defer cleanCacheForTool(t)

	out, err := (retrieveInfo{}).Execute(context.Background(), json.RawMessage(`{"query":"2026年8月3日北京天气"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "本地知识缓存命中") || !strings.Contains(out, "晴朗") {
		t.Fatalf("hit output wrong: %q", out)
	}
}

func TestRetrieveInfoToolMissBlocked(t *testing.T) {
	cleanCacheForTool(t)
	out, err := (retrieveInfo{}).Execute(context.Background(), json.RawMessage(`{"query":"完全没查过的问题"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	// 未命中 + 零 policy → 提示需联网授权，绝不静默联网
	if !strings.Contains(out, "未命中") || !strings.Contains(out, "联网") {
		t.Fatalf("miss output wrong: %q", out)
	}
}

func TestRetrieveInfoToolEmptyQuery(t *testing.T) {
	if _, err := (retrieveInfo{}).Execute(context.Background(), json.RawMessage(`{"query":"  "}`)); err == nil {
		t.Fatal("empty query must error")
	}
}

func cleanCacheForTool(t *testing.T) {
	t.Helper()
	dir := filepath.Join(mustCacheDir(t), "websearch")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, de := range entries {
		_ = os.Remove(filepath.Join(dir, de.Name()))
	}
}

func mustCacheDir(t *testing.T) string {
	t.Helper()
	dir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("cache dir: %v", err)
	}
	return filepath.Join(dir, "reasonix")
}
