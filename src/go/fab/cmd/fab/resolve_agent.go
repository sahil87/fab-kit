package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// resolveAgentCmd implements `fab resolve-agent <stage|role>` — a pure query (no
// side effects) in the same family as `fab resolve`. Its argument is either a
// pipeline STAGE (mapped through the fixed stage→role mapping) or a ROLE name
// directly — the latter serves `fab agent` and the operator launcher's role-level
// resolution. A name shared by a stage and a role (review, hydrate) is a fixed
// point (that stage maps to that same-named role), so role-first dispatch resolves
// it identically either way. It resolves the role to a concrete
// {cli, model, effort} through the fill precedence implemented in
// internal/agent (an agent.profiles.<role> override, the role's depth knob
// agent.session/agent.workers, then the resolved CLI's own per-role fills),
// and echoes the result VERBATIM — no validation against any CLI's accepted
// set (CLI neutrality).
//
// Output (byte-stable for the same config): a `model=` line always, then optional
// `effort=`, `provider=`, and `dispatch=` lines:
//
//	model=<id>
//	effort=<level>
//	provider=<name>
//	dispatch=<command>
//
// The effort line is omitted when the resolved profile has no effort; the CLI
// line is omitted when it has no CLI. An empty model emits an empty `model=`
// line, signaling "inherit the session/orchestrator model". The dispatch line is
// derived from dispatch.mode, CLI capabilities, and $TMUX: native omits it,
// pane emits interactive_command, and headless emits headless_command. A CLI with
// no reachable rung errors actionably. Command fields never substitute for one
// another. Other non-zero exits cover malformed/unreadable config and unknown
// stage/role names.
//
// The optional `--alias` flag is the Claude-Code Agent-tool adapter: when set,
// the resolved model is mapped to its short alias (opus/sonnet/haiku/fable) on the
// `model=` line via agent.ModelAlias, since the Agent tool's `model` enum rejects
// full IDs. Default (absent) is the resolved model verbatim — an alias for the
// built-in claude fills, a full ID when one is configured. The `effort=`/`provider=`
// lines are unaffected by `--alias`; empty/non-Claude models pass through verbatim.
// The `dispatch=` line ALWAYS embeds the resolved model verbatim, even under
// `--alias` — CLI dispatch never aliases (an external CLI's model flag may take only
// concrete IDs); the
// {model}/{effort} placeholders are substituted via internal/spawn.WithProfile
// (reused, not reimplemented) using the role's own resolved model/effort.
//
// INVOCATION-TIME OVERRIDES: `--cli <name>`, `--model <id>`, and
// `--effort <level>` are the top rung of the fill precedence
//
//	cli:      --cli  >  agent.profiles.<role>.cli  >  depth knob  >  claude
//	model/effort:  flag  >  agent.profiles.<role>.<field>
//	               >  clis.<p>.profiles.<role>.<field>
//	               >  clis.<p>.profiles.default.<field>  >  empty
//
// and they ride the SAME single resolution call (agent.ResolveRoleWith — the
// precedence lives in internal/agent, not here). `--cli` swaps the CLI and
// re-derives the `dispatch=` line from dispatch.mode plus the NAMED CLI's
// capabilities, so the resolved rung and emitted-line presence can differ from the
// stage's unoverridden result. That is a QUERY RESULT, not an adapter move.
// `fab dispatch start` takes no override flags
// and re-resolves the stage from config itself, so relocating a stage between native
// Agent-tool dispatch and CLI dispatch takes a config override (a depth knob, or
// agent.profiles.<role>.cli), never an invocation flag. A swap re-derives
// model/effort from the NEW CLI's own per-role fills — an explicit
// agent.profiles.<role>.model still wins, since a pin the user wrote is not
// inheritance.
//
// `--model`/`--effort` are valid WITHOUT `--cli` here — a within-role
// override of the profile this pure query would otherwise print.
// `fab agent` accepts the same bare overrides (a final post-refill layer
// on every addressing form), so the two commands agree on override legality.
//
// An unknown `--cli` name is a LOOKUP failure (non-zero exit naming the
// resolvable CLIs) — not validation of any command's content
// (document-don't-validate stands: resolved strings still pass through verbatim).
// The error is byte-identical to `fab agent`'s because both call the shared
// unknownCLIError helper (agent.go).
func resolveAgentCmd() *cobra.Command {
	var alias bool
	var cli, provider string
	var model string
	var effort string
	cmd := &cobra.Command{
		Use:   "resolve-agent <stage|role>",
		Short: "Resolve a pipeline stage (or agent role) to its {cli, model, effort} agent profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Config-only command: it resolves a profile, never change state, so
			// with no fab/ project it degrades to env > system > built-in
			// defaults instead of failing closed (see loadRepoConfig in agent.go
			// and cmd/fab/skill.md's config-free-commands rule).
			var cfg *config.Config
			if fabRoot, err := resolve.FabRoot(); err == nil {
				cfg, err = config.Load(fabRoot)
				if err != nil {
					return err
				}
			} else if !errors.Is(err, resolve.ErrNoFabRoot) {
				// Broken environment (unreadable cwd), not "no project".
				return err
			} else {
				cfg = config.LoadNoProject()
			}

			role, err := agent.RoleForName(args[0])
			if err != nil {
				return err
			}

			// Overrides key on whether the flag was SUPPLIED (cobra's
			// Flag.Changed), not on value emptiness — so `--model=` explicitly
			// clears the role's model (emitting the inherit signal) instead of
			// being silently ignored, and `--cli=` resolves an empty CLI
			// (the lookup failure below) rather than falling back to the depth knob.
			cli, cliSet, err := resolveCLIFlag(cmd, cli, provider)
			if err != nil {
				return err
			}
			kind := "stage"
			if agent.IsRoleName(args[0]) {
				kind = "role"
			}
			resolution, err := composeAgentResolution(cfg, resolutionRequest{
				Selector: args[0],
				Kind:     kind,
				Role:     role,
				Overrides: agent.Overrides{
					CLI:       cli,
					CLISet:    cliSet,
					Model:     model,
					ModelSet:  cmd.Flags().Changed("model"),
					Effort:    effort,
					EffortSet: cmd.Flags().Changed("effort"),
				},
				Dispatch: true,
				TmuxEnv:  os.Getenv("TMUX"),
			})
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), resolution.Lines(alias))
			return nil
		},
	}
	cmd.Flags().BoolVar(&alias, "alias", false, "emit the Claude-Code short model alias (opus/sonnet/haiku/fable) on the model= line instead of a versioned ID (Agent-tool adapter)")
	addCLIFlag(cmd, &cli, &provider, "override the resolved CLI (re-derives dispatch= from that CLI; unoverridden model/effort refill from its per-role fills, then empty)")
	cmd.Flags().StringVar(&model, "model", "", "override the resolved model (valid without --cli — a within-role override)")
	cmd.Flags().StringVar(&effort, "effort", "", "override the resolved effort (valid without --cli — a within-role override)")
	return cmd
}

