package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mkEmptySessionWithPeer builds dir/name.jsonl (empty, mtime at mod) with a
// live non-empty peer sharing topicID, and returns the empty session path.
func mkEmptySessionWithPeer(t *testing.T, dir, name, topicID string, mod time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name+".jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write empty session: %v", err)
	}
	os.Chtimes(path, mod, mod)
	writeTestBranchMeta(t, path, topicID)

	peer := filepath.Join(dir, name+"-live.jsonl")
	if err := os.WriteFile(peer, []byte("real turn content"), 0o644); err != nil {
		t.Fatalf("write live peer: %v", err)
	}
	writeTestBranchMeta(t, peer, topicID)
	return path
}

func writeTestBranchMeta(t *testing.T, path, topicID string) {
	t.Helper()
	meta := BranchMeta{ID: filepath.Base(path), TopicID: topicID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := SaveBranchMeta(path, meta); err != nil {
		t.Fatalf("save meta for %s: %v", path, err)
	}
}

func TestReclaimableEmptySessionsCollectsOnlyIdleTopicCoveredEmpties(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	fresh := time.Now().Add(-time.Hour)

	// Empty + idle + topic covered: reclaimable.
	reclaimable := mkEmptySessionWithPeer(t, dir, "empty", "topic-a", old)
	// Empty + fresh (inside grace): not reclaimable yet.
	mkEmptySessionWithPeer(t, dir, "fresh", "topic-b", fresh)
	// Empty + idle but no live peer (only sibling is another empty).
	noPeer := filepath.Join(dir, "orphan.jsonl")
	if err := os.WriteFile(noPeer, nil, 0o644); err != nil {
		t.Fatalf("write orphan: %v", err)
	}
	os.Chtimes(noPeer, old, old)
	writeTestBranchMeta(t, noPeer, "topic-c")
	writeTestBranchMeta(t, filepath.Join(dir, "orphan-sibling.jsonl"), "topic-c")
	if err := os.WriteFile(filepath.Join(dir, "orphan-sibling.jsonl"), nil, 0o644); err != nil {
		t.Fatalf("write orphan sibling: %v", err)
	}

	got, err := ReclaimableEmptySessions(dir, time.Now(), EmptySessionGracePeriod)
	if err != nil {
		t.Fatalf("ReclaimableEmptySessions: %v", err)
	}
	if len(got) != 1 || got[0] != reclaimable {
		t.Fatalf("reclaimable = %v, want exactly [%s]", got, reclaimable)
	}
}

func TestTrashEmptySessionMovesToTrashAndRejectsNonEmpty(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	path := mkEmptySessionWithPeer(t, dir, "victim", "topic-d", old)

	if err := TrashEmptySession(path, dir); err != nil {
		t.Fatalf("TrashEmptySession: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("empty session still present after trash: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".trash")); err != nil {
		t.Fatalf("no trash directory created: %v", err)
	}

	// Non-empty session must be rejected (nothing moves).
	nonEmpty := filepath.Join(dir, "content.jsonl")
	if err := os.WriteFile(nonEmpty, []byte("turns"), 0o644); err != nil {
		t.Fatalf("write non-empty: %v", err)
	}
	os.Chtimes(nonEmpty, old, old)
	writeTestBranchMeta(t, nonEmpty, "topic-d")
	if err := TrashEmptySession(nonEmpty, dir); err == nil {
		t.Fatal("TrashEmptySession accepted a non-empty session")
	}
	if _, err := os.Stat(nonEmpty); err != nil {
		t.Fatalf("non-empty session was moved: %v", err)
	}
}
