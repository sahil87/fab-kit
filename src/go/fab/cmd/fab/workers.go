package main

import (
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
