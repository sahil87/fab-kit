package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// paneRole names what a launch subcommand does with PANE mode. It is the one
// axis on which `start`, `open`, and `restart` genuinely differ now that pane
// mode's entry moved to its own verb, so it is modelled as a single enum on the
// shared flag struct rather than a scatter of per-subcommand booleans.
type paneRole int

const (
	// paneRejected — `fab dispatch start`. Pane mode's entry is `fab dispatch
	// open`, because a pane worker is now spawned and delivered to in two
	// separate steps with an agent-driven readiness gate between them, and Go
	// cannot run that gate. `start` therefore launches only the headless arm; a
	// pane flag, or a configuration that resolves to pane, is an actionable
	// error pointing at `open`.
	paneRejected paneRole = iota
	// paneForced — `fab dispatch open`. Pane is the verb's entire purpose, so it
	// is selected explicitly and a missing prerequisite is a hard error rather
	// than a descent (an `open` that quietly became a headless launch would be
	// the opposite of what its caller asked for).
	paneForced
	// paneDerived — `fab dispatch restart`. Mode is re-derived from the current
	// environment as before; a headless landing relaunches fully, while a pane
	// landing performs the spawn-only `open` step and hands the gate back to the
	// orchestrator.
	paneDerived
)

// launchFlags holds the mode/launch flags the launch subcommands share, bound by
// addLaunchFlags. It exists so they cannot drift on their flag SURFACE the way
// runDispatchLaunch stops them drifting on the launch path: the registrations,
// the guards, and the mode ladder all live once.
//
// Only the flag surface and the launch path are shared — each subcommand keeps
// its own Use/Short/Long/Example strings, since the prompt-source difference
// (stdin vs. the persisted file) and the pane role are exactly what their help
// text has to explain.
type launchFlags struct {
	timeout      int
	paneMode     bool
	headlessMode bool
	server       string
	role         paneRole
}

// addLaunchFlags registers the launch flags this subcommand's pane role calls for
// and returns the struct they bind into.
//
// The registered surface differs by role, but every flag a caller could
// reasonably still type stays REGISTERED so it can be answered with guidance
// rather than a bare `unknown flag`:
//
//	paneRejected (start)   --timeout / --headless, plus HIDDEN --pane / --server
//	                       that error with the `fab dispatch open` route
//	paneForced   (open)    --server only — pane is implicit, and the headless-only
//	                       flags have nothing to mean
//	paneDerived  (restart) all four, with the cobra flag group for --pane/--headless
//
// That group fires during ValidateFlagGroups — BEFORE any RunE work — so the
// conflict is a genuine usage error (exit 2) that structurally cannot leave
// partial state behind, matching resolve.go / pane_capture.go / panemap.go.
func addLaunchFlags(cmd *cobra.Command, role paneRole) *launchFlags {
	f := &launchFlags{role: role}
	if role == paneForced {
		cmd.Flags().StringVarP(&f.server, "server", "L", "", "Target tmux socket label (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket")
		return f
	}

	cmd.Flags().IntVar(&f.timeout, "timeout", 0, "Enforce a POSIX `timeout <secs>` inside the launch wrapper (0 = none); implies --headless")
	cmd.Flags().BoolVar(&f.headlessMode, "headless", false, "Force a detached headless worker, overriding dispatch.mode for this launch")
	cmd.Flags().BoolVar(&f.paneMode, "pane", false, "Force an interactive tmux worker — a pane split into your own window, or a new window when you are not in one (requires a reachable tmux server)")
	cmd.Flags().StringVarP(&f.server, "server", "L", "", "Target tmux socket label (passed as 'tmux -L <name>'); implies pane mode. Defaults to $TMUX / tmux default socket; ignored in headless mode")

	if role == paneRejected {
		// Kept registered but hidden: a caller reaching for the old spelling gets
		// the route to the new verb instead of `unknown flag`, while the help
		// output advertises only what `start` can actually do.
		_ = cmd.Flags().MarkHidden("pane")
		_ = cmd.Flags().MarkHidden("server")
		return f
	}
	cmd.MarkFlagsMutuallyExclusive("pane", "headless")
	return f
}

