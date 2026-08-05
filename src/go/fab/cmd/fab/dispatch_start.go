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
	var server string
	cmd := &cobra.Command{
		Use:   "start <change> <stage>",
		Short: "Launch a stage worker — headless and detached by default, or interactively in a tmux window with --pane",
		Example: `  # Launch the apply stage's dispatch command headless, prompt on stdin
  fab dispatch start b91h apply < prompt.md

  # Enforce a 30-minute POSIX timeout inside the launch wrapper
  fab dispatch start --timeout 1800 b91h apply < prompt.md

  # Run the stage interactively in a tmux window you can watch and steer
  fab dispatch start b91h review --pane < prompt.md

  # Target a specific tmux socket (works from outside tmux)
  fab dispatch start b91h review --pane --server work < prompt.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --timeout is implemented as POSIX `timeout N` INSIDE the headless
			// `sh -c` wrapper, which pane mode never constructs. Silently ignoring
			// it under --pane would advertise a bound nothing enforces, so the
			// combination is a usage error. Keyed on Flags().Changed (was the flag
			// SUPPLIED) rather than the value, mirroring agent.go's guard style, so
			// an explicit `--timeout 0` is still caught.
			if paneMode && cmd.Flags().Changed("timeout") {
				return fmt.Errorf("--pane and --timeout are mutually exclusive: --timeout is enforced by the headless launch wrapper (POSIX `timeout`), which pane mode does not use")
			}
			return runDispatchStart(cmd, args[0], args[1], timeout, paneMode, server)
		},
	}
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Enforce a POSIX `timeout <secs>` inside the launch wrapper (0 = none); headless only")
	cmd.Flags().BoolVar(&paneMode, "pane", false, "Run the stage worker interactively in a tmux window (requires a reachable tmux server) instead of headless")
	cmd.Flags().StringVarP(&server, "server", "L", "", "Target tmux socket label for --pane (passed as 'tmux -L <name>'). Defaults to $TMUX / tmux default socket; ignored without --pane")
	return cmd
}

// runDispatchStart resolves the stage's tier → provider, refuses if a dispatch
// for this (change, stage) is already running, persists the stdin prompt, then
// launches the worker in one of two MODES and persists {stage}.yaml:
//
//	headless (default) — the provider's dispatch_command, detached via the
//	                     `sh -c` wrapper; tmux-independent (launchHeadless)
//	--pane             — the provider's session_command, interactive in a tmux
//	                     window the user can watch and steer (launchPane)
//
// Everything up to the launch is shared: the two modes differ only in WHICH
// provider command they compose and HOW they start it. The no-cross-fallback
// rule holds in both directions — headless never falls back to session_command,
// pane never falls back to dispatch_command.
func runDispatchStart(cmd *cobra.Command, changeArg, stage string, timeout int, paneMode bool, server string) error {
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
	baseCmd, err := modeCommand(paneMode, prov, stage, profile.Provider)
	if err != nil {
		return err
	}
	resolvedCmd := spawn.WithProfile(baseCmd, profile.Model, profile.Effort)

	// Pane mode requires a reachable tmux server. Probe BEFORE any state write so
	// an unreachable server leaves no partial dispatch behind. The headless path
	// performs no tmux probe at all — its tmux-independence guarantee is unchanged.
	if paneMode {
		if err := dispatch.ServerReachable(server); err != nil {
			return err
		}
	}

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
	if paneMode {
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

	fmt.Fprintf(cmd.OutOrStdout(), "dispatched %s/%s (%s)\n", id, stage, report)
	return nil
}

// modeCommand picks the provider command field the requested mode composes and
// errors with the matching config-key hint when it is absent.
//
// The two fields are deliberately NOT merged and never fall back to each other
// (they are different invocations of the same binary — see the `providers:`
// contract): headless runs ONE headless task via dispatch_command, pane mode
// opens an interactive SESSION via session_command — the same string `fab agent`
// composes, which is why pane mode needs no new provider config field.
func modeCommand(paneMode bool, prov config.ProviderConfig, stage, providerName string) (string, error) {
	field, cmdStr := "dispatch_command", prov.DispatchCommand
	if paneMode {
		field, cmdStr = "session_command", prov.SessionCommand
	}
	if cmdStr == "" {
		tier, _ := agent.TierForStage(stage)
		return "", fmt.Errorf("stage %q resolves to tier %q (provider %q), which has no %s; configure providers.%s.%s to dispatch this stage",
			stage, tier, providerName, field, providerName, field)
	}
	return cmdStr, nil
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
