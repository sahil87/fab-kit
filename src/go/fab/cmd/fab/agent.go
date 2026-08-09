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
// `fab agent [role] [--provider <name> [--model <id>] [--effort <level>]] [--workers <provider>] [--print] [--repo <path>]`
// — launch (or print) the resolved agent session command in the current shell. It
// replaces `fab spawn-command`, with a semantic upgrade: the printed/exec'd command
// is profile-resolved (model/effort substituted), not placeholder-stripped.
//
// Two mutually exclusive ADDRESSING MODES compose the command:
//
//   - Role-addressed (the `[role]` positional, `default` when omitted; any of the
//     six role names accepted): resolves the role profile — whose provider comes
//     from an agent.profiles override, else the role's depth knob — then composes
//     providers.<profile.provider>.interactive_command with the role's {model}/{effort}.
//   - Provider-addressed (`--provider <name>`): BYPASSES role resolution and looks
//     up providers.<name> directly (project config per-field merged over fab-kit's
//     built-in table, exactly as the role path's provider lookup does), composing
//     its interactive_command with the `--model`/`--effort` values. Omitted values are
//     empty and follow spawn.WithProfile's documented empty-value rule (template
//     mode drops the placeholder's token plus a preceding `-`-flag; append mode
//     omits the flag) — so `fab agent --provider codex --print` composes a bare
//     `codex` invocation and the CLI's own default model applies.
//
// `--model`/`--effort` are scoped to the provider mode: on the role path the
// profile IS the role's (resolved through the fill precedence), so a bare `--model`
// would either invent an undocumented role-override surface or be silently ignored
// — both worse than a usage error.
//
// Both guards (and the mode selection itself) key on cobra's Flag.Changed —
// whether the flag was SUPPLIED — rather than on its value being non-empty. So
// `fab agent doing --provider=` is still the mutual-exclusion error and
// `fab agent --model= --print` is still the requires-`--provider` error; neither
// falls through to the role path.
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
//   - `--workers <provider>`: sets FAB_AGENT_WORKERS=<provider> in the exec
//     environment, REPLACING any entry inherited from the parent rather than
//     appending a second one (duplicate resolution is unspecified). It does not
//     alter `--print` output or validate the value.
var execAgent = syscall.Exec

func agentCmd() *cobra.Command {
	var printOnly bool
	var repo string
	var provider string
	var model string
	var effort string
	cmd := &cobra.Command{
		Use:   "agent [role]",
		Short: "Launch (or --print) the resolved agent session command in the current shell",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			role := ""
			if len(args) == 1 {
				role = args[0]
			}

			// The guards key on whether a flag was SUPPLIED (cobra's
			// Flag.Changed), not on whether its value is non-empty: an
			// explicitly-empty `--provider=` / `--model=` is still a supplied
			// flag, and testing emptiness would let those invocations fall
			// silently through to the role path instead of erroring.
			providerSet := cmd.Flags().Changed("provider")
			profileFlagSet := cmd.Flags().Changed("model") || cmd.Flags().Changed("effort")

			// Mutual exclusion is hand-checked rather than declared via cobra's
			// MarkFlagsMutuallyExclusive because that helper relates two FLAGS —
			// the role here is a positional, which it cannot reference.
			if role != "" && providerSet {
				return fmt.Errorf("the [role] positional and --provider are mutually exclusive: %q names a role (whose provider comes from the role's override or depth knob), --provider addresses a provider directly", role)
			}
			if !providerSet {
				if profileFlagSet {
					return fmt.Errorf("--model/--effort require --provider (on the role path the model and effort come from the resolved role profile)")
				}
				if role == "" {
					role = agent.RoleDefault
				}
			}

			return runAgent(cmd, role, provider, providerSet, model, effort, printOnly, repo)
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "print the fully-resolved command instead of executing it")
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	cmd.Flags().StringVar(&provider, "provider", "", "address a provider directly (bypasses role resolution); mutually exclusive with the [role] positional")
	cmd.Flags().StringVar(&model, "model", "", "model id for the --provider form (empty: the provider command's model token is dropped, so its CLI default applies)")
	cmd.Flags().StringVar(&effort, "effort", "", "reasoning effort for the --provider form (empty: the effort token is dropped)")
	cmd.Flags().String("workers", "", "set FAB_AGENT_WORKERS for the launched agent session")
	return cmd
}

// runAgent composes the session command for whichever addressing mode the caller
// selected and either prints it or execs it. `providerSet` (cobra's
// Flag.Changed for --provider) — NOT the emptiness of `provider` — selects the
// mode, so an explicitly-empty `--provider=` takes the provider path and fails
// its lookup rather than silently resolving the default role.
func runAgent(cmd *cobra.Command, role, provider string, providerSet bool, model, effort string, printOnly bool, repo string) error {
	cfg, err := loadRepoConfig(repo)
	if err != nil {
		return err
	}

	providerName, profileModel, profileEffort := provider, model, effort
	if !providerSet {
		profile, err := agent.ResolveRole(cfg, role)
		if err != nil {
			return err
		}
		providerName, profileModel, profileEffort = profile.Provider, profile.Model, profile.Effort
	}

	prov, known := agent.ResolveProvider(cfg, providerName)
	if providerSet && !known {
		return unknownProviderError(cfg, providerName)
	}
	if !known || prov.InteractiveCommand == "" {
		if providerSet {
			return fmt.Errorf("provider %q has no interactive_command; configure providers.%s.interactive_command", providerName, providerName)
		}
		return fmt.Errorf("role %q resolves to provider %q, which has no interactive_command; configure providers.%s.interactive_command",
			role, providerName, providerName)
	}

	resolvedCmd := spawn.WithProfile(prov.InteractiveCommand, profileModel, profileEffort)

	if printOnly {
		fmt.Fprintln(cmd.OutOrStdout(), resolvedCmd)
		return nil
	}

	// Exec the composed command in the current shell so shell expansions expand
	// at invocation time and the agent CLI replaces this process. No TTY guard:
	// the agent CLI surfaces its own error when stdin is not a terminal.
	env := os.Environ()
	if workers, set := workersOverride(cmd); set {
		env = envWithWorkers(env, workers)
	}
	return execAgent("/bin/sh", []string{"/bin/sh", "-c", resolvedCmd}, env)
}

// unknownProviderError is the shared `--provider <name>` LOOKUP failure for the
// two commands that accept the flag (`fab agent`'s provider-addressed mode and
// `fab resolve-agent`'s invocation-time override). Both need byte-identical
// phrasing — the error is a user-facing contract documented once in
// `_cli-fab.md` — so it lives here rather than being restated at each call site.
//
// It is a lookup failure, NOT validation of the resolved command's content
// (document-don't-validate stands). The `providers.<name>` hint names a config key
// path, so an explicitly-empty `--provider=` would otherwise render the malformed
// `providers.` — a `<name>` placeholder is substituted so the suggested path is
// always a valid key.
//
// Callers gate on cobra's Flag.Changed (the flag was SUPPLIED) together with a
// failed agent.ResolveProvider lookup; this helper only formats the error.
func unknownProviderError(cfg *config.Config, name string) error {
	hintName := name
	if hintName == "" {
		hintName = "<name>"
	}
	return fmt.Errorf("unknown provider %q (available: %s); configure it under providers.%s in fab/project/config.yaml",
		name, strings.Join(agent.ProviderNames(cfg), ", "), hintName)
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