// resolveMode enforces this subcommand's pane-role guards and then resolves the
// launch mode. Called at the top of every launch subcommand's RunE.
//
// For paneRejected, a supplied --pane/--server is answered with the `open` route
// before anything else happens. For paneForced, pane is selected explicitly with
// no ladder consultation at all — the verb IS the selection. Otherwise the
// explicit-first ladder runs exactly as before.
//
// --timeout is implemented as POSIX `timeout N` INSIDE the headless `sh -c`
// wrapper, which pane mode never constructs. Silently ignoring it under --pane
// would advertise a bound nothing enforces, so the combination is a usage error.
// Keyed on Flags().Changed (was the flag SUPPLIED) rather than the value,
// mirroring agent.go's guard style, so an explicit `--timeout 0` is still caught.
//
// Only the EXPLICIT --pane conflicts: --timeout is itself a headless signal in the
// selection ladder, so `--timeout` inside tmux selects headless rather than
// erroring (it must not break scripted invocations that never mention panes).
//
// The ladder's caller-supplied signals are likewise read as "was the flag
// SUPPLIED", so `--timeout 0` / `--server ""` still count.
//
// It resolves the pane PLACEMENT in the same breath (see paneTarget). Both env
// reads — $TMUX and $TMUX_PANE — live HERE, in the cobra layer, so
// internal/dispatch stays pure and table-testable: the SelectMode precedent.
// Resolving both here is also what gives `restart` and `open` the split behavior
// for free — they share this method, so the subcommands cannot drift on either
// decision.
func (f *launchFlags) resolveMode(cmd *cobra.Command, cfg *config.Config, prov config.ProviderConfig) (dispatch.Mode, dispatch.AutoReason, paneTarget, error) {
	serverSet := cmd.Flags().Changed("server")

	if f.role == paneRejected {
		for _, flag := range []string{"pane", "server"} {
			if cmd.Flags().Changed(flag) {
				return "", "", paneTarget{}, paneFlagRetiredError(flag)
			}
		}
	}
	if f.role == paneForced {
		return dispatch.ModePane, dispatch.ReasonExplicit, f.paneTargetFor(dispatch.ModePane, serverSet), nil
	}

	if f.paneMode && cmd.Flags().Changed("timeout") {
		return "", "", paneTarget{}, fmt.Errorf("--pane and --timeout are mutually exclusive: --timeout is enforced by the headless launch wrapper (POSIX `timeout`), which pane mode does not use")
	}
	tmux := dispatch.TmuxAbsent
	if os.Getenv("TMUX") != "" {
		tmux = dispatch.TmuxAvailable
	}
	mode, reason, err := dispatch.SelectMode(f.paneMode, f.headlessMode,
		cmd.Flags().Changed("timeout"), serverSet, cfg.GetDispatchMode(), prov.Native,
		prov.InteractiveCommand != "", prov.HeadlessCommand != "", tmux)
	if err != nil {
		return "", "", paneTarget{}, err
	}
	return mode, reason, f.paneTargetFor(mode, serverSet), nil
}

// paneTargetFor resolves the pane placement inputs that come from the
// environment. Shared by both resolveMode exits so the two cannot drift.
func (f *launchFlags) paneTargetFor(mode dispatch.Mode, serverSet bool) paneTarget {
	dispatcherPane := os.Getenv("TMUX_PANE")
	return paneTarget{
		shape:          dispatch.SelectPaneShape(mode == dispatch.ModePane, serverSet, dispatcherPane),
		dispatcherPane: dispatcherPane,
	}
}

// paneFlagRetiredError answers a pane flag typed at `fab dispatch start` with the
// route to the verb that now owns pane mode, rather than letting cobra emit a
// bare `unknown flag`. A pane worker is spawned and delivered to in two steps
// with an agent-driven readiness gate between them, so a single-shot `start
// --pane` has nothing to map onto.
func paneFlagRetiredError(flag string) error {
	return fmt.Errorf("`fab dispatch start` launches only the headless arm, so --%s is not accepted here; pane mode's entry is `fab dispatch open <change> <stage>`, followed by `fab dispatch ready` and `fab dispatch deliver`", flag)
}

// paneDispatchError is the pane counterpart of nativeDispatchError: `start`'s
// automatic resolution landed on a rung this subcommand cannot launch, so it
// stops before any state write and names the verb that can.
func paneDispatchError(stage, provider string) error {
	return fmt.Errorf("stage %q resolves to pane mode for provider %q, but `fab dispatch start` launches only the headless arm; run `fab dispatch open <change> %s` (then `fab dispatch ready` and `fab dispatch deliver`) to open a pane worker, or pass --headless to force this launch headless",
		stage, provider, stage)
}

