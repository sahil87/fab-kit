package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// requiredTools lists prerequisites that must be available on PATH.
var requiredTools = []string{"git", "bash", "yq", "direnv"}

// agentConfig describes how to deploy skills to a specific AI agent.
type agentConfig struct {
	Label   string // display name
	CLI     string // command to check on PATH
	BaseDir string // target directory relative to repo root
	Format  string // "directory" or "flat"
	Mode    string // "copy" or "symlink"
}

// Sync performs the full workspace sync using the cached kit directory.
// systemVersion is the embedded version of the fab-kit binary.
// shimOnly runs steps 1-5 only; projectOnly runs step 6 only.
func Sync(systemVersion string, shimOnly, projectOnly bool) error {
	// Resolve repo root via git
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return fmt.Errorf("cannot determine repo root: %w", err)
	}

	fabDir := filepath.Join(repoRoot, "fab")

	// Resolve fab_version from config.yaml
	cfg, err := ResolveConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("not in a fab-managed repo. Run 'fab init' to set one up")
	}
	fabVersion := cfg.FabVersion

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
		kitVersion := fabVersion

		// Step 4: Workspace scaffolding (all from cache)
		scaffoldDirectories(repoRoot, fabDir, cachedKitDir, kitVersion)

		scaffoldDir := filepath.Join(cachedKitDir, "scaffold")
		if dirExists(scaffoldDir) {
			if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
				return fmt.Errorf("scaffold tree-walk failed: %w", err)
			}
		}

		deploySkills(repoRoot, cachedKitDir)

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

	fmt.Println("Done.")
	return nil
}

// versionGuard ensures fab_version <= system fab-kit version.
// If the system version is too old, attempts auto-update via Update().
func versionGuard(fabVersion, systemVersion string) error {
	if systemVersion == "dev" {
		return nil // dev build, skip guard
	}
	cmp := compareSemver(fabVersion, systemVersion)
	if cmp <= 0 {
		return nil // fab_version <= system version
	}

	fmt.Printf("Project needs v%s but system has v%s. Attempting update...\n", fabVersion, systemVersion)
	if err := Update(systemVersion); err != nil {
		return fmt.Errorf("system fab-kit v%s is older than project fab_version %s. Run 'fab update' manually: %w",
			systemVersion, fabVersion, err)
	}

	// After update, we can't re-check the version in-process (the binary hasn't reloaded).
	// Trust that brew upgraded to latest. If still insufficient, next run will catch it.
	return nil
}

// compareSemver compares two semver strings. Returns -1, 0, or 1.
func compareSemver(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)
	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

// parseSemver splits a version string into [major, minor, patch].
func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
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

// checkPrerequisites validates that all required tools are available.
func checkPrerequisites() error {
	var missing []string
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s. Install with: brew install %s",
			strings.Join(missing, ", "), strings.Join(missing, " "))
	}

	// Validate yq is v4+
	out, err := exec.Command("yq", "--version").Output()
	if err == nil {
		verStr := string(out)
		// Extract version number — yq output is like "yq (https://...) version v4.x.y"
		// or "yq version 4.x.y"
		parts := strings.Fields(verStr)
		for _, p := range parts {
			p = strings.TrimPrefix(p, "v")
			if len(p) > 0 && p[0] >= '0' && p[0] <= '9' {
				major := strings.Split(p, ".")[0]
				if major < "4" {
					return fmt.Errorf("yq version 4+ required (found %s). Install the Go version: brew install yq", p)
				}
				break
			}
		}
	}

	return nil
}

