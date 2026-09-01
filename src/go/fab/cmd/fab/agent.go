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
	"gopkg.in/yaml.v3"
)

// agentCmd implements
// `fab agent [role|stage] [--provider <name>] [--model <id>] [--effort <level>] [--headless] [-t|--template] [-o <format>] [--workers <provider>] [-p|--print] [--repo <path>] [-- <agent-args>...]`
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
//     and composes providers.<profile.provider>.interactive_command with the
//     role's {model}/{effort}.
//   - `--provider <name>` with NO selector: BYPASSES role resolution and looks up
//     providers.<name> directly (project config per-field merged over fab-kit's
//     built-in table), composing its interactive_command with the `--model`/
//     `--effort` values. Omitted values are empty and follow spawn.WithProfile's
//     documented empty-value rule (template mode drops the placeholder's token
//     plus a preceding `-`-flag; append mode omits the flag) — so
//     `fab agent --provider codex --print` composes a bare `codex` invocation.
//   - A selector WITH `--provider`: re-resolves the role's profile from the named
//     provider's own fills (agent.ResolveRoleWith with Overrides.Provider) —
//     unnamed flags then apply verbatim over the refilled values.
//
// `--model`/`--effort` are general (not provider-mode-only): wherever they are
// legal they are applied as verbatim final overrides after role resolution /
// provider refill — verbatim pass-through, no validation (document-don't-validate).
//
// Output modes, one sink per invocation:
//
//   - Default: EXECs the composed command in the current shell (via `sh -c`, so
//     shell expansions like $(basename "$(pwd)") expand at invocation). No TTY
//     guard — exec-and-let-the-agent-CLI-handle-it (document-don't-validate).
//   - `-p, --print`: prints the fully-resolved command instead of executing (the
//     `fab spawn-command` replacement — profile-resolved, not stripped).
//   - `-t, --template`: prints the selected provider's command template
//     UNsubstituted (a tap BEFORE the fill step — `{model}`/`{effort}` placeholders
//     intact). Implies print; rejects `--model`/`--effort` with a usage error
//     (they feed a step that never runs).
//   - `-o, --output yaml`: prints the resolution as structured YAML (keys
//     selector/kind/role/provider/model/effort/command — minimal in this
//     change; the schema extends additively later).
//   - `--headless` selects the provider's `headless_command` instead of
//     `interactive_command`, valid only on the three print-family sinks; exec of a
//     headless command is a usage error. A provider with no headless_command
//     hard-errors naming the config key.
//
// `--model`/`--effort` legality and every other guard still keys on cobra's
// Flag.Changed — whether the flag was SUPPLIED — rather than on its value being
// non-empty. Explicitly-empty `--provider=` / `--model=` still resolves/errors the
// same as a supplied flag; an explicitly-empty `--model=` clears the model field
// on the selector paths.
//
// An unknown `--provider` name is a LOOKUP failure (non-zero exit naming the
// available providers), not validation of the command's content — resolved command
// strings still pass through verbatim (document-don't-validate). The error's
// config-key hint substitutes a `<name>` placeholder when the supplied name is
// empty (`--provider=`), so the suggested `providers.<key>` path is never malformed.
//
// Common to all modes:
//
//   - `--repo <path>`: reads <path>/fab/project/config.yaml instead of the current
//     repo (the operator's fetch-another-repo's-command use case).
//   - `--workers <provider>`: sets FAB_AGENT_WORKERS=<provider> in the exec
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
	var provider string
	var model string
	var effort string
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

			providerSet := cmd.Flags().Changed("provider")

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

			return runAgent(cmd, selector, provider, providerSet, model, effort, printOnly, template, headless, output, repo, passthrough)
		},
	}
	cmd.Flags().BoolVarP(&printOnly, "print", "p", false, "print the fully-resolved command instead of executing it")
	cmd.Flags().BoolVarP(&template, "template", "t", false, "print the selected provider's command template un-substituted (placeholders intact); implies --print")
	cmd.Flags().BoolVar(&headless, "headless", false, "resolve the provider's headless_command instead of interactive_command (print-family modes only)")
	cmd.Flags().StringVarP(&output, "output", "o", "", `print the resolution as a structured document; only "yaml" is accepted`)
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	cmd.Flags().StringVar(&provider, "provider", "", "provider override: with a [role|stage] selector, re-resolve from that provider's fills; alone, address the provider directly (bypasses role resolution)")
	cmd.Flags().StringVar(&model, "model", "", "final model override (empty: on the bare --provider form the provider command's model token is dropped, so its CLI default applies)")
	cmd.Flags().StringVar(&effort, "effort", "", "final effort override (empty: on the bare --provider form the effort token is dropped)")
	cmd.Flags().String("workers", "", "set FAB_AGENT_WORKERS for the launched agent session")
	return cmd
}

