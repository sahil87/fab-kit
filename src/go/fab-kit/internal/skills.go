package internal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// agentConfig describes how to deploy skills to a specific AI agent.
type agentConfig struct {
	Label   string // display name
	CLI     string // command to check on PATH
	BaseDir string // target directory relative to repo root
	Format  string // "directory" or "flat"
	Mode    string // "copy" or "symlink"
}

// deploySkills deploys skill files to agent-specific directories.
// Returns a non-nil error when any agent's deployment had write failures,
// so Sync exits non-zero instead of reporting stale skills as success.
func deploySkills(repoRoot, kitDir string) error {
	// Collect canonical skill list
	skillsDir := filepath.Join(kitDir, "skills")
	skills := listSkills(skillsDir)
	if len(skills) == 0 {
		return nil
	}

	// Define agent configurations
	agents := []agentConfig{
		{Label: "Claude Code", CLI: "claude", BaseDir: filepath.Join(repoRoot, ".claude", "skills"), Format: "directory", Mode: "copy"},
		{Label: "OpenCode", CLI: "opencode", BaseDir: filepath.Join(repoRoot, ".opencode", "commands"), Format: "flat", Mode: "copy"},
		{Label: "Codex", CLI: "codex", BaseDir: filepath.Join(repoRoot, ".agents", "skills"), Format: "directory", Mode: "copy"},
		{Label: "Gemini", CLI: "gemini", BaseDir: filepath.Join(repoRoot, ".gemini", "skills"), Format: "directory", Mode: "copy"},
	}

	agentsFound := 0
	var errs []error
	for _, agent := range agents {
		if !agentAvailable(agent.CLI) {
			fmt.Printf("Skipping %s: %s not found in PATH\n", agent.Label, agent.CLI)
			continue
		}

		if err := syncAgentSkills(agent, skills, skillsDir); err != nil {
			errs = append(errs, err)
		}
		cleanStaleSkills(agent.BaseDir, agent.Format, skills, repoRoot)
		agentsFound++
	}

	if agentsFound == 0 {
		fmt.Println("Warning: No agent CLIs found in PATH. Skills were not deployed to any agent.")
	}

	return errors.Join(errs...)
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
// Write/symlink/read failures are counted per-skill (never as created/
// repaired), surfaced on stderr, and returned as a non-nil error so Sync
// exits non-zero — deployed skills are the instructions agents execute, and
// a silent deploy failure means agents run stale skill versions.
func syncAgentSkills(agent agentConfig, skills []string, skillsDir string) error {
	if err := os.MkdirAll(agent.BaseDir, 0755); err != nil {
		return fmt.Errorf("%s: cannot create %s: %w", agent.Label, agent.BaseDir, err)
	}

	created, repaired, ok, failed := 0, 0, 0, 0
	var firstErr error
	fail := func(skill string, err error) {
		failed++
		if firstErr == nil {
			firstErr = err
		}
		fmt.Fprintf(os.Stderr, "WARN: %s: failed to deploy %s: %v\n", agent.Label, skill, err)
	}

	for _, skill := range skills {
		src := filepath.Join(skillsDir, skill+".md")
		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("WARN: kit/skills/%s.md missing in cache — skipping\n", skill)
			continue
		}

		var dest string
		if agent.Format == "directory" {
			if err := os.MkdirAll(filepath.Join(agent.BaseDir, skill), 0755); err != nil {
				fail(skill, err)
				continue
			}
			dest = filepath.Join(agent.BaseDir, skill, "SKILL.md")
		} else {
			dest = filepath.Join(agent.BaseDir, skill+".md")
		}

		if agent.Mode == "copy" {
			srcData, err := os.ReadFile(src)
			if err != nil {
				fail(skill, fmt.Errorf("cannot read source: %w", err))
				continue
			}

			if info, err := os.Lstat(dest); err == nil && info.Mode()&os.ModeSymlink == 0 {
				// File exists and is not a symlink — compare content
				destData, _ := os.ReadFile(dest)
				if string(srcData) == string(destData) {
					ok++
				} else if err := os.WriteFile(dest, srcData, 0644); err != nil {
					fail(skill, err)
				} else {
					repaired++
				}
			} else if _, err := os.Lstat(dest); err == nil {
				// Exists as symlink or something else — replace. A failed
				// remove must not fall through to WriteFile: writing through
				// a leftover symlink would modify its target (e.g. the kit
				// cache) instead of replacing the entry.
				if err := os.Remove(dest); err != nil {
					fail(skill, fmt.Errorf("cannot replace existing entry: %w", err))
				} else if err := os.WriteFile(dest, srcData, 0644); err != nil {
					fail(skill, err)
				} else {
					repaired++
				}
			} else if err := os.WriteFile(dest, srcData, 0644); err != nil {
				fail(skill, err)
			} else {
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
					if err := os.Symlink(target, dest); err != nil {
						fail(skill, err)
					} else {
						repaired++
					}
				}
			} else if _, err := os.Lstat(dest); err == nil {
				os.Remove(dest)
				if err := os.Symlink(target, dest); err != nil {
					fail(skill, err)
				} else {
					repaired++
				}
			} else if err := os.Symlink(target, dest); err != nil {
				fail(skill, err)
			} else {
				created++
			}
		}
	}

	total := created + repaired + ok
	if failed > 0 {
		fmt.Printf("%-12s %d/%d (created %d, repaired %d, already valid %d, failed %d)\n",
			agent.Label+":", total, len(skills), created, repaired, ok, failed)
		return fmt.Errorf("%s: %d skill deployment(s) failed (first: %w)", agent.Label, failed, firstErr)
	}
	fmt.Printf("%-12s %d/%d (created %d, repaired %d, already valid %d)\n",
		agent.Label+":", total, len(skills), created, repaired, ok)
	return nil
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
