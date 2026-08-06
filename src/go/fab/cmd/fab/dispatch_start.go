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

func dispatchStartCmd() *cobra.Command {
	var timeout int
	var paneMode bool
	var headlessMode bool
	var server string
	cmd := &cobra.Command{
		Use:   "start <change> <stage>",
		Short: "Launch a stage worker — mode defaults to auto (a tmux window inside tmux, headless outside); force with --pane / --headless",
		Example: `  # Auto mode: a watchable tmux window inside tmux, headless outside
  fab dispatch start b91h apply < prompt.md

  # Force headless for an unattended run that happens to live inside a tmux tab
  fab dispatch start b91h apply --headless < prompt.md

  # Enforce a 30-minute POSIX timeout (implies headless)
  fab dispatch start --timeout 1800 b91h apply < prompt.md

  # Force the stage into a tmux window you can watch and steer
  fab dispatch start b91h review --pane < prompt.md

  # Target a specific tmux socket (implies pane; works from outside tmux)
  fab dispatch start b91h review --server work < prompt.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --timeout is implemented as POSIX `timeout N` INSIDE the headless
			// `sh -c` wrapper, which pane mode never constructs. Silently ignoring
			// it under --pane would advertise a bound nothing enforces, so the
			// combination is a usage error. Keyed on Flags().Changed (was the flag
			// SUPPLIED) rather than the value, mirroring agent.go's guard style, so
			// an explicit `--timeout 0` is still caught.
			//
			// Only the EXPLICIT --pane conflicts: --timeout is itself a headless
			// signal in the selection ladder, so `--timeout` inside tmux selects
			// headless rather than erroring (it must not break scripted invocations
			// that never mention panes).
			if paneMode && cmd.Flags().Changed("timeout") {
				return fmt.Errorf("--pane and --timeout are mutually exclusive: --timeout is enforced by the headless launch wrapper (POSIX `timeout`), which pane mode does not use")
			}
			// Resolve the mode from the explicit-first ladder; the caller-supplied
			// signals are read as "was the flag SUPPLIED" so `--timeout 0` /
			// `--server ""` still count.
			mode, reason := dispatch.SelectMode(paneMode, headlessMode,
				cmd.Flags().Changed("timeout"), cmd.Flags().Changed("server"), os.Getenv("TMUX"))
			return runDispatchStart(cmd, args[0], args[1], timeout, mode, reason, server)
		},
	}
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Enforce a POSIX `timeout <secs>` inside the launch wrapper (0 = none); implies --headless")
	cmd.Flags().BoolVar(&paneMode, "pane", false, "Force an interactive tmux-window worker (requires a reachable tmux server) instead of the auto default")
	cmd.Flags().BoolVar(&headlessMode, "headless", false, "Force a detached headless worker, opting out of auto pane selection inside tmux")
	cmd.Flags().StringVarP(&server, "server", "L", "", "Target tmux socket label (passed as 'tmux -L <name>'); implies pane mode. Defaults to $TMUX / tmux default socket; ignored in headless mode")
	// Cobra's flag group fires during ValidateFlagGroups — BEFORE any RunE work —
	// so the conflict is a genuine usage error (exit 2) that structurally cannot
	// leave partial state behind, matching resolve.go / pane_capture.go / panemap.go.
	cmd.MarkFlagsMutuallyExclusive("pane", "headless")
	return cmd
}

// runDispatchStart resolves the stage's tier → provider, refuses if a dispatch
// for this (change, stage) is already running, persists the stdin prompt, then
// launches the worker in the ALREADY-RESOLVED mode and persists {stage}.yaml:
//
//	dispatch.ModeHeadless — the provider's dispatch_command, detached via the
//	                        `sh -c` wrapper; tmux-independent (launchHeadless)
//	dispatch.ModePane     — the provider's session_command, interactive in a tmux
//	                        window the user can watch and steer (launchPane)
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
func runDispatchStart(cmd *cobra.Command, changeArg, stage string, timeout int, mode dispatch.Mode, reason dispatch.AutoReason, server string) error {
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

	// Persist the prompt from stdin BEFORE launching: the headless wrapper
	// redirects it into the command's stdin, and pane mode points the interactive
	// worker at it (the file must exist by the time the worker reads the pointer).
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dispatch dir: %w", err)
	}
	promptPath := dispatch.PromptPath(dir, stage)
	prompt, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return fmt.Errorf("read prompt from stdin: %w", err)
	}
	if err := os.WriteFile(promptPath, prompt, 0o644); err != nil {
		return fmt.Errorf("write prompt: %w", err)
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
		report, err = launchPane(rec, resolvedCmd, repoRoot, promptPath, id, stage, server)
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

// launchPane opens the interactive worker in a tmux window and records the pane
// identity on rec. The worker receives a ONE-LINE POINTER to the persisted prompt
// file as the interactive command's single quoted argument (the `_cli-agents`
// § Spawn Composition form) — the multi-thousand-token prompt itself never rides
// tmux, and embedding at spawn sidesteps the printed-prompt trap entirely.
//
// The pointer path is repo-relative because the window's cwd IS the repo root.
// dispatch.WindowCommand shell-quotes the pointer, so a repo path containing a
// single quote cannot break out of the embedded argument.
func launchPane(rec *dispatch.Dispatch, resolvedCmd, repoRoot, promptPath, id, stage, server string) (string, error) {
	relPrompt, err := filepath.Rel(repoRoot, promptPath)
	if err != nil {
		relPrompt = promptPath
	}
	window := dispatch.WindowName(id, stage)
	windowCmd := dispatch.WindowCommand(resolvedCmd, dispatch.PointerPrompt(relPrompt))

	paneID, err := dispatch.OpenWindow(server, window, repoRoot, windowCmd)
	if err != nil {
		return "", err
	}
	rec.Pane = paneID
	rec.Window = window
	rec.Server = server
	return fmt.Sprintf("pane %s, window %s", paneID, window), nil
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
