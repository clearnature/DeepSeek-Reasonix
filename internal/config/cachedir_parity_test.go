package config_test

import (
	"testing"

	"reasonix/internal/config"
	"reasonix/internal/store"
)

// The provider packages reach the cache root through store.UserCacheDir
// because config imports them; the two implementations must stay in step.
func TestStoreUserCacheDirMatchesConfig(t *testing.T) {
	if got, want := store.UserCacheDir(), config.CacheDir(); got != want {
		t.Fatalf("store.UserCacheDir() = %q, config.CacheDir() = %q", got, want)
	}
}
