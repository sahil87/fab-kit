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
// Write failures are propagated (return err) — a silently failed
// .kit-migration-version write would disable migration discovery in Upgrade.
// This is the contract across the scaffold walk: scaffoldTreeWalk,
// jsonMergePermissions, and lineEnsureMerge all return their os.WriteFile /
// append errors rather than swallowing them, so a half-scaffolded tree
// (disk full, permissions, read-only mount) surfaces as a setup failure
// instead of looking successful.
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

// gitignoreNormalize reduces a gitignore line to its core directory token by
// stripping a single leading "/" and a single trailing "/*" or "/". It is the
// shared primitive for the gitignore-aware dedup: a deeper nested path like
// "/.claude/commands/" normalizes to ".claude/commands" (which still contains a
// slash) and therefore never equals a directory token like ".claude". The
// caller compares normalized forms for exact equality, so the residual slash is
// what keeps deeper paths from spuriously "covering" the directory entry.
func gitignoreNormalize(line string) string {
	s := strings.TrimRight(line, "\r")
	s = strings.TrimPrefix(s, "/")
	if strings.HasSuffix(s, "/*") {
		s = strings.TrimSuffix(s, "/*")
	} else {
		s = strings.TrimSuffix(s, "/")
	}
	return s
}

// gitignoreCovers reports whether an existing .gitignore line already covers the
// fragment entry under directory-token equivalence. For an entry like "/.claude"
// the covering set is { /.claude, /.claude/, /.claude/*, .claude, .claude/,
// .claude/* } — leading slash optional, optional trailing "/" or "/*". A deeper
// nested path (e.g. "/.claude/commands/") does NOT cover the entry (its
// normalized form retains an internal slash and so differs from the core token).
func gitignoreCovers(existingLine, entry string) bool {
	return gitignoreNormalize(existingLine) == gitignoreNormalize(entry)
}

// gitignoreHasNegation reports whether any destination line negates the entry's
// core directory token (a line like "!/.claude/..." or "!.claude/..."). It is
// the binding Guardrail B hard-stop: when present, sync must never append a
// broader ignore for that entry — regardless of whether a broader "/.claude/*"
// exclusion precedes the negation, and regardless of the variant-coverage check.
func gitignoreHasNegation(destLines []string, entry string) bool {
	token := gitignoreNormalize(entry)
	if token == "" {
		return false
	}
	for _, dl := range destLines {
		s := strings.TrimRight(dl, "\r")
		if !strings.HasPrefix(s, "!") {
			continue
		}
		// Strip the "!" then a single leading "/", and check whether the
		// remainder is the token itself or a path under it (token + "/").
		neg := strings.TrimPrefix(strings.TrimPrefix(s, "!"), "/")
		if neg == token || strings.HasPrefix(neg, token+"/") {
			return true
		}
	}
	return false
}

// lineEnsureMerge appends non-duplicate, non-comment lines from source to dest.
//
// Dedup is literal string equality except for a destination whose basename is
// ".gitignore", where it is gitignore-aware: a fragment entry is treated as
// already present when an existing line covers it under directory-token
// equivalence (gitignoreCovers), and the append is suppressed outright when the
// destination already negates the entry's core token (gitignoreHasNegation,
// Guardrail B). Non-.gitignore destinations (e.g. .envrc) keep strict literal
// equality (Guardrail A).
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
			if err := os.WriteFile(dest, resolved, 0644); err != nil {
				return fmt.Errorf("cannot write %s: %w", label, err)
			}
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
			if err := os.WriteFile(dest, []byte(entry+"\n"), 0644); err != nil {
				return fmt.Errorf("cannot write %s: %w", label, err)
			}
			added = append(added, entry)
		} else {
			// Check if entry already exists
			destData, err := os.ReadFile(dest)
			if err != nil {
				return err
			}
			destLines := strings.Split(string(destData), "\n")
			isGitignore := filepath.Base(label) == ".gitignore"
			found := false
			// Guardrail B (.gitignore only): a present negation for the entry's
			// core token is a hard stop — never append a broader ignore.
			if isGitignore && gitignoreHasNegation(destLines, entry) {
				found = true
			}
			for _, dl := range destLines {
				if found {
					break
				}
				if isGitignore {
					// Gitignore-aware coverage (directory-token equivalence).
					if gitignoreCovers(dl, entry) {
						found = true
					}
				} else if strings.TrimRight(dl, "\r") == entry {
					// Non-.gitignore (e.g. .envrc): strict literal equality.
					found = true
				}
			}
			if !found {
				// Append with newline
				f, err := os.OpenFile(dest, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(f, "\n%s\n", entry); err != nil {
					f.Close()
					return fmt.Errorf("cannot append to %s: %w", label, err)
				}
				if err := f.Close(); err != nil {
					return fmt.Errorf("cannot close %s: %w", label, err)
				}
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
