package main

import (
	"encoding/json"
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
	var cli, provider, role, dir string
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "open --cli <name>",
		Short: "Spawn a CLI's interactive command in a tmux pane (no dispatch record)",
		Long: "Resolve a CLI's interactive_command — project config per-field merged over\n" +
			"the built-in CLI table, exactly as `fab agent` resolves it — substitute the\n" +
			"{model}/{effort} fills via the standard precedence with the CLI pinned at\n" +
			"invocation time, and spawn the composed command in a tmux pane. The new pane's id\n" +
			"is printed on stdout (plus the socket label when --server is non-default).\n\n" +
			"When run from inside a tmux pane ($TMUX_PANE set, no --server) the spawn is a\n" +
			"plain split of the current window; otherwise it is a new unnamed window. No\n" +
			"worker-column placement, no title, no dispatch record, no .fab-dispatch/ state —\n" +
			"this is the CLI-generic primitive; `fab dispatch open` is the record-keeping\n" +
			"binding for pipeline workers.\n\n" +
			"--json emits a single {\"pane\",\"cli\",\"server\"} object (plus a deprecated\n" +
			"duplicate \"provider\" key for one minor release) instead of the text\n" +
			"report (server is null for the default socket). Probe the pane with\n" +
			"`fab pane ready`, then hand it a prompt with `fab pane deliver`.\n\n" +
			"Exit codes: 0 spawned; 1 resolution/operational failure (unknown CLI, no\n" +
			"interactive_command); 3 tmux failure (unreachable server, failed spawn).",
		Example: `  fab pane open --cli kimi
  fab pane open --cli claude --role fast -c /path/to/repo
  fab pane open --cli kimi --server work --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, cliSet, err := resolveCLIFlag(cmd, cli, provider)
			if err != nil {
				return err
			}
			if !cliSet {
				return fmt.Errorf(`required flag(s) "cli" not set`)
			}
			return runPaneOpen(cmd, cli, role, dir, jsonFlag)
		},
	}
	addCLIFlag(cmd, &cli, &provider, "CLI whose interactive_command is spawned (required)")
	cmd.Flags().StringVar(&role, "role", agent.RoleDefault, "role whose {model}/{effort} fills apply")
	cmd.Flags().StringVarP(&dir, "cwd", "c", "", "working directory for the new pane (default: current directory)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Emit structured JSON output")
	// --cli is required, but not through cobra's MarkFlagRequired: that would
	// reject a script still passing the hidden deprecated --provider alias.
	return cmd
}

// runPaneOpen resolves the CLI's interactive command and spawns it in a
// plain tmux pane.
//
// Config resolution matches the pane family's no-FabRoot-guard posture: when a
// fab root is resolvable its project config is loaded (per-field merge over the
// built-in table); outside a repo the empty config still yields the built-in
// CLI table, so `fab pane open --cli kimi` works from a scratch dir.
//
// The resolution/validation errors are RunE errors (exit 1); the two tmux
// failures — an unreachable server and a refused spawn — exit 3 via the
// pane-family in-handler os.Exit pattern, because nothing was spawned and the
// caller should not confuse them with a resolution problem.
func runPaneOpen(cmd *cobra.Command, cli, role, dir string, jsonFlag bool) error {
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

	// The CLI is pinned at invocation time (the top rung of the fill
	// precedence); --role selects whose model/effort fills apply.
	profile, err := agent.ResolveRoleWith(cfg, role, agent.Overrides{CLI: cli, CLISet: true})
	if err != nil {
		return err
	}
	prov, known := agent.ResolveCLI(cfg, cli)
	if !known {
		return unknownCLIError(cfg, cli)
	}
	if prov.InteractiveCommand == "" {
		return fmt.Errorf("CLI %q has no interactive_command; configure clis.%s.interactive_command", cli, cli)
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
	paneID, err := pane.OpenPlainPane(server, split, absDir, spawn.WithInteractiveShell(resolvedCmd))
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		os.Exit(3)
	}

	out := cmd.OutOrStdout()
	if jsonFlag {
		return json.NewEncoder(out).Encode(paneOpenJSON{
			Pane:     paneID,
			CLI:      cli,
			Provider: cli,
			Server:   toNullable(server),
		})
	}
	fmt.Fprintf(out, "opened pane %s (CLI %s)\n", paneID, cli)
	if server != "" {
		fmt.Fprintf(out, "server: %s\n", server)
	}
	return nil
}

// paneOpenJSON is the --json output shape for `fab pane open`. Server is
// nullable via the shared toNullable contract: the default socket reports
// null rather than "".
type paneOpenJSON struct {
	Pane string `json:"pane"`
	CLI  string `json:"cli"`
	// Provider duplicates CLI for the provider → cli deprecation window
	// (261120-f2ly); removed with the other pre-2.31 spellings.
	Provider string  `json:"provider"`
	Server   *string `json:"server"`
}
