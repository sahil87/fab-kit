package main

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lockfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"github.com/spf13/cobra"
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
		statusPlanCmd(),
		statusConfidenceCmd(),
		statusValidateCmd(),
		statusStartCmd(),
		statusAdvanceCmd(),
		statusFinishCmd(),
		statusResetCmd(),
		statusSkipCmd(),
		statusFailCmd(),
		statusSetChangeTypeCmd(),
		statusSetSummaryCmd(),
		statusGetSummaryCmd(),
		statusSetAcceptanceCmd(),
		statusSetChecklistRemovedCmd(),
		statusSetConfidenceCmd(),
		statusSetConfidenceFuzzyCmd(),
		statusAddIssueCmd(),
		statusGetIssuesCmd(),
		statusAddPRCmd(),
		statusGetPRsCmd(),
	)

	return cmd
}

// loadStatus resolves and loads a .status.yaml for read-only subcommands.
// Readers stay lock-free — Save's temp+rename atomicity guarantees they never
// observe a torn document. Mutating subcommands go through withStatusLock.
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

// withStatusLock resolves the change, then runs fn with a freshly loaded
// StatusFile while holding the .status.yaml sibling flock, so the whole
// load-mutate-save cycle serializes against concurrent writers — the
// artifact-write hook and fab status invocations in other panes — instead of
// last-writer-wins over the whole document (mz4q F03). Every mutating
// subcommand routes through here.
func withStatusLock(changeArg string, fn func(statusFile *sf.StatusFile, statusPath, fabRoot string) error) error {
	fabRoot, err := resolve.FabRoot()
	if err != nil {
		return err
	}
	statusPath, err := resolve.ToAbsStatus(fabRoot, changeArg)
	if err != nil {
		return err
	}
	return lockfile.WithLock(statusPath, func() error {
		statusFile, err := sf.Load(statusPath)
		if err != nil {
			return err
		}
		return fn(statusFile, statusPath, fabRoot)
	})
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

func statusPlanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan <change>",
		Short: "Extract plan fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, statusPath, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			// Acceptance truth: prefer the live count from plan.md `## Acceptance`
			// checkboxes over the persisted counter (the write-time cache), so a
			// hook-bypassing edit (sed, direct edit) cannot make `status plan`
			// report stale acceptance progress. Falls back to the cache when
			// plan.md / its ## Acceptance section is absent. (b)
			acceptanceCompleted := sf.Plan.AcceptanceCompleted
			acceptanceCount := sf.Plan.AcceptanceCount
			if done, total, ok := status.LiveAcceptance(filepath.Dir(statusPath)); ok {
				acceptanceCompleted = done
				acceptanceCount = total
			}
			fmt.Printf("generated:%v\n", sf.Plan.Generated)
			fmt.Printf("task_count:%d\n", sf.Plan.TaskCount)
			fmt.Printf("acceptance_count:%d\n", acceptanceCount)
			fmt.Printf("acceptance_completed:%d\n", acceptanceCompleted)
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
			// confidence.indicative is retired (1.10.0): no longer emitted.
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
			driver, from, reason := optArg(args, 2), optArg(args, 3), optArg(args, 4)
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				return status.Start(st, statusPath, fabRoot, args[1], driver, from, reason)
			})
		},
	}
}

func statusAdvanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "advance <change> <stage> [driver]",
		Short: "active → ready",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			driver := optArg(args, 2)
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.Advance(st, statusPath, args[1], driver)
			})
		},
	}
}

func statusFinishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "finish <change> <stage> [driver]",
		Short: "{active,ready} → done (+auto-activate next)",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			driver := optArg(args, 2)
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				return status.Finish(st, statusPath, fabRoot, args[1], driver)
			})
		},
	}
}

func statusResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset <change> <stage> [driver] [from] [reason]",
		Short: "{done,ready,skipped} → active (+cascade)",
		Args:  cobra.RangeArgs(2, 5),
		RunE: func(cmd *cobra.Command, args []string) error {
			driver, from, reason := optArg(args, 2), optArg(args, 3), optArg(args, 4)
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				return status.Reset(st, statusPath, fabRoot, args[1], driver, from, reason)
			})
		},
	}
}

func statusSkipCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skip <change> <stage> [driver]",
		Short: "{pending,active} → skipped (+cascade)",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			driver := optArg(args, 2)
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				return status.Skip(st, statusPath, fabRoot, args[1], driver)
			})
		},
	}
}

func statusFailCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fail <change> <stage> [driver] [rework]",
		Short: "active → failed (review/review-pr only)",
		Args:  cobra.RangeArgs(2, 4),
		RunE: func(cmd *cobra.Command, args []string) error {
			driver, rework := optArg(args, 2), optArg(args, 3)
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				return status.Fail(st, statusPath, fabRoot, args[1], driver, rework)
			})
		},
	}
}

func statusSetChangeTypeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-change-type <change> <type>",
		Short: "Set change_type",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.SetChangeType(st, statusPath, args[1])
			})
		},
	}
}

func statusSetAcceptanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-acceptance <change> <field> <value>",
		Short: "Update plan field",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.SetAcceptance(st, statusPath, args[1], args[2])
			})
		},
	}
}

// statusSetChecklistRemovedCmd surfaces the strict-error message for the
// removed `set-checklist` command. The Cobra layer matches the literal
// command name and returns the pointer-to-set-acceptance error.
func statusSetChecklistRemovedCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "set-checklist [args...]",
		Short:  "Removed — use set-acceptance",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return status.SetChecklistRemovedError()
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
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.SetConfidence(st, statusPath, certain, confident, tentative, unresolved, score)
			})
		},
	}

	// --indicative is retired (1.10.0): accepted-but-ignored no-op for one
	// release so existing scripts do not break. It writes nothing.
	cmd.Flags().BoolVar(&indicative, "indicative", false, "Deprecated no-op (retained for script back-compat)")
	return cmd
}

func statusSetConfidenceFuzzyCmd() *cobra.Command {
	var indicative bool

	cmd := &cobra.Command{
		Use:   "set-confidence-fuzzy <change> <certain> <confident> <tentative> <unresolved> <score> <mean_s> <mean_r> <mean_a> <mean_d>",
		Short: "Replace confidence block with dimensions",
		Args:  cobra.ExactArgs(10),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.SetConfidenceFuzzy(st, statusPath, certain, confident, tentative, unresolved, score, meanS, meanR, meanA, meanD)
			})
		},
	}

	// --indicative is retired (1.10.0): accepted-but-ignored no-op (see set-confidence).
	cmd.Flags().BoolVar(&indicative, "indicative", false, "Deprecated no-op (retained for script back-compat)")
	return cmd
}

func statusAddIssueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-issue <change> <id>",
		Short: "Append issue ID (idempotent)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.AddIssue(st, statusPath, args[1])
			})
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
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.AddPR(st, statusPath, args[1])
			})
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

func statusSetSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-summary <change> <text>",
		Short: "Set the per-change log summary",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, _ string) error {
				return status.SetSummary(st, statusPath, args[1])
			})
		},
	}
}

func statusGetSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-summary <change>",
		Short: "Print the per-change log summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			// Empty summary prints an empty line (graceful absence — the FKF
			// log.md generator falls back to the change slug).
			fmt.Println(sf.Summary)
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
