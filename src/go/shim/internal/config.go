package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const configRelPath = "fab/project/config.yaml"

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
