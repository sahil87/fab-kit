package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Init handles `fab init` — scaffolds a new fab project or updates an existing one.
func Init() error {
	// 1. Resolve latest version
	fmt.Println("Resolving latest fab-kit version...")
	latest, err := LatestVersion()
	if err != nil {
		return fmt.Errorf("cannot resolve latest version: %w", err)
	}
	fmt.Printf("Latest version: %s\n", latest)

	// 2. Ensure cached
	_, err = EnsureCached(latest)
	if err != nil {
		return err
	}

	// 3. Get CWD as repo root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	// 4. Create/update config.yaml with fab_version
	configPath := filepath.Join(cwd, "fab", "project", "config.yaml")
	if err := setFabVersion(configPath, latest); err != nil {
		return fmt.Errorf("cannot update config.yaml: %w", err)
	}
	fmt.Printf("Set fab_version: %s in config.yaml\n", latest)

	// 5. Run sync
	fmt.Println("Setting up project...")
	if err := Sync(latest, false, false); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("\nfab initialized (v%s). Run /fab-setup in your AI agent to configure.\n", latest)
	return nil
}

// setFabVersion creates or updates config.yaml with the fab_version field.
func setFabVersion(path string, version string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	var data map[string]interface{}

	content, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(content, &data); err != nil {
			return fmt.Errorf("cannot parse existing config.yaml: %w", err)
		}
	}

	if data == nil {
		data = make(map[string]interface{})
	}
	data["fab_version"] = version

	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// copyDir copies src directory to dst, creating dst if needed.
// Existing files in dst are overwritten; existing files not in src are left alone.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Skip .gitkeep in bin/ — we want bin/ to stay clean
		if strings.HasSuffix(relPath, "bin/.gitkeep") {
			// Still create the directory
			return os.MkdirAll(filepath.Dir(destPath), 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}
