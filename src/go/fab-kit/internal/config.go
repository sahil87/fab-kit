package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const configRelPath = "fab/project/config.yaml"

// dotFabVersionRelPath is the plain-text sibling that holds the pinned engine
// version as of 260708-j0qm — fab_version moved out of config.yaml to here (a
// one-line file, sibling to fab/.kit-migration-version). config.yaml's
// fab_version: key is read only as a one-compat-window fallback.
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

func resolveConfigFrom(startDir string) (*ConfigResult, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, configRelPath)
		if _, err := os.Stat(candidate); err == nil {
			version, err := readFabVersion(dir, candidate)
			if err != nil {
				return nil, err
			}
			return &ConfigResult{
				ConfigPath: candidate,
				RepoRoot:   dir,
				FabVersion: version,
			}, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return nil, nil
		}
		dir = parent
	}
}

// readFabVersion resolves the pinned engine version for a repo. As of 260708-j0qm
// the version lives in the plain-text sibling fab/.fab-version; readFabVersion
// reads it FIRST and, for one compat window, falls back to a config.yaml
// fab_version: key (repos not yet migrated by 2.14.0-to-2.15.0). repoRoot anchors
// the .fab-version lookup; configPath is the located config.yaml. An empty result
// from both sources is a real error — the router needs a pinned version.
func readFabVersion(repoRoot, configPath string) (string, error) {
	// 1. fab/.fab-version (authoritative post-migration).
	if data, err := os.ReadFile(filepath.Join(repoRoot, dotFabVersionRelPath)); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			return v, nil
		}
	}

	// 2. Fallback: config.yaml fab_version: key (pre-migration compat window).
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", configPath, err)
	}
	var cfg struct {
		FabVersion string `yaml:"fab_version"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("cannot parse %s: %w", configPath, err)
	}
	if cfg.FabVersion == "" {
		return "", fmt.Errorf("no fab version found in fab/.fab-version or config.yaml. Run 'fab init' to set one")
	}
	return cfg.FabVersion, nil
}
