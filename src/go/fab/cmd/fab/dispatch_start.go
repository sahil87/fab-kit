package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

// launchFlags holds the four mode/launch flags `start` and `restart` share, bound
// by addLaunchFlags. It exists so the two subcommands cannot drift on their flag
// SURFACE the way runDispatchLaunch stops them drifting on the launch path: the
// registrations, the --pane/--timeout guard, and the mode ladder all live once.
//
// Only the flag surface is shared — each subcommand keeps its own Use/Short/Long/
// Example strings, since the prompt-source difference (stdin vs. the persisted
// file) is exactly what their help text has to explain.
type launchFlags struct {
	timeout      int
	paneMode     bool
	headlessMode bool
	server       string
}

// addLaunchFlags registers the shared launch flags on cmd and returns the struct
// they bind into. It also installs the cobra flag group for --pane/--headless:
// that fires during ValidateFlagGroups — BEFORE any RunE work — so the conflict is
// a genuine usage error (exit 2) that structurally cannot leave partial state
// behind, matching resolve.go / pane_capture.go / panemap.go.
func addLaunchFlags(cmd *cobra.Command) *launchFlags {
	f := &launchFlags{}
	cmd.Flags().IntVar(&f.timeout, "timeout", 0, "Enforce a POSIX `timeout <secs>` inside the launch wrapper (0 = none); implies --headless")
	cmd.Flags().BoolVar(&f.paneMode, "pane", false, "Force an interactive tmux worker — a pane split into your own window, or a new window when you are not in one (requires a reachable tmux server)")
	cmd.Flags().BoolVar(&f.headlessMode, "headless", false, "Force a detached headless worker, opting out of auto pane selection inside tmux")
	cmd.Flags().StringVarP(&f.server, "server", "L", "", "Target tmux socket label (passed as 'tmux -L <name>'); implies pane mode. Defaults to $TMUX / tmux default socket; ignored in headless mode")
	cmd.MarkFlagsMutuallyExclusive("pane", "headless")
	return f
}