// runAgent selects the addressing + output modes the user's flags picked and
// delegates to resolveAgentInvocation for the composition.
func runAgent(cmd *cobra.Command, selector, provider string, providerSet bool, model, effort string, printOnly, template, headless bool, output string, repo string, passthrough []string) error {
	m := agentMode{
		Provider:    provider,
		ProviderSet: providerSet,
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
	}
	return resolveAgentInvocation(cmd, m)
}

// agentMode is the one-input resolveAgentInvocation call bundle — what
// cmd/runAgent normalized out of the cobra flag set. Field comments name each
// semantic; normalize them here ONCE so resolution reads a struct instead of a
// bare field list.
type agentMode struct {
	Provider    string
	ProviderSet bool // cobra's Flag.Changed for --provider (an explicitly-empty --provider= is still a supplied flag)
	Model       string
	ModelSet    bool
	Effort      string
	EffortSet   bool
	Selector    string // a role or stage name; empty = default role (unless ProviderSet)
	Print       bool
	Template    bool
	Headless    bool
	Output      string
	Repo        string
	Passthrough []string
}

// agentResolutionYAML is the one `-o yaml` output document — the minimal
// projection over the resolved invocation. The kind/selector/role triple
// reports provenance for the ADDRESSING form (bare provider = kind "provider"
// and role empty); provider/model/effort come from the resolved profile; the
// command is the composed line including passthrough args (the same line
// --print would emit).
type agentResolutionYAML struct {
	Selector string `yaml:"selector"`
	Kind     string `yaml:"kind"`
	Role     string `yaml:"role"`
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Effort   string `yaml:"effort"`
	Command  string `yaml:"command"`
}

func resolveAgentInvocation(cmd *cobra.Command, m agentMode) error {
	cfg, err := loadRepoConfig(m.Repo)
	if err != nil {
		return err
	}

	// The input-splitting above reduced the addressing to exactly one of
	// these forms; resolve against each and pick the (template, interactive,
	// or headless) command slot afterwards.
	var profile agent.Profile
	var cmdSlot string // provider's raw command template (interactive or headless)
	var kind string    // "role" | "stage" | "provider"
	var selector string

	if m.ProviderSet && m.Selector == "" {
		// BARE provider form: below role resolution entirely — compose the
		// provider's command with the invocation's --model/--effort only
		// (providers.<name>.profiles is deliberately not consulted).
		prov, known := agent.ResolveProvider(cfg, m.Provider)
		if !known {
			return unknownProviderError(cfg, m.Provider)
		}
		if cmdSlot, err = m.commandSlot(prov, m.Provider); err != nil {
			return err
		}
		profile = agent.Profile{Provider: m.Provider, Model: m.Model, Effort: m.Effort}
		kind = "provider"
		selector = ""
	} else {
		if m.Selector == "" {
			m.Selector = agent.RoleDefault
		}
		isRole := agent.IsRoleName(m.Selector)
		kind = "stage"
		if isRole {
			kind = "role"
		}
		if _, isStage := agent.RoleForStage(m.Selector); !isRole && !isStage {
			// Neither a role nor a known stage — a combined error naming both
			// valid sets rather than agent.RoleForName's stage-only message.
			return fmt.Errorf("unknown selector %q (valid roles: %s; valid stages: %s)", m.Selector, strings.Join(agent.RoleNames(), ", "), strings.Join(agent.StageNames(), ", "))
		}
		role, _ := roleForKind(m.Selector, isRole)
		profile, err = agent.ResolveRoleWith(cfg, role, agent.Overrides{Provider: m.Provider, ProviderSet: m.ProviderSet, Model: m.Model, ModelSet: m.ModelSet, Effort: m.Effort, EffortSet: m.EffortSet})
		if err != nil {
			return err
		}
		prov, known := agent.ResolveProvider(cfg, profile.Provider)
		if !known && m.ProviderSet {
			// A flag-named provider that resolves to nothing is the shared
			// lookup failure, on the selector path exactly as on the bare
			// --provider path (unknownProviderError is the one phrasing).
			return unknownProviderError(cfg, m.Provider)
		}
		slotName := "interactive_command"
		cmdSlot = prov.InteractiveCommand
		if m.Headless {
			slotName = "headless_command"
			cmdSlot = prov.HeadlessCommand
		}
		if cmdSlot == "" {
			// Selector-path errors name the requested selector + resolved
			// provider (mirroring roleSessionCommand's present-tense contract).
			// An explicitly-empty `--provider=` resolves an empty provider —
			// substitute the same <name> placeholder unknownProviderError uses,
			// so the hint never suggests a malformed `providers.` path.
			hintName := profile.Provider
			if hintName == "" {
				hintName = "<name>"
			}
			return fmt.Errorf("role %q resolves to provider %q, which has no %s; configure providers.%s.%s",
				m.Selector, profile.Provider, slotName, hintName, slotName)
		}
		selector = m.Selector
	}

	// Resolve the command slot (template mode requires raw-with-placeholders;
	// substitutions happen next unless the sink is -t's unsubstituted tap).
	composed := cmdSlot
	if !m.Template {
		composed = spawn.WithProfile(cmdSlot, profile.Model, profile.Effort)
	}

	// Passthrough args are appended as SHELL-QUOTED tokens, because the composed
	// line is handed to `sh -c` (below) and reaches --print as copy-pasteable
	// text. Quoting here is what keeps `fab agent -- -p "two words"` a single
	// argument instead of two, and what stops a value containing shell
	// metacharacters from being re-interpreted.
	composed = appendPassthrough(composed, m.Passthrough)

	if m.Template || m.Print || m.Output != "" {
		if m.Output != "" {
			// `-o yaml` emits the MINIMAL projection: bare-provider form
			// reports a provider kind and an empty role. Selector kind reports
			// the role-first/stage classification; the stage→role mapping is
			// still visible via role.
			doc := agentResolutionYAML{Selector: selector, Kind: kind, Role: "", Provider: profile.Provider, Model: profile.Model, Effort: profile.Effort, Command: composed}
			if kind != "provider" {
				role, _ := roleForKind(selector, agent.IsRoleName(selector))
				doc.Role = role
			}
			b, err := yaml.Marshal(doc)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(b))
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), composed)
		return nil
	}

	// Exec the composed command in the current shell so shell expansions expand
	// at invocation time and the agent CLI replaces this process. No TTY guard:
	// the agent CLI surfaces its own error when stdin is not a terminal.
	env := os.Environ()
	if workers, set := workersOverride(cmd); set {
		env = envWithWorkers(env, workers)
	}
	return execAgent("/bin/sh", []string{"/bin/sh", "-c", composed}, env)
}

