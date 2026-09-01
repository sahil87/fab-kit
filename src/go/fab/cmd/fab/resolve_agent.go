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
// derived from dispatch.mode, provider capabilities, and $TMUX: native omits it,
// pane emits interactive_command, and headless emits headless_command. A provider with
// no reachable rung errors actionably. Command fields never substitute for one
// another. Other non-zero exits cover malformed/unreadable config and unknown
// stage/role names.
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
// re-derives the `dispatch=` line from dispatch.mode plus the NAMED provider's
// capabilities, so the resolved rung and emitted-line presence can differ from the
// stage's unoverridden result. That is a QUERY RESULT, not an adapter move.
// `fab dispatch start` takes no override flags
// and re-resolves the stage from config itself, so relocating a stage between native
// Agent-tool dispatch and CLI dispatch takes a config override (a depth knob, or
// agent.profiles.<role>.provider), never an invocation flag. A swap re-derives
// model/effort from the NEW provider's own per-role fills — an explicit
// agent.profiles.<role>.model still wins, since a pin the user wrote is not
// inheritance.
//
// `--model`/`--effort` are valid WITHOUT `--provider` here — a within-role
// override of the profile this pure query would otherwise print.
// `fab agent` accepts the same bare overrides (a final post-refill layer
// on every addressing form), so the two commands agree on override legality.
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
			cmdTemplate, err := dispatchLineFor(prov, cfg.GetDispatchMode(), os.Getenv("TMUX"))
			if err != nil {
				return noDispatchCapabilityError(profile.Provider, cfg.GetDispatchMode(), err)
			}
			var dispatchLine string
			if cmdTemplate != "" {
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

// dispatchLineFor returns the unsubstituted command for the resolved automatic
// rung, or "" for native mode. It shares internal/dispatch.SelectMode with the
// launcher; $TMUX presence is the resolver seam's pane-availability signal.
func dispatchLineFor(prov config.ProviderConfig, preference, tmuxEnv string) (string, error) {
	tmux := dispatch.TmuxAbsent
	if tmuxEnv != "" {
		tmux = dispatch.TmuxAvailable
	}
	mode, _, err := dispatch.SelectMode(false, false, false, false, preference,
		prov.Native, prov.InteractiveCommand != "", prov.HeadlessCommand != "", tmux)
	if err != nil {
		return "", err
	}
	switch mode {
	case dispatch.ModeNative:
		return "", nil
	case dispatch.ModePane:
		return prov.InteractiveCommand, nil
	case dispatch.ModeHeadless:
		return prov.HeadlessCommand, nil
	default:
		return "", fmt.Errorf("unexpected dispatch mode %q", mode)
	}
}

// noDispatchCapabilityError reports that no rung at or below the configured
// dispatch.mode preference is reachable for this provider. Selection descends
// pane → native → headless and never ascends, so the remedies name only the
// rungs the preference can actually reach — suggesting interactive_command under a
// `headless` preference would point at a capability the ladder would skip.
func noDispatchCapabilityError(provider, preference string, cause error) error {
	var remedies []string
	if preference == string(dispatch.ModePane) {
		remedies = append(remedies, fmt.Sprintf("providers.%s.interactive_command for pane", provider))
	}
	if preference == string(dispatch.ModePane) || preference == string(dispatch.ModeNative) {
		remedies = append(remedies, fmt.Sprintf("providers.%s.native for native", provider))
	}
	remedies = append(remedies, fmt.Sprintf("providers.%s.headless_command for headless", provider))
	return fmt.Errorf("provider %q has no dispatch capability at or below dispatch.mode %q; configure %s: %w",
		provider, preference, joinRemedies(remedies), cause)
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

// formatAgentProfile renders a resolved profile as the byte-stable stdout
// contract: a `model=<id>` line always, an `effort=<level>` line only when the
// effort is non-empty, a `provider=<name>` line only when the provider is
// non-empty, and a `dispatch=<command>` line only when dispatchLine is non-empty.
// An empty model emits an empty `model=` line (the "inherit" signal). dispatchLine
// is the ALREADY-substituted command (placeholders resolved via internal/spawn) —
// the caller passes "" to omit the line (native Agent-tool dispatch). Extracted so
// the omit-when-empty branches are unit-testable without needing a config whose
// RESOLVED effort/provider/headless_command is empty.
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
