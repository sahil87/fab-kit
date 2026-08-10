package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

func paneOpenCmd() *cobra.Command {
	var provider, role, dir string
	cmd := &cobra.Command{
		Use:   "open --provider <name>",
		Short: "Spawn a provider's interactive command in a tmux pane (no dispatch record)",
		Long: "Resolve a provider's interactive_command — project config per-field merged over\n" +
			"the built-in provider table, exactly as `fab agent` resolves it — substitute the\n" +
			"{model}/{effort} fills via the standard precedence with the provider pinned at\n" +
			"invocation time, and spawn the composed command in a tmux pane. The new pane's id\n" +
			"is printed on stdout (plus the socket label when --server is non-default).\n\n" +
			"When run from inside a tmux pane ($TMUX_PANE set, no --server) the spawn is a\n" +
			"plain split of the current window; otherwise it is a new unnamed window. No\n" +
			"worker-column placement, no title, no dispatch record, no .fab-dispatch/ state —\n" +
			"this is the provider-generic primitive; `fab dispatch open` is the record-keeping\n" +
			"binding for pipeline workers.\n\n" +
			"Probe the pane with `fab pane ready`, then hand it a prompt with\n" +
			"`fab pane deliver`.\n\n" +
			"Exit codes: 0 spawned; 1 resolution/operational failure (unknown provider, no\n" +
			"interactive_command); 3 tmux failure (unreachable server, failed spawn).",
		Example: `  fab pane open --provider kimi
  fab pane open --provider claude --role fast -c /path/to/repo
  fab pane open --provider kimi --server work`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPaneOpen(cmd, provider, role, dir)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider whose interactive_command is spawned (required)")
	cmd.Flags().StringVar(&role, "role", agent.RoleDefault, "role whose {model}/{effort} fills apply")
	cmd.Flags().StringVarP(&dir, "cwd", "c", "", "working directory for the new pane (default: current directory)")
	_ = cmd.MarkFlagRequired("provider")
	return cmd
}

// runPaneOpen resolves the provider's interactive command and spawns it in a
// plain tmux pane.
//
// Config resolution matches the pane family's no-FabRoot-guard posture: when a
// fab root is resolvable its project config is loaded (per-field merge over the
// built-in table); outside a repo the empty config still yields the built-in
// provider table, so `fab pane open --provider kimi` works from a scratch dir.
//
// The resolution/validation errors are RunE errors (exit 1); the two tmux
// failures — an unreachable server and a refused spawn — exit 3 via the
// pane-family in-handler os.Exit pattern, because nothing was spawned and the
// caller should not confuse them with a resolution problem.
func runPaneOpen(cmd *cobra.Command, provider, role, dir string) error {
	server, _ := cmd.Flags().GetString("server")
	serverSet := cmd.Flags().Changed("server")

	cfg := &config.Config{}
	if fabRoot, err := resolve.FabRoot(); err == nil {
		loaded, err := config.Load(fabRoot)
		if err != nil {
			return err
		}
		cfg = loaded
	}

	// The provider is pinned at invocation time (the top rung of the fill
	// precedence); --role selects whose model/effort fills apply.
	profile, err := agent.ResolveRoleWith(cfg, role, agent.Overrides{Provider: provider, ProviderSet: true})
	if err != nil {
		return err
	}
	prov, known := agent.ResolveProvider(cfg, provider)
	if !known {
		return unknownProviderError(cfg, provider)
	}
	if prov.InteractiveCommand == "" {
		return fmt.Errorf("provider %q has no interactive_command; configure providers.%s.interactive_command", provider, provider)
	}
	resolvedCmd := spawn.WithProfile(prov.InteractiveCommand, profile.Model, profile.Effort)

	// Reachability before spawn: a dead socket must never half-launch.
	if err := pane.ServerReachable(server); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		dir = wd
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// A pane id is meaningless cross-socket, so --server always opens a window;
	// only an invoker that IS a tmux pane on the target server gets the split.
	split := os.Getenv("TMUX_PANE") != "" && !serverSet
	paneID, err := pane.OpenPlainPane(server, split, absDir, resolvedCmd)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "opened pane %s (provider %s)\n", paneID, provider)
	if server != "" {
		fmt.Fprintf(out, "server: %s\n", server)
	}
	return nil
}
