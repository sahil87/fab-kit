package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cache directory names under ~/.fab-kit/
const (
	// KitPathEnv names the per-process kit content override.
	KitPathEnv     = "FAB_KIT_PATH"
	localCacheDir  = ".fab-kit/local-versions" // populated by `just build`, always takes priority
	remoteCacheDir = ".fab-kit/versions"       // populated by shim auto-fetch from GitHub releases
)

func fabKitHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return home
}

// LocalCacheDir returns ~/.fab-kit/local-versions/{version}/.
func LocalCacheDir(version string) string {
	return filepath.Join(fabKitHome(), localCacheDir, version)
}

// RemoteCacheDir returns ~/.fab-kit/versions/{version}/.
func RemoteCacheDir(version string) string {
	return filepath.Join(fabKitHome(), remoteCacheDir, version)
}

// CacheDir returns the path to ~/.fab-kit/versions/{version}/ (remote cache).
// Used by Download to know where to extract release archives.
func CacheDir(version string) string {
	return RemoteCacheDir(version)
}

// CachedKitDir returns the kit/ directory for a version, preferring local over remote.
func CachedKitDir(version string) string {
	localKit := filepath.Join(LocalCacheDir(version), "kit")
	if dirExists(localKit) {
		return localKit
	}
	return filepath.Join(RemoteCacheDir(version), "kit")
}

// KitPathOverride resolves and validates the per-process kit content override.
// When absolutization succeeds, the returned path remains available on later
// validation errors so informational callers such as doctor can report it.
func KitPathOverride() (dir string, set bool, err error) {
	raw := os.Getenv(KitPathEnv)
	if raw == "" {
		return "", false, nil
	}

	absolute, err := filepath.Abs(raw)
	if err != nil {
		return raw, true, fmt.Errorf("%s is set but %q cannot be made absolute: %w", KitPathEnv, raw, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return absolute, true, fmt.Errorf("%s is set but %s is not a directory: %w", KitPathEnv, absolute, err)
	}
	if !info.IsDir() {
		return absolute, true, fmt.Errorf("%s is set but %s is not a directory", KitPathEnv, absolute)
	}
	return absolute, true, nil
}

// ResolveKitDir returns the reader-facing kit directory. The environment
// override wins when active; otherwise cache resolution remains unchanged.
func ResolveKitDir(version string) (string, error) {
	if dir, set, err := KitPathOverride(); set || err != nil {
		return dir, err
	}
	return CachedKitDir(version), nil
}

// RefuseKitPathOverride protects lifecycle commands that stamp release state.
func RefuseKitPathOverride(command string) error {
	if os.Getenv(KitPathEnv) == "" {
		return nil
	}
	return fmt.Errorf("%s is set — unset it before running 'fab %s' (the override must not influence version stamping)", KitPathEnv, command)
}

// ResolveBinary returns the path to the fab-go binary for a version,
// checking local-versions first, then versions.
func ResolveBinary(version string) (string, bool) {
	// Local builds take priority
	localBin := filepath.Join(LocalCacheDir(version), "fab-go")
	if isExecutable(localBin) {
		return localBin, true
	}

	// Fall back to downloaded releases
	remoteBin := filepath.Join(RemoteCacheDir(version), "fab-go")
	if isExecutable(remoteBin) {
		return remoteBin, true
	}

	return remoteBin, false
}

// EnsureCached resolves the binary for a version. Checks local-versions first,
// then versions. If neither exists, downloads from GitHub releases.
// Returns the path to the fab-go binary.
func EnsureCached(version string) (string, error) {
	if path, found := ResolveBinary(version); found {
		return path, nil
	}

	fmt.Fprintf(os.Stderr, "Fetching fab-kit v%s...\n", version)
	if err := Download(version); err != nil {
		return "", fmt.Errorf("failed to fetch v%s: %w", version, err)
	}

	if path, found := ResolveBinary(version); found {
		return path, nil
	}
	return "", fmt.Errorf("download completed but fab-go binary not found in cache")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
