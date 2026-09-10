package store

import (
	"os"
	"path/filepath"
	"strings"
)

// UserCacheDir mirrors config.CacheDir for packages that config already
// imports: config pulls in the provider packages for effort-capability
// discovery, so a provider package importing config back would close a cycle.
// internal/config/cachedir_parity_test.go asserts the two agree.
func UserCacheDir() string {
	if dir := cleanEnvDir("REASONIX_CACHE_HOME"); dir != "" {
		return dir
	}
	if dir := cleanEnvDir("REASONIX_HOME"); dir != "" {
		return filepath.Join(dir, "cache")
	}
	dir, err := os.UserCacheDir()
	if err != nil || dir == "" {
		return ""
	}
	return filepath.Join(dir, "reasonix")
}

// cleanEnvDir resolves one directory-valued environment variable the way
// config does: expand vars, expand a leading ~, then require an absolute path.
func cleanEnvDir(name string) string {
	dir := strings.TrimSpace(os.Getenv(name))
	if dir == "" {
		return ""
	}
	dir = os.ExpandEnv(dir)
	if dir == "~" || strings.HasPrefix(dir, "~/") || strings.HasPrefix(dir, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		if dir == "~" {
			dir = home
		} else {
			dir = filepath.Join(home, dir[2:])
		}
	}
	if !filepath.IsAbs(dir) {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return ""
		}
		dir = abs
	}
	return dir
}
