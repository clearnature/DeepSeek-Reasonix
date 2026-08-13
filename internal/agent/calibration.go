package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"reasonix/internal/config"
)

// Calibration persistence: a per-model token/char ratio survives restarts so
// the first request after a relaunch uses the last session's calibration
// instead of the cold fallback. Best-effort; admission never blocks on IO.

func calibrationFilePath(modelRef string) string {
	home := config.ReasonixHomeDir()
	if strings.TrimSpace(home) == "" || strings.TrimSpace(modelRef) == "" {
		return ""
	}
	safe := strings.NewReplacer("/", "__", "\\", "__", ":", "_").Replace(modelRef)
	return filepath.Join(home, "calibration", safe+".json")
}

// loadPersistedCalibration returns a stored calibration for the model, or nil
// when absent or malformed.
func loadPersistedCalibration(modelRef string) *promptTokenCalibration {
	path := calibrationFilePath(modelRef)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var rec struct {
		PromptTokens int   `json:"prompt_tokens"`
		CompactChars int64 `json:"compact_chars"`
	}
	if err := json.Unmarshal(data, &rec); err != nil || rec.CompactChars <= 0 {
		return nil
	}
	return &promptTokenCalibration{
		promptTokens: rec.PromptTokens,
		compactChars: rec.CompactChars,
	}
}

// persistCalibration writes the calibration atomically (tmp + rename).
func persistCalibration(modelRef string, cal *promptTokenCalibration) {
	if cal == nil || cal.compactChars <= 0 {
		return
	}
	path := calibrationFilePath(modelRef)
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	rec := struct {
		Model        string  `json:"model"`
		PromptTokens int     `json:"prompt_tokens"`
		CompactChars int64   `json:"compact_chars"`
		TokPerChar   float64 `json:"tok_per_char"`
		UpdatedAt    string  `json:"updated_at"`
	}{
		Model: modelRef, PromptTokens: cal.promptTokens, CompactChars: cal.compactChars,
		TokPerChar: float64(cal.promptTokens) / float64(cal.compactChars),
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// absRatioDelta returns |a-b|, used for the persist debounce.
func absRatioDelta(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

// restorePersistedCalibration loads the per-model calibration so the first
// request after a relaunch uses the last session's real ratio (best-effort).
func (a *Agent) restorePersistedCalibration(modelRef string) {
	if cal := loadPersistedCalibration(strings.TrimSpace(modelRef)); cal != nil {
		a.sess.output.promptCalibration.Store(cal)
	}
}

// calibrationKey is the per-model persistence key.
func (a *Agent) calibrationKey() string {
	if a == nil {
		return ""
	}
	if ref := strings.TrimSpace(a.modelRef); ref != "" {
		return ref
	}
	return "default"
}
