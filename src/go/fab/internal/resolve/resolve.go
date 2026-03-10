package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FabRoot returns the fab/ directory path by searching upward from cwd.
func FabRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "fab")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("fab/ directory not found")
		}
		dir = parent
	}
}

// ToFolder resolves a change reference to a full folder name.
// If override is empty, reads .fab-status.yaml symlink at repo root.
func ToFolder(fabRoot, override string) (string, error) {
	changesDir := filepath.Join(fabRoot, "changes")

	if override != "" {
		return resolveOverride(changesDir, override)
	}
	return resolveFromCurrent(fabRoot, changesDir)
}

// ExtractID extracts the 4-char change ID from a YYMMDD-XXXX-slug folder name.
func ExtractID(folder string) string {
	parts := strings.SplitN(folder, "-", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ToDir returns the directory path relative to repo root.
func ToDir(fabRoot, override string) (string, error) {
	folder, err := ToFolder(fabRoot, override)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("fab/changes/%s/", folder), nil
}

// ToStatus returns the .status.yaml path relative to repo root.
func ToStatus(fabRoot, override string) (string, error) {
	folder, err := ToFolder(fabRoot, override)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("fab/changes/%s/.status.yaml", folder), nil
}

// ToAbsDir returns the absolute directory path.
func ToAbsDir(fabRoot, override string) (string, error) {
	folder, err := ToFolder(fabRoot, override)
	if err != nil {
		return "", err
	}
	return filepath.Join(fabRoot, "changes", folder), nil
}

// ToAbsStatus returns the absolute .status.yaml path.
func ToAbsStatus(fabRoot, override string) (string, error) {
	folder, err := ToFolder(fabRoot, override)
	if err != nil {
		return "", err
	}
	return filepath.Join(fabRoot, "changes", folder, ".status.yaml"), nil
}

func resolveOverride(changesDir, override string) (string, error) {
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return "", fmt.Errorf("fab/changes/ not found.")
	}

	folders, err := listChangeFolders(changesDir)
	if err != nil {
		return "", err
	}
	if len(folders) == 0 {
		return "", fmt.Errorf("No active changes found.")
	}

	overrideLower := strings.ToLower(override)

	// Exact match
	for _, f := range folders {
		if strings.ToLower(f) == overrideLower {
			return f, nil
		}
	}

	// Substring match
	var partials []string
	for _, f := range folders {
		if strings.Contains(strings.ToLower(f), overrideLower) {
			partials = append(partials, f)
		}
	}

	if len(partials) == 1 {
		return partials[0], nil
	}
	if len(partials) > 1 {
		return "", fmt.Errorf("Multiple changes match \"%s\": %s.", override, strings.Join(partials, ", "))
	}

	return "", fmt.Errorf("No change matches \"%s\".", override)
}

func resolveFromCurrent(fabRoot, changesDir string) (string, error) {
	// Read .fab-status.yaml symlink at repo root
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	if target, err := os.Readlink(symlinkPath); err == nil {
		// target is "fab/changes/{name}/.status.yaml"
		name := ExtractFolderFromSymlink(target)
		if name != "" {
			return name, nil
		}
	}

	// Fallback: single-change guess
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return "", fmt.Errorf("No active change.")
	}

	var candidates []string
	entries, _ := os.ReadDir(changesDir)
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		statusPath := filepath.Join(changesDir, e.Name(), ".status.yaml")
		if _, err := os.Stat(statusPath); err == nil {
			candidates = append(candidates, e.Name())
		}
	}

	if len(candidates) == 1 {
		fmt.Fprintf(os.Stderr, "(resolved from single active change)\n")
		return candidates[0], nil
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("No active change.")
	}
	return "", fmt.Errorf("No active change (multiple changes exist — use /fab-switch).")
}

// ExtractFolderFromSymlink extracts the change folder name from a symlink target path.
// Expected format: "fab/changes/{name}/.status.yaml"
func ExtractFolderFromSymlink(target string) string {
	// Normalize separators for cross-platform
	target = filepath.ToSlash(target)
	const prefix = "fab/changes/"
	const suffix = "/.status.yaml"
	if strings.HasPrefix(target, prefix) && strings.HasSuffix(target, suffix) {
		name := target[len(prefix) : len(target)-len(suffix)]
		if name != "" && !strings.Contains(name, "/") {
			return name
		}
	}
	return ""
}

func listChangeFolders(changesDir string) ([]string, error) {
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		return nil, err
	}
	var folders []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "archive" {
			folders = append(folders, e.Name())
		}
	}
	return folders, nil
}
