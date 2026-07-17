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
	cmd := &cobra.Command{
		Use:   "start <change> <stage>",
		Short: "Launch a stage's resolved spawn command detached, reading the prompt on stdin",
		Example: `  # Launch the apply stage's dispatch command, prompt on stdin
  fab dispatch start b91h apply < prompt.md

  # Enforce a 30-minute POSIX timeout inside the launch wrapper
  fab dispatch start --timeout 1800 b91h apply < prompt.md`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDispatchStart(cmd, args[0], args[1], timeout)
		},
	}
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Enforce a POSIX `timeout <secs>` inside the launch wrapper (0 = none)")
	return cmd
}

// runDispatchStart resolves the stage's tier → provider → dispatch_command,
// refuses if a dispatch for this (change, stage) is already running, launches the
// command detached, and persists {stage}.yaml. The prompt is read from stdin
// (cmd.InOrStdin) into {stage}-prompt.md. A resolved tier whose provider has no
// dispatch_command is an error (no fallback to a session command).
func runDispatchStart(cmd *cobra.Command, changeArg, stage string, timeout int) error {
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

	// Resolve the stage's tier → provider → dispatch_command, substituting
	// {model}/{effort} via internal/spawn — the same path fab resolve-agent uses.
	cfg, err := config.Load(fabRoot)
	if err != nil {
		return err
	}
	profile, err := agent.Resolve(cfg, stage)
	if err != nil {
		return err
	}
	prov, _ := agent.ResolveProvider(cfg, profile.Provider)
	if prov.DispatchCommand == "" {
		tier, _ := agent.TierForStage(stage)
		return fmt.Errorf("stage %q resolves to tier %q (provider %q), which has no dispatch_command; configure providers.%s.dispatch_command to dispatch this stage", stage, tier, profile.Provider, profile.Provider)
	}
	resolvedCmd := spawn.WithProfile(prov.DispatchCommand, profile.Model, profile.Effort)

	// Refuse-if-running: a live prior dispatch for this exact (change, stage)
	// must be killed first. A completed prior attempt (done/failed/orphaned) is
	// overwritten — last-attempt-only, no per-attempt history.
	if prior, err := dispatch.Load(dir, stage); err == nil {
		exitPresent, _, exErr := dispatch.ReadExit(dir, stage)
		if exErr != nil {
			return exErr
		}
		if !exitPresent && dispatch.Alive(prior.PID) {
			return fmt.Errorf("a dispatch for %s/%s is already running (pid %d); run `fab dispatch kill` first", changeArg, stage, prior.PID)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	// Persist the prompt from stdin BEFORE launching (the wrapper redirects it
	// into the command's stdin).
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

	// Launch detached: setsid sh -c '<cmd> < prompt > log 2>&1; echo $? > exit'.
	argv := dispatch.WrapperArgv(resolvedCmd,
		promptPath, dispatch.LogPath(dir, stage), dispatch.ExitPath(dir, stage), timeout)
	pid, pgid, err := dispatch.Launch(argv, repoRoot)
	if err != nil {
		return err
	}

	rec := &dispatch.Dispatch{
		PID:       pid,
		PGID:      pgid,
		SpawnCmd:  resolvedCmd,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Timeout:   timeout,
	}
	if err := dispatch.Save(dir, stage, rec); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "dispatched %s/%s (pid %d, pgid %d)\n", id, stage, pid, pgid)
	return nil
}
