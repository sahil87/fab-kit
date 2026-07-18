package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/score"
	"github.com/spf13/cobra"
)

func scoreCmd() *cobra.Command {
	var checkGate bool
	var stage string

	cmd := &cobra.Command{
		Use:   "score <change>",
		Short: "Compute confidence score from Assumptions table",
		Example: `  # Compute and persist the intake confidence score
  fab score b91h

  # Read-only gate check — exits non-zero below the threshold
  fab score --check-gate --stage intake b91h`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}

			if checkGate {
				result, err := score.CheckGate(fabRoot, args[0], stage)
				if err != nil {
					return err
				}
				fmt.Println(score.FormatGateYAML(result))
				if result.Gate == "fail" {
					// The gate result must be observable via exit code — the
					// /fab-ff and /fab-fff intake gate keys on it. The YAML
					// report stays on stdout; the error reaches stderr via
					// main's handler.
					return fmt.Errorf("intake gate failed: score %.1f below threshold %.1f", result.Score, result.Threshold)
				}
				return nil
			}

			result, err := score.Compute(fabRoot, args[0], stage)
			if err != nil {
				return err
			}
			fmt.Print(score.FormatScoreYAML(result))
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkGate, "check-gate", false, "Gate check mode (read-only)")
	cmd.Flags().StringVar(&stage, "stage", "intake", "Stage for scoring (intake; spec retired in 1.10.0)")

	return cmd
}
