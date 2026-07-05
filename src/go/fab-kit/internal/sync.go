package internal

// sync.go holds the Sync orchestrator, its version guard, and the small
// step helpers the orchestrator owns directly (repo-root resolution, direnv
// allow, project sync scripts). The mechanics it dispatches to live in
// sibling files: semver.go (version parsing/compare), prereqs.go (tool
// checks), scaffold.go (tree-walk + merge mini-engines), skills.go (agent
// skill deployment + legacy cleanup).

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// runSync indirects Sync for Init/Upgrade so tests can stub the sync step
// (full Sync needs git/yq/direnv/network). Same seam pattern as isBrewInstalled.
var runSync = Sync

// Sync performs the full workspace sync using the cached kit directory.
// systemVersion is the embedded version of the fab-kit binary (feeds the
// version guard). kitVersion is the kit content version to sync; when empty
// (the plain `fab sync` path) it is read from fab_version in config.yaml.
// Init and Upgrade pass kitVersion explicitly so config.yaml can be stamped
// only AFTER a successful sync (and so the guard compares the real binary
// version, not the kit version against itself).
// shimOnly runs steps 1-5 only; projectOnly runs step 6 only.
func Sync(systemVersion, kitVersion string, shimOnly, projectOnly bool) error {
	// Resolve the kit version from config.yaml unless the caller provided it.
	// The managed-repo check is a fab/project/config.yaml walk-up that does not
	// depend on git, so it MUST gate before gitRepoRoot(): a directory that is
	// neither git-tracked nor fab-managed exits with ExitNotManaged (3), the
	// distinguishable "not a fab-managed repo" signal, rather than collapsing to
	// the git-root-resolution error's generic exit 1. This keeps `fab sync`
	// symmetric with `fab-kit migrations-status`, which has no git precondition
	// and already exits 3 in the same directory. Init/Upgrade pass kitVersion
	// explicitly (config.yaml is not yet stamped), so they skip this check.
	fabVersion := kitVersion
	if fabVersion == "" {
		cfg, err := RequireManagedRepo()
		if err != nil {
			return err
		}
		fabVersion = cfg.FabVersion
	}

	// Resolve repo root via git. A managed repo without git context fails here
	// with a genuine error → exit 1 (R2: real failures are unchanged).
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return fmt.Errorf("cannot determine repo root: %w", err)
	}

	fabDir := filepath.Join(repoRoot, "fab")

	// Collected (non-aborting) deployment failure — sync continues its
	// remaining repair steps but MUST exit non-zero at the end.
	var deployErr error

	if !projectOnly {
		// Step 1: Prerequisites check
		if err := checkPrerequisites(); err != nil {
			return err
		}

		// Step 2: Version guard
		if err := versionGuard(fabVersion, systemVersion); err != nil {
			return err
		}

		// Step 3: Ensure cache
		fmt.Printf("Resolving kit v%s from cache...\n", fabVersion)
		if _, err := EnsureCached(fabVersion); err != nil {
			return err
		}

		cachedKitDir := CachedKitDir(fabVersion)

		// Step 4: Workspace scaffolding (all from cache)
		if err := scaffoldDirectories(repoRoot, fabDir, cachedKitDir, fabVersion); err != nil {
			return fmt.Errorf("scaffolding failed: %w", err)
		}

		scaffoldDir := filepath.Join(cachedKitDir, "scaffold")
		if dirExists(scaffoldDir) {
			if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
				return fmt.Errorf("scaffold tree-walk failed: %w", err)
			}
		}

		deployErr = deploySkills(repoRoot, cachedKitDir)

		// Hook sync — registers inline fab hook commands
		settingsPath := filepath.Join(repoRoot, ".claude", "settings.local.json")
		msg, err := syncHooks(settingsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: hook sync failed: %v\n", err)
		} else {
			fmt.Println(msg)
		}

		cleanLegacyAgents(repoRoot, cachedKitDir)

		// Step 5: Direnv allow
		runDirenvAllow(repoRoot)
	}

	if !shimOnly {
		// Step 6: Project-level sync scripts
		if err := runProjectSyncScripts(fabDir, repoRoot); err != nil {
			return err
		}
	}

	if deployErr != nil {
		return fmt.Errorf("skill deployment failed: %w", deployErr)
	}

	fmt.Println("Done.")
	return nil
}

// versionGuard ensures fab_version <= system fab-kit version.
// If the system version is too old, attempts auto-update via Update() and
// then verifies POST-STATE: it re-checks the installed binary version on PATH
// instead of trusting Update's return value (which is nil on the brew
// release-lag no-op, and used to be nil on the not-brew path too).
//
// When the guard trips, it ALWAYS returns an error — either "updated,
// re-run 'fab sync'" (the on-disk binary is now new enough, but this process
// still runs the old code) or actionable too-old instructions. It never
// continues the current sync on a binary known to be older than fab_version.
func versionGuard(fabVersion, systemVersion string) error {
	if systemVersion == "dev" {
		return nil // dev build, skip guard
	}
	if compareSemver(fabVersion, systemVersion) <= 0 {
		return nil // fab_version <= system version
	}

	fmt.Printf("Project needs v%s but system has v%s. Attempting update...\n", fabVersion, systemVersion)
	updateErr := Update(systemVersion, false)

	// Post-state check: what version is actually installed now?
	installed, verErr := installedBinaryVersion()
	if verErr == nil && compareSemver(fabVersion, installed) <= 0 {
		return fmt.Errorf("fab-kit was updated to v%s — re-run 'fab sync'", installed)
	}

	manualHint := "update fab-kit manually (brew upgrade fab-kit, or reinstall: brew install sahil87/tap/fab-kit), then re-run 'fab sync'"
	if updateErr != nil {
		return fmt.Errorf("system fab-kit v%s is older than project fab_version %s and auto-update did not succeed (%v) — %s",
			systemVersion, fabVersion, updateErr, manualHint)
	}
	if verErr != nil {
		return fmt.Errorf("system fab-kit v%s is older than project fab_version %s and the installed version could not be verified after update (%v) — %s",
			systemVersion, fabVersion, verErr, manualHint)
	}
	return fmt.Errorf("installed fab-kit v%s is still older than project fab_version %s after update — the Homebrew tap may lag the release; %s",
		installed, fabVersion, manualHint)
}

// gitRepoRoot resolves the repo root via git rev-parse.
func gitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repo: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// runDirenvAllow runs direnv allow if .envrc exists.
func runDirenvAllow(repoRoot string) {
	envrc := filepath.Join(repoRoot, ".envrc")
	if _, err := os.Stat(envrc); err == nil {
		cmd := exec.Command("direnv", "allow")
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run() // best-effort, don't fail sync on direnv issues
	}
}

// runProjectSyncScripts discovers and executes fab/sync/*.sh scripts.
func runProjectSyncScripts(fabDir, repoRoot string) error {
	syncDir := filepath.Join(fabDir, "sync")
	if !dirExists(syncDir) {
		return nil
	}

	entries, err := os.ReadDir(syncDir)
	if err != nil {
		return nil
	}

	var scripts []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sh") {
			scripts = append(scripts, e.Name())
		}
	}
	sort.Strings(scripts)

	if len(scripts) > 0 {
		fmt.Println("Running project-level sync scripts...")
	}

	for _, script := range scripts {
		scriptPath := filepath.Join(syncDir, script)
		fmt.Printf("  -> %s\n", script)
		cmd := exec.Command("bash", scriptPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = repoRoot
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("project sync script %s failed: %w", script, err)
		}
	}

	return nil
}
