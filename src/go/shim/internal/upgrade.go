package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Upgrade handles `fab upgrade [version]` — updates fab/.kit/ and config.yaml.
func Upgrade(targetVersion string) error {
	// Must be in a fab repo
	cfg, err := ResolveConfig()
	if err != nil {
		// If the error is about missing fab_version, that's OK for upgrade
		// Try to find config.yaml without requiring fab_version
		cwd, wdErr := os.Getwd()
		if wdErr != nil {
			return err
		}
		configPath := filepath.Join(cwd, "fab", "project", "config.yaml")
		if _, statErr := os.Stat(configPath); statErr != nil {
			return fmt.Errorf("not in a fab-managed repo. Run 'fab init' to set one up")
		}
		// config exists but fab_version missing — proceed with upgrade
		cfg = &ConfigResult{
			ConfigPath: configPath,
			RepoRoot:   cwd,
			FabVersion: "",
		}
	}
	if cfg == nil {
		return fmt.Errorf("not in a fab-managed repo. Run 'fab init' to set one up")
	}

	currentVersion := cfg.FabVersion

	// Resolve target version
	if targetVersion == "" {
		fmt.Println("Resolving latest version...")
		latest, err := LatestVersion()
		if err != nil {
			return fmt.Errorf("cannot resolve latest version: %w", err)
		}
		targetVersion = latest
	}
	targetVersion = strings.TrimPrefix(targetVersion, "v")

	// Check if already up to date
	if currentVersion == targetVersion {
		fmt.Printf("Already on the latest version (%s). No update needed.\n", currentVersion)
		return nil
	}

	if currentVersion != "" {
		fmt.Printf("Current version: %s\n", currentVersion)
	}
	fmt.Printf("Target version: %s\n", targetVersion)

	// Ensure target is cached
	_, err = EnsureCached(targetVersion)
	if err != nil {
		return err
	}

	// Atomic swap of fab/.kit/
	kitSrc := CachedKitDir(targetVersion)
	kitDst := filepath.Join(cfg.RepoRoot, "fab", ".kit")

	// Verify source kit has a VERSION file
	if _, err := os.Stat(filepath.Join(kitSrc, "VERSION")); err != nil {
		return fmt.Errorf("cached kit for v%s is missing VERSION file", targetVersion)
	}

	fmt.Println("Updating fab/.kit/...")

	// Extract to temp, then swap
	tmpKit := kitDst + ".upgrade-tmp"
	os.RemoveAll(tmpKit)
	if err := copyDir(kitSrc, tmpKit); err != nil {
		os.RemoveAll(tmpKit)
		return fmt.Errorf("cannot prepare kit update: %w", err)
	}

	// Swap: remove old, rename new
	if err := os.RemoveAll(kitDst); err != nil {
		os.RemoveAll(tmpKit)
		return fmt.Errorf("cannot remove old kit: %w", err)
	}
	if err := os.Rename(tmpKit, kitDst); err != nil {
		return fmt.Errorf("cannot install new kit: %w", err)
	}

	// Update fab_version in config.yaml
	if err := setFabVersion(cfg.ConfigPath, targetVersion); err != nil {
		return fmt.Errorf("cannot update config.yaml: %w", err)
	}

	// Run fab-sync.sh
	syncScript := filepath.Join(kitDst, "scripts", "fab-sync.sh")
	if _, err := os.Stat(syncScript); err == nil {
		fmt.Println("Running fab-sync.sh...")
		cmd := exec.Command("bash", syncScript)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = cfg.RepoRoot
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: fab-sync.sh failed: %s\n", err)
		}
	}

	// Display result
	if currentVersion != "" {
		fmt.Printf("\nUpdated: %s -> %s\n", currentVersion, targetVersion)
	} else {
		fmt.Printf("\nInstalled: %s\n", targetVersion)
	}

	// Migration reminder
	migrationVersionFile := filepath.Join(cfg.RepoRoot, "fab", ".kit-migration-version")
	if data, err := os.ReadFile(migrationVersionFile); err == nil {
		migVersion := strings.TrimSpace(string(data))
		if migVersion != targetVersion {
			fmt.Printf("\nRun '/fab-setup migrations' to update project files (%s -> %s)\n", migVersion, targetVersion)
		}
	}

	return nil
}
