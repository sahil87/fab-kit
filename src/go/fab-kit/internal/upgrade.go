package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Upgrade handles `fab upgrade-repo [version]` — re-syncs skills to the
// target version and then updates fab_version in config.yaml.
// systemVersion is the embedded version of the fab-kit binary, threaded into
// Sync so the version guard compares against the real binary version (F22).
//
// Ordering contract (F18): Sync runs FIRST (with the kit version passed
// explicitly) and fab_version is stamped only after Sync succeeds. A failed
// sync therefore exits non-zero, leaves config.yaml on the old version, and
// a re-run retries instead of short-circuiting on "Already on the latest
// version".
func Upgrade(systemVersion, targetVersion string) error {
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

	// Run sync FIRST, passing the kit version explicitly. On failure,
	// propagate the error (non-zero exit) without stamping fab_version or
	// printing a success line — config.yaml stays on the old version, so a
	// re-run of `fab upgrade-repo` retries the upgrade.
	fmt.Println("Running sync...")
	if err := runSync(systemVersion, targetVersion, false, false); err != nil {
		return fmt.Errorf("sync failed: %w — run 'fab sync' to repair, then re-run 'fab upgrade-repo'", err)
	}

	// Stamp fab_version in config.yaml only after a successful sync (F18)
	if err := setFabVersion(cfg.ConfigPath, targetVersion); err != nil {
		return fmt.Errorf("cannot update config.yaml: %w", err)
	}

	// Display result
	if currentVersion != "" {
		fmt.Printf("\nUpdated: %s -> %s\n", currentVersion, targetVersion)
	} else {
		fmt.Printf("\nInstalled: %s\n", targetVersion)
	}

	// Migration detection + reminder.
	//
	// Mechanically discover whether any migration genuinely applies between the
	// local version and the target, rather than comparing version strings. The
	// three terminal cases (overlap / applicable / no-op) and the missing-file
	// case mirror the /fab-setup migrations skill's discovery rules.
	migrationVersionFile := filepath.Join(cfg.RepoRoot, "fab", ".kit-migration-version")
	if data, err := os.ReadFile(migrationVersionFile); err == nil {
		migVersion := strings.TrimSpace(string(data))
		migrationsDir := filepath.Join(CachedKitDir(targetVersion), "migrations")
		result, derr := DiscoverMigrations(migrationsDir, migVersion, targetVersion)
		switch {
		case derr != nil:
			// Discovery failure (e.g. missing migrations dir) is non-fatal to the
			// upgrade itself — warn so the user knows discovery did not run, and
			// skip the stamp (we cannot confirm there is nothing to migrate).
			fmt.Fprintf(os.Stderr, "WARNING: migration discovery failed: %s\n", derr)
		case len(result.Overlaps) > 0:
			// Malformed migration set — warn with detail, do NOT stamp.
			fmt.Printf("\nWARNING: overlapping migration ranges detected:\n")
			for _, o := range result.Overlaps {
				fmt.Printf("  %s\n", o)
			}
			fmt.Println("Run '/fab-setup migrations' to resolve.")
		case len(result.Applicable) > 0:
			// Migrations genuinely apply — print a styled reminder, do NOT stamp
			// (the skill owns the write after it applies each file).
			reminder := fmt.Sprintf("Run '/fab-setup migrations' to update project files (%s -> %s)", migVersion, targetVersion)
			fmt.Printf("\n%s\n", boldYellow(reminder))
		default:
			// Nothing applies and no overlap — silently stamp to the target so the
			// local version stops drifting behind the engine. Reuse the same-package
			// helper rather than reimplementing the write inline.
			if err := stampMigrationVersion(cfg.RepoRoot, targetVersion); err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: could not update .kit-migration-version: %s\n", err)
			}
		}
	}

	return nil
}

// isTTY reports whether f is a character device (an interactive terminal),
// using only the standard library — no golang.org/x/term or go-isatty (per
// Constitution I: minimal single-binary dependencies).
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// boldYellow wraps s in bold-yellow ANSI codes when os.Stdout is a TTY, and
// returns s unchanged otherwise (so logs and pipes stay free of escape codes).
func boldYellow(s string) string {
	if !isTTY(os.Stdout) {
		return s
	}
	return "\033[1;33m" + s + "\033[0m"
}
