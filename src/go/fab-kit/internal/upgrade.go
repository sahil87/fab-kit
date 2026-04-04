package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Upgrade handles `fab upgrade-repo [version]` — updates fab_version in config.yaml and re-syncs skills.
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

	// Verify cached kit has a VERSION file
	kitSrc := CachedKitDir(targetVersion)
	if _, err := os.Stat(filepath.Join(kitSrc, "VERSION")); err != nil {
		return fmt.Errorf("cached kit for v%s is missing VERSION file", targetVersion)
	}

	fmt.Printf("Upgrading to %s...\n", targetVersion)

	// Update fab_version in config.yaml
	if err := setFabVersion(cfg.ConfigPath, targetVersion); err != nil {
		return fmt.Errorf("cannot update config.yaml: %w", err)
	}

	// Run sync
	fmt.Println("Running sync...")
	if err := Sync(targetVersion, false, false); err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: sync failed: %s\n", err)
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
