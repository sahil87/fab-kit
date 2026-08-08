package kitpath

import (
	"fmt"
	"os"
	"path/filepath"
)

// KitPathEnv names the per-process kit content override.
const KitPathEnv = "FAB_KIT_PATH"

// overrideDir allows tests to override the kit directory resolution.
// When non-empty, KitDir() returns this value instead of resolving from the executable.
var overrideDir string

// SetOverride sets an override kit directory for testing purposes.
// Pass empty string to clear the override.
func SetOverride(dir string) {
	overrideDir = dir
}

// KitDir resolves the kit/ directory sibling to the running executable.
// It evaluates symlinks so that symlinked binaries resolve to the real kit location.
func KitDir() (string, error) {
	if overrideDir != "" {
		return overrideDir, nil
	}
	if dir := os.Getenv(KitPathEnv); dir != "" {
		absolute, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("%s is set but %q cannot be made absolute: %w", KitPathEnv, dir, err)
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return "", fmt.Errorf("%s is set but %s is not a directory: %w", KitPathEnv, absolute, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%s is set but %s is not a directory", KitPathEnv, absolute)
		}
		return absolute, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot determine executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("cannot resolve executable symlinks: %w", err)
	}
	kitDir := filepath.Join(filepath.Dir(exe), "kit")
	if info, err := os.Stat(kitDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("kit directory not found at %s", kitDir)
	}
	return kitDir, nil
}
