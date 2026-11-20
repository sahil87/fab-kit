package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// The CLI-tool selector flag was `--provider` until kit 2.31.0 (261120-f2ly). It is
// now `--cli`; `--provider` stays registered as a HIDDEN deprecated alias for one
// minor release so scripts and released tooling keep working. Every command that
// takes the flag registers the pair through addCLIFlag and resolves it through
// resolveCLIFlag, so the alias semantics live in one place.

// addCLIFlag registers `--cli` (canonical, shown in help) and the hidden
// deprecated `--provider` alias on cmd. The two flags bind separate variables —
// resolveCLIFlag merges them — so cobra's Flag.Changed still tells apart "not
// supplied" from an explicit empty value for each spelling.
func addCLIFlag(cmd *cobra.Command, cli, provider *string, usage string) {
	cmd.Flags().StringVar(cli, "cli", "", usage)
	cmd.Flags().StringVar(provider, "provider", "", "deprecated alias of --cli")
	_ = cmd.Flags().MarkHidden("provider")
}

// resolveCLIFlag merges --cli and its deprecated --provider alias into one value
// and a "supplied" bit (cobra's Flag.Changed semantics, so an explicit empty
// `--cli=` is still a supplied flag). Using --provider prints the deprecation line
// on stderr; supplying both with different values is a usage error, the same value
// is accepted.
func resolveCLIFlag(cmd *cobra.Command, cli, provider string) (value string, set bool, err error) {
	cliSet := cmd.Flags().Changed("cli")
	providerSet := cmd.Flags().Changed("provider")
	if providerSet {
		fmt.Fprintln(cmd.ErrOrStderr(), "fab: warning: --provider is deprecated, use --cli")
	}
	switch {
	case cliSet && providerSet && cli != provider:
		return "", false, fmt.Errorf("--cli %q and --provider %q conflict — --provider is a deprecated alias of --cli; pass --cli only", cli, provider)
	case providerSet && !cliSet:
		return provider, true, nil
	default:
		return cli, cliSet, nil
	}
}