// resolveMode enforces the --pane/--timeout usage error and then resolves the
// launch mode from the explicit-first ladder. Called at the top of both
// subcommands' RunE.
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
// Resolving both here is also what gives `restart` the split behavior for free —
// it shares this method, so the two subcommands cannot drift on either decision.
func (f *launchFlags) resolveMode(cmd *cobra.Command) (dispatch.Mode, dispatch.AutoReason, paneTarget, error) {
	if f.paneMode && cmd.Flags().Changed("timeout") {
		return "", "", paneTarget{}, fmt.Errorf("--pane and --timeout are mutually exclusive: --timeout is enforced by the headless launch wrapper (POSIX `timeout`), which pane mode does not use")
	}
	serverSet := cmd.Flags().Changed("server")
	mode, reason := dispatch.SelectMode(f.paneMode, f.headlessMode,
		cmd.Flags().Changed("timeout"), serverSet, os.Getenv("TMUX"))
	dispatcherPane := os.Getenv("TMUX_PANE")
	return mode, reason, paneTarget{
		shape:          dispatch.SelectPaneShape(mode == dispatch.ModePane, serverSet, dispatcherPane),
		dispatcherPane: dispatcherPane,
	}, nil
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
		Short: "Launch a stage worker — mode defaults to auto (a watchable tmux pane inside tmux, headless outside); force with --pane / --headless",
		Example: `  # Auto mode: a watchable tmux pane inside tmux, headless outside
  fab dispatch start b91h apply < prompt.md

  # Force headless for an unattended run that happens to live inside a tmux tab
  fab dispatch start b91h apply --headless < prompt.md

  # Enforce a 30-minute POSIX timeout (implies headless)
  fab dispatch start --timeout 1800 b91h apply < prompt.md

  # Force the stage into a tmux pane you can watch and steer (splits your window)
  fab dispatch start b91h review --pane < prompt.md

  # Target a specific tmux socket (implies pane; works from outside tmux)
  fab dispatch start b91h review --server work < prompt.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, reason, target, err := f.resolveMode(cmd)
			if err != nil {
				return err
			}
			return runDispatchLaunch(cmd, args[0], args[1], f.timeout, mode, reason, f.server, target, promptFromStdin)
		},
	}
	f = addLaunchFlags(cmd)
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

// runDispatchLaunch resolves the stage's tier → provider, refuses if a dispatch
// for this (change, stage) is already running, obtains the prompt from the given
// promptSource, then launches the worker in the ALREADY-RESOLVED mode and persists
// {stage}.yaml:
//
//	dispatch.ModeHeadless — the provider's dispatch_command, detached via the
//	                        `sh -c` wrapper; tmux-independent (launchHeadless)
//	dispatch.ModePane     — the provider's session_command, interactive in a tmux
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
// The mode arrives resolved (dispatch.SelectMode, called in the cobra RunE) along
// with the selection SOURCE (reason), which governs two things here: whether the
// output line explains the choice, and whether a pane path that cannot proceed is
// a hard error (explicit selection) or a soft fallback to headless (auto). Both
// pane-path failure shapes — an unreachable tmux server and a provider with no
// session_command — take that same asymmetry; see validatePane.
//
// Everything up to the launch is shared: the two modes differ only in WHICH
// provider command they compose and HOW they start it. The no-cross-fallback
// rule holds in both directions — headless never falls back to session_command,
// pane never falls back to dispatch_command (and the soft fallback is a MODE
// change that re-composes from dispatch_command, not a cross-field fallback).
func runDispatchLaunch(cmd *cobra.Command, changeArg, stage string, timeout int, mode dispatch.Mode, reason dispatch.AutoReason, server string, target paneTarget, prompt promptSource) error {
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

	// Resolve the stage's tier → provider, substituting {model}/{effort} via
	// internal/spawn — the same path fab resolve-agent uses. WHICH of the
	// provider's two command fields is composed depends on the mode; they are
	// never merged and never fall back to each other.
	cfg, err := config.Load(fabRoot)
	if err != nil {
		return err
	}
	profile, err := agent.Resolve(cfg, stage)
	if err != nil {
		return err
	}
	prov, _ := agent.ResolveProvider(cfg, profile.Provider)

	// The third placement input (the other two are env-derived, resolved in
	// resolveMode). Filled here because this is the first point config exists;
	// ignored entirely outside the split shape.
	target.columnWidth = cfg.GetDispatchColumnWidth()

	// Validate pane mode BEFORE composing any command and before any state write,
	// so a pane path that cannot proceed leaves no partial dispatch behind — and,
	// under AUTO selection, can still degrade to headless. Composition is deferred
	// past this point precisely so the degrade is reachable: composing the pane
	// command first would hard-fail a dispatch_command-only provider before the
	// fallback decision, turning a previously-working headless dispatch into an
	// error merely because the caller sits inside tmux.
	if mode == dispatch.ModePane {
		if fallback := validatePane(prov, server, stage, profile.Provider); fallback != nil {
			// Asymmetric by SELECTION SOURCE: an explicit pane selection propagates
			// the error (nothing launched, nothing persisted) because the caller
			// asked for watchability, while an AUTO-selected pane degrades to
			// headless with a one-line notice.
			if reason != dispatch.ReasonAutoTmux {
				return fallback.err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), fallback.notice)
			mode, reason = dispatch.ModeHeadless, fallback.reason
		}
	}

	// The mode is final now, so exactly ONE command is composed. A degrade
	// composes from dispatch_command, so the no-cross-fallback rule survives the
	// mode change: a provider carrying neither field still errors here with the
	// dispatch_command config-key hint.
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
		report, err = launchPane(cmd, rec, resolvedCmd, repoRoot, promptPath, id, stage, server, target)
	} else {
		report, err = launchHeadless(rec, resolvedCmd, repoRoot, dir, stage, promptPath, timeout)
	}
	if err != nil {
		return err
	}

	if err := dispatch.Save(dir, stage, rec); err != nil {
		return err
	}

	// The report names the mode's identity; an AUTO selection appends its source so
	// a surprising mode (or a soft fallback) is explainable from the output alone —
	// the compliance-visibility principle. An explicitly-selected mode's line is
	// byte-identical to before auto selection existed (ReasonExplicit is empty).
	if reason != dispatch.ReasonExplicit {
		report += ", " + string(reason)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "dispatched %s/%s (%s)\n", id, stage, report)
	return nil
}

// paneFallback describes ONE way the pane path cannot proceed, in the two shapes
// the caller needs: the stderr notice + AutoReason an AUTO-selected pane degrades
// with, and the error an EXPLICITLY-selected pane fails with. Bundling them keeps
// the asymmetry a single branch on the selection source rather than one branch per
// failure shape.
type paneFallback struct {
	notice string
	reason dispatch.AutoReason
	err    error
}

// validatePane reports whether pane mode can proceed, returning nil when it can
// and the matching paneFallback when it cannot. It performs no state writes and
// composes no command, so BOTH outcomes are safe before anything is persisted.
// stage/providerName are only ever used to render shape (b)'s hard error.
//
// The two failure shapes, in probe order:
//
//	(a) the tmux server does not answer — a stale $TMUX inherited from a killed
//	    server, or no tmux at all (ServerReachable's actionable error).
//	(b) tmux answered, but the resolved provider carries no session_command —
//	    there is no interactive invocation to open a window on.
//
// Shape (b) is checked HERE rather than at composition time so that an
// auto-selected pane can fall back instead of aborting: a dispatch_command-only
// provider's stages ran headless before auto selection existed and must keep
// running headless inside tmux. The headless path performs no tmux probe at all —
// its tmux-independence guarantee is unchanged.
func validatePane(prov config.ProviderConfig, server, stage, providerName string) *paneFallback {
	if probeErr := dispatch.ServerReachable(server); probeErr != nil {
		return &paneFallback{
			notice: dispatch.FallbackNotice,
			reason: dispatch.ReasonAutoUnreachable,
			err:    probeErr,
		}
	}
	if prov.SessionCommand == "" {
		return &paneFallback{
			notice: dispatch.FallbackNoticeNoSessionCommand,
			reason: dispatch.ReasonAutoNoSessionCommand,
			err:    missingCommandError(stage, providerName, "session_command"),
		}
	}
	return nil
}

// modeCommand picks the provider command field the requested mode composes and
// errors with the matching config-key hint when it is absent.
//
// Under `dispatch start` the pane branch's absent-field error is diagnosed EARLIER
// by validatePane (so an auto selection can fall back instead of aborting), which
// raises the identical missingCommandError. The check is kept here too: it is this
// function's own contract that a mode never composes an empty command, and both
// paths share one message so they cannot drift.
//
// The two fields are deliberately NOT merged and never fall back to each other
// (they are different invocations of the same binary — see the `providers:`
// contract): headless runs ONE headless task via dispatch_command, pane mode
// opens an interactive SESSION via session_command — the same string `fab agent`
// composes, which is why pane mode needs no new provider config field.
func modeCommand(mode dispatch.Mode, prov config.ProviderConfig, stage, providerName string) (string, error) {
	field, cmdStr := "dispatch_command", prov.DispatchCommand
	if mode == dispatch.ModePane {
		field, cmdStr = "session_command", prov.SessionCommand
	}
	if cmdStr == "" {
		return "", missingCommandError(stage, providerName, field)
	}
	return cmdStr, nil
}

// missingCommandError renders the actionable "this provider cannot dispatch this
// stage" error naming the stage, its resolved tier, the provider, and the exact
// config key to set. It lives in one place because two callers raise it —
// modeCommand (at composition) and validatePane (shape (b), which must diagnose
// the missing session_command BEFORE composition so the auto fallback stays
// reachable) — and the two must not drift.
func missingCommandError(stage, providerName, field string) error {
	tier, _ := agent.TierForStage(stage)
	return fmt.Errorf("stage %q resolves to tier %q (provider %q), which has no %s; configure providers.%s.%s to dispatch this stage",
		stage, tier, providerName, field, providerName, field)
}

// launchPane opens the interactive worker in a tmux pane and records the pane
// identity on rec. The worker receives a ONE-LINE POINTER to the persisted prompt
// file as the interactive command's single quoted argument (the `_cli-agents`
// § Spawn Composition form) — the multi-thousand-token prompt itself never rides
// tmux, and embedding at spawn sidesteps the printed-prompt trap entirely.
//
// The pointer path is repo-relative because the new pane's cwd IS the repo root.
// dispatch.WindowCommand shell-quotes the pointer, so a repo path containing a
// single quote cannot break out of the embedded argument.
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
func launchPane(cmd *cobra.Command, rec *dispatch.Dispatch, resolvedCmd, repoRoot, promptPath, id, stage, server string, target paneTarget) (string, error) {
	relPrompt, err := filepath.Rel(repoRoot, promptPath)
	if err != nil {
		relPrompt = promptPath
	}
	title := dispatch.WindowName(id, stage)
	windowCmd := dispatch.WindowCommand(resolvedCmd, dispatch.PointerPrompt(relPrompt))

	var paneID, report string
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
				probeErr, describePlacement(place))
		}
		var warnings []error
		paneID, warnings, err = dispatch.OpenSplitPane(server, place, title, repoRoot, windowCmd)
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
		paneID, err = dispatch.OpenWindow(server, title, repoRoot, windowCmd)
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

// describePlacement renders a resolved SplitPlacement as the human half of the
// degraded-probe warning. It exists so the warning states the OUTCOME in the
// stacked-column vocabulary the rule is documented in ("carving a new column" /
// "stacking under"), rather than leaking the bare tmux `-h`/`-v` flag the placement
// carries.
func describePlacement(place dispatch.SplitPlacement) string {
	if place.Direction == dispatch.SplitRight {
		return fmt.Sprintf("carving a new worker column off pane %s", place.Target)
	}
	return fmt.Sprintf("stacking the worker under pane %s", place.Target)
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
