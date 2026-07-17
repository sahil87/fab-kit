package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

// newRootCmd assembles the fab-go root command with all subcommands
// registered. Extracted from main() so tests can walk the live command tree
// (e.g. the router-allowlist collision test sources top-level names from the
// help-dump tree of this root).
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "fab",
		Short:         "Fab workflow engine — single binary replacement for kit shell scripts",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		resolveCmd(),
		resolveAgentCmd(),
		configCmd(),
		logCmd(),
		statusCmd(),
		preflightCmd(),
		changeCmd(),
		scoreCmd(),
		paneCmd(),
		dispatchCmd(),
		fabHelpCmd(),
		operatorCmd(),
		agentCmd(),
		batchCmd(),
		kitPathCmd(),
		impactCmd(),
		prMetaCmd(),
		memoryIndexCmd(),
		shellInitCmd(),
		helpDumpCmd(),
	)

	return root
}

// markRunReached wraps every RunE in the assembled command tree so a shared
// flag records whether the resolved command's run phase actually began. This is
// the seam that classifies usage errors: cobra surfaces flag-parse, arg-count,
// unknown-subcommand, and flag-group errors during execute() BEFORE any RunE
// runs, whereas an operational error originates from inside a RunE. Setting the
// flag at the last moment before the real handler — rather than in a
// PersistentPreRunE — is deliberate: PersistentPreRunE runs before cobra's
// ValidateFlagGroups, so a mutually-exclusive flags-group conflict would be
// misclassified as operational. Classification thus rides on execution phase,
// never on message-string matching (mirroring paneValidationExitCode's
// error-value discipline).
func markRunReached(cmd *cobra.Command, reached *bool) {
	for _, sub := range cmd.Commands() {
		markRunReached(sub, reached)
	}
	if orig := cmd.RunE; orig != nil {
		cmd.RunE = func(c *cobra.Command, args []string) error {
			*reached = true
			return orig(c, args)
		}
	}
}

// run executes the fab CLI and returns the process exit code, so the mapping is
// unit-testable (mirrors pane_exitcode_test.go's classifier-test shape).
//
// Exit-code convention (toolkit principle №4): 0 = success, 1 = operational
// failure, 2 = usage error. A usage error is any malformed invocation caught at
// parse/validation time (unknown/malformed flag, arg-count violation, unknown
// subcommand, mutually-exclusive flags-group conflict) — all surface before the
// resolved command's RunE begins, so `reached` is still false. Operational
// errors originate from inside a RunE (`reached` is true) and stay exit 1.
//
// Domain-specific in-handler exit codes (pane 2/3, memory-index 0/1/2) call
// os.Exit directly from within their RunE and so bypass this mapping entirely —
// their codes are preserved unchanged (the no-renumbering coexistence rule).
func run(args []string, outW, errW io.Writer) int {
	root := newRootCmd()
	root.SetArgs(args)
	root.SetOut(outW)

	// Inject cobra's auto-generated help/completion commands BEFORE wrapping, so
	// their RunE handlers participate in the reached-run-phase signal. Execute()
	// would otherwise add these AFTER markRunReached ran (see ExecuteC), leaving
	// the completion subcommands' RunE unwrapped — a returned operational error
	// (e.g. a write failure in `completion bash`) would then be misclassified as
	// a usage error (exit 2) since `reached` never flips. Both calls are
	// idempotent, so Execute()'s later re-invocation is a no-op. (The `help` and
	// `__complete` commands use Run, not RunE, so they carry no returned error to
	// classify; initializing help here is harmless and keeps the seam uniform.)
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	reached := false
	markRunReached(root, &reached)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(errW, "ERROR: %s\n", err)
		if reached {
			return 1
		}
		return 2
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
