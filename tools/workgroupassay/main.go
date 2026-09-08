// Command workgroupassay measures how much of ordinary use has execution worth
// folding. It reads durable wire logs, keeps the turns the frozen protocol
// admits, and refuses to answer until the sample is whole.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"reasonix/internal/config"
	"reasonix/internal/eventwire"
)

// turn is one authored turn: the frames it owns and where in time it sits.
type turn struct {
	session  string
	at       time.Time
	index    int
	authored int
	frames   []eventwire.Event
}

func main() {
	root := flag.String("root", "", "session root to scan (default: the configured one)")
	flag.Parse()
	dirs := scanRoots(*root)
	turns, skipped := collect(dirs)
	sort.Slice(turns, func(i, j int) bool {
		if !turns[i].at.Equal(turns[j].at) {
			return turns[i].at.Before(turns[j].at)
		}
		return turns[i].index < turns[j].index
	})
	report(turns, skipped)
}

func scanRoots(root string) []string {
	if strings.TrimSpace(root) != "" {
		return []string{root}
	}
	seen := map[string]bool{}
	var out []string
	for _, d := range append([]string{config.SessionDir()}, projectSessionDirs()...) {
		if d != "" && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// projectSessionDirs finds the per-project session directories a Studio window
// writes into, which is where ordinary use lands.
func projectSessionDirs() []string {
	home := config.ReasonixHomeDir()
	if home == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(home, "projects"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(home, "projects", e.Name(), "sessions"))
		}
	}
	return out
}

// skipReasons counts what the protocol turned away, so a small sample can say
// why rather than look like nothing happened.
type skipReasons struct {
	beforeFreeze  int
	truncatedLogs int
	unnamedTurns  int
}

func collect(dirs []string) ([]turn, skipReasons) {
	var out []turn
	var skipped skipReasons
	frozen := frozenTime()
	for _, dir := range dirs {
		logs, _ := filepath.Glob(filepath.Join(dir, "*.wire.jsonl"))
		for _, path := range logs {
			at, ok := sessionStart(path)
			if !ok || at.Before(frozen) {
				skipped.beforeFreeze++
				continue
			}
			if truncated(path) {
				skipped.truncatedLogs++
				continue
			}
			turns, unnamed := turnsIn(path, at)
			skipped.unnamedTurns += unnamed
			out = append(out, turns...)
		}
	}
	return out, skipped
}

// sessionStart reads when a session began from its file name, which is the only
// clock the durable log carries for frames that never ran a tool.
func sessionStart(path string) (time.Time, bool) {
	name := filepath.Base(path)
	if i := strings.Index(name, "-"); i > 0 && len(name) > i+7 {
		if t, err := time.ParseInLocation("20060102-150405", name[:i+7], time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// truncated reads the log's own record of what it does not contain. A prefix
// cannot answer a question about a whole turn.
func truncated(logPath string) bool {
	meta := strings.TrimSuffix(logPath, ".wire.jsonl") + ".wire.meta.json"
	body, err := os.ReadFile(meta)
	if err != nil {
		return false
	}
	var m struct {
		Truncated bool `json:"truncated"`
	}
	return json.Unmarshal(body, &m) == nil && m.Truncated
}

// turnsIn splits a log into authored turns. A turn_started with no authoredTurn
// is a synthetic continuation of one the host already named, never a turn of
// its own; it is counted as skipped and its frames stay with nothing.
func turnsIn(path string, at time.Time) ([]turn, int) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, 0
	}
	var out []turn
	var cur *turn
	unnamed := 0
	idx := 0
	for line := range strings.SplitSeq(strings.TrimRight(string(body), "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var e eventwire.Event
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		if e.Kind == "turn_started" {
			if cur != nil {
				out = append(out, *cur)
			}
			cur = nil
			if e.AuthoredTurn == nil {
				unnamed++
				continue
			}
			idx++
			cur = &turn{session: filepath.Base(path), at: at, index: idx, authored: *e.AuthoredTurn}
		}
		if cur != nil {
			cur.frames = append(cur.frames, e)
		}
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out, unnamed
}

func report(turns []turn, skipped skipReasons) {
	fmt.Printf("P2-2d natural-value assay\n")
	fmt.Printf("  protocol frozen at %s; sample is the first %d eligible authored turns after it\n",
		frozenAt, sampleSize)
	fmt.Printf("  eligible so far: %d\n", len(turns))
	fmt.Printf("  turned away: %d session(s) before the freeze, %d truncated, %d unnamed turns\n",
		skipped.beforeFreeze, skipped.truncatedLogs, skipped.unnamedTurns)
	if len(turns) < sampleSize {
		fmt.Printf("\nNo result yet. %d more eligible authored turn(s) are owed.\n", sampleSize-len(turns))
		fmt.Printf("Reading a partial sample is the sequential decision this protocol exists to prevent,\n")
		fmt.Printf("so nothing further is printed until the sample is whole.\n")
		return
	}
	measure(turns[:sampleSize])
}
