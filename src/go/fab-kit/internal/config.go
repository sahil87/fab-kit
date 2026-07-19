package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configRelPath = "fab/project/config.yaml"

// dotFabVersionRelPath is the plain-text sibling that holds the pinned engine
// version as of 260708-j0qm — fab_version moved out of config.yaml to here (a
// one-line file, sibling to fab/.kit-migration-version). It is the sole version
// source; config.yaml's fab_version: key is no longer consulted.
const dotFabVersionRelPath = "fab/.fab-version"

// ExitNotManaged is the process exit code the fab-kit binary uses when a
// command that requires a fab-managed repo is run outside one (ResolveConfig
// walked to the filesystem root without finding fab/project/config.yaml). It is
// deliberately distinct from the generic exit 1 (main() returns 1 for any other
// RunE error) so external callers (wt's default init, hop, operator scripts) can
// branch on "not applicable here" vs. "a real sync failure" without replicating
// fab's config.yaml walk-up. Mirrors the fab binary's in-handler os.Exit(N)
// tiering (pane_window_name.go, memory_index.go). Genuine failures — corrupt
// config, failed writes, version-guard trips — still return a normal error and
// exit 1, unchanged.
const ExitNotManaged = 3

// ConfigResult holds the resolved config path and fab_version.
type ConfigResult struct {
	ConfigPath string
	RepoRoot   string
	FabVersion string
}

// ResolveConfig walks up from CWD to find fab/project/config.yaml and reads fab_version.
func ResolveConfig() (*ConfigResult, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("cannot determine working directory: %w", err)
	}
	return resolveConfigFrom(cwd)
}

// RequireManagedRepo resolves the config and enforces that the caller is inside
// a fab-managed repo. A genuine ResolveConfig error (corrupt/unparseable config,
// missing fab_version) is returned unchanged for the caller to propagate — those
// still collapse to exit 1 in main(). The "not a fab-managed repo" case
// (ResolveConfig returned (nil, nil)) is terminal: it prints the actionable
// message to stderr and exits with ExitNotManaged, so it never reaches main()'s
// generic exit 1. Callers therefore only ever observe a non-nil *ConfigResult or
// a real error. Shared by Sync and the migrations-status command so the check
// (and its distinct exit code) lives in exactly one place.
func RequireManagedRepo() (*ConfigResult, error) {
	cfg, err := ResolveConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "not in a fab-managed repo. Run 'fab init' to set one up")
		os.Exit(ExitNotManaged)
	}
	return cfg, nil
}

// findConfigDir walks up from startDir looking for fab/project/config.yaml and
// returns the directory that contains the fab/ tree (the repo root) plus the
// located config.yaml path. When no config.yaml is found up to the filesystem
// root it returns ("", "") — "not a fab-managed repo". This is the walk-up
// primitive shared by resolveConfigFrom and the Upgrade recovery path, so both
// locate the repo root identically regardless of the CWD depth.
func findConfigDir(startDir string) (repoRoot, configPath string) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, configRelPath)
		if _, err := os.Stat(candidate); err == nil {
			return dir, candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", ""
		}
		dir = parent
	}
}

func resolveConfigFrom(startDir string) (*ConfigResult, error) {
	repoRoot, configPath := findConfigDir(startDir)
	if repoRoot == "" {
		// Reached filesystem root without finding config.yaml
		return nil, nil
	}
	version, err := readFabVersion(repoRoot)
	if err != nil {
		return nil, err
	}
	return &ConfigResult{
		ConfigPath: configPath,
		RepoRoot:   repoRoot,
		FabVersion: version,
	}, nil
}

// readFabVersion resolves the pinned engine version for a repo from the plain-text
// sibling fab/.fab-version (the sole source since 260708-j0qm; config.yaml is no
// longer consulted). repoRoot anchors the lookup. An absent or empty file is a
// real error with the init/upgrade guidance — the router needs a pinned version.
// A non-ENOENT read error (permission denied, I/O error) is surfaced separately
// with the path and underlying cause, so a present-but-unreadable pin is not
// misreported as "run init/upgrade" (which would not help).
func readFabVersion(repoRoot string) (string, error) {
	path := filepath.Join(repoRoot, dotFabVersionRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no fab version found in fab/.fab-version. Run 'fab init' (new repo) or 'fab upgrade-repo' (existing repo) to set one")
		}
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	if v := strings.TrimSpace(string(data)); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("no fab version found in fab/.fab-version. Run 'fab init' (new repo) or 'fab upgrade-repo' (existing repo) to set one")
}
