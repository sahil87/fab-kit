package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// agentCmd implements
// `fab agent [role|stage] [--cli <name>] [--model <id>] [--effort <level>] [--headless] [-t|--template] [-o <format>] [--workers <cli>] [-p|--print] [--repo <path>] [-- <agent-args>...]`
// — launch (or print) the resolved agent session command in the current shell. It
// replaces `fab spawn-command`, with a semantic upgrade: the printed/exec'd command
// is profile-resolved (model/effort substituted), not placeholder-stripped.
//
// One SELECTOR is addressable:
//
//   - A role name or a stage name (intake/apply/review/hydrate/ship/review-pr)
//     as the positional, defaulting to the `default` role: resolves the role
//     profile (role-first via agent.RoleForName — the review/hydrate stage↔role
//     collisions are fixed points, so kind affects only the `-o yaml` kind key)
//     and composes clis.<profile.cli>.interactive_command with the
//     role's {model}/{effort}.
//   - `--cli <name>` with NO selector: BYPASSES role resolution and looks up
//     clis.<name> directly (project config per-field merged over fab-kit's
//     built-in table), composing its interactive_command with the `--model`/
//     `--effort` values. Omitted values are empty and follow spawn.WithProfile's
//     documented empty-value rule (template mode drops the placeholder's token
//     plus a preceding `-`-flag; append mode omits the flag) — so
//     `fab agent --cli codex --print` composes a bare `codex` invocation.
//   - A selector WITH `--cli`: re-resolves the role's profile from the named
//     CLI's own fills (agent.ResolveRoleWith with Overrides.CLI) —
//     unnamed flags then apply verbatim over the refilled values.
//
// `--model`/`--effort` are general (not CLI-mode-only): wherever they are
// legal they are applied as verbatim final overrides after role resolution /
// CLI refill — verbatim pass-through, no validation (document-don't-validate).
//
// Output modes, one sink per invocation:
//
//   - Default: EXECs the composed command in the current shell (via `sh -c`, so
//     shell expansions like $(basename "$(pwd)") expand at invocation). No TTY
//     guard — exec-and-let-the-agent-CLI-handle-it (document-don't-validate).
//   - `-p, --print`: prints the fully-resolved command instead of executing (the
//     `fab spawn-command` replacement — profile-resolved, not stripped).
//   - `-t, --template`: prints the selected CLI's command template
//     UNsubstituted (a tap BEFORE the fill step — `{model}`/`{effort}` placeholders
//     intact). Implies print; rejects `--model`/`--effort` with a usage error
//     (they feed a step that never runs).
//   - `-o, --output yaml`: prints the complete structured resolution, including
//     the addressing/profile/command fields, alias, template/fill mode,
//     provenance, and the selected non-native dispatch rung when present.
//   - `--headless` selects the CLI's `headless_command` instead of
//     `interactive_command`, valid only on the three print-family sinks; exec of a
//     headless command is a usage error. A CLI with no headless_command
//     hard-errors naming the config key.
//
// `--model`/`--effort` legality and every other guard still keys on cobra's
// Flag.Changed — whether the flag was SUPPLIED — rather than on its value being
// non-empty. Explicitly-empty `--cli=` / `--model=` still resolves/errors the
// same as a supplied flag; an explicitly-empty `--model=` clears the model field
// on the selector paths.
//
// An unknown `--cli` name is a LOOKUP failure (non-zero exit naming the
// available CLIs), not validation of the command's content — resolved command
// strings still pass through verbatim (document-don't-validate). The error's
// config-key hint substitutes a `<name>` placeholder when the supplied name is
// empty (`--cli=`), so the suggested `clis.<key>` path is never malformed.
//
// Common to all modes:
//
//   - `--repo <path>`: reads <path>/fab/project/config.yaml instead of the current
//     repo (the operator's fetch-another-repo's-command use case).
//   - `--preset <name>`: chooses an agent preset for THIS resolution only, as if
//     FAB_AGENT_PRESET=<name> were set (the env layer: it beats system/project
//     agent.* fields, loses to explicit FAB_AGENT_* fields; --cli/--model/--effort
//     stay on top; `none` disables a configured preset; an unknown name warns and
//     resolves as unset). Resolution only — it does NOT export FAB_AGENT_PRESET into
//     an exec'd session (unlike --workers), and flags never reach `fab dispatch
//     start`: to make a preset reach dispatched workers set agent.preset in config or
//     export FAB_AGENT_PRESET. Rejected with the selector-less `--cli` form, which
//     bypasses the role resolution a preset feeds.
//   - `--workers <cli>`: sets FAB_AGENT_WORKERS=<cli> in the exec
//     environment, REPLACING any entry inherited from the parent rather than
//     appending a second one (duplicate resolution is unspecified). It does not
//     alter `--print` output or validate the value.
var execAgent = syscall.Exec