// scaffoldDirectories creates required directories and .gitkeep files.
func scaffoldDirectories(repoRoot, fabDir, kitDir, kitVersion string) {
	docsDir := filepath.Join(repoRoot, "docs")
	dirs := []string{
		filepath.Join(fabDir, "changes"),
		filepath.Join(fabDir, "changes", "archive"),
		filepath.Join(docsDir, "memory"),
		filepath.Join(docsDir, "specs"),
	}

	for _, dir := range dirs {
		if !dirExists(dir) {
			os.MkdirAll(dir, 0755)
			rel, _ := filepath.Rel(repoRoot, dir)
			fmt.Printf("Created: %s\n", rel)
		}
	}

	// .gitkeep files
	for _, name := range []string{
		filepath.Join(fabDir, "changes", ".gitkeep"),
		filepath.Join(fabDir, "changes", "archive", ".gitkeep"),
	} {
		if _, err := os.Stat(name); os.IsNotExist(err) {
			os.WriteFile(name, nil, 0644)
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
			os.Remove(oldVersionFile)
			fmt.Println("Cleaned: stale fab/project/VERSION (migrated to fab/.kit-migration-version)")
		} else {
			// Old exists, new doesn't — migrate
			os.Rename(oldVersionFile, migrationVersionFile)
			fmt.Printf("Migrated: fab/project/VERSION -> fab/.kit-migration-version (%s)\n", oldVerStr)
		}
	}

	if _, err := os.Stat(migrationVersionFile); err == nil {
		content, _ := os.ReadFile(migrationVersionFile)
		fmt.Printf("fab/.kit-migration-version: OK (%s)\n", strings.TrimSpace(string(content)))
	} else if _, err := os.Stat(filepath.Join(fabDir, "project", "config.yaml")); err == nil {
		// Existing project: set base version
		os.WriteFile(migrationVersionFile, []byte("0.1.0\n"), 0644)
		fmt.Println("Created: fab/.kit-migration-version (0.1.0 — existing project, run `/fab-setup migrations` to migrate)")
	} else {
		// New project: match engine version
		versionSrc := filepath.Join(kitDir, "VERSION")
		data, _ := os.ReadFile(versionSrc)
		os.WriteFile(migrationVersionFile, data, 0644)
		fmt.Printf("Created: fab/.kit-migration-version (%s)\n", kitVersion)
	}
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

// deploySkills deploys skill files to agent-specific directories.
func deploySkills(repoRoot, kitDir string) {
	// Collect canonical skill list
	skillsDir := filepath.Join(kitDir, "skills")
	skills := listSkills(skillsDir)
	if len(skills) == 0 {
		return
	}

	// Define agent configurations
	agents := []agentConfig{
		{Label: "Claude Code", CLI: "claude", BaseDir: filepath.Join(repoRoot, ".claude", "skills"), Format: "directory", Mode: "copy"},
		{Label: "OpenCode", CLI: "opencode", BaseDir: filepath.Join(repoRoot, ".opencode", "commands"), Format: "flat", Mode: "copy"},
		{Label: "Codex", CLI: "codex", BaseDir: filepath.Join(repoRoot, ".agents", "skills"), Format: "directory", Mode: "copy"},
		{Label: "Gemini", CLI: "gemini", BaseDir: filepath.Join(repoRoot, ".gemini", "skills"), Format: "directory", Mode: "copy"},
	}

	agentsFound := 0
	for _, agent := range agents {
		if !agentAvailable(agent.CLI) {
			fmt.Printf("Skipping %s: %s not found in PATH\n", agent.Label, agent.CLI)
			continue
		}

		syncAgentSkills(agent, skills, skillsDir)
		cleanStaleSkills(agent.BaseDir, agent.Format, skills, repoRoot)
		agentsFound++
	}

	if agentsFound == 0 {
		fmt.Println("Warning: No agent CLIs found in PATH. Skills were not deployed to any agent.")
	}
}

// listSkills returns the base names (without .md) of all skill files.
func listSkills(skillsDir string) []string {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}
	var skills []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".md") {
			skills = append(skills, strings.TrimSuffix(name, ".md"))
		}
	}
	return skills
}

// agentAvailable checks if an agent CLI is available.
// Respects FAB_AGENTS env var override.
func agentAvailable(cli string) bool {
	if fabAgents, ok := os.LookupEnv("FAB_AGENTS"); ok {
		for _, a := range strings.Fields(fabAgents) {
			if a == cli {
				return true
			}
		}
		return false
	}
	_, err := exec.LookPath(cli)
	return err == nil
}

