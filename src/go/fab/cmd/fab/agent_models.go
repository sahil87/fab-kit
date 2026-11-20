package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/agentmodels"
	"github.com/spf13/cobra"
)

func agentModelsCmd() *cobra.Command {
	var jsonOutput bool
	var repo string
	cmd := &cobra.Command{
		Use:   "models [cli]",
		Short: "List models offered by agent CLI tools and CLI fills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadRepoConfig(repo)
			if err != nil {
				return err
			}
			names := agent.CLINames(cfg)
			if len(args) == 1 {
				if _, known := agent.ResolveCLI(cfg, args[0]); !known {
					return unknownCLIError(cfg, args[0])
				}
				names = []string{args[0]}
			}

			rows := make([]agentmodels.Row, 0)
			for _, name := range names {
				discovered, warnings := agentmodels.Discover(cfg, name)
				rows = append(rows, discovered...)
				for _, warning := range warnings {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s: %s\n", name, warning)
				}
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
			}
			return writeModelTable(cmd, rows)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print a JSON array of model rows")
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	return cmd
}

func writeModelTable(cmd *cobra.Command, rows []agentmodels.Row) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "CLI\tMODEL\tLABEL\tEFFORTS\tSOURCE\tDEFAULT")
	for _, row := range rows {
		defaultMarker := ""
		if row.Default {
			defaultMarker = "*"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
			row.CLI, row.ID, row.Label, strings.Join(row.Efforts, ","), row.Source, defaultMarker)
	}
	return writer.Flush()
}
