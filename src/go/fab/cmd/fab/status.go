package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lockfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/refresh"
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
		statusRefreshCmd(),
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

// encodeJSON writes v to the command's stdout as indented JSON (two-space
// indent, trailing newline via Encode), matching the `fab dispatch status
// --json` precedent (dispatchStatusJSON). Shared by the read-only `fab status`
// query subcommands' --json branches so the encoder mechanics are single-sourced.
func encodeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// JSON output shapes for the read-only `fab status` query subcommands. snake_case
// keys match the .status.yaml field names. Ordered/list subcommands emit bare
// arrays (a Go map would marshal alphabetically and destroy stage order) and are
// not modeled as structs here.
type (
	confidenceJSON struct {
		Certain    int     `json:"certain"`
		Confident  int     `json:"confident"`
		Tentative  int     `json:"tentative"`
		Unresolved int     `json:"unresolved"`
		Score      float64 `json:"score"`
	}
	planJSON struct {
		Generated           bool `json:"generated"`
		TaskCount           int  `json:"task_count"`
		AcceptanceCount     int  `json:"acceptance_count"`
		AcceptanceCompleted int  `json:"acceptance_completed"`
	}
	stageStateJSON struct {
		Stage string `json:"stage"`
		State string `json:"state"`
	}
	currentStageJSON struct {
		Stage string `json:"stage"`
	}
	summaryJSON struct {
		Summary string `json:"summary"`
	}
)

func statusAllStagesCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "all-stages",
		Short: "List all stage IDs in order",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			stages := status.AllStages()
			if jsonFlag {
				return encodeJSON(cmd, stages)
			}
			for _, s := range stages {
				fmt.Println(s)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func statusProgressMapCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "progress-map <change>",
		Short: "Extract stage:state pairs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			pairs := status.ProgressMap(sf)
			if jsonFlag {
				// Array (not a map) so pipeline stage order is preserved.
				out := make([]stageStateJSON, 0, len(pairs))
				for _, ss := range pairs {
					out = append(out, stageStateJSON{Stage: ss.Stage, State: ss.State})
				}
				return encodeJSON(cmd, out)
			}
			for _, ss := range pairs {
				fmt.Printf("%s:%s\n", ss.Stage, ss.State)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
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
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "current-stage <change>",
		Short: "Detect active stage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			stage := status.CurrentStage(sf)
			if jsonFlag {
				return encodeJSON(cmd, currentStageJSON{Stage: stage})
			}
			fmt.Println(stage)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func statusDisplayStageCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "display-stage <change>",
		Short: "Display stage as stage:state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			stage, state := status.DisplayStage(sf)
			if jsonFlag {
				return encodeJSON(cmd, stageStateJSON{Stage: stage, State: state})
			}
			fmt.Printf("%s:%s\n", stage, state)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func statusPlanCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
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
			//
			// Compute once here; the text and --json paths below both render this
			// single source of truth (so the two rendering paths cannot drift).
			acceptanceCompleted := sf.Plan.AcceptanceCompleted
			acceptanceCount := sf.Plan.AcceptanceCount
			if done, total, ok := status.LiveAcceptance(filepath.Dir(statusPath)); ok {
				acceptanceCompleted = done
				acceptanceCount = total
			}
			if jsonFlag {
				return encodeJSON(cmd, planJSON{
					Generated:           sf.Plan.Generated,
					TaskCount:           sf.Plan.TaskCount,
					AcceptanceCount:     acceptanceCount,
					AcceptanceCompleted: acceptanceCompleted,
				})
			}
			fmt.Printf("generated:%v\n", sf.Plan.Generated)
			fmt.Printf("task_count:%d\n", sf.Plan.TaskCount)
			fmt.Printf("acceptance_count:%d\n", acceptanceCount)
			fmt.Printf("acceptance_completed:%d\n", acceptanceCompleted)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func statusConfidenceCmd() *cobra.Command {
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "confidence <change>",
		Short: "Extract confidence fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			if jsonFlag {
				return encodeJSON(cmd, confidenceJSON{
					Certain:    sf.Confidence.Certain,
					Confident:  sf.Confidence.Confident,
					Tentative:  sf.Confidence.Tentative,
					Unresolved: sf.Confidence.Unresolved,
					Score:      sf.Confidence.Score,
				})
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
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
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
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				// Self-heal artifact-derived fields before the forward
				// transition, so a just-written intake.md/plan.md is reflected
				// before the next stage reads them (the pull-based successor to
				// the removed artifact-write hook). The recompute and the
				// transition persist in this same locked load/Save.
				selfHealRefresh(fabRoot, statusPath, st)
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
				// Self-heal artifact-derived fields before the forward
				// transition (see statusAdvanceCmd). Finish's own Save persists
				// both the heal and the transition.
				selfHealRefresh(fabRoot, statusPath, st)
				return status.Finish(st, statusPath, fabRoot, args[1], driver)
			})
		},
	}
}