func agentCmd() *cobra.Command {
	var printOnly bool
	var template bool
	var headless bool
	var output string
	var repo string
	var cli, provider string
	var model string
	var effort string
	var preset string
	cmd := &cobra.Command{
		Use:   "agent [role|stage] [-- <agent-args>...]",
		Short: "Launch (or --print) the resolved agent session command in the current shell",
		// Everything after `--` is passthrough for the launched agent CLI, so the
		// count of POSITIONALS cannot be validated by arity alone — only the args
		// BEFORE the dash are fab's. cobra records that boundary in
		// ArgsLenAtDash(); MaximumNArgs(1) would count the passthrough as extra
		// roles and reject `fab agent -- --resume`.
		Args: func(cmd *cobra.Command, args []string) error {
			if n := cmd.ArgsLenAtDash(); n > 1 {
				return fmt.Errorf("accepts at most 1 arg before `--`, received %d (everything after `--` is passed to the agent CLI)", n)
			} else if n < 0 && len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d (to pass arguments to the agent CLI, put them after `--`)", len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// ArgsLenAtDash is the count of args before `--`, or -1 when no `--`
			// was given. It is the ONLY reliable split: cobra strips the dash
			// itself, so the passthrough is otherwise indistinguishable from a
			// second positional.
			selector, passthrough := "", []string(nil)
			if n := cmd.ArgsLenAtDash(); n >= 0 {
				if n == 1 {
					selector = args[0]
				}
				passthrough = args[n:]
			} else if len(args) == 1 {
				selector = args[0]
			}

			cli, cliSet, err := resolveCLIFlag(cmd, cli, provider)
			if err != nil {
				return err
			}

			// A preset feeds role resolution; the bare --cli form bypasses it, so the
			// pair is rejected rather than silently ignoring the preset.
			if cmd.Flags().Changed("preset") && cliSet && selector == "" {
				return fmt.Errorf("--preset has no effect with a bare --cli (no [role|stage] selector): that form bypasses role resolution; add a selector or drop --preset")
			}

			// Template is implied print-mode; --model/--effort feed the fill step
			// that it skips, so they are rejected rather than silently ignored.
			if template && (cmd.Flags().Changed("model") || cmd.Flags().Changed("effort")) {
				return fmt.Errorf("--model/--effort have no effect with --template (substitution never runs)")
			}

			// `-o yaml` is its own print sink — combination usage errors are
			// hand-checked (cobra's MarkFlagsMutuallyExclusive pairs strings
			// indistinguishably for output vs field names).
			if cmd.Flags().Changed("output") {
				if output != "yaml" {
					return fmt.Errorf("--output accepts exactly <yaml>")
				}
				if printOnly {
					return fmt.Errorf("--output and --print are mutually exclusive (one output sink per invocation)")
				}
				if template {
					return fmt.Errorf("--output and --template are mutually exclusive (one output sink per invocation)")
				}
			}
			// --headless picks the headless command slot; exec of a headless
			// command is a usage error, so the three print sinks are the only
			// legal combinations.
			if headless && !printOnly && !template && !cmd.Flags().Changed("output") {
				return fmt.Errorf("--headless is valid only in the print-family modes (--print, -t, -o yaml); exec of a headless command is not supported")
			}

			return runAgent(cmd, selector, cli, cliSet, model, effort, printOnly, template, headless, output, repo, passthrough, preset)
		},
	}
	cmd.Flags().BoolVarP(&printOnly, "print", "p", false, "print the fully-resolved command instead of executing it")
	cmd.Flags().BoolVarP(&template, "template", "t", false, "print the selected CLI's command template un-substituted (placeholders intact); implies --print")
	cmd.Flags().BoolVar(&headless, "headless", false, "resolve the CLI's headless_command instead of interactive_command (print-family modes only)")
	cmd.Flags().StringVarP(&output, "output", "o", "", `print the resolution as a structured document; only "yaml" is accepted`)
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	addCLIFlag(cmd, &cli, &provider, "CLI override: with a [role|stage] selector, re-resolve from that CLI's fills; alone, address the CLI directly (bypasses role resolution)")
	cmd.Flags().StringVar(&model, "model", "", "final model override (empty: on the bare --cli form the CLI command's model token is dropped, so its CLI default applies)")
	cmd.Flags().StringVar(&effort, "effort", "", "final effort override (empty: on the bare --cli form the effort token is dropped)")
	cmd.Flags().String("workers", "", "set FAB_AGENT_WORKERS for the launched agent session")
	cmd.Flags().StringVar(&preset, "preset", "", "agent preset for this resolution only (as if FAB_AGENT_PRESET were set; `none` disables a configured one). "+
		"Resolution only: it is not exported into the launched session, and flags do not reach `fab dispatch start` — "+
		"to make a preset reach dispatched workers, set agent.preset in config or export FAB_AGENT_PRESET")
	cmd.AddCommand(agentModelsCmd())
	cmd.AddCommand(agentPresetsCmd())
	return cmd
}