// noDispatchCapabilityError reports that no rung at or below the configured
// dispatch.mode preference is reachable for this CLI. Selection descends
// pane → native → headless and never ascends, so the remedies name only the
// rungs the preference can actually reach — suggesting interactive_command under a
// `headless` preference would point at a capability the ladder would skip. When
// native was skipped for an alias-less model, the error names that cause and
// offers an alias, empty model, or ToolDefaultModel instead of suggesting native capability again.
func noDispatchCapabilityError(profile agent.Profile, native dispatch.NativeAvailability, preference string, cause error) error {
	cli := profile.CLI
	var remedies []string
	if preference == string(dispatch.ModePane) {
		remedies = append(remedies, fmt.Sprintf("clis.%s.interactive_command for pane", cli))
	}
	if native != dispatch.NativeNoAlias && (preference == string(dispatch.ModePane) || preference == string(dispatch.ModeNative)) {
		remedies = append(remedies, fmt.Sprintf("clis.%s.native for native", cli))
	}
	remedies = append(remedies, fmt.Sprintf("clis.%s.headless_command for headless", cli))
	if native == dispatch.NativeNoAlias && preference != string(dispatch.ModeHeadless) {
		remedies = append(remedies, "set the model to an alias (opus/sonnet/haiku/fable), leave it empty to inherit, or set it to "+agent.ToolDefaultModel)
		return fmt.Errorf("CLI %q has no dispatch capability at or below dispatch.mode %q: model %q has no native Agent-tool alias; configure %s: %w",
			cli, preference, profile.Model, joinRemedies(remedies), cause)
	}
	return fmt.Errorf("CLI %q has no dispatch capability at or below dispatch.mode %q; configure %s: %w",
		cli, preference, joinRemedies(remedies), cause)
}

// joinRemedies renders a remedy list as prose: "A", "A or B", "A, B, or C".
func joinRemedies(remedies []string) string {
	switch len(remedies) {
	case 1:
		return remedies[0]
	case 2:
		return remedies[0] + " or " + remedies[1]
	default:
		return strings.Join(remedies[:len(remedies)-1], ", ") + ", or " + remedies[len(remedies)-1]
	}
}
