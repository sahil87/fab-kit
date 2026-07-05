package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configRelPath = "fab/project/config.yaml"

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
			version, err := readFabVersion(candidate)
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

// readFabVersion reads the fab_version field from a config.yaml file.
func readFabVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}

	var cfg struct {
		FabVersion string `yaml:"fab_version"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("cannot parse %s: %w", path, err)
	}

	if cfg.FabVersion == "" {
		return "", fmt.Errorf("no fab_version in config.yaml. Run 'fab init' to set one")
	}
	return cfg.FabVersion, nil
}
