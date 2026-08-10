package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

func paneDeliverCmd() *cobra.Command {
	var promptFile, text string
	cmd := &cobra.Command{
		Use:   "deliver <pane> (--prompt-file <path> | --text <string>)",
		Short: "Type a prompt into a tmux pane, verifying every step",
		Long: "Type a one-line prompt into a tmux pane and VERIFY that it landed: readiness\n" +
			"probe, clear the input line, type the payload, check that it echoed, press\n" +
			"Enter, confirm the pane reacted. Any failed check costs one attempt; there is\n" +
			"exactly one retry, and a second failure reports the pane's screen instead of\n" +
			"retrying again.\n\n" +
			"Verification is the point: a payload that never echoes or an Enter that changes\n" +
			"nothing is the silent-drop failure fire-and-forget sending cannot see.\n\n" +
			"--prompt-file types the pointer line `Read <path> and execute it.` (dispatch\n" +
			"parity) after checking the file exists — the path is typed as supplied, so make\n" +
			"it meaningful from the pane's own cwd. --text types its argument literally.\n" +
			"Exactly one of the two is required.\n\n" +
			"Exit codes: 0 delivered; 1 operational failure (missing prompt file, delivery\n" +
			"that never verified); 2 pane missing; 3 other tmux failure.",
		Example: `  fab pane deliver %12 --text "echo hi"
  fab pane deliver %12 --prompt-file .fab-dispatch/b91h/apply-prompt.md
  fab pane deliver %3 --prompt-file prompt.md --server work`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneDeliver(cmd, args[0], promptFile, text)
		},
	}
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "type the pointer line naming this prompt file (Read <path> and execute it.)")
	cmd.Flags().StringVar(&text, "text", "", "type this text literally")
	cmd.MarkFlagsMutuallyExclusive("prompt-file", "text")
	cmd.MarkFlagsOneRequired("prompt-file", "text")
	return cmd
}

// runPaneDeliver resolves the payload and runs the verified send-keys
// choreography against the named pane — the same gate `fab dispatch deliver`
// binds over, addressed by pane id with no dispatch record and no completion
// signals to stash.
//
// The prompt-file existence check precedes any typing: a pointer at a file
// that is not there would type cleanly, verify cleanly, and leave the pane
// reading nothing — the silent failure this whole choreography exists to
// prevent.
func runPaneDeliver(cmd *cobra.Command, paneID, promptFile, text string) error {
	server, _ := cmd.Flags().GetString("server")

	if err := pane.ValidatePane(paneID, server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(paneValidationExitCode(err))
	}

	payload, source := text, "(text)"
	if promptFile != "" {
		if _, err := os.Stat(promptFile); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("no prompt at %s — nothing to deliver; point --prompt-file at a file that exists", promptFile)
			}
			return err
		}
		payload = pane.PointerPrompt(promptFile)
		source = fmt.Sprintf("(prompt %s)", promptFile)
	}

	warnings, snippet, err := pane.NewGate(server).Deliver(paneID, payload)
	for _, w := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", w)
	}
	if err != nil {
		if snippet != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "--- pane %s, last %d lines ---\n%s",
				paneID, pane.SnippetLines, snippet)
		}
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "delivered %s %s\n", paneID, source)
	return nil
}
