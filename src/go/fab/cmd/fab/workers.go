package main

import (
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
	"github.com/spf13/cobra"
)

const agentWorkersEnv = "FAB_AGENT_WORKERS"

// workersOverride returns the --workers value together with whether the flag
// was supplied. Flag.Changed deliberately distinguishes an omitted flag from
// --workers= so the CLI remains pure pass-through sugar for the environment.
func workersOverride(cmd *cobra.Command) (string, bool) {
	flag := cmd.Flags().Lookup("workers")
	if flag == nil || !flag.Changed {
		return "", false
	}
	value, _ := cmd.Flags().GetString("workers")
	return value, true
}

// withWorkersEnv prefixes a shell command with a safely quoted environment
// assignment. tmux receives the result as one shell command string.
func withWorkersEnv(command, workers string, set bool) string {
	if !set {
		return command
	}
	return agentWorkersEnv + "=" + shellquote.Single(workers) + " " + command
}

// envWithWorkers returns env with FAB_AGENT_WORKERS set to workers, dropping any
// entry the parent environment already carried. syscall.Exec passes the slice
// through verbatim, so appending alone would leave two entries and duplicate
// resolution is unspecified — a direct exec's getenv takes the FIRST match, while
// a shell rebuilding its own table takes the last. Replacing keeps the override
// authoritative for either consumer.
func envWithWorkers(env []string, workers string) []string {
	prefix := agentWorkersEnv + "="
	out := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		out = append(out, entry)
	}
	return append(out, prefix+workers)
}