// paneTarget carries the pane-placement inputs to launchPane: WHICH shape the
// worker pane takes, — for the split shape — WHICH pane the dispatching process
// itself occupies ($TMUX_PANE, the split's anchor), and how wide the worker column
// is carved.
//
// shape and dispatcherPane travel together because they come from the same env read
// and are meaningless apart: a shape of ShapeSplit with no dispatcherPane cannot be
// placed, and SelectPaneShape is exactly the function that rules that pair out.
// Both are resolved in resolveMode, the cobra layer.
//
// columnWidth comes from CONFIG rather than the environment, so it is filled by
// runDispatchLaunch once config is loaded — resolveMode has no config in hand. It
// rides this struct anyway because it is the third input to the same placement
// decision, and threading it as yet another launchPane parameter would only widen an
// already-wide signature.
type paneTarget struct {
	shape          dispatch.PaneShape
	dispatcherPane string
	columnWidth    int
}

func dispatchStartCmd() *cobra.Command {
	// Declared before the command literal so RunE can close over it; addLaunchFlags
	// (below) binds it to the registered flags before any RunE can run.
	var f *launchFlags
	cmd := &cobra.Command{
		Use:   "start <change> <stage>",
		Short: "Launch a HEADLESS stage worker (pane mode's entry is `fab dispatch open`)",
		Long: "Launch a detached headless stage worker from the prompt on stdin.\n\n" +
			"`start` composes only the resolved provider's headless_command. Pane mode has\n" +
			"its own entry — `fab dispatch open`, followed by `fab dispatch ready` and\n" +
			"`fab dispatch deliver` — because a pane worker is spawned and delivered to in\n" +
			"two steps with an agent-driven readiness gate between them. A configuration\n" +
			"that resolves to pane therefore stops here with that route instead of\n" +
			"launching.",
		Example: `  # Resolve automatically from dispatch.mode and provider capability
  fab dispatch start b91h apply < prompt.md

  # Force headless for an unattended run that happens to live inside a tmux tab
  fab dispatch start b91h apply --headless < prompt.md

  # Enforce a 30-minute POSIX timeout (implies headless)
  fab dispatch start --timeout 1800 b91h apply < prompt.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchLaunch(cmd, args[0], args[1], f, promptFromStdin)
		},
	}
	f = addLaunchFlags(cmd, paneRejected)
	return cmd
}

// promptSource obtains the stage prompt bytes for a launch. It is the ONE seam
// `start` and `restart` differ at: `start` reads stdin (promptFromStdin) and
// `restart` reads the already-persisted {stage}-prompt.md (promptFromStateDir in
// dispatch_restart.go). Everything else about the launch — resolution, pane
// validation, refuse-if-running, stale-file clearing, launch, save, report — is
// shared, so the two subcommands cannot drift.
//
// It is called with the resolved dispatch dir and stage AFTER the refuse-if-running
// check, so a running dispatch refuses before either source is touched. Bytes
// rather than an io.Reader: both sources are fully read into memory before the
// launch anyway (the file must exist on disk by the time the worker reads it), and
// bytes keep the shared path free of stream lifetime concerns.
//
// The second return value reports whether the shared path should PERSIST the
// bytes to {stage}-prompt.md. `start` persists (that is where the prompt comes
// from); `restart` does not — the file IS its input, so rewriting it with its own
// content is a no-op that only risks corruption on a partial write.
type promptSource func(cmd *cobra.Command, dir, stage string) (prompt []byte, persist bool, err error)

// promptFromStdin is `start`'s prompt source: the full stage prompt arrives on
// stdin and is persisted to {stage}-prompt.md before the launch (the headless
// wrapper redirects it into the command's stdin, and pane mode points the
// interactive worker at it).
func promptFromStdin(cmd *cobra.Command, _, _ string) ([]byte, bool, error) {
	prompt, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return nil, false, fmt.Errorf("read prompt from stdin: %w", err)
	}
	return prompt, true, nil
}

// runDispatchLaunch resolves the stage's role → provider, refuses if a dispatch
// for this (change, stage) is already running, obtains the prompt from the given
// promptSource, resolves the mode, then launches the worker and persists
// {stage}.yaml:
//
//	dispatch.ModeHeadless — the provider's headless_command, detached via the
//	                        `sh -c` wrapper; tmux-independent (launchHeadless)
//	dispatch.ModePane     — the provider's interactive_command, interactive in a tmux
//	                        pane the user can watch and steer — split into the
//	                        dispatching agent's own window, or in a new window
//	                        when the dispatcher has no pane to split (launchPane)
//
// It backs BOTH `fab dispatch start` (source: stdin) and `fab dispatch restart`
// (source: the persisted {stage}-prompt.md). A restart is a fresh attempt under
// the existing last-attempt-only semantics — same prologue, same mode ladder, same
// output/record shape — so there is deliberately no restart-specific branch here
// and no new state string, attempt counter, or `restarted:` marker anywhere.
//
// Mode selection uses dispatch.SelectMode after config/provider resolution. Its
// reason governs output and the pane-probe asymmetry: explicit pane propagates a
// probe error, while automatic pane re-runs the shared descent with tmux marked
// unreachable and may land on native or headless.
//
// Everything up to the launch is shared: the two launchable modes differ only in
// which provider command they compose and how they start it. The ladder changes
// modes; a command field never substitutes for another.
func runDispatchLaunch(cmd *cobra.Command, changeArg, stage string, flags *launchFlags, prompt promptSource) error {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}
	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		return err
	}
	id := resolve.ExtractID(folder)
	if id == "" {
		return fmt.Errorf("could not extract change ID from %q", folder)
	}
	repoRoot := filepath.Dir(fabRoot)
	dir := dispatch.DirFor(repoRoot, id)

	// Resolve the stage's role → provider, substituting {model}/{effort} via
	// internal/spawn — the same path fab resolve-agent uses. Provider fields are
	// independent capability data; SelectMode applies the configured policy.
	cfg, err := config.Load(fabRoot)
	if err != nil {
		return err
	}
	profile, err := agent.Resolve(cfg, stage)
	if err != nil {
		return err
	}
	prov, _ := agent.ResolveProvider(cfg, profile.Provider)
	mode, reason, target, err := flags.resolveMode(cmd, cfg, prov)
	if err != nil {
		// Only a genuinely empty ladder gets the capability framing. resolveMode's
		// other refusals — a retired pane flag, the --pane/--timeout conflict — are
		// already actionable on their own, and wrapping them in "configure
		// providers.X.…" would point the caller at a config key that has nothing to
		// do with what they typed.
		if errors.Is(err, dispatch.ErrNoMode) {
			return noDispatchCapabilityError(profile.Provider, cfg.GetDispatchMode(), err)
		}
		return err
	}
	if strings.Contains(string(reason), "(descended:") {
		fmt.Fprintf(cmd.ErrOrStderr(), "dispatch selection: %s\n", reason)
	}
	if mode == dispatch.ModeNative {
		return nativeDispatchError(stage, profile.Provider)
	}

	// The third placement input (the other two are env-derived, resolved in
	// resolveMode). Filled here because this is the first point config exists;
	// ignored entirely outside the split shape.
	target.columnWidth = cfg.GetDispatchColumnWidth()

	// Validate pane mode BEFORE composing any command and before any state write,
	// so a pane path that cannot proceed leaves no partial dispatch behind — and,
	// under automatic selection, can still descend against the failed probe.
	// Composition is deferred until the final rung is known.
	if mode == dispatch.ModePane {
		if fallback := validatePane(prov, flags.server, stage, profile.Provider); fallback != nil {
			// Asymmetric by selection source: explicit pane propagates the error;
			// automatic pane descends again with the current probe result.
			if reason == dispatch.ReasonExplicit {
				return fallback.err
			}
			tmux := dispatch.TmuxAvailable
			if fallback.tmuxUnavailable {
				tmux = dispatch.TmuxUnreachable
			}
			mode, reason, err = dispatch.SelectMode(false, false, false, false,
				cfg.GetDispatchMode(), prov.Native, prov.InteractiveCommand != "",
				prov.HeadlessCommand != "", tmux)
			if err != nil {
				return noDispatchCapabilityError(profile.Provider, cfg.GetDispatchMode(), err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "dispatch selection: %s\n", reason)
			if mode == dispatch.ModeNative {
				return nativeDispatchError(stage, profile.Provider)
			}
		}
	}

	// A pane landing `start` cannot launch, refused AFTER the descent has had its
	// chance. Ordering is load-bearing: a stale $TMUX makes the ladder pick pane,
	// and descending to headless on the failed probe is exactly what keeps an
	// unattended `start` working there. Only a pane rung that survives validation
	// is a genuine "you wanted a watchable worker" landing, and that one gets the
	// route to the verb that opens one. `restart` and `open` accept it instead and
	// land on the spawn-only path.
	if mode == dispatch.ModePane && flags.role == paneRejected {
		return paneDispatchError(stage, profile.Provider)
	}

	// The mode is final now, so exactly one mode-specific command is composed.
	baseCmd, err := modeCommand(mode, prov, stage, profile.Provider)
	if err != nil {
		return err
	}
	resolvedCmd := spawn.WithProfile(baseCmd, profile.Model, profile.Effort)

	// Refuse-if-running: a live prior dispatch for this exact (change, stage)
	// must be killed first. A completed prior attempt (done/failed/orphaned) is
	// overwritten — last-attempt-only, no per-attempt history. Liveness is read
	// per the PRIOR record's own mode: a prior pane dispatch is live while its
	// pane lives, a prior headless one while its pid lives.
	if prior, err := dispatch.Load(dir, stage); err == nil {
		running, err := priorRunning(dir, stage, prior)
		if err != nil {
			return err
		}
		if running {
			return fmt.Errorf("a dispatch for %s/%s is already running (%s); run `fab dispatch kill` first",
				changeArg, stage, priorIdentity(prior))
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	// Obtain the prompt AFTER the refusal check (so a running dispatch refuses
	// before either source is touched) and persist it BEFORE launching: the
	// headless wrapper redirects it into the command's stdin, and pane mode points
	// the interactive worker at it (the file must exist by the time the worker
	// reads the pointer). A `restart` source reads that very file and asks for no
	// re-write.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dispatch dir: %w", err)
	}
	promptPath := dispatch.PromptPath(dir, stage)
	promptBytes, persistPrompt, err := prompt(cmd, dir, stage)
	if err != nil {
		return err
	}
	if persistPrompt {
		if err := os.WriteFile(promptPath, promptBytes, 0o644); err != nil {
			return fmt.Errorf("write prompt: %w", err)
		}
	}

	// Overwrite of a completed prior attempt: clear the stale exit/result/log so
	// the new run's status is not contaminated by the previous attempt's files.
	for _, p := range []string{
		dispatch.ExitPath(dir, stage),
		dispatch.ResultPath(dir, stage),
		dispatch.LogPath(dir, stage),
	} {
		_ = os.Remove(p)
	}

	rec := &dispatch.Dispatch{
		SpawnCmd:  resolvedCmd,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	var report string
	if mode == dispatch.ModePane {
		report, err = launchPane(cmd, rec, resolvedCmd, repoRoot, id, stage, flags.server, target)
	} else {
		report, err = launchHeadless(rec, resolvedCmd, repoRoot, dir, stage, promptPath, flags.timeout)
	}
	if err != nil {
		return err
	}

	if err := dispatch.Save(dir, stage, rec); err != nil {
		return err
	}

	// The report names the mode's identity; automatic selection appends its reason so
	// a surprising mode or descent is explainable from the output alone —
	// the compliance-visibility principle. An explicitly-selected mode's line is
	// byte-identical to before auto selection existed (ReasonExplicit is empty).
	if reason != dispatch.ReasonExplicit {
		report += ", " + string(reason)
	}
	// The verb is derived from the MODE, not the subcommand: a headless launch
	// hands the worker its prompt on stdin and is therefore dispatched, while a
	// pane launch only opens the pane — delivery is a separate, verified step —
	// so calling it "dispatched" would assert something that has not happened yet.
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s/%s (%s)\n", launchVerb(mode), id, stage, report)

	// `restart`'s caller asked for a full relaunch and got half of one, so the
	// missing half is named. `open`'s caller asked for exactly a spawn, so it is
	// not told what its own verb already says.
	if mode == dispatch.ModePane && flags.role == paneDerived {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"note: the pane holds no prompt yet — run `fab dispatch ready %s %s`, clear any wall it reports, then `fab dispatch deliver %s %s`\n",
			changeArg, stage, changeArg, stage)
	}
	return nil
}

// launchVerb names what a launch of this mode actually accomplished.
func launchVerb(mode dispatch.Mode) string {
	if mode == dispatch.ModePane {
		return "opened"
	}
	return "dispatched"
}

// paneFallback describes why the pane path cannot proceed. The caller propagates
// the error for explicit pane or re-runs automatic selection with the observed tmux
// availability.
type paneFallback struct {
	tmuxUnavailable bool
	err             error
}

// validatePane reports whether pane mode can proceed, returning nil when it can
// and the matching paneFallback when it cannot. It performs no state writes and
// composes no command, so both outcomes are safe before anything is persisted.
//
// The two failure shapes, in probe order:
//
//	(a) the tmux server does not answer — a stale $TMUX inherited from a killed
//	    server, or no tmux at all (ServerReachable's actionable error).
//	(b) tmux answered, but the resolved provider carries no interactive_command —
//	    there is no interactive invocation to open a window on.
//
// Shape (b) is normally filtered by SelectMode for automatic selection, but is
// checked here for explicit pane and as a defensive invariant. Headless performs
// no tmux probe.
func validatePane(prov config.ProviderConfig, server, stage, providerName string) *paneFallback {
	if probeErr := dispatch.ServerReachable(server); probeErr != nil {
		return &paneFallback{
			tmuxUnavailable: true,
			err:             probeErr,
		}
	}
	if prov.InteractiveCommand == "" {
		return &paneFallback{
			err: missingCommandError(stage, providerName, "interactive_command"),
		}
	}
	return nil
}

func nativeDispatchError(stage, provider string) error {
	return fmt.Errorf("stage %q resolves to native mode for provider %q, but `fab dispatch` cannot launch the native Agent-tool adapter; re-run `fab resolve-agent %s --alias` and dispatch natively when the `dispatch=` line is absent",
		stage, provider, stage)
}

// modeCommand picks the provider command field the requested mode composes and
// errors with the matching config-key hint when it is absent.
//
// Under `dispatch start` the pane branch's absent-field error is diagnosed EARLIER
// by validatePane for explicit pane and defensively after selection, which
// raises the identical missingCommandError. The check is kept here too: it is this
// function's own contract that a mode never composes an empty command, and both
// paths share one message so they cannot drift.
//
// The two fields are deliberately NOT merged and never fall back to each other
// (they are different invocations of the same binary — see the `providers:`
// contract): headless runs ONE headless task via headless_command, pane mode
// opens an interactive SESSION via interactive_command — the same string `fab agent`
// composes, which is why pane mode needs no new provider config field.
func modeCommand(mode dispatch.Mode, prov config.ProviderConfig, stage, providerName string) (string, error) {
	field, cmdStr := "headless_command", prov.HeadlessCommand
	if mode == dispatch.ModePane {
		field, cmdStr = "interactive_command", prov.InteractiveCommand
	}
	if cmdStr == "" {
		return "", missingCommandError(stage, providerName, field)
	}
	return cmdStr, nil
}

// missingCommandError renders the actionable "this provider cannot dispatch this
// stage" error naming the stage, its resolved role, the provider, and the exact
// config key to set. It lives in one place because two callers raise it —
// modeCommand (at composition) and validatePane (shape (b), which must diagnose
// the missing interactive_command before composition) — and the two must not drift.
func missingCommandError(stage, providerName, field string) error {
	role, _ := agent.RoleForStage(stage)
	return fmt.Errorf("stage %q resolves to role %q (provider %q), which has no %s; configure providers.%s.%s to dispatch this stage",
		stage, role, providerName, field, providerName, field)
}

// launchPane opens the interactive worker in a tmux pane and records the pane
// identity on rec. It delivers NO PROMPT: the composed interactive_command is
// passed to tmux verbatim, as pure launch grammar.
//
// Prompt delivery is a separate, VERIFIED step (`fab dispatch deliver`, behind
// the readiness gate). Appending the pointer as a spawn argument — the previous
// behavior — was fire-and-forget: a provider CLI that silently drops a positional
// prompt left the worker at an empty prompt while the dispatch read `running`,
// and requiring one tied pane capability to whether a provider's CLI happens to
// accept a positional prompt at all. See docs/specs/harness-adapters.md § 3.
//
// WHERE the pane opens is dispatch.SelectPaneShape's decision, resolved in the
// cobra RunE and delivered as target (see paneTarget):
//
//	ShapeSplit  — split the DISPATCHING agent's own window, so its stage workers
//	              stack in a right-hand column beside it (the two-tier hierarchy's
//	              inner tier). The identity string rides the pane TITLE.
//	ShapeWindow — a new window named for the dispatch (the pre-split behavior),
//	              for a dispatcher that has no pane of its own to split.
//
// WITHIN the split shape, which pane is split and how wide the column is carved is
// dispatch.SplitTarget's decision — record-keyed sibling detection (pane titles are
// clobbered by the harness running in the worker) plus the configured column width
// on the carving split only. Both of its non-fatal outcomes come back as warnings.
//
// Both shapes record the SAME identity string in rec.Window and both are keyed by
// pane ID afterwards, so status/kill/capture need no shape awareness.
func launchPane(cmd *cobra.Command, rec *dispatch.Dispatch, resolvedCmd, repoRoot, id, stage, server string, target paneTarget) (string, error) {
	title := dispatch.WindowName(id, stage)

	var paneID, report string
	var err error
	if target.shape == dispatch.ShapeSplit {
		// Placement degrades before it fails: a failing sibling probe — an
		// unreadable dispatch record as much as a failing tmux list-panes — still
		// yields a usable placement, so the dispatch launches and only the warning
		// records that the probe was degraded. The message quotes the probe's own
		// error (which names WHICH half failed) and then WHERE the worker actually
		// landed, because a partial record read can still find a sibling — only a
		// total failure falls all the way back to carving off the dispatcher.
		place, probeErr := dispatch.SplitTarget(server, target.dispatcherPane, repoRoot, target.columnWidth)
		if probeErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: worker-column placement probe failed (%v); %s\n",
				probeErr, place.Describe())
		}
		var warnings []error
		paneID, warnings, err = dispatch.OpenSplitPane(server, place, title, repoRoot, resolvedCmd)
		// Cosmetic-only failures (a size tmux would not take, a title it would not
		// set): the worker is running and its pane ID — the real identity — is
		// already recorded, so these warn rather than aborting.
		for _, w := range warnings {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", w)
		}
		if err != nil {
			return "", err
		}
		report = fmt.Sprintf("pane %s, split, title %s", paneID, title)
	} else {
		paneID, err = dispatch.OpenWindow(server, title, repoRoot, resolvedCmd)
		if err != nil {
			return "", err
		}
		report = fmt.Sprintf("pane %s, window %s", paneID, title)
	}
	rec.Pane = paneID
	rec.Window = title
	rec.Server = server
	return report, nil
}

// launchHeadless launches the detached wrapper and records the pid/pgid on rec:
// setsid semantics over `sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'`.
func launchHeadless(rec *dispatch.Dispatch, resolvedCmd, repoRoot, dir, stage, promptPath string, timeout int) (string, error) {
	argv := dispatch.WrapperArgv(resolvedCmd,
		promptPath, dispatch.LogPath(dir, stage), dispatch.ExitPath(dir, stage), timeout)
	pid, pgid, err := dispatch.Launch(argv, repoRoot)
	if err != nil {
		return "", err
	}
	rec.PID = pid
	rec.PGID = pgid
	rec.Timeout = timeout
	return fmt.Sprintf("pid %d, pgid %d", pid, pgid), nil
}

// priorRunning reports whether a prior dispatch record is still live, reading
// liveness per that record's OWN mode (a prior pane dispatch may predate or
// follow a headless one for the same stage). In BOTH modes it applies the same
// finished-signal `fab dispatch status` derives its state from, so the two
// commands can never disagree about whether an attempt is still going:
//
//	headless — {stage}.exit present ⇒ finished (the shell recorded a code), so
//	           running is "no exit file AND the pid is alive" (DeriveState).
//	pane     — {stage}-result.yaml present ⇒ finished, and result presence WINS
//	           over pane liveness (DerivePaneState): an interactive worker never
//	           exits on completion, it sits at its prompt, so a liveness-only
//	           rule would refuse forever after a successful pane run and make a
//	           `done` attempt un-overwritable.
func priorRunning(dir, stage string, prior *dispatch.Dispatch) (bool, error) {
	if prior.IsPane() {
		if dispatch.ResultPresent(dir, stage) {
			return false, nil
		}
		return dispatch.PaneAlive(prior.Pane, prior.Server), nil
	}
	exitPresent, _, err := dispatch.ReadExit(dir, stage)
	if err != nil {
		return false, err
	}
	return !exitPresent && dispatch.Alive(prior.PID), nil
}

// priorIdentity renders the live prior dispatch's identity for the refusal
// message — the pane ID in pane mode, the pid in headless mode.
func priorIdentity(prior *dispatch.Dispatch) string {
	if prior.IsPane() {
		return fmt.Sprintf("pane %s", prior.Pane)
	}
	return fmt.Sprintf("pid %d", prior.PID)
}