// runAgent selects the addressing + output modes the user's flags picked and
// delegates to resolveAgentInvocation for the composition.
func runAgent(cmd *cobra.Command, selector, cli string, cliSet bool, model, effort string, printOnly, template, headless bool, output string, repo string, passthrough []string, preset string) error {
	m := agentMode{
		CLI:         cli,
		CLISet:      cliSet,
		Model:       model,
		ModelSet:    cmd.Flags().Changed("model"),
		Effort:      effort,
		EffortSet:   cmd.Flags().Changed("effort"),
		Selector:    selector,
		Print:       printOnly,
		Template:    template,
		Headless:    headless,
		Output:      output,
		Repo:        repo,
		Passthrough: passthrough,
		Preset:      preset,
	}
	return resolveAgentInvocation(cmd, m)
}

// agentMode is the one-input resolveAgentInvocation call bundle — what
// cmd/runAgent normalized out of the cobra flag set. Field comments name each
// semantic; normalize them here ONCE so resolution reads a struct instead of a
// bare field list.
type agentMode struct {
	CLI         string
	CLISet      bool // cobra's Flag.Changed for --cli (an explicitly-empty --cli= is still a supplied flag)
	Model       string
	ModelSet    bool
	Effort      string
	EffortSet   bool
	Selector    string // a role or stage name; empty = default role (unless CLISet)
	Print       bool
	Template    bool
	Headless    bool
	Output      string
	Repo        string
	Passthrough []string
	Preset      string // --preset: chosen at the env layer for this one resolution ("" = not supplied)
}

