package main

import (
	"encoding/json"
	"sort"

	"github.com/spf13/cobra"
)

// HelpDoc is the top-level shll.ai "command reference" contract document.
// Field order is significant — it determines JSON key order, which must match
// the frozen reference sample at sahil87/shll.ai:help/wt.json.
//
// Per the toolkit help-dump standard, the envelope is exactly
// {tool, version, schema_version, root} — no captured_at. The capture timestamp
// is owned by shll.ai (a tool cannot know its own capture time); the puller
// stamps it after capture.
type HelpDoc struct {
	Tool          string `json:"tool"`           // literal "fab" (the user-facing binary, not "fab-kit")
	Version       string `json:"version"`        // from main.version (ldflags); never hardcoded
	SchemaVersion int    `json:"schema_version"` // literal 1
	Root          Node   `json:"root"`
}

// Node mirrors a single cobra command in the dumped tree. The Commands slice is
// always non-nil so leaves serialize as [] (matching the reference), never null.
type Node struct {
	Name     string `json:"name"`     // cmd.Name()
	Path     string `json:"path"`     // cmd.CommandPath() — e.g. "fab change new"
	Short    string `json:"short"`    // cmd.Short
	Usage    string `json:"usage"`    // cmd.UseLine()
	Text     string `json:"text"`     // cmd.UsageString() — raw -h body, byte-for-byte
	Commands []Node `json:"commands"` // recursive; [] for a leaf (never null)
}

// helpDumpCmd is a hidden, CI/build-time-only command that serializes the live
// cobra command tree of the rich `fab` CLI to the frozen shll.ai contract JSON
// on stdout. It is Hidden so it stays out of `fab --help` and out of its own
// dumped tree (buildNode filters Hidden commands).
func helpDumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "help-dump",
		Short:  "Emit the CLI help tree as shll.ai contract JSON (CI/build-time)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			doc := dumpDoc(cmd.Root(), version)
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false) // preserve raw -h bytes; do not escape <, >, &
			return enc.Encode(doc)
		},
	}
}

// dumpDoc builds the full contract document from the assembled root command and
// the injected version string.
func dumpDoc(root *cobra.Command, version string) HelpDoc {
	return HelpDoc{
		Tool:          "fab",
		Version:       version,
		SchemaVersion: 1,
		Root:          buildNode(root),
	}
}

// buildNode recursively serializes a cobra command and its surviving children.
// At every level it drops cobra's auto-generated "completion" and "help"
// subcommands and any Hidden command (which self-excludes help-dump), then sorts
// the survivors by Name() for byte-stable output.
func buildNode(cmd *cobra.Command) Node {
	node := Node{
		Name:     cmd.Name(),
		Path:     cmd.CommandPath(),
		Short:    cmd.Short,
		Usage:    cmd.UseLine(),
		Text:     cmd.UsageString(),
		Commands: []Node{}, // non-nil so leaves serialize as [], not null
	}

	children := make([]*cobra.Command, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Name() == "completion" || child.Name() == "help" || child.Hidden {
			continue
		}
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})

	for _, child := range children {
		node.Commands = append(node.Commands, buildNode(child))
	}

	return node
}