// statusRefreshCmd recomputes the artifact-derived .status.yaml fields
// (change_type + confidence from intake.md; plan.generated/task_count/
// acceptance counts from plan.md) from the on-disk artifacts. It is the
// pull-based successor to the removed artifact-write PostToolUse hook: a
// hook-bypassing edit (sed, direct write) or a non-Claude agent that never
// fires the hook can no longer leave these fields stale. Respects
// change_type_source: explicit; a missing artifact is a safe no-op.
func statusRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh <change>",
		Short: "Recompute change_type/confidence (intake.md) + plan counts (plan.md) from artifacts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withStatusLock(args[0], func(st *sf.StatusFile, statusPath, fabRoot string) error {
				changeDir := filepath.Dir(statusPath)
				dirty, err := refresh.Refresh(fabRoot, changeDir, st)
				if err != nil {
					return err
				}
				if dirty {
					return st.Save(statusPath)
				}
				return nil
			})
		},
	}
}

// selfHealRefresh runs the artifact-derived recompute inside a transition's
// already-held status lock, mutating the in-memory StatusFile so the following
// transition's own Save persists both the heal and the transition in one write
// (no second Save). It is best-effort: a refresh error MUST NOT abort the
// transition (advance/finish own the state machine; a scoring hiccup on a
// just-written artifact should not block a stage move), matching the removed
// hook's swallow-on-error posture. The dirty flag is intentionally ignored —
// the transition Saves unconditionally, and refresh's in-memory mutations ride
// along.
func selfHealRefresh(fabRoot, statusPath string, st *sf.StatusFile) {
	changeDir := filepath.Dir(statusPath)
	_, _ = refresh.Refresh(fabRoot, changeDir, st)
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
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "get-issues <change>",
		Short: "List issue IDs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			if jsonFlag {
				// Non-nil slice so an empty list marshals as [] (never null).
				ids := make([]string, 0, len(sf.Issues))
				ids = append(ids, sf.Issues...)
				return encodeJSON(cmd, ids)
			}
			for _, id := range sf.Issues {
				fmt.Println(id)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
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
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "get-prs <change>",
		Short: "List PR URLs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			if jsonFlag {
				// Non-nil slice so an empty list marshals as [] (never null).
				urls := make([]string, 0, len(sf.PRs))
				urls = append(urls, sf.PRs...)
				return encodeJSON(cmd, urls)
			}
			for _, url := range sf.PRs {
				fmt.Println(url)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
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
	var jsonFlag bool
	cmd := &cobra.Command{
		Use:   "get-summary <change>",
		Short: "Print the per-change log summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, _, _, err := loadStatus(args[0])
			if err != nil {
				return err
			}
			if jsonFlag {
				// Object-wrapped (not a bare string) so fields can be added
				// additively; an empty summary emits {"summary":""}.
				return encodeJSON(cmd, summaryJSON{Summary: sf.Summary})
			}
			// Empty summary prints an empty line (graceful absence — the FKF
			// log.md generator falls back to the change slug).
			fmt.Println(sf.Summary)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func optArg(args []string, idx int) string {
	if idx < len(args) {
		return args[idx]
	}
	return ""
}