func resolveAgentInvocation(cmd *cobra.Command, m agentMode) error {
	cfg, err := loadRepoConfigPreset(m.Repo, m.Preset)
	if err != nil {
		return err
	}

	cliOnly := m.CLISet && m.Selector == ""
	selector, kind, role := m.Selector, "cli", ""
	if !cliOnly {
		if selector == "" {
			selector = agent.RoleDefault
		}
		isRole := agent.IsRoleName(selector)
		kind = "stage"
		if isRole {
			kind = "role"
		}
		if _, isStage := agent.RoleForStage(selector); !isRole && !isStage {
			return fmt.Errorf("unknown selector %q (valid roles: %s; valid stages: %s)", selector, strings.Join(agent.RoleNames(), ", "), strings.Join(agent.StageNames(), ", "))
		}
		role, _ = roleForKind(selector, isRole)
	}

	resolution, err := composeAgentResolution(cfg, resolutionRequest{
		Selector: selector,
		Kind:     kind,
		Role:     role,
		CLIOnly:  cliOnly,
		Overrides: agent.Overrides{
			CLI:       m.CLI,
			CLISet:    m.CLISet,
			Model:     m.Model,
			ModelSet:  m.ModelSet,
			Effort:    m.Effort,
			EffortSet: m.EffortSet,
		},
		Headless:    m.Headless,
		Passthrough: m.Passthrough,
		Dispatch:    m.Output != "",
		TmuxEnv:     os.Getenv("TMUX"),
	})
	if err != nil {
		return err
	}

	slotName := "interactive_command"
	if m.Headless {
		slotName = "headless_command"
	}
	if resolution.Template == "" {
		if cliOnly {
			return fmt.Errorf("CLI %q has no %s; configure clis.%s.%s", resolution.CLI, slotName, resolution.CLI, slotName)
		}
		hintName := resolution.CLI
		if hintName == "" {
			hintName = "<name>"
		}
		return fmt.Errorf("role %q resolves to CLI %q, which has no %s; configure clis.%s.%s",
			selector, resolution.CLI, slotName, hintName, slotName)
	}

	if m.Template || m.Print || m.Output != "" {
		if m.Output != "" {
			b, err := resolution.YAML()
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(b))
			return nil
		}
		if m.Template {
			fmt.Fprintln(cmd.OutOrStdout(), appendPassthrough(resolution.Template, m.Passthrough))
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), resolution.Command)
		}
		return nil
	}

	// Exec the composed command in the current shell so shell expansions expand
	// at invocation time and the agent CLI replaces this process. No TTY guard:
	// the agent CLI surfaces its own error when stdin is not a terminal.
	env := os.Environ()
	if workers, set := workersOverride(cmd); set {
		env = envWithWorkers(env, workers)
	}
	return execAgent("/bin/sh", []string{"/bin/sh", "-c", resolution.Command}, env)
}

// roleForKind resolves the role agentMode's selector names: a role name when
// isRole, else a stage name → its mapped role. The collisions (review,
// hydrate) are fixed points, so kind decides the -o yaml report only — the
// resolved role puns either way.
func roleForKind(selector string, isRole bool) (string, bool) {
	if isRole {
		return selector, true
	}
	role, ok := agent.RoleForStage(selector)
	return role, ok
}

// unknownCLIError is the shared `--cli <name>` LOOKUP failure for the
// two commands that accept the flag (`fab agent`'s CLI-addressed mode and
// `fab resolve-agent`'s invocation-time override). Both need byte-identical
// phrasing — the error is a user-facing contract documented once in
// `_cli-fab.md` — so it lives here rather than being restated at each call site.
//
// It is a lookup failure, NOT validation of the resolved command's content
// (document-don't-validate stands). The `clis.<name>` hint names a config key
// path, so an explicitly-empty `--cli=` would otherwise render the malformed
// `clis.` — a `<name>` placeholder is substituted so the suggested path is
// always a valid key.
//
// Callers gate on cobra's Flag.Changed (the flag was SUPPLIED) together with a
// failed agent.ResolveCLI lookup; this helper only formats the error.
func unknownCLIError(cfg *config.Config, name string) error {
	hintName := name
	if hintName == "" {
		hintName = "<name>"
	}
	return fmt.Errorf("unknown CLI %q (available: %s); configure it under clis.%s in fab/project/config.yaml",
		name, strings.Join(agent.CLINames(cfg), ", "), hintName)
}

