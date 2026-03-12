package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/resolve"
	"github.com/wvrdz/fab-kit/src/go/fab/internal/status"
	sf "github.com/wvrdz/fab-kit/src/go/fab/internal/statusfile"
)

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Manage workflow stages, states, and .status.yaml",
	}

	cmd.AddCommand(
		statusAllStagesCmd(),
		statusProgressMapCmd(),
		statusProgressLineCmd(),
		statusCurrentStageCmd(),
		statusDisplayStageCmd(),
		statusChecklistCmd(),
		statusConfidenceCmd(),
		statusValidateCmd(),
		statusStartCmd(),
		statusAdvanceCmd(),
		statusFinishCmd(),
		statusResetCmd(),
		statusSkipCmd(),
		statusFailCmd(),
		statusSetChangeTypeCmd(),
		statusSetChecklistCmd(),
		statusSetConfidenceCmd(),
		statusSetConfidenceFuzzyCmd(),
		statusAddIssueCmd(),
		statusGetIssuesCmd(),
		statusAddPRCmd(),
		statusGetPRsCmd(),
	)

	return cmd
}

func loadStatus(changeArg string) (*sf.StatusFile, string, string, error) {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return nil, "", "", err
	}
	statusPath, err := resolve.ToAbsStatus(fabRoot, changeArg)
	if err != nil {
		return nil, "", "", err
	}
	statusFile, err := sf.Load(statusPath)
	if err != nil {
		return nil, "", "", err
	}
	return statusFile, statusPath, fabRoot, nil
}

func statusAllStagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all-stages",
		Short: "List all stage IDs in order",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, s := range status.AllStages() {
				fmt.Println(s)
			}
			return nil
		},
	}
}

func statusProgressMapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "progress-map <change>",
		Short: "Extract stage:state pairs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			for _, ss := range status.ProgressMap(sf) {
				fmt.Printf("%s:%s\n", ss.Stage, ss.State)
			}
			return nil
		},
	}
}

func statusProgressLineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "progress-line <change>",
		Short: "Single-line visual progress",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			line := status.ProgressLine(sf)
			if line != "" {
				fmt.Println(line)
			}
			return nil
		},
	}
}

func statusCurrentStageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current-stage <change>",
		Short: "Detect active stage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			fmt.Println(status.CurrentStage(sf))
			return nil
		},
	}
}

func statusDisplayStageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "display-stage <change>",
		Short: "Display stage as stage:state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			stage, state := status.DisplayStage(sf)
			fmt.Printf("%s:%s\n", stage, state)
			return nil
		},
	}
}

func statusChecklistCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checklist <change>",
		Short: "Extract checklist fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("generated:%v\n", sf.Checklist.Generated)
			fmt.Printf("completed:%d\n", sf.Checklist.Completed)
			fmt.Printf("total:%d\n", sf.Checklist.Total)
			return nil
		},
	}
}

func statusConfidenceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confidence <change>",
		Short: "Extract confidence fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("certain:%d\n", sf.Confidence.Certain)
			fmt.Printf("confident:%d\n", sf.Confidence.Confident)
			fmt.Printf("tentative:%d\n", sf.Confidence.Tentative)
			fmt.Printf("unresolved:%d\n", sf.Confidence.Unresolved)
			fmt.Printf("score:%.1f\n", sf.Confidence.Score)
			indicative := false
			if sf.Confidence.Indicative != nil && *sf.Confidence.Indicative {
				indicative = true
			}
			fmt.Printf("indicative:%v\n", indicative)
			return nil
		},
	}
}

func statusValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-status-file <change>",
		Short: "Validate .status.yaml against schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			return status.Validate(sf)
		},
	}
}

func statusStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <change> <stage> [driver] [from] [reason]",
		Short: "{pending,failed} → active",
		Args:  cobra.RangeArgs(2, 5),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, fabRoot, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			driver, from, reason := optArg(args, 2), optArg(args, 3), optArg(args, 4)
			return status.Start(sf, statusPath, fabRoot, args[1], driver, from, reason)
		},
	}
}

func statusAdvanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "advance <change> <stage> [driver]",
		Short: "active → ready",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			driver := optArg(args, 2)
			return status.Advance(sf, statusPath, args[1], driver)
		},
	}
}

func statusFinishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finish <change> <stage> [driver]",
		Short: "{active,ready} → done (+auto-activate next)",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, fabRoot, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			driver := optArg(args, 2)
			return status.Finish(sf, statusPath, fabRoot, args[1], driver)
		},
	}
}

func statusResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <change> <stage> [driver] [from] [reason]",
		Short: "{done,ready,skipped} → active (+cascade)",
		Args:  cobra.RangeArgs(2, 5),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, fabRoot, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			driver, from, reason := optArg(args, 2), optArg(args, 3), optArg(args, 4)
			return status.Reset(sf, statusPath, fabRoot, args[1], driver, from, reason)
		},
	}
}

func statusSkipCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skip <change> <stage> [driver]",
		Short: "{pending,active} → skipped (+cascade)",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, fabRoot, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			driver := optArg(args, 2)
			return status.Skip(sf, statusPath, fabRoot, args[1], driver)
		},
	}
}

func statusFailCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fail <change> <stage> [driver] [rework]",
		Short: "active → failed (review/review-pr only)",
		Args:  cobra.RangeArgs(2, 4),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, fabRoot, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			driver, rework := optArg(args, 2), optArg(args, 3)
			return status.Fail(sf, statusPath, fabRoot, args[1], driver, rework)
		},
	}
}

func statusSetChangeTypeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-change-type <change> <type>",
		Short: "Set change_type",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			return status.SetChangeType(sf, statusPath, args[1])
		},
	}
}

func statusSetChecklistCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-checklist <change> <field> <value>",
		Short: "Update checklist field",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			return status.SetChecklist(sf, statusPath, args[1], args[2])
		},
	}
}

func statusSetConfidenceCmd() *cobra.Command {
	var indicative bool

	cmd := &cobra.Command{
		Use:   "set-confidence <change> <certain> <confident> <tentative> <unresolved> <score>",
		Short: "Replace confidence block",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			certain, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid value for 'certain' (%q): %w", args[1], err)
			}
			confident, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid value for 'confident' (%q): %w", args[2], err)
			}
			tentative, err := strconv.Atoi(args[3])
			if err != nil {
				return fmt.Errorf("invalid value for 'tentative' (%q): %w", args[3], err)
			}
			unresolved, err := strconv.Atoi(args[4])
			if err != nil {
				return fmt.Errorf("invalid value for 'unresolved' (%q): %w", args[4], err)
			}
			score, err := strconv.ParseFloat(args[5], 64)
			if err != nil {
				return fmt.Errorf("invalid value for 'score' (%q): %w", args[5], err)
			}
			return status.SetConfidence(sf, statusPath, certain, confident, tentative, unresolved, score, indicative)
		},
	}

	cmd.Flags().BoolVar(&indicative, "indicative", false, "Mark score as indicative (from intake)")
	return cmd
}

func statusSetConfidenceFuzzyCmd() *cobra.Command {
	var indicative bool

	cmd := &cobra.Command{
		Use:   "set-confidence-fuzzy <change> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>",
		Short: "Replace confidence block with dimensions",
		Args:  cobra.ExactArgs(10),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			certain, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid value for 'certain' (%q): %w", args[1], err)
			}
			confident, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid value for 'confident' (%q): %w", args[2], err)
			}
			tentative, err := strconv.Atoi(args[3])
			if err != nil {
				return fmt.Errorf("invalid value for 'tentative' (%q): %w", args[3], err)
			}
			unresolved, err := strconv.Atoi(args[4])
			if err != nil {
				return fmt.Errorf("invalid value for 'unresolved' (%q): %w", args[4], err)
			}
			score, err := strconv.ParseFloat(args[5], 64)
			if err != nil {
				return fmt.Errorf("invalid value for 'score' (%q): %w", args[5], err)
			}
			meanS, err := strconv.ParseFloat(args[6], 64)
			if err != nil {
				return fmt.Errorf("invalid value for 'mean_s' (%q): %w", args[6], err)
			}
			meanR, err := strconv.ParseFloat(args[7], 64)
			if err != nil {
				return fmt.Errorf("invalid value for 'mean_r' (%q): %w", args[7], err)
			}
			meanA, err := strconv.ParseFloat(args[8], 64)
			if err != nil {
				return fmt.Errorf("invalid value for 'mean_a' (%q): %w", args[8], err)
			}
			meanD, err := strconv.ParseFloat(args[9], 64)
			if err != nil {
				return fmt.Errorf("invalid value for 'mean_d' (%q): %w", args[9], err)
			}
			return status.SetConfidenceFuzzy(sf, statusPath, certain, confident, tentative, unresolved, score, meanS, meanR, meanA, meanD, indicative)
		},
	}

	cmd.Flags().BoolVar(&indicative, "indicative", false, "Mark score as indicative (from intake)")
	return cmd
}

func statusAddIssueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-issue <change> <id>",
		Short: "Append issue ID (idempotent)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			return status.AddIssue(sf, statusPath, args[1])
		},
	}
}

func statusGetIssuesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-issues <change>",
		Short: "List issue IDs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			for _, id := range sf.Issues {
				fmt.Println(id)
			}
			return nil
		},
	}
}

func statusAddPRCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-pr <change> <url>",
		Short: "Append PR URL (idempotent)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			return status.AddPR(sf, statusPath, args[1])
		},
	}
}

func statusGetPRsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-prs <change>",
		Short: "List PR URLs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			for _, url := range sf.PRs {
				fmt.Println(url)
			}
			return nil
		},
	}
}

func optArg(args []string, idx int) string {
	if idx < len(args) {
		return args[idx]
	}
	return ""
}


