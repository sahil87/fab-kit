package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// agentCmd implements `fab agent [tier] [--print] [--repo <path>]` — launch (or
// print) the resolved agent session command in the current shell. It replaces
// `fab spawn-command`, with a semantic upgrade: the printed/exec'd command is
// profile-resolved (model/effort substituted), not placeholder-stripped.
//
//   - Resolves the tier profile (default when the positional [tier] is omitted;
//     any of the six role-tier names accepted), then composes
//     providers.<profile.provider>.session_command with {model}/{effort}
//     substituted (or Claude-style flags appended for a non-templated command),
//     via internal/spawn.WithProfile.
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
	cmd := &cobra.Command{
		Use:   "agent [tier]",
		Short: "Launch (or --print) the resolved agent session command in the current shell",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tier := agent.TierDefault
			if len(args) == 1 {
				tier = args[0]
			}
			return runAgent(cmd, tier, printOnly, repo)
		},
	}
	cmd.Flags().BoolVar(&printOnly, "print", false, "print the fully-resolved command instead of executing it")
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	return cmd
}

func runAgent(cmd *cobra.Command, tier string, printOnly bool, repo string) error {
	cfg, err := loadRepoConfig(repo)
	if err != nil {
		return err
	}

	profile, err := agent.ResolveTier(cfg, tier)
	if err != nil {
		return err
	}

	prov, ok := agent.ResolveProvider(cfg, profile.Provider)
	if !ok || prov.SessionCommand == "" {
		return fmt.Errorf("tier %q resolves to provider %q, which has no session_command; configure providers.%s.session_command",
			tier, profile.Provider, profile.Provider)
	}

	resolvedCmd := spawn.WithProfile(prov.SessionCommand, profile.Model, profile.Effort)

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