// commandSlot picks the command slot for the BARE-provider form: the
// provider's interactive_command, or its headless_command when --headless
// is set. With no selector in play there is nothing else to name, so its
// empty-slot error names the provider outright — keeping the provider hint
// format on one side and the selector hint format on the other (the inline
// site in resolveAgentInvocation).
func (m agentMode) commandSlot(prov config.ProviderConfig, providerName string) (string, error) {
	slot := prov.InteractiveCommand
	slotKey := "interactive_command"
	if m.Headless {
		slot, slotKey = prov.HeadlessCommand, "headless_command"
	}
	if slot == "" {
		return "", fmt.Errorf("provider %q has no %s; configure providers.%s.%s", providerName, slotKey, providerName, slotKey)
	}
	return slot, nil
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

// roleSessionCommand composes the interactive session command for ROLE: the
// role's provider's interactive_command with the role's {model, effort}
// substituted. It is the ONE implementation of that resolution chain — both
// `fab agent <role>` and the operator's tmux-tab launcher go through it.
//
// It returns an EMPTY command (and a nil error) when the role's provider cannot
// supply one — either the provider NAME does not resolve at all, or it resolves
// but carries no interactive_command. The two are deliberately one signal: from
// a caller's view both mean "this role names no launchable session", and both
// have always produced the same outcome. That is deliberately not an error here,
// because the two callers legitimately differ on what it means — and keeping
// that divergence at the call sites, with the resolution itself shared, is the
// whole reason this helper exists:
//
//   - `fab agent` surfaces it as an actionable error naming the role and the
//     provider: a session the user explicitly asked for cannot be composed.
//   - the operator falls back to spawn.DefaultSpawnCommand — it must always
//     launch, so a broken provider entry cannot strand the coordinator.
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
	prov, known := agent.ResolveProvider(cfg, profile.Provider)
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
// built-in provider table then supplies the default session command).
func loadRepoConfig(repo string) (*config.Config, error) {
	if repo != "" {
		return config.LoadPath(filepath.Join(repo, "fab", "project", "config.yaml"))
	}
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		if !errors.Is(err, resolve.ErrNoFabRoot) {
			// A cwd that cannot be read is a broken environment, not "no
			// project" — surface it instead of silently reading built-ins.
			return nil, err
		}
		// No fab/ project anywhere up the tree. `fab agent` is a CONFIG-ONLY
		// command — it needs a resolved {provider, model, effort}, not change
		// state — so it degrades to the project-free cascade (env > system >
		// built-in defaults) rather than failing closed. That is what makes
		// `fab agent` usable as a plain session launcher from any directory,
		// and it keeps a machine-wide preference in ~/.fab-kit/config.yaml
		// authoritative outside a project too. Project-state commands keep
		// erroring here; see cmd/fab/skill.md's config-free-commands rule.
		return config.LoadNoProject(), nil
	}
	return config.Load(fabRoot)
}
