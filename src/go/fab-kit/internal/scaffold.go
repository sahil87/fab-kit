package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scaffoldDirectories creates required directories and .gitkeep files.
// Write failures are propagated — a silently failed .kit-migration-version
// write would silently disable migration discovery in Upgrade.
func scaffoldDirectories(repoRoot, fabDir, kitDir, kitVersion string) error {
	docsDir := filepath.Join(repoRoot, "docs")
	dirs := []string{
		filepath.Join(fabDir, "changes"),
		filepath.Join(fabDir, "changes", "archive"),
		filepath.Join(docsDir, "memory"),
		filepath.Join(docsDir, "specs"),
	}

	for _, dir := range dirs {
		if !dirExists(dir) {
			rel, _ := filepath.Rel(repoRoot, dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("cannot create %s: %w", rel, err)
			}
			fmt.Printf("Created: %s\n", rel)
		}
	}

	// .gitkeep files
	for _, name := range []string{
		filepath.Join(fabDir, "changes", ".gitkeep"),
		filepath.Join(fabDir, "changes", "archive", ".gitkeep"),
	} {
		if _, err := os.Stat(name); os.IsNotExist(err) {
			if err := os.WriteFile(name, nil, 0644); err != nil {
				return fmt.Errorf("cannot write %s: %w", name, err)
			}
		}
	}

	// fab/.kit-migration-version — dual version model
	migrationVersionFile := filepath.Join(fabDir, ".kit-migration-version")

	// Backward compat: migrate old fab/project/VERSION to new location
	oldVersionFile := filepath.Join(fabDir, "project", "VERSION")
	if _, err := os.Stat(oldVersionFile); err == nil {
		oldVer, _ := os.ReadFile(oldVersionFile)
		oldVerStr := strings.TrimSpace(string(oldVer))
		if _, err := os.Stat(migrationVersionFile); err == nil {
			// Both exist — new file takes precedence, remove old
			if err := os.Remove(oldVersionFile); err != nil {
				return fmt.Errorf("cannot remove stale fab/project/VERSION: %w", err)
			}
			fmt.Println("Cleaned: stale fab/project/VERSION (migrated to fab/.kit-migration-version)")
		} else {
			// Old exists, new doesn't — migrate
			if err := os.Rename(oldVersionFile, migrationVersionFile); err != nil {
				return fmt.Errorf("cannot migrate fab/project/VERSION to fab/.kit-migration-version: %w", err)
			}
			fmt.Printf("Migrated: fab/project/VERSION -> fab/.kit-migration-version (%s)\n", oldVerStr)
		}
	}

	if _, err := os.Stat(migrationVersionFile); err == nil {
		content, _ := os.ReadFile(migrationVersionFile)
		fmt.Printf("fab/.kit-migration-version: OK (%s)\n", strings.TrimSpace(string(content)))
	} else if _, err := os.Stat(filepath.Join(fabDir, "project", "config.yaml")); err == nil {
		// Existing project: set base version
		if err := os.WriteFile(migrationVersionFile, []byte("0.1.0\n"), 0644); err != nil {
			return fmt.Errorf("cannot write fab/.kit-migration-version: %w", err)
		}
		fmt.Println("Created: fab/.kit-migration-version (0.1.0 — existing project, run `/fab-setup migrations` to migrate)")
	} else {
		// New project: match engine version
		versionSrc := filepath.Join(kitDir, "VERSION")
		data, err := os.ReadFile(versionSrc)
		if err != nil {
			return fmt.Errorf("cannot read kit VERSION (%s): %w", versionSrc, err)
		}
		if err := os.WriteFile(migrationVersionFile, data, 0644); err != nil {
			return fmt.Errorf("cannot write fab/.kit-migration-version: %w", err)
		}
		fmt.Printf("Created: fab/.kit-migration-version (%s)\n", kitVersion)
	}

	return nil
}

// scaffoldTreeWalk walks the scaffold directory and dispatches by filename convention.
func scaffoldTreeWalk(scaffoldDir, repoRoot string) error {
	var files []string
	err := filepath.Walk(scaffoldDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, scaffoldFile := range files {
		relPath, _ := filepath.Rel(scaffoldDir, scaffoldFile)
		dirPart := filepath.Dir(relPath)
		fileName := filepath.Base(relPath)

		isFragment := strings.HasPrefix(fileName, "fragment-")
		if isFragment {
			fileName = strings.TrimPrefix(fileName, "fragment-")
		}

		var destPath string
		if dirPart == "." {
			destPath = fileName
		} else {
			destPath = filepath.Join(dirPart, fileName)
		}

		dest := filepath.Join(repoRoot, destPath)
		os.MkdirAll(filepath.Dir(dest), 0755)

		if isFragment {
			if strings.HasSuffix(fileName, ".json") {
				if err := jsonMergePermissions(scaffoldFile, dest, destPath); err != nil {
					return err
				}
			} else {
				if err := lineEnsureMerge(scaffoldFile, dest, destPath); err != nil {
					return err
				}
			}
		} else {
			// copy-if-absent
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				data, err := os.ReadFile(scaffoldFile)
				if err != nil {
					return err
				}
				if err := os.WriteFile(dest, data, 0644); err != nil {
					return err
				}
				fmt.Printf("Created: %s\n", destPath)
			}
		}
	}

	return nil
}

