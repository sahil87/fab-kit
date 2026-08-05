package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// agentCmd implements
// `fab agent [tier] [--provider <name> [--model <id>] [--effort <level>]] [--print] [--repo <path>]`
// — launch (or print) the resolved agent session command in the current shell. It
// replaces `fab spawn-command`, with a semantic upgrade: the printed/exec'd command
// is profile-resolved (model/effort substituted), not placeholder-stripped.
//
// Two mutually exclusive ADDRESSING MODES compose the command:
//
//   - Tier-addressed (the `[tier]` positional, `default` when omitted; any of the
//     six role-tier names accepted): resolves the tier profile, then composes
//     providers.<profile.provider>.session_command with the tier's {model}/{effort}.
//   - Provider-addressed (`--provider <name>`): BYPASSES tier resolution and looks
//     up providers.<name> directly (project config per-field merged over fab-kit's
//     built-in table, exactly as the tier path's provider lookup does), composing
//     its session_command with the `--model`/`--effort` values. Omitted values are
//     empty and follow spawn.WithProfile's documented empty-value rule (template
//     mode drops the placeholder's token plus a preceding `-`-flag; append mode
//     omits the flag) — so `fab agent --provider codex --print` composes a bare
//     `codex` invocation and the CLI's own default model applies.
//
// `--model`/`--effort` are scoped to the provider mode: on the tier path the
// profile IS the tier's (resolved through inheritance), so a bare `--model` would
// either invent an undocumented tier-override surface or be silently ignored —
// both worse than a usage error.
//
// Both guards (and the mode selection itself) key on cobra's Flag.Changed —
// whether the flag was SUPPLIED — rather than on its value being non-empty. So
// `fab agent doing --provider=` is still the mutual-exclusion error and
// `fab agent --model= --print` is still the requires-`--provider` error; neither
// falls through to the tier path.
//
// An unknown `--provider` name is a LOOKUP failure (non-zero exit naming the
// available providers), not validation of the command's content — resolved command
// strings still pass through verbatim (document-don't-validate). The error's
// config-key hint substitutes a `<name>` placeholder when the supplied name is
// empty (`--provider=`), so the suggested `providers.<key>` path is never malformed.
//
// Common to both modes:
//
//   - Default: EXECs the composed command in the current shell (via `sh -c`, so
//     shell expansions like $(basename "$(pwd)") expand at invocation). No TTY
//     guard — exec-and-let-the-agent-CLI-handle-it (document-don't-validate).
//   - `--print`: prints the fully-resolved command instead of executing (the
//     `fab spawn-command` replacement — profile-resolved, not stripped).
//   - `--repo <path>`: reads <path>/fab/project/config.yaml instead of the current
//     repo (the operator's fetch-another-repo's-command use case).
func agentCmd() *cobra.Command {
	var printOnly bool
	var repo string
	var provider string
	var model string
	var effort string
	cmd := &cobra.Command{
		Use:   "agent [tier]",
		Short: "Launch (or --print) the resolved agent session command in the current shell",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tier := ""
			if len(args) == 1 {
				tier = args[0]
			}

			// The guards key on whether a flag was SUPPLIED (cobra's
			// Flag.Changed), not on whether its value is non-empty: an
			// explicitly-empty `--provider=` / `--model=` is still a supplied
			// flag, and testing emptiness would let those invocations fall
			// silently through to the tier path instead of erroring.
			providerSet := cmd.Flags().Changed("provider")
			profileFlagSet := cmd.Flags().Changed("model") || cmd.Flags().Changed("effort")

			// Mutual exclusion is hand-checked rather than declared via cobra's
			// MarkFlagsMutuallyExclusive because that helper relates two FLAGS —
			// the tier here is a positional, which it cannot reference.
			if tier != "" && providerSet {
				return fmt.Errorf("the [tier] positional and --provider are mutually exclusive: %q names a role tier (whose provider comes from the tier), --provider addresses a provider directly", tier)
			}
			if !providerSet {
				if profileFlagSet {
					return fmt.Errorf("--model/--effort require --provider (on the tier path the model and effort come from the resolved tier)")
				}
				if tier == "" {
					tier = agent.TierDefault
				}
			}

			return runAgent(cmd, tier, provider, providerSet, model, effort, printOnly, repo)
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "print the fully-resolved command instead of executing it")
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	cmd.Flags().StringVar(&provider, "provider", "", "address a provider directly (bypasses tier resolution); mutually exclusive with the [tier] positional")
	cmd.Flags().StringVar(&model, "model", "", "model id for the --provider form (empty: the provider command's model token is dropped, so its CLI default applies)")
	cmd.Flags().StringVar(&effort, "effort", "", "reasoning effort for the --provider form (empty: the effort token is dropped)")
	return cmd
}

// runAgent composes the session command for whichever addressing mode the caller
// selected and either prints it or execs it. `providerSet` (cobra's
// Flag.Changed for --provider) — NOT the emptiness of `provider` — selects the
// mode, so an explicitly-empty `--provider=` takes the provider path and fails
// its lookup rather than silently resolving the default tier.
func runAgent(cmd *cobra.Command, tier, provider string, providerSet bool, model, effort string, printOnly bool, repo string) error {
	cfg, err := loadRepoConfig(repo)
	if err != nil {
		return err
	}

	providerName, profileModel, profileEffort := provider, model, effort
	if !providerSet {
		profile, err := agent.ResolveTier(cfg, tier)
		if err != nil {
			return err
		}
		providerName, profileModel, profileEffort = profile.Provider, profile.Model, profile.Effort
	}

	prov, known := agent.ResolveProvider(cfg, providerName)
	if providerSet && !known {
		// The hint names a config key path, so an explicitly-empty `--provider=`
		// would otherwise render the malformed `providers.` — substitute a
		// placeholder so the suggested path is always a valid key.
		hintName := providerName
		if hintName == "" {
			hintName = "<name>"
		}
		return fmt.Errorf("unknown provider %q (available: %s); configure it under providers.%s in fab/project/config.yaml",
			providerName, strings.Join(agent.ProviderNames(cfg), ", "), hintName)
	}
	if !known || prov.SessionCommand == "" {
		if providerSet {
			return fmt.Errorf("provider %q has no session_command; configure providers.%s.session_command", providerName, providerName)
		}
		return fmt.Errorf("tier %q resolves to provider %q, which has no session_command; configure providers.%s.session_command",
			tier, providerName, providerName)
	}

	resolvedCmd := spawn.WithProfile(prov.SessionCommand, profileModel, profileEffort)

	if printOnly {
		fmt.Fprintln(cmd.OutOrStdout(), resolvedCmd)
		return nil
	}

	// Exec the composed command in the current shell so shell expansions expand
	// at invocation time and the agent CLI replaces this process. No TTY guard:
	// the agent CLI surfaces its own error when stdin is not a terminal.
	return syscall.Exec("/bin/sh", []string{"/bin/sh", "-c", resolvedCmd}, os.Environ())
}

// loadRepoConfig loads the config from an explicit repo root (--repo) or the
// current repo's fab/ (upward search). The path-based load mirrors the former
// `fab spawn-command --repo` behavior; a missing file yields an empty config (the
// built-in provider table then supplies the default session command).
func loadRepoConfig(repo string) (*config.Config, error) {
	if repo != "" {
		return config.LoadPath(filepath.Join(repo, "fab", "project", "config.yaml"))
	}
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return nil, err
	}
	return config.Load(fabRoot)
}
