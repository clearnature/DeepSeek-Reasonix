package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCalibrationPersistAndReload pins the restart path: a persisted
// per-model calibration restores tokPerChar on a fresh agent (8/13: restart
// lost the in-memory calibration, first compaction fell back to 0.000).
func TestCalibrationPersistAndReload(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REASONIX_HOME", home)
	model := "deepseek-responses/deepseek-v4-flash"

	persistCalibration(model, &promptTokenCalibration{promptTokens: 580, compactChars: 1000})

	got := loadPersistedCalibration(model)
	if got == nil {
		t.Fatalf("persisted calibration not loaded")
	}
	if r := float64(got.promptTokens) / float64(got.compactChars); r < 0.57 || r > 0.59 {
		t.Fatalf("restored ratio = %v, want ~0.58", r)
	}

	// A fresh agent starts from the persisted calibration, not the fallback.
	a := &Agent{agentConfig: agentConfig{modelRef: model}, sess: sessionRuntime{}}
	if cal := loadPersistedCalibration(a.calibrationKey()); cal == nil {
		t.Fatalf("fresh agent did not pick up persisted calibration")
	}
	a.sess.output.promptCalibration.Store(loadPersistedCalibration(a.calibrationKey()))
	if tp := a.tokPerChar(); tp < 0.57 || tp > 0.59 {
		t.Fatalf("fresh agent tokPerChar = %v, want ~0.58 (persisted)", tp)
	}
}

// TestCalibrationDebounce pins the persist gate: a near-identical ratio does
// not rewrite the file (mtime stable), so per-request calibration does not
// hammer the disk.
func TestCalibrationDebounce(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REASONIX_HOME", home)
	model := "m/deepseek-v4-flash"

	a := &Agent{agentConfig: agentConfig{modelRef: model}}
	path := calibrationFilePath(model)
	shape := requestCalibrationShape{requestChars: 1000, compactChars: 1000}

	a.setPromptTokenCalibration(600, shape)
	st1, err := os.Stat(path)
	if err != nil {
		t.Fatalf("first persist failed: %v", err)
	}
	a.setPromptTokenCalibration(601, shape) // +0.17% < 2%: debounced
	st2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after debounce failed: %v", err)
	}
	if !st2.ModTime().Equal(st1.ModTime()) {
		t.Fatalf("debounced write touched the file: %v -> %v", st1.ModTime(), st2.ModTime())
	}
	a.setPromptTokenCalibration(630, shape) // +5%: persisted
	st3, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after significant change failed: %v", err)
	}
	if st3.ModTime().Equal(st1.ModTime()) {
		t.Fatalf("significant change was not persisted")
	}
	_ = filepath.Join(home) // keep home referenced
}