// jsonMergePermissions merges permissions.allow arrays between source and dest JSON files.
func jsonMergePermissions(source, dest, label string) error {
	srcData, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("cannot read scaffold %s: %w", label, err)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		// Copy source to dest
		os.MkdirAll(filepath.Dir(dest), 0755)
		if err := os.WriteFile(dest, srcData, 0644); err != nil {
			return err
		}
		// Count permissions
		var srcJSON map[string]interface{}
		json.Unmarshal(srcData, &srcJSON)
		count := 0
		if perms, ok := srcJSON["permissions"].(map[string]interface{}); ok {
			if allow, ok := perms["allow"].([]interface{}); ok {
				count = len(allow)
			}
		}
		fmt.Printf("Created: %s (%d permission rules)\n", label, count)
		return nil
	}

	// Merge: read both files and merge permissions.allow arrays
	destData, err := os.ReadFile(dest)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", label, err)
	}

	var srcJSON, destJSON map[string]interface{}
	if err := json.Unmarshal(srcData, &srcJSON); err != nil {
		return fmt.Errorf("cannot parse scaffold JSON %s: %w", label, err)
	}
	if err := json.Unmarshal(destData, &destJSON); err != nil {
		return fmt.Errorf("cannot parse existing JSON %s: %w", label, err)
	}

	// Extract permissions.allow from both
	srcAllow := extractPermissionsAllow(srcJSON)
	destAllow := extractPermissionsAllow(destJSON)

	// Find new entries (in src but not in dest)
	existing := make(map[string]bool)
	for _, entry := range destAllow {
		existing[fmt.Sprintf("%v", entry)] = true
	}

	var newEntries []interface{}
	for _, entry := range srcAllow {
		key := fmt.Sprintf("%v", entry)
		if !existing[key] {
			newEntries = append(newEntries, entry)
		}
	}

	if len(newEntries) > 0 {
		// Add new entries to dest
		destAllow = append(destAllow, newEntries...)
		setPermissionsAllow(destJSON, destAllow)

		merged, err := json.MarshalIndent(destJSON, "", "  ")
		if err != nil {
			return err
		}
		merged = append(merged, '\n')
		if err := os.WriteFile(dest, merged, 0644); err != nil {
			return err
		}
		fmt.Printf("Updated: %s (added %d permission rules)\n", label, len(newEntries))
	} else {
		fmt.Printf("%s: OK\n", label)
	}

	return nil
}

// extractPermissionsAllow extracts the permissions.allow array from a JSON object.
func extractPermissionsAllow(obj map[string]interface{}) []interface{} {
	perms, ok := obj["permissions"].(map[string]interface{})
	if !ok {
		return nil
	}
	allow, ok := perms["allow"].([]interface{})
	if !ok {
		return nil
	}
	return allow
}

// setPermissionsAllow sets the permissions.allow array in a JSON object.
func setPermissionsAllow(obj map[string]interface{}, allow []interface{}) {
	perms, ok := obj["permissions"].(map[string]interface{})
	if !ok {
		perms = make(map[string]interface{})
		obj["permissions"] = perms
	}
	perms["allow"] = allow
}

// lineEnsureMerge appends non-duplicate, non-comment lines from source to dest.
func lineEnsureMerge(source, dest, label string) error {
	srcData, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("cannot read scaffold %s: %w", label, err)
	}

	// Legacy migration: if target is a symlink, resolve to real file
	if info, err := os.Lstat(dest); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, _ := os.ReadFile(dest)
		os.Remove(dest)
		if len(resolved) > 0 {
			os.WriteFile(dest, resolved, 0644)
		}
		fmt.Printf("%s: migrated from symlink to file\n", label)
	}

	existed := false
	if _, err := os.Stat(dest); err == nil {
		existed = true
	}

	var added []string
	srcLines := strings.Split(string(srcData), "\n")

	for _, line := range srcLines {
		entry := strings.TrimRight(line, "\r")
		if entry == "" || strings.HasPrefix(entry, "#") {
			continue
		}

		if _, err := os.Stat(dest); os.IsNotExist(err) {
			// Create the file with this entry
			os.WriteFile(dest, []byte(entry+"\n"), 0644)
			added = append(added, entry)
		} else {
			// Check if entry already exists
			destData, err := os.ReadFile(dest)
			if err != nil {
				return err
			}
			destLines := strings.Split(string(destData), "\n")
			found := false
			for _, dl := range destLines {
				if strings.TrimRight(dl, "\r") == entry {
					found = true
					break
				}
			}
			if !found {
				// Append with newline
				f, err := os.OpenFile(dest, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				fmt.Fprintf(f, "\n%s\n", entry)
				f.Close()
				added = append(added, entry)
			}
		}
	}

	if len(added) > 0 {
		if !existed {
			fmt.Printf("Created: %s (added %s)\n", label, strings.Join(added, " "))
		} else {
			fmt.Printf("Updated: %s (added %s)\n", label, strings.Join(added, " "))
		}
	} else {
		fmt.Printf("%s: OK\n", label)
	}

	return nil
}
