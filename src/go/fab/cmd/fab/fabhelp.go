package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/frontmatter"
	"github.com/sahil87/fab-kit/src/go/fab/internal/kitpath"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

// skillGroup defines a display group and its order.
type skillGroup struct {
	name    string
	members []string // skill names in this group (populated during scan)
}

// skillToGroupMap maps skill names to their display group.
var skillToGroupMap = map[string]string{
	"fab-new":             "Start & Navigate",
	"fab-draft":           "Start & Navigate",
	"fab-switch":          "Start & Navigate",
	"fab-status":          "Start & Navigate",
	"fab-discuss":         "Start & Navigate",
	"fab-continue":        "Planning",
	"fab-ff":              "Planning",
	"fab-fff":             "Planning",
	"fab-clarify":         "Planning",
	"fab-archive":         "Completion",
	"git-pr":              "Completion",
	"docs-hydrate-specs":  "Maintenance",
	"docs-reorg-specs":    "Maintenance",
	"docs-reorg-memory":   "Maintenance",
	"fab-setup":           "Setup",
	"fab-help":            "Setup",
	"docs-hydrate-memory": "Setup",
}

// groupOrder defines the display order of groups.
var groupOrder = []string{
	"Start & Navigate",
	"Planning",
	"Completion",
	"Maintenance",
	"Setup",
	"Batch Operations",
}

func fabHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fab-help",
		Short: "Show fab workflow overview and available commands",
		Args:  cobra.NoArgs,
		RunE:  runFabHelp,
	}
}

func runFabHelp(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()
	_, err := resolve.FabRoot()
	if err != nil {
		return err
	}

	kitDir, err := kitpath.KitDir()
	if err != nil {
		return fmt.Errorf("cannot resolve kit directory: %w", err)
	}
	version := readKitVersion(kitDir)

	// Scan skills
	skills := scanSkills(filepath.Join(kitDir, "skills"))

	// Compute max display name length for alignment
	maxLen := computeMaxNameLen(skills)

	// Also account for batch commands and hardcoded entries
	batchEntries := getBatchEntries()
	for _, e := range batchEntries {
		if len(e.display) > maxLen {
			maxLen = len(e.display)
		}
	}
	hardcoded := "fab sync"
	if len(hardcoded) > maxLen {
		maxLen = len(hardcoded)
	}

	padTo := maxLen + 2

	// Render output
	fmt.Fprintf(w, "Fab Kit v%s \u2014 Specification-Driven Development\n", version)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "WORKFLOW")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  /fab-new \u2500\u2192 /fab-continue (or /fab-ff) \u2500\u2192 /fab-archive")
	fmt.Fprintln(w, "               \u2195 /fab-clarify")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Planning stages: spec \u2192 tasks")
	fmt.Fprintln(w, "  Execution stages: apply \u2192 review \u2192 hydrate")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "COMMANDS")

	// Track which skills have been rendered
	rendered := make(map[string]bool)

	for _, group := range groupOrder {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s\n", group)

		// Render skills in this group
		for _, s := range skills {
			if skillToGroupMap[s.name] == group {
				formatEntry(w, "/"+s.name, s.desc, padTo)
				rendered[s.name] = true
			}
		}

		// Render batch commands in this group
		if group == "Batch Operations" {
			for _, e := range batchEntries {
				formatEntry(w, e.display, e.desc, padTo)
			}
		}

		// Render hardcoded non-skill entries for Setup group
		if group == "Setup" {
			formatEntry(w, "fab sync", "Repair directories and agents (no LLM needed)", padTo)
		}
	}

	// Render "Other" group for unmapped skills
	var other []skillEntry
	for _, s := range skills {
		if !rendered[s.name] {
			other = append(other, s)
		}
	}
	if len(other) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  Other\n")
		for _, s := range other {
			formatEntry(w, "/"+s.name, s.desc, padTo)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "TYPICAL FLOW")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Quick change:  /fab-new \u2192 /fab-ff \u2192 /fab-archive")
	fmt.Fprintln(w, "  Careful change: /fab-new \u2192 /fab-continue (repeat) \u2192 /fab-archive")
	fmt.Fprintln(w, "  Maintain docs:  /docs-hydrate-memory, /docs-hydrate-specs, /docs-reorg-specs, /docs-reorg-memory")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "PACKAGES")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    wt create, wt list, wt open, wt delete, wt init          Git worktree management")
	fmt.Fprintln(w, "    idea                                                     Per-repo backlog (fab/backlog.md)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "    Run <command> help for details.")

	return nil
}

// skillEntry holds a discovered skill name and description.
type skillEntry struct {
	name string
	desc string
}

// batchEntry holds a batch command display name and description.
type batchEntry struct {
	display string
	desc    string
}

// scanSkills reads skill files from the given directory and returns entries
// sorted alphabetically, excluding partials (_*) and internal-* skills.
func scanSkills(skillsDir string) []skillEntry {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil
	}

	var skills []skillEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		base := strings.TrimSuffix(name, ".md")

		// Exclude partials and internal skills
		if strings.HasPrefix(base, "_") || strings.HasPrefix(base, "internal-") {
			continue
		}

		path := filepath.Join(skillsDir, name)
		if !frontmatter.HasFrontmatter(path) {
			continue
		}

		skillName := frontmatter.Field(path, "name")
		skillDesc := frontmatter.Field(path, "description")
		if skillName == "" || skillDesc == "" {
			continue
		}

		skills = append(skills, skillEntry{name: skillName, desc: skillDesc})
	}

	sort.Slice(skills, func(i, j int) bool {
		return skills[i].name < skills[j].name
	})

	return skills
}

// getBatchEntries returns the batch command entries from cobra metadata.
func getBatchEntries() []batchEntry {
	var entries []batchEntry
	for _, sub := range batchCmd().Commands() {
		entries = append(entries, batchEntry{
			display: "fab batch " + sub.Name(),
			desc:    sub.Short,
		})
	}
	return entries
}

// readKitVersion reads the VERSION file from the kit directory.
func readKitVersion(kitDir string) string {
	data, err := os.ReadFile(filepath.Join(kitDir, "VERSION"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// computeMaxNameLen finds the longest display name across all skills.
func computeMaxNameLen(skills []skillEntry) int {
	maxLen := 0
	for _, s := range skills {
		display := "/" + s.name
		if len(display) > maxLen {
			maxLen = len(display)
		}
	}
	return maxLen
}

// formatEntry prints a formatted help entry with aligned columns.
func formatEntry(w interface{ Write([]byte) (int, error) }, display, desc string, padTo int) {
	spaces := padTo - len(display)
	if spaces < 2 {
		spaces = 2
	}
	fmt.Fprintf(w, "    %s%s%s\n", display, strings.Repeat(" ", spaces), desc)
}
