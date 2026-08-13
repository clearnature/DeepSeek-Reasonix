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

	persistCalibration(model, &promptTokenCalibration{promptTokens: 580, compactChars: 1000, requestChars: 1000})

	got := loadPersistedCalibration(model)
	if got == nil {
		t.Fatalf("persisted calibration not loaded")
	}
	if r := float64(got.promptTokens) / float64(got.compactChars); r < 0.57 || r > 0.59 {
		t.Fatalf("restored ratio = %v, want ~0.58", r)
	}
	if got.requestChars != 1000 {
		t.Fatalf("restored request_chars = %d, want 1000", got.requestChars)
	}

	// A fresh agent starts from the persisted calibration, not the fallback.
	a := &Agent{agentConfig: agentConfig{modelRef: model}, sess: sessionRuntime{}}
	a.sess.output.promptCalibration.Store(loadPersistedCalibration(a.calibrationKey()))
	if tp := a.tokPerChar(); tp < 0.57 || tp > 0.59 {
		t.Fatalf("fresh agent tokPerChar = %v, want ~0.58 (persisted)", tp)
	}
	// The restored calibration must drive the token estimate too, not just the
	// telemetry ratio (8/13: missing request_chars bailed to the fallback):
	// 0.58 calibrates to 580 where the 0.25 fallback would estimate 250.
	shape := requestCalibrationShape{requestChars: 1000, compactChars: 1000}
	est, ok := a.calibratedPromptTokens(shape)
	if !ok {
		t.Fatalf("restored calibration not used by calibratedPromptTokens")
	}
	if est < 560 || est > 600 {
		t.Fatalf("calibrated estimate = %d, want ~580 (0.58 ratio)", est)
	}
}

// TestCalibrationRejectsTinySample pins the guard against junk calibration
// files: a 1-token/1-char write from a minimal request (observed 8/13) must
// not price the whole session at ratio 1.0.
func TestCalibrationRejectsTinySample(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REASONIX_HOME", home)
	model := "deepseek/deepseek-v4-flash"

	if err := os.MkdirAll(filepath.Dir(calibrationFilePath(model)), 0o755); err != nil {
		t.Fatalf("mkdir calibration dir: %v", err)
	}
	if err := os.WriteFile(calibrationFilePath(model),
		[]byte(`{"model":"deepseek/deepseek-v4-flash","prompt_tokens":1,"compact_chars":1,"tok_per_char":1,"updated_at":"2026-08-13T04:39:00Z"}`), 0o644); err != nil {
		t.Fatalf("write junk calibration: %v", err)
	}
	if cal := loadPersistedCalibration(model); cal != nil {
		t.Fatalf("tiny-sample calibration loaded: %+v", cal)
	}
}

// TestCalibrationLegacyFileFallback pins compatibility with files written by
// the initial bfb8029a8 shape (prompt_tokens + compact_chars only): the
// restored calibration still calibrates the estimate via the compact ratio.
func TestCalibrationLegacyFileFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("REASONIX_HOME", home)
	model := "deepseek/deepseek-v4-flash"

	if err := os.MkdirAll(filepath.Dir(calibrationFilePath(model)), 0o755); err != nil {
		t.Fatalf("mkdir calibration dir: %v", err)
	}
	if err := os.WriteFile(calibrationFilePath(model),
		[]byte(`{"model":"deepseek/deepseek-v4-flash","prompt_tokens":406,"compact_chars":1000,"tok_per_char":0.406,"updated_at":"2026-08-13T06:00:00Z"}`), 0o644); err != nil {
		t.Fatalf("write legacy calibration: %v", err)
	}

	cal := loadPersistedCalibration(model)
	if cal == nil {
		t.Fatalf("legacy calibration not loaded")
	}
	if cal.requestChars <= 0 || cal.requestChars != cal.compactChars {
		t.Fatalf("legacy fallback request_chars = %d, want compact_chars fallback %d", cal.requestChars, cal.compactChars)
	}

	a := &Agent{agentConfig: agentConfig{modelRef: model}, sess: sessionRuntime{}}
	a.sess.output.promptCalibration.Store(cal)
	shape := requestCalibrationShape{requestChars: 1000, compactChars: 1000}
	est, ok := a.calibratedPromptTokens(shape)
	if !ok {
		t.Fatalf("legacy calibration not used by calibratedPromptTokens")
	}
	if est < 390 || est > 430 {
		t.Fatalf("legacy calibrated estimate = %d, want ~406", est)
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