// syncAgentSkills deploys skills to an agent's directory.
func syncAgentSkills(agent agentConfig, skills []string, skillsDir string) {
	os.MkdirAll(agent.BaseDir, 0755)

	created, repaired, ok := 0, 0, 0

	for _, skill := range skills {
		src := filepath.Join(skillsDir, skill+".md")
		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("WARN: kit/skills/%s.md missing in cache — skipping\n", skill)
			continue
		}

		var dest string
		if agent.Format == "directory" {
			os.MkdirAll(filepath.Join(agent.BaseDir, skill), 0755)
			dest = filepath.Join(agent.BaseDir, skill, "SKILL.md")
		} else {
			dest = filepath.Join(agent.BaseDir, skill+".md")
		}

		if agent.Mode == "copy" {
			srcData, err := os.ReadFile(src)
			if err != nil {
				continue
			}

			if info, err := os.Lstat(dest); err == nil && info.Mode()&os.ModeSymlink == 0 {
				// File exists and is not a symlink — compare content
				destData, _ := os.ReadFile(dest)
				if string(srcData) == string(destData) {
					ok++
				} else {
					os.WriteFile(dest, srcData, 0644)
					repaired++
				}
			} else if _, err := os.Lstat(dest); err == nil {
				// Exists as symlink or something else — replace
				os.Remove(dest)
				os.WriteFile(dest, srcData, 0644)
				repaired++
			} else {
				os.WriteFile(dest, srcData, 0644)
				created++
			}
		} else {
			// Symlink mode — target is the absolute path in the cache
			target := filepath.Join(skillsDir, skill+".md")
			if info, err := os.Lstat(dest); err == nil && info.Mode()&os.ModeSymlink != 0 {
				// Symlink exists — check if target resolves
				if _, err := os.Stat(dest); err == nil {
					ok++
				} else {
					os.Remove(dest)
					os.Symlink(target, dest)
					repaired++
				}
			} else if _, err := os.Lstat(dest); err == nil {
				os.Remove(dest)
				os.Symlink(target, dest)
				repaired++
			} else {
				os.Symlink(target, dest)
				created++
			}
		}
	}

	total := created + repaired + ok
	fmt.Printf("%-12s %d/%d (created %d, repaired %d, already valid %d)\n",
		agent.Label+":", total, len(skills), created, repaired, ok)
}

// cleanStaleSkills removes skill entries not present in the canonical skills list.
func cleanStaleSkills(baseDir, format string, skills []string, repoRoot string) {
	if !dirExists(baseDir) {
		return
	}

	skillSet := make(map[string]bool)
	for _, s := range skills {
		skillSet[s] = true
	}

	removed := 0
	if format == "directory" {
		entries, _ := os.ReadDir(baseDir)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if !skillSet[e.Name()] {
				os.RemoveAll(filepath.Join(baseDir, e.Name()))
				removed++
			}
		}
	} else {
		entries, _ := os.ReadDir(baseDir)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".md") {
				continue
			}
			stem := strings.TrimSuffix(name, ".md")
			if !skillSet[stem] {
				os.Remove(filepath.Join(baseDir, name))
				removed++
			}
		}
	}

	if removed > 0 {
		rel, _ := filepath.Rel(repoRoot, baseDir)
		fmt.Printf("Cleaned: %d stale entries from %s\n", removed, rel)
	}
}

// cleanLegacyAgents removes .claude/agents/ files matching known skill names.
func cleanLegacyAgents(repoRoot, kitDir string) {
	agentsDir := filepath.Join(repoRoot, ".claude", "agents")
	if !dirExists(agentsDir) {
		return
	}

	skills := listSkills(filepath.Join(kitDir, "skills"))
	skillSet := make(map[string]bool)
	for _, s := range skills {
		skillSet[s] = true
	}

	staleAgents := 0
	entries, _ := os.ReadDir(agentsDir)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		stem := strings.TrimSuffix(name, ".md")
		if skillSet[stem] {
			os.Remove(filepath.Join(agentsDir, name))
			staleAgents++
		}
	}

	if staleAgents > 0 {
		fmt.Printf("Cleaned: %d stale agent files from .claude/agents/\n", staleAgents)
	}
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
