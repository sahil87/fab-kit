package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const brewFormula = "fab-kit"

// Update self-updates the fab-kit binary via Homebrew.
func Update(currentVersion string) error {
	// Guard: only works if installed via Homebrew
	self, err := os.Executable()
	if err == nil && !strings.Contains(self, "/Cellar/fab-kit/") {
		fmt.Printf("fab-kit v%s was not installed via Homebrew.\n", currentVersion)
		fmt.Println("Update manually, or reinstall with: brew install sahil87/tap/fab-kit")
		return nil
	}

	fmt.Printf("Current version: v%s\n", currentVersion)

	// Refresh Homebrew index
	fmt.Println("Checking for updates...")
	cmd := exec.Command("brew", "update", "--quiet")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := runWithTimeout(cmd, 30*time.Second); err != nil {
		return fmt.Errorf("could not check for updates (brew update failed): %w", err)
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

	cmd = exec.Command("brew", "upgrade", brewFormula)
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
