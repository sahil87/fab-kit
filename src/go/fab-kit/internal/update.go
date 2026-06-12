package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const brewFormula = "fab-kit"

// ErrNotBrewInstalled is returned by Update when fab-kit was not installed
// via Homebrew, so callers (the `fab update` command, versionGuard) can
// distinguish "cannot self-update" from a successful update.
var ErrNotBrewInstalled = errors.New("fab-kit was not installed via Homebrew")

// Update self-updates the fab-kit binary via Homebrew.
func Update(currentVersion string, skipBrewUpdate bool) error {
	// Guard: only works if installed via Homebrew
	if !isBrewInstalled() {
		fmt.Printf("fab-kit v%s was not installed via Homebrew.\n", currentVersion)
		fmt.Println("Update manually, or reinstall with: brew install sahil87/tap/fab-kit")
		return ErrNotBrewInstalled
	}

	fmt.Printf("Current version: v%s\n", currentVersion)

	// Refresh Homebrew index (unless skipped)
	if !skipBrewUpdate {
		fmt.Println("Checking for updates...")
		cmd := exec.Command("brew", "update", "--quiet")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := runWithTimeout(cmd, 30*time.Second); err != nil {
			return fmt.Errorf("could not check for updates (brew update failed): %w", err)
		}
	}

	// Query latest version from Homebrew
	latest, err := brewLatestVersion()
	if err != nil {
		return fmt.Errorf("could not determine latest version: %w", err)
	}

	if latest == currentVersion {
		fmt.Printf("Already up to date (v%s).\n", currentVersion)
		return nil
	}

	fmt.Printf("Updating v%s → v%s...\n", currentVersion, latest)

	cmd := exec.Command("brew", "upgrade", brewFormula)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := runWithTimeout(cmd, 120*time.Second); err != nil {
		return fmt.Errorf("brew upgrade failed: %w", err)
	}

	fmt.Printf("Updated to v%s.\n", latest)
	return nil
}

// brewLatestVersion queries Homebrew for the latest stable version of fab-kit.
func brewLatestVersion() (string, error) {
	out, err := exec.Command("brew", "info", "--json=v2", brewFormula).Output()
	if err != nil {
		return "", err
	}

	var info struct {
		Formulae []struct {
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return "", err
	}
	if len(info.Formulae) == 0 || info.Formulae[0].Versions.Stable == "" {
		return "", fmt.Errorf("no stable version found in brew info output")
	}
	return info.Formulae[0].Versions.Stable, nil
}

// isBrewInstalled checks whether fab-kit was installed via Homebrew by resolving
// the executable's symlink and looking for /Cellar/ in the real path. It is a
// package-level var so tests can override the brew-install guard (the test binary's
// path never contains /Cellar/); production behavior is unchanged.
var isBrewInstalled = func() bool {
	self, err := os.Executable()
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(self)
	if err != nil {
		return false
	}
	return strings.Contains(real, "/Cellar/")
}

// installedBinaryVersion queries the fab-kit binary on PATH for its version
// (the post-state check versionGuard relies on instead of trusting Update's
// return value — after `brew upgrade`, the PATH symlink points at the new
// Cellar binary even though the running process is still the old one).
// Output format is cobra's stable `fab-kit version vX.Y.Z`. Package-level var
// so tests can override (same seam pattern as isBrewInstalled).
var installedBinaryVersion = func() (string, error) {
	path, err := exec.LookPath("fab-kit")
	if err != nil {
		return "", fmt.Errorf("fab-kit not found on PATH: %w", err)
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("cannot query fab-kit version: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return "", fmt.Errorf("cannot parse fab-kit --version output %q", string(out))
	}
	return strings.TrimPrefix(fields[len(fields)-1], "v"), nil
}

// runWithTimeout runs a command with a timeout.
func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("timed out after %s", timeout)
	}
}
