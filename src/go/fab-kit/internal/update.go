package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const brewFormula = "fab-kit"

// Seams for testing: overridden in tests to drive Update's flow without a real
// Homebrew install. Production wiring keeps the exec.Command calls direct.
var (
	isBrewInstalledFn   = isBrewInstalled
	brewLatestVersionFn = brewLatestVersion
)

// Update self-updates the fab-kit binary via Homebrew.
//
// When skipBrewUpdate is true, the internal `brew update --quiet` tap-metadata
// refresh is skipped; the brew info version check, the up-to-date
// short-circuit, and brew upgrade all still run unchanged.
func Update(currentVersion string, skipBrewUpdate bool) error {
	// Guard: only works if installed via Homebrew
	if !isBrewInstalledFn() {
		fmt.Printf("fab-kit v%s was not installed via Homebrew.\n", currentVersion)
		fmt.Println("Update manually, or reinstall with: brew install sahil87/tap/fab-kit")
		return nil
	}

	fmt.Printf("Current version: v%s\n", currentVersion)

	// Refresh Homebrew index (tap metadata), unless explicitly skipped.
	if skipBrewUpdate {
		fmt.Println("Skipping brew update (--skip-brew-update); checking for updates...")
	} else {
		fmt.Println("Checking for updates...")
		cmd := exec.Command("brew", "update", "--quiet")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := runWithTimeout(cmd, 30*time.Second); err != nil {
			return fmt.Errorf("could not check for updates (brew update failed): %w", err)
		}
	}

	// Query latest version from Homebrew
	latest, err := brewLatestVersionFn()
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
// the executable's symlink and looking for /Cellar/ in the real path.
func isBrewInstalled() bool {
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
