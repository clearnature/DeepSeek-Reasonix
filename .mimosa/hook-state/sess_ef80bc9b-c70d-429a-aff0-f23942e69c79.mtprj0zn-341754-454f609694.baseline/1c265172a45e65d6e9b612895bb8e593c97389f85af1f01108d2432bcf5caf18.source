package config

import (
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

// P4 foreground→background, T4: foreground_backgroundize_seconds config key.
// Default 120s; explicit 0 disables auto-backgroundize (the /background
// command still works); REASONIX_AUTO_BACKGROUND_MS overrides in ms.

func TestForegroundBackgroundizeDefault(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "")
	cfg := Default()
	if cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds != nil {
		t.Fatalf("default raw foreground_backgroundize_seconds = %v, want nil", *cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds)
	}
	if got := cfg.ForegroundBackgroundize(); got != 120*time.Second {
		t.Fatalf("ForegroundBackgroundize() = %v, want 120s", got)
	}
}

func TestForegroundBackgroundizeExplicitZeroDisables(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "")
	cfg := Default()
	cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds = intPtr(0)
	if got := cfg.ForegroundBackgroundize(); got != 0 {
		t.Fatalf("explicit 0 ForegroundBackgroundize() = %v, want disabled (0)", got)
	}
}

func TestForegroundBackgroundizeParsesExplicitZero(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "")
	cfg := Default()
	if _, err := toml.Decode("[tools.background_jobs]\nforeground_backgroundize_seconds = 0\n", cfg); err != nil {
		t.Fatalf("decode explicit zero: %v", err)
	}
	if cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds == nil {
		t.Fatal("explicit zero decoded as nil")
	}
	if got := cfg.ForegroundBackgroundize(); got != 0 {
		t.Fatalf("decoded explicit 0 ForegroundBackgroundize() = %v, want disabled", got)
	}
}

func TestForegroundBackgroundizeNegativeFallsBackDefault(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "")
	cfg := Default()
	cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds = intPtr(-5)
	if got := cfg.ForegroundBackgroundize(); got != 120*time.Second {
		t.Fatalf("negative ForegroundBackgroundize() = %v, want the 120s default (BashTimeoutSeconds pattern)", got)
	}
}

func TestForegroundBackgroundizeCustomSeconds(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "")
	cfg := Default()
	cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds = intPtr(30)
	if got := cfg.ForegroundBackgroundize(); got != 30*time.Second {
		t.Fatalf("custom ForegroundBackgroundize() = %v, want 30s", got)
	}
}

func TestForegroundBackgroundizeEnvOverridesMilliseconds(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "2500")
	cfg := Default()
	cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds = intPtr(120)
	if got := cfg.ForegroundBackgroundize(); got != 2500*time.Millisecond {
		t.Fatalf("env override ForegroundBackgroundize() = %v, want 2500ms", got)
	}
}

func TestForegroundBackgroundizeEnvZeroOrNegativeDisables(t *testing.T) {
	for _, v := range []string{"0", "-1"} {
		t.Run("env="+v, func(t *testing.T) {
			t.Setenv(autoBackgroundizeEnv, v)
			cfg := Default()
			cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds = intPtr(120)
			if got := cfg.ForegroundBackgroundize(); got != 0 {
				t.Fatalf("env %q ForegroundBackgroundize() = %v, want disabled (explicit 0 semantics)", v, got)
			}
		})
	}
}

func TestForegroundBackgroundizeEnvUnparsableFallsBackToConfig(t *testing.T) {
	t.Setenv(autoBackgroundizeEnv, "abc")
	cfg := Default()
	cfg.Tools.BackgroundJobs.ForegroundBackgroundizeSeconds = intPtr(120)
	if got := cfg.ForegroundBackgroundize(); got != 120*time.Second {
		t.Fatalf("unparsable env ForegroundBackgroundize() = %v, want the config value 120s", got)
	}
}
