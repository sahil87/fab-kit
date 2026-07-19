package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runConfigUpgrade shells out to the pinned fab-go binary's `fab config upgrade`
// to reconcile fab/project/config.yaml after a sync. FAIL-OPEN by contract: any
// failure (a fab-go that predates the subcommand → unknown-command non-zero exit,
// or any other error) prints a reminder and returns nil — an upgrade must never
// break on the config step (decision 4). Both binaries ship in one brew package,
// so binary/kit skew only occurs on explicit-version upgrades.
func runConfigUpgrade(fabGoBin, repoRoot string) {
	cmd := exec.Command(fabGoBin, "config", "upgrade")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Note: could not auto-run `fab config upgrade` (%s). "+
			"Run it manually after upgrading to refresh config.yaml's reference fence.\n",
			strings.TrimSpace(string(out)))
		return
	}
	if trimmed := strings.TrimSpace(string(out)); trimmed != "" {
		fmt.Println(trimmed)
	}
}

// Upgrade handles `fab upgrade-repo [version] [--latest]` — re-syncs skills to
// the target version and then stamps fab/.fab-version + auto-runs the config
// upgrader. systemVersion is the embedded version of the fab-kit binary, threaded
// into Sync so the version guard compares against the real binary version (F22).
//
// Target resolution precedence (first match wins):
//   - an explicit targetVersion arg always wins (the GitHub API is not called);
//   - else useLatest queries GitHub via LatestVersion() (opt-in network call —
//     the pre-change default);
//   - else the running binary's own systemVersion (offline, authoritative) when
//     it is a real release tag (not empty and not "dev");
//   - else a network fallback via LatestVersion() for a "dev"/unstamped binary,
//     which has no real release tag to sync to.
//
// Ordering contract (F18): Sync runs FIRST (with the kit version passed
// explicitly) and fab/.fab-version is stamped only after Sync succeeds. A failed
// sync therefore exits non-zero, leaves fab/.fab-version on the old version, and
// a re-run retries instead of short-circuiting on "Already on the latest
// version".
func Upgrade(systemVersion, targetVersion string, useLatest bool) error {
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

	// Resolve target version.
	//   - explicit arg wins
	//   - --latest queries GitHub (opt-in network call)
	//   - default: the running binary's own version (offline, authoritative)
	if targetVersion == "" {
		switch {
		case useLatest:
			fmt.Println("Resolving latest version...")
			latest, err := LatestVersion()
			if err != nil {
				return fmt.Errorf("cannot resolve latest version: %w", err)
			}
			targetVersion = latest
		case systemVersion != "" && systemVersion != "dev":
			targetVersion = systemVersion
		default:
			// A dev/just-built shim (version == "dev") or an unstamped binary has no
			// real release tag to sync to — fall back to the network so it can still
			// resolve a published release.
			fmt.Println("Resolving latest version...")
			latest, err := LatestVersion()
			if err != nil {
				return fmt.Errorf("cannot resolve latest version: %w", err)
			}
			targetVersion = latest
		}
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

	// Ensure target is cached — the returned path is the pinned fab-go binary the
	// post-sync `fab config upgrade` auto-run shells out to.
	fabGoBin, err := EnsureCached(targetVersion)
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
	// propagate the error (non-zero exit) without stamping the version or
	// printing a success line — the pin stays on the old version, so a
	// re-run of `fab upgrade-repo` retries the upgrade.
	fmt.Println("Running sync...")
	if err := runSync(systemVersion, targetVersion, false, false); err != nil {
		return fmt.Errorf("sync failed: %w — run 'fab sync' to repair, then re-run 'fab upgrade-repo'", err)
	}

	// Stamp fab/.fab-version only after a successful sync (F18). config.yaml is no
	// longer version-stamped (260708-j0qm) — the version lives in the plain-text
	// sibling, and fab config upgrade is config.yaml's only writer.
	if err := stampFabVersion(cfg.RepoRoot, targetVersion); err != nil {
		return fmt.Errorf("cannot write fab/.fab-version: %w", err)
	}
	warnIfFabVersionIgnored(cfg.RepoRoot)

	// Auto-run the config upgrader against the pinned fab-go: reconcile
	// config.yaml (regenerate the managed fence, park removals, carry renames).
	// FAIL-OPEN: if the pinned fab-go predates `fab config upgrade` (non-zero exit
	// / unknown command), print a reminder and continue — an upgrade must never
	// break on the config step (decision 4).
	runConfigUpgrade(fabGoBin, cfg.RepoRoot)

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
