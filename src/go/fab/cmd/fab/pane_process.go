package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/shellparse"
	"github.com/spf13/cobra"
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

// agentBinaryNames returns the process basenames that identify a live agent.
// The provider table is the merged resolvable set (built-ins plus project
// overrides); interactive_command is the relevant grammar because operator
// panes use the interactive spawn shape. Shell names are deliberately excluded
// so a configured shell wrapper can never become positive liveness evidence.
func agentBinaryNames(cfg *config.Config) map[string]bool {
	names := map[string]bool{
		"claude":      true,
		"claude-code": true,
	}
	for _, providerName := range agent.ProviderNames(cfg) {
		provider, ok := agent.ResolveProvider(cfg, providerName)
		if !ok {
			continue
		}
		name := interactiveCommandBinary(provider.InteractiveCommand)
		if name != "" && !pane.IsShellCommand(name) {
			names[name] = true
		}
	}
	return names
}

// interactiveCommandBinary returns the basename of the quote-aware leading
// command word, skipping only true POSIX NAME=value assignment prefixes.
// Provider command strings are document-don't-validate data, so an empty or
// assignment-only string simply contributes no name.
func interactiveCommandBinary(command string) string {
	return processBinaryName(shellparse.LeadingCommand(command))
}

// processBinaryName normalizes comm/cmdline tokens for exact set matching.
func processBinaryName(token string) string {
	token = strings.Trim(token, "\"'")
	if token == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(token))
}

// classifyProcess classifies a process from its comm name first, then uses a
// bounded cmdline fallback. Only the first two cmdline token basenames are
// considered so prompt text later in argv can never become liveness evidence.
func classifyProcess(comm, cmdline string, agents map[string]bool) string {
	lower := processBinaryName(comm)
	if agents[lower] {
		return "agent"
	}
	fields := strings.Fields(cmdline)
	if len(fields) > 2 {
		fields = fields[:2]
	}
	for _, token := range fields {
		if agents[processBinaryName(token)] {
			return "agent"
		}
	}
	switch {
	case lower == "node":
		return "node"
	case lower == "git" || lower == "gh":
		return "git"
	default:
		return "other"
	}
}

// loadAgentBinaryNames resolves project config when available and otherwise
// degrades to the built-in provider table. A config read failure is best-effort
// and silent: built-in names still provide useful classification.
func loadAgentBinaryNames() map[string]bool {
	var cfg *config.Config
	if fabRoot, err := resolve.FabRoot(); err == nil {
		cfg, _ = config.Load(fabRoot)
	}
	return agentBinaryNames(cfg)
}

// parsePSCmdlines parses `ps -axo pid=,args=` output into a PID→args map.
// Each line is a (right-aligned) numeric PID followed by the full command
// line; pid is numeric-first and the remainder is args, so the parse is
// robust against spaces inside the command line. Lines whose first field is
// not a PID are skipped. Lives in this un-tagged file (consumed by the
// darwin process discovery) so the parsing is unit-testable on every
// platform.
func parsePSCmdlines(out string) map[int]string {
	m := make(map[int]string)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, " ", 2)
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		args := ""
		if len(fields) == 2 {
			args = strings.TrimSpace(fields[1])
		}
		m[pid] = args
	}
	return m
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

// paneAgentAlive reports positive live-agent evidence for a pane. Any pane-PID
// or process-discovery error returns false so callers fail toward emitting
// agent_exited rather than suppressing it without evidence.
func paneAgentAlive(paneID string, agents map[string]bool) bool {
	pid, err := pane.GetPanePID(paneID, "")
	if err != nil {
		return false
	}
	tree, err := discoverProcessTree(pid, agents)
	if err != nil {
		return false
	}
	return hasAgentInTree(tree)
}

func runPaneProcess(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	jsonFlag, _ := cmd.Flags().GetBool("json")
	server, _ := cmd.Flags().GetString("server")

	// Validate pane exists — the pane-family in-handler exit scheme (2 = pane
	// missing, 3 = any other tmux failure), so operator scripts branch on cause
	// uniformly across the family.
	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	// Get pane PID
	pid, err := pane.GetPanePID(paneID, server)
	if err != nil {
		return fmt.Errorf("get pane PID: %w", err)
	}

	// Discover process tree (platform-specific) with the same provider-aware
	// classifier the operator tick uses for liveness confirmation.
	tree, err := discoverProcessTree(pid, loadAgentBinaryNames())
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
