package main

import (
	"fmt"
	"os"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// resolveAgentCmd implements `fab resolve-agent <stage|role>` — a pure query (no
// side effects) in the same family as `fab resolve`. Its argument is either a
// pipeline STAGE (mapped through the fixed stage→role mapping) or a ROLE name
// directly — the latter serves `fab agent` and the operator launcher's role-level
// resolution. A name shared by a stage and a role (review, hydrate) is a fixed
// point (that stage maps to that same-named role), so role-first dispatch resolves
// it identically either way. It resolves the role to a concrete
// {provider, model, effort} through the fill precedence implemented in
// internal/agent (an agent.profiles.<role> override, the role's depth knob
// agent.session/agent.workers, then the resolved provider's own per-role fills),
// and echoes the result VERBATIM — no validation against any provider's accepted
// set (provider neutrality).
//
// Output (byte-stable for the same config): a `model=` line always, then optional
// `effort=`, `provider=`, and `dispatch=` lines:
//
//	model=<id>
//	effort=<level>
//	provider=<name>
//	dispatch=<command>
//
// The effort line is omitted when the resolved profile has no effort; the provider
// line is omitted when it has no provider. An empty model emits an empty `model=`
// line, signaling "inherit the session/orchestrator model". The dispatch line is
// emitted when the resolved provider carries a dispatch_command (the CLI-dispatch
// opt-in) — or, with `dispatch.watchable: true` and the orchestrator inside tmux,
// from a session_command-only provider (the watchable pane opt-in; see
// dispatchLineFor). Its absence signals native Agent-tool dispatch, and a HEADLESS
// dispatch never falls back to a session command. Non-zero exit only on a real
// error: malformed/unreadable config, or an unknown stage/role name.
//
// The optional `--alias` flag is the Claude-Code Agent-tool adapter: when set,
// the resolved model is mapped to its short alias (opus/sonnet/haiku/fable) on the
// `model=` line via agent.ModelAlias, since the Agent tool's `model` enum rejects
// full IDs. Default (absent) is the full ID. The `effort=`/`provider=` lines are
// unaffected by `--alias`; empty/non-Claude models pass through verbatim. The
// `dispatch=` line ALWAYS embeds the FULL model ID even under `--alias` — CLI
// dispatch never aliases (an external CLI's --model flag takes a full ID); the
// {model}/{effort} placeholders are substituted via internal/spawn.WithProfile
// (reused, not reimplemented) using the role's own resolved model/effort.
//
// INVOCATION-TIME OVERRIDES: `--provider <name>`, `--model <id>`, and
// `--effort <level>` are the top rung of the fill precedence
//
//	provider:      --provider  >  agent.profiles.<role>.provider  >  depth knob  >  claude
//	model/effort:  flag  >  agent.profiles.<role>.<field>
//	               >  providers.<p>.profiles.<role>.<field>
//	               >  providers.<p>.profiles.default.<field>  >  empty
//
// and they ride the SAME single resolution call (agent.ResolveRoleWith — the
// precedence lives in internal/agent, not here). `--provider` swaps the provider and
// re-derives the `dispatch=` line from the NAMED provider's dispatch_command — so
// the emitted `dispatch=` presence can differ from the stage's unoverridden one.
// That is a QUERY RESULT, not an adapter move: this command only reports what the
// named provider's dispatch_command is. `fab dispatch start` takes no override flags
// and re-resolves the stage from config itself, so relocating a stage between native
// Agent-tool dispatch and CLI dispatch takes a config override (a depth knob, or
// agent.profiles.<role>.provider), never an invocation flag. A swap re-derives
// model/effort from the NEW provider's own per-role fills — an explicit
// agent.profiles.<role>.model still wins, since a pin the user wrote is not
// inheritance.
//
// `--model`/`--effort` are valid WITHOUT `--provider` here — a within-role
// override of the profile this pure query would otherwise print. This is a
// deliberate, documented asymmetry with `fab agent`, where they remain usage
// errors without `--provider`: `fab agent` is a session launcher with two mutually
// exclusive addressing modes, where a bare `--model` would invent an undocumented
// role-override surface.
//
// An unknown `--provider` name is a LOOKUP failure (non-zero exit naming the
// resolvable providers) — not validation of any command's content
// (document-don't-validate stands: resolved strings still pass through verbatim).
// The error is byte-identical to `fab agent`'s because both call the shared
// unknownProviderError helper (agent.go).
func resolveAgentCmd() *cobra.Command {
	var alias bool
	var provider string
	var model string
	var effort string
	cmd := &cobra.Command{
		Use:   "resolve-agent <stage|role>",
		Short: "Resolve a pipeline stage (or agent role) to its {provider, model, effort} agent profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			cfg, err := config.Load(fabRoot)
			if err != nil {
				return err
			}

			role, err := agent.RoleForName(args[0])
			if err != nil {
				return err
			}

			// Overrides key on whether the flag was SUPPLIED (cobra's
			// Flag.Changed), not on value emptiness — so `--model=` explicitly
			// clears the role's model (emitting the inherit signal) instead of
			// being silently ignored, and `--provider=` resolves an empty provider
			// (the lookup failure below) rather than falling back to the depth knob.
			providerSet := cmd.Flags().Changed("provider")
			profile, err := agent.ResolveRoleWith(cfg, role, agent.Overrides{
				Provider:    provider,
				ProviderSet: providerSet,
				Model:       model,
				ModelSet:    cmd.Flags().Changed("model"),
				Effort:      effort,
				EffortSet:   cmd.Flags().Changed("effort"),
			})
			if err != nil {
				return err
			}

			prov, known := agent.ResolveProvider(cfg, profile.Provider)
			if providerSet && !known {
				// A supplied --provider that resolves to nothing is a lookup
				// failure naming the resolvable set — shared verbatim with
				// `fab agent` via unknownProviderError.
				return unknownProviderError(cfg, profile.Provider)
			}
			// The dispatch= command ALWAYS embeds the full resolved model ID (CLI
			// dispatch never aliases), so substitute placeholders from the full
			// model BEFORE --alias overwrites profile.Model with the short alias.
			// $TMUX is read HERE (the cobra layer) so dispatchLineFor stays pure —
			// the internal/dispatch.SelectMode precedent.
			var dispatchLine string
			if cmdTemplate := dispatchLineFor(prov, known, cfg.GetDispatchWatchable(), os.Getenv("TMUX")); cmdTemplate != "" {
				dispatchLine = spawn.WithProfile(cmdTemplate, profile.Model, profile.Effort)
			}

			if alias {
				profile.Model = agent.ModelAlias(profile.Model)
			}

			fmt.Fprint(cmd.OutOrStdout(), formatAgentProfile(profile, dispatchLine))
			return nil
		},
	}
	cmd.Flags().BoolVar(&alias, "alias", false, "emit the Claude-Code short model alias (opus/sonnet/haiku/fable) on the model= line instead of the full ID (Agent-tool adapter)")
	cmd.Flags().StringVar(&provider, "provider", "", "override the resolved provider (re-derives dispatch= from that provider; unoverridden model/effort refill from its per-role fills, then empty)")
	cmd.Flags().StringVar(&model, "model", "", "override the resolved model (valid without --provider — a within-role override)")
	cmd.Flags().StringVar(&effort, "effort", "", "override the resolved effort (valid without --provider — a within-role override)")
	return cmd
}

