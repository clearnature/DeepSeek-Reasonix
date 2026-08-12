package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Empty-session garbage collection. Desktop creates a fresh session file when
// a topic is opened or a mode/profile is rebuilt before the first turn writes
// content; files that never received a single turn (0-byte .jsonl) accumulate
// as duplicate-looking history entries. Recovery branches were covered by
// recovery_gc.go; empty sessions were not. A 0-byte session is reclaimable
// when it sat idle past the grace period and its topic still has a live
// (non-empty) session to represent it.

// EmptySessionGracePeriod is how long a reclaimable empty session must sit
// idle before GC may collect it. A fresh file may simply be a just-created
// topic that has not received its first turn yet.
const EmptySessionGracePeriod = 24 * time.Hour

// ReclaimableEmptySessions returns session paths in dir that are 0-byte,
// idle past grace, topic-covered by a live peer, and not lease-held.
func ReclaimableEmptySessions(dir string, now time.Time, grace time.Duration) ([]string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".events.jsonl") {
			continue
		}
		path := filepath.Join(dir, name)
		if !IsVisibleSession(path) || SessionLeaseHeld(path) {
			continue
		}
		info, err := e.Info()
		if err != nil || info.IsDir() {
			continue
		}
		if info.Size() != 0 {
			continue
		}
		if now.Sub(info.ModTime()) < grace {
			continue
		}
		if !topicHasLivePeer(dir, path) {
			continue
		}
		out = append(out, path)
	}
	return out, nil
}

// topicHasLivePeer reports whether the session's topic is still represented
// by a non-empty sibling session. Without a topic (or a live peer) the file
// may be the only trace of that conversation, so it is preserved.
func topicHasLivePeer(dir, path string) bool {
	meta, ok, err := LoadBranchMeta(path)
	if err != nil || !ok || strings.TrimSpace(meta.TopicID) == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	base := filepath.Base(path)
	for _, e := range entries {
		name := e.Name()
		if name == base || !strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".events.jsonl") {
			continue
		}
		peer := filepath.Join(dir, name)
		if !IsVisibleSession(peer) {
			continue
		}
		if info, err := e.Info(); err == nil && info.Size() == 0 {
			continue
		}
		peerMeta, peerOK, _ := LoadBranchMeta(peer)
		if peerOK && peerMeta.TopicID == meta.TopicID {
			return true
		}
	}
	return false
}

// TrashEmptySession moves a 0-byte, idle, topic-covered session into the same
// recoverable .trash layout used by Desktop. Emptyness and topic coverage are
// re-proven under the removal guard so a concurrent write cannot be trashed.
func TrashEmptySession(path, parentDir string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	parentDir = filepath.Clean(strings.TrimSpace(parentDir))
	if path == "." || parentDir == "." || filepath.Dir(path) != parentDir {
		return fmt.Errorf("empty session must be a direct child of its session directory")
	}
	key := filepath.Base(path)
	if !strings.HasSuffix(key, ".jsonl") || strings.HasSuffix(key, ".events.jsonl") {
		return fmt.Errorf("invalid session path")
	}
	guard, err := TryAcquireSessionRemovalGuard(path)
	if err != nil {
		return err
	}
	defer guard.Release()
	info, err := os.Stat(path)
	if err != nil || info.Size() != 0 {
		return errors.New("session is not empty")
	}
	if time.Since(info.ModTime()) < EmptySessionGracePeriod {
		return errors.New("session is still inside its safety grace period")
	}
	if !topicHasLivePeer(parentDir, path) {
		return errors.New("session topic has no live peer")
	}
	stageDir, err := reserveRecoveryTrashStage(parentDir)
	if err != nil {
		return err
	}
	if err := prepareRecoveryTrashStage(path, key, stageDir); err != nil {
		return err
	}
	return finishRecoveryTrashStage(parentDir, path, key, stageDir, guard)
}
