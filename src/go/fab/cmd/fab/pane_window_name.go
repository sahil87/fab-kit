package main

import (
	"bytes"
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

	if char == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error: <char> must be non-empty")
		os.Exit(3)
	}

	name, stderr, err := pane.ReadWindowName(paneID, server)
	if err != nil {
		printTmuxErr(cmd.ErrOrStderr(), stderr, err)
		os.Exit(tmuxExitCode(stderr))
	}

	if strings.HasPrefix(name, char) {
		emitResult(cmd.OutOrStdout(), paneID, name, name, "noop", asJSON)
		return nil
	}

	newName := char + name
	if stderr, err := renameWindow(server, paneID, newName); err != nil {
		printTmuxErr(cmd.ErrOrStderr(), stderr, err)
		os.Exit(tmuxExitCode(stderr))
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

	name, stderr, err := pane.ReadWindowName(paneID, server)
	if err != nil {
		printTmuxErr(cmd.ErrOrStderr(), stderr, err)
		os.Exit(tmuxExitCode(stderr))
	}

	if !strings.HasPrefix(name, from) {
		emitResult(cmd.OutOrStdout(), paneID, name, name, "noop", asJSON)
		return nil
	}

	newName := to + strings.TrimPrefix(name, from)
	if stderr, err := renameWindow(server, paneID, newName); err != nil {
		printTmuxErr(cmd.ErrOrStderr(), stderr, err)
		os.Exit(tmuxExitCode(stderr))
	}
	emitResult(cmd.OutOrStdout(), paneID, name, newName, "renamed", asJSON)
	return nil
}

// renameWindow runs `tmux rename-window -t <pane> <newName>` with captured
// stderr. Returns stderr bytes and any exec error. Callers map the error to an
// exit code via tmuxExitCode.
func renameWindow(server, paneID, newName string) ([]byte, error) {
	cmd := exec.Command("tmux", renameArgs(server, paneID, newName)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.Bytes(), err
}

// renameArgs is the testable argv builder for `tmux rename-window`.
// When server is non-empty, the argv is prepended with `-L <server>`.
func renameArgs(server, paneID, newName string) []string {
	return pane.WithServer(server, "rename-window", "-t", paneID, newName)
}

// tmuxExitCode maps tmux stderr content to the documented exit-code scheme.
// Pane-missing messages (from display-message or rename-window on a vanished
// pane) → 2; everything else — including tmux-not-running / socket errors /
// permission denied / other tmux failures — → 3.
func tmuxExitCode(stderr []byte) int {
	s := strings.ToLower(string(stderr))
	if strings.Contains(s, "can't find pane") ||
		strings.Contains(s, "no such pane") ||
		(strings.Contains(s, "pane") && strings.Contains(s, "not found")) {
		return 2
	}
	return 3
}

// printTmuxErr emits a useful error line — prefers the tmux-supplied stderr
// when present (so users see tmux's actual message), falls back to the Go exec
// error otherwise.
func printTmuxErr(w io.Writer, stderr []byte, err error) {
	msg := strings.TrimSpace(string(stderr))
	if msg != "" {
		fmt.Fprintln(w, msg)
		return
	}
	fmt.Fprintf(w, "Error: %s\n", err)
}

type windowNameResult struct {
	Pane   string `json:"pane"`
	Old    string `json:"old"`
	New    string `json:"new"`
	Action string `json:"action"`
}

// emitResult writes the operation result to w in either plain or JSON form.
// Plain form: `renamed: <old> -> <new>\n` on a rename, empty on a no-op.
// JSON form: a single `{"pane","old","new","action"}` object per call, with
// the trailing newline added by json.Encoder.
func emitResult(w io.Writer, paneID, oldName, newName, action string, asJSON bool) {
	if asJSON {
		result := windowNameResult{Pane: paneID, Old: oldName, New: newName, Action: action}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to encode JSON output: %s\n", err)
		}
		return
	}
	if action == "renamed" {
		fmt.Fprintf(w, "renamed: %s -> %s\n", oldName, newName)
	}
}
