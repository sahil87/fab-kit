package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

const cacheBaseDir = ".fab-kit/versions"

// CacheDir returns the path to ~/.fab-kit/versions/{version}/.
func CacheDir(version string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback; should not happen in practice
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, cacheBaseDir, version)
}

// CachedBinary returns the path to the cached fab-go binary for a version.
func CachedBinary(version string) string {
	return filepath.Join(CacheDir(version), "fab-go")
}

// CachedKitDir returns the path to the cached kit/ directory for a version.
func CachedKitDir(version string) string {
	return filepath.Join(CacheDir(version), "kit")
}

// IsCached checks if a version's fab-go binary exists in the cache.
func IsCached(version string) bool {
	info, err := os.Stat(CachedBinary(version))
	if err != nil {
		return false
	}
	// Check executable bit
	return info.Mode()&0111 != 0
}

// EnsureCached checks if the version is cached. If not, downloads it.
// Returns the path to the cached fab-go binary.
func EnsureCached(version string) (string, error) {
	binary := CachedBinary(version)
	if IsCached(version) {
		return binary, nil
	}

	fmt.Fprintf(os.Stderr, "Fetching fab-kit v%s...\n", version)
	if err := Download(version); err != nil {
		return "", fmt.Errorf("failed to fetch v%s: %w", version, err)
	}

	if !IsCached(version) {
		return "", fmt.Errorf("download completed but fab-go binary not found at %s", binary)
	}
	return binary, nil
}
