package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func paneProcessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process <pane>",
		Short: "Detect the process tree running in a tmux pane",
		Args:  cobra.ExactArgs(1),
		RunE:  runPaneProcess,
	}
	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

// ProcessNode represents a single process in the tree.
type ProcessNode struct {
	PID            int           `json:"pid"`
	PPID           int           `json:"ppid"`
	Comm           string        `json:"comm"`
	Cmdline        string        `json:"cmdline"`
	Classification string        `json:"classification"`
	Children       []ProcessNode `json:"children"`
}

// processJSON is the JSON output structure for pane process.
type processJSON struct {
	Pane      string        `json:"pane"`
	PanePID   int           `json:"pane_pid"`
	Processes []ProcessNode `json:"processes"`
	HasAgent  bool          `json:"has_agent"`
}

// ClassifyProcess classifies a process by its comm name.
func ClassifyProcess(comm string) string {
	lower := strings.ToLower(comm)
	switch {
	case lower == "claude" || lower == "claude-code":
		return "agent"
	case lower == "node":
		return "node"
	case lower == "git" || lower == "gh":
		return "git"
	default:
		return "other"
	}
}

// hasAgentInTree checks recursively if any process is classified as "agent".
func hasAgentInTree(nodes []ProcessNode) bool {
	for _, n := range nodes {
		if n.Classification == "agent" {
			return true
		}
		if hasAgentInTree(n.Children) {
			return true
		}
	}
	return false
}

func runPaneProcess(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Validate pane exists
	if err := pane.ValidatePane(paneID); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(1)
	}

	// Get pane PID
	pid, err := pane.GetPanePID(paneID)
	if err != nil {
		return fmt.Errorf("get pane PID: %w", err)
	}

	// Discover process tree (platform-specific)
	tree, err := discoverProcessTree(pid)
	if err != nil {
		return fmt.Errorf("process discovery: %w", err)
	}

	hasAgent := hasAgentInTree(tree)

	if jsonFlag {
		out := processJSON{
			Pane:      paneID,
			PanePID:   pid,
			Processes: tree,
			HasAgent:  hasAgent,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Human-readable output
	printProcessTree(cmd, paneID, pid, tree, hasAgent)
	return nil
}

// printProcessTree prints a human-readable process tree.
func printProcessTree(cmd *cobra.Command, paneID string, panePID int, nodes []ProcessNode, hasAgent bool) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Pane %s (PID %d)\n", paneID, panePID)

	for _, n := range nodes {
		printNode(w, n, "")
	}

	if hasAgent {
		fmt.Fprintln(w, "\nAgent process detected.")
	}
}

// printNode prints a single process node with indentation.
func printNode(w io.Writer, node ProcessNode, indent string) {
	classification := ""
	if node.Classification != "other" {
		classification = fmt.Sprintf(" [%s]", node.Classification)
	}
	fmt.Fprintf(w, "%s%d %s%s\n", indent, node.PID, node.Comm, classification)
	for _, child := range node.Children {
		printNode(w, child, indent+"  ")
	}
}
