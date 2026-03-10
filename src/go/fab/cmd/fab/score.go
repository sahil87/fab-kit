package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/resolve"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/score"
)

func scoreCmd() *cobra.Command {
	var checkGate bool
	var stage string

	cmd := &cobra.Command{
		Use:   "score <change>",
		Short: "Compute confidence score from Assumptions table",
		Args:  cobra.ExactArgs(1),
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
	cmd.Flags().StringVar(&stage, "stage", "spec", "Stage for scoring (intake or spec)")

	return cmd
}
