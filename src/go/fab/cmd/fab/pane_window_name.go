package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

func paneWindowNameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "window-name",
		Short: "Window-name prefix operations",
		Long:  "Window-name prefix operations: ensure-prefix, replace-prefix",
	}
	cmd.AddCommand(
		paneWindowNameEnsurePrefixCmd(),
		paneWindowNameReplacePrefixCmd(),
	)
	return cmd
}

func paneWindowNameEnsurePrefixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ensure-prefix <pane> <char>",
		Short: "Idempotently prepend <char> to the tmux window name",
		Args:  cobra.ExactArgs(2),
		RunE:  runEnsurePrefix,
	}
	cmd.Flags().Bool("json", false, "Emit structured JSON output")
	return cmd
}

func paneWindowNameReplacePrefixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replace-prefix <pane> <from> <to>",
		Short: "Atomically replace a literal prefix <from> with <to> on the tmux window name",
		Args:  cobra.ExactArgs(3),
		RunE:  runReplacePrefix,
	}
	cmd.Flags().Bool("json", false, "Emit structured JSON output")
	return cmd
}

func runEnsurePrefix(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	char := args[1]
	server, _ := cmd.Flags().GetString("server")
	asJSON, _ := cmd.Flags().GetBool("json")

	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "tmux not running")
		os.Exit(1)
	}

	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(2)
	}

	name, err := pane.ReadWindowName(paneID, server)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	if strings.HasPrefix(name, char) {
		emitResult(cmd.OutOrStdout(), paneID, name, name, "noop", asJSON)
		return nil
	}

	newName := char + name
	if err := exec.Command("tmux", renameArgs(server, paneID, newName)...).Run(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}
	emitResult(cmd.OutOrStdout(), paneID, name, newName, "renamed", asJSON)
	return nil
}

func runReplacePrefix(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	from := args[1]
	to := args[2]
	server, _ := cmd.Flags().GetString("server")
	asJSON, _ := cmd.Flags().GetBool("json")

	if from == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: <from> must be non-empty")
		os.Exit(3)
	}

	if os.Getenv("TMUX") == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "tmux not running")
		os.Exit(1)
	}

	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(2)
	}

	name, err := pane.ReadWindowName(paneID, server)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	if !strings.HasPrefix(name, from) {
		emitResult(cmd.OutOrStdout(), paneID, name, name, "noop", asJSON)
		return nil
	}

	newName := to + strings.TrimPrefix(name, from)
	if err := exec.Command("tmux", renameArgs(server, paneID, newName)...).Run(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}
	emitResult(cmd.OutOrStdout(), paneID, name, newName, "renamed", asJSON)
	return nil
}

// renameArgs is the testable argv builder for `tmux rename-window`.
// When server is non-empty, the argv is prepended with `-L <server>`.
func renameArgs(server, paneID, newName string) []string {
	return pane.WithServer(server, "rename-window", "-t", paneID, newName)
}

type windowNameResult struct {
	Pane   string `json:"pane"`
	Old    string `json:"old"`
	New    string `json:"new"`
	Action string `json:"action"`
}

// emitResult writes the operation result to w in either plain or JSON form.
// Plain form: `renamed: <old> -> <new>\n` on a rename, empty on a no-op.
// JSON form: a single `{"pane","old","new","action"}` object per call.
func emitResult(w io.Writer, paneID, oldName, newName, action string, asJSON bool) {
	if asJSON {
		result := windowNameResult{Pane: paneID, Old: oldName, New: newName, Action: action}
		b, _ := json.Marshal(result)
		fmt.Fprintln(w, string(b))
		return
	}
	if action == "renamed" {
		fmt.Fprintf(w, "renamed: %s -> %s\n", oldName, newName)
	}
}