// dispatchLineFor returns the UNSUBSTITUTED provider command the `dispatch=` line
// should carry, or "" to omit the line (native Agent-tool dispatch). It is a PURE
// function — the caller reads $TMUX and the config and passes them in, the
// internal/dispatch.SelectMode precedent — so the whole emission matrix is
// table-testable.
//
// Two triggers, in precedence order:
//
//  1. The resolved provider carries a dispatch_command ⇒ emit it. Unchanged
//     behavior, and it wins outright: a provider that opted into CLI dispatch is
//     dispatched by its own headless command regardless of tmux or the opt-in.
//  2. WATCHABLE PANE OPT-IN: the provider has NO dispatch_command, but
//     `dispatch.watchable: true` AND $TMUX is set AND the provider carries a
//     session_command ⇒ emit the session_command.
//
// Why trigger 2 is sound even though it emits a SESSION command on a line named
// `dispatch=`: the skills' dispatch seam branches on the line's PRESENCE and never
// executes its value — `fab dispatch start` re-resolves the stage from config
// itself. Inside tmux, start's auto ladder selects PANE mode, which composes the
// provider's session_command (a session_command-only provider dispatches fine
// under pane mode — shipped l9ng/zxe0 behavior). So the emitted value is
// informational and matches what start will actually run.
//
// Why $TMUX and not a tmux probe: this is a DEFAULTING signal, exactly as in
// SelectMode — "would a pane be visible to the caller?". With $TMUX unset the line
// is omitted, so the stage stays on NATIVE Agent-tool dispatch (never headless CLI:
// headless remains gated on a real dispatch_command). tmux presence is what decides
// pane-vs-native, which is the whole point of the opt-in.
//
// Known edge (documented, not solved): if tmux dies between this resolve and
// `fab dispatch start`, start's auto ladder soft-falls-back to headless and then
// errors on the missing dispatch_command. Rare, self-explaining at the CLI.
func dispatchLineFor(prov config.ProviderConfig, known, watchable bool, tmuxEnv string) string {
	if !known {
		return ""
	}
	if prov.DispatchCommand != "" {
		return prov.DispatchCommand
	}
	if watchable && tmuxEnv != "" && prov.SessionCommand != "" {
		return prov.SessionCommand
	}
	return ""
}

// formatAgentProfile renders a resolved profile as the byte-stable stdout
// contract: a `model=<id>` line always, an `effort=<level>` line only when the
// effort is non-empty, a `provider=<name>` line only when the provider is
// non-empty, and a `dispatch=<command>` line only when dispatchLine is non-empty.
// An empty model emits an empty `model=` line (the "inherit" signal). dispatchLine
// is the ALREADY-substituted command (placeholders resolved via internal/spawn) —
// the caller passes "" to omit the line (native Agent-tool dispatch). Extracted so
// the omit-when-empty branches are unit-testable without needing a config whose
// RESOLVED effort/provider/dispatch_command is empty.
func formatAgentProfile(p agent.Profile, dispatchLine string) string {
	out := fmt.Sprintf("model=%s\n", p.Model)
	if p.Effort != "" {
		out += fmt.Sprintf("effort=%s\n", p.Effort)
	}
	if p.Provider != "" {
		out += fmt.Sprintf("provider=%s\n", p.Provider)
	}
	if dispatchLine != "" {
		out += fmt.Sprintf("dispatch=%s\n", dispatchLine)
	}
	return out
}
