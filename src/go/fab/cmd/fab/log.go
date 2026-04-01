package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/sahil87/fab-kit/src/go/fab/internal/log"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

func logCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Append-only JSON logging to .history.jsonl",
	}

	cmd.AddCommand(logCommandCmd(), logConfidenceCmd(), logReviewCmd(), logTransitionCmd())
	return cmd
}

func logCommandCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "command <cmd> [change] [args]",
		Short: "Log a skill invocation",
		Args:  cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			cmdName := args[0]
			changeArg := ""
			extraArgs := ""
			if len(args) >= 2 {
				changeArg = args[1]
			}
			if len(args) >= 3 {
				extraArgs = args[2]
			}
			return log.Command(fabRoot, cmdName, changeArg, extraArgs)
		},
	}
}

func logConfidenceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confidence <change> <score> <delta> <trigger>",
		Short: "Log a confidence score change",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			score, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return fmt.Errorf("invalid score: %s", args[1])
			}
			return log.ConfidenceLog(fabRoot, args[0], score, args[2], args[3])
		},
	}
}

func logReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review <change> <result> [rework]",
		Short: "Log a review outcome",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			rework := ""
			if len(args) >= 3 {
				rework = args[2]
			}
			return log.Review(fabRoot, args[0], args[1], rework)
		},
	}
}

func logTransitionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "transition <change> <stage> <action> [from] [reason] [driver]",
		Short: "Log a stage transition",
		Args:  cobra.RangeArgs(3, 6),
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			from, reason, driver := "", "", ""
			if len(args) >= 4 {
				from = args[3]
			}
			if len(args) >= 5 {
				reason = args[4]
			}
			if len(args) >= 6 {
				driver = args[5]
			}
			return log.Transition(fabRoot, args[0], args[1], args[2], from, reason, driver)
		},
	}
}