// roleSessionCommand composes the interactive session command for ROLE: the
// role's CLI's interactive_command with the role's {model, effort}
// substituted. It is the ONE implementation of that resolution chain — both
// `fab agent <role>` and the operator's tmux-tab launcher go through it.
//
// It returns an EMPTY command (and a nil error) when the role's CLI cannot
// supply one — either the CLI NAME does not resolve at all, or it resolves
// but carries no interactive_command. The two are deliberately one signal: from
// a caller's view both mean "this role names no launchable session", and both
// have always produced the same outcome. That is deliberately not an error here,
// because the two callers legitimately differ on what it means — and keeping
// that divergence at the call sites, with the resolution itself shared, is the
// whole reason this helper exists:
//
//   - `fab agent` surfaces it as an actionable error naming the role and the
//     cli: a session the user explicitly asked for cannot be composed.
//   - the operator falls back to spawn.DefaultSpawnCommand — it must always
//     launch, so a broken CLI entry cannot strand the coordinator.
//
// The resolved profile comes back alongside so a falling-back caller can still
// substitute the role's {model, effort} into whatever command it picks.
//
// A genuine role-resolution failure (an unknown role NAME) is returned as an
// error — that is a caller mistake, not a policy choice.
//
// Before this existed the chain was written twice, and the copies drifted on the
// no-project case: the operator fell back to a nil config (silently discarding
// ~/.fab-kit/config.yaml) while `fab agent` failed closed. Sharing the chain is
// what keeps that from recurring.
func roleSessionCommand(cfg *config.Config, role string) (string, agent.Profile, error) {
	profile, err := agent.ResolveRole(cfg, role)
	if err != nil {
		return "", agent.Profile{}, err
	}
	prov, known := agent.ResolveCLI(cfg, profile.CLI)
	if !known || prov.InteractiveCommand == "" {
		return "", profile, nil
	}
	return spawn.WithProfile(prov.InteractiveCommand, profile.Model, profile.Effort), profile, nil
}

// appendPassthrough appends caller-supplied agent-CLI arguments to a composed
// command line, one shell-quoted token each.
//
// The composed line is a STRING executed via `sh -c` and printed verbatim by
// --print, so the arguments have to survive one more round of shell word
// splitting. Single-quoting each token is what makes `-p "two words"` arrive as
// one argument, and what keeps a token containing $, backticks or ; inert
// instead of being re-interpreted by that shell.
//
// An empty slice returns the command unchanged, so a no-passthrough invocation
// is byte-identical to before this existed.
func appendPassthrough(command string, args []string) string {
	for _, a := range args {
		command += " " + shellquote.Single(a)
	}
	return command
}

// loadRepoConfig loads the config from an explicit repo root (--repo) or the
// current repo's fab/ (upward search). The path-based load mirrors the former
// `fab spawn-command --repo` behavior; a missing file yields an empty config (the
// built-in CLI table then supplies the default session command).
func loadRepoConfig(repo string) (*config.Config, error) {
	return loadRepoConfigPreset(repo, "")
}

// loadRepoConfigPreset is loadRepoConfig with an invocation-time agent preset (the
// `fab agent --preset` choice, applied at the env layer for this one load). An empty
// preset is the plain cascade.
func loadRepoConfigPreset(repo, preset string) (*config.Config, error) {
	if repo != "" {
		return config.LoadPathWithPreset(filepath.Join(repo, "fab", "project", "config.yaml"), preset)
	}
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		if !errors.Is(err, resolve.ErrNoFabRoot) {
			// A cwd that cannot be read is a broken environment, not "no
			// project" — surface it instead of silently reading built-ins.
			return nil, err
		}
		// No fab/ project anywhere up the tree. `fab agent` is a CONFIG-ONLY
		// command — it needs a resolved {cli, model, effort}, not change
		// state — so it degrades to the project-free cascade (env > system >
		// built-in defaults) rather than failing closed. That is what makes
		// `fab agent` usable as a plain session launcher from any directory,
		// and it keeps a machine-wide preference in ~/.fab-kit/config.yaml
		// authoritative outside a project too. Project-state commands keep
		// erroring here; see cmd/fab/skill.md's config-free-commands rule.
		return config.LoadNoProjectWithPreset(preset), nil
	}
	return config.LoadWithPreset(fabRoot, preset)
}
