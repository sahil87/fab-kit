package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const latestReleaseURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"

// RunInit handles the `fab init` command.
func RunInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Check if already initialized.
	configPath := filepath.Join(cwd, configRelPath)
	if cfg, err := parseConfigFile(configPath); err == nil && cfg.FabVersion != "" {
		fmt.Printf("Already initialized (fab_version: %s). Edit fab_version in config.yaml to change versions.\n", cfg.FabVersion)
		return nil
	}

	// Determine latest version.
	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("determining latest version: %w\nCheck your network connectivity and try again.", err)
	}

	fmt.Printf("Initializing fab-kit %s...\n", latest)

	// Ensure the version is cached.
	_, err = EnsureCached(latest)
	if err != nil {
		return err
	}

	// Run fab-sync.sh from the cached version.
	vdir, err := VersionDir(latest)
	if err != nil {
		return err
	}
	syncScript := filepath.Join(vdir, "fab/.kit/scripts/fab-sync.sh")
	if _, err := os.Stat(syncScript); err == nil {
		cmd := exec.Command("bash", syncScript)
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running fab-sync.sh: %w", err)
		}
	}

	// Create or update config.yaml with fab_version.
	if err := setFabVersion(configPath, latest); err != nil {
		return fmt.Errorf("updating config.yaml: %w", err)
	}

	fmt.Printf("Done. fab_version set to %s in config.yaml.\n", latest)
	return nil
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(latestReleaseURL)
	if err != nil {
		return "", fmt.Errorf("querying GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	// Strip leading "v" from tag name (e.g., "v0.41.0" → "0.41.0").
	version := strings.TrimPrefix(release.TagName, "v")
	if version == "" {
		return "", fmt.Errorf("empty version from GitHub release")
	}

	return version, nil
}

func parseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setFabVersion(configPath, version string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Read existing config to preserve other fields.
	existing := make(map[string]interface{})
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("parsing existing config: %w", err)
		}
	}

	existing["fab_version"] = version

	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, data, 0o644)
}
