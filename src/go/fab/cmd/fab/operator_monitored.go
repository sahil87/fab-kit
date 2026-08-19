package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Monitored-set verbs (fab-operator.md §4 Monitored Set). enroll owns BOTH the
// monitored entry and its branch_map entry — enrollment is the documented
// moment branch_map gains its pair. remove retains branch_map (documented
// persistence policy: downstream dependency resolution needs it). The »/›
// window-name renames stay separate `fab pane window-name` primitives — these
// verbs touch only the state file.

func operatorEnrollCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enroll <change-id>",
		Short: "Create (or replace) a monitored entry and record its branch_map pair",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorEnroll,
	}
	cmd.Flags().String("pane", "", "pane ID the agent runs in (required)")
	cmd.Flags().String("repo", "", "absolute main-worktree root of the agent's repo (required)")
	cmd.Flags().String("session", "", "tmux session the agent's window lives in (required)")
	cmd.Flags().String("branch", "", "the change's branch name (required)")
	cmd.Flags().String("stage", "", "current pipeline stage")
	cmd.Flags().String("agent", "", "last-known agent state (passed through verbatim)")
	cmd.Flags().String("stop-stage", "", "stage to park at (default: full pipeline)")
	cmd.Flags().String("spawned-by", "", "watch name if spawned by a watch")
	cmd.Flags().StringSlice("depends-on", nil, "comma-separated change IDs this change depends on")
	_ = cmd.MarkFlagRequired("pane")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("session")
	_ = cmd.MarkFlagRequired("branch")
	return cmd
}

func runOperatorEnroll(cmd *cobra.Command, args []string) error {
	id := args[0]
	f := cmd.Flags()
	pane, _ := f.GetString("pane")
	repo, _ := f.GetString("repo")
	session, _ := f.GetString("session")
	branch, _ := f.GetString("branch")
	stage, _ := f.GetString("stage")
	agent, _ := f.GetString("agent")
	stopStage, err := optionalStageFlag(cmd, "stop-stage")
	if err != nil {
		return err
	}
	spawnedBy, _ := f.GetString("spawned-by")
	dependsOn, _ := f.GetStringSlice("depends-on")
	if stage != "" && !validStage(stage) {
		return fmt.Errorf("invalid --stage %q (valid: intake, apply, review, hydrate, ship, review-pr)", stage)
	}

	now := nowRFC3339()
	entry := monitoredEntry{
		Pane:           pane,
		Repo:           repo,
		Session:        session,
		Stage:          stage,
		Agent:          agent,
		StopStage:      stopStage,
		SpawnedBy:      nil,
		DependsOn:      dependsOn,
		Branch:         branch,
		EnrolledAt:     now,
		LastTransition: now,
	}
	if spawnedBy != "" {
		entry.SpawnedBy = &spawnedBy
	}
	if entry.DependsOn == nil {
		entry.DependsOn = []string{}
	}

	return mutateOperatorState(func(data map[string]interface{}) error {
		monitored := map[string]monitoredEntry{}
		if err := operatorSection(data, "monitored", &monitored); err != nil {
			return err
		}
		// Wholesale replace on re-enroll — fresh enrolled_at is the honest
		// reading of a re-enrollment.
		monitored[id] = entry
		data["monitored"] = monitored

		bm := map[string]branchMapEntry{}
		if err := operatorSection(data, "branch_map", &bm); err != nil {
			return err
		}
		bm[id] = branchMapEntry{Branch: branch, Repo: repo}
		data["branch_map"] = bm
		return nil
	})
}

func operatorUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <change-id>",
		Short: "Mutate per-tick observed fields of a monitored entry",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorUpdate,
	}
	cmd.Flags().String("stage", "", "current pipeline stage")
	cmd.Flags().String("agent", "", "last-known agent state (passed through verbatim)")
	cmd.Flags().String("stop-stage", "", "stage to park at (empty string clears)")
	return cmd
}

func runOperatorUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]
	f := cmd.Flags()
	stage, _ := f.GetString("stage")
	agent, _ := f.GetString("agent")
	stopStage, err := optionalStageFlag(cmd, "stop-stage")
	if err != nil {
		return err
	}
	if f.Changed("stage") && stage != "" && !validStage(stage) {
		return fmt.Errorf("invalid --stage %q (valid: intake, apply, review, hydrate, ship, review-pr)", stage)
	}

	return mutateOperatorState(func(data map[string]interface{}) error {
		monitored := map[string]monitoredEntry{}
		if err := operatorSection(data, "monitored", &monitored); err != nil {
			return err
		}
		entry, ok := monitored[id]
		if !ok {
			return fmt.Errorf("no monitored entry for %s", id)
		}
		if f.Changed("stage") {
			if entry.Stage != stage {
				entry.Stage = stage
				// last_transition moves only when the stored stage changes.
				entry.LastTransition = nowRFC3339()
			}
		}
		if f.Changed("agent") {
			entry.Agent = agent
		}
		if f.Changed("stop-stage") {
			entry.StopStage = stopStage
		}
		monitored[id] = entry
		data["monitored"] = monitored
		return nil
	})
}

func operatorRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <change-id>",
		Short: "Remove a monitored entry (its branch_map entry is retained)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorRemove,
	}
}

func runOperatorRemove(cmd *cobra.Command, args []string) error {
	id := args[0]
	return mutateOperatorState(func(data map[string]interface{}) error {
		monitored := map[string]monitoredEntry{}
		if err := operatorSection(data, "monitored", &monitored); err != nil {
			return err
		}
		if _, ok := monitored[id]; !ok {
			return fmt.Errorf("no monitored entry for %s", id)
		}
		delete(monitored, id)
		data["monitored"] = monitored
		// branch_map deliberately untouched — entries persist for downstream
		// dependency resolution until explicitly cleared (branch-map rm).
		return nil
	})
}

// optionalStageFlag reads an optional stage-valued flag: absent → nil (leave
// untouched); present-but-empty → nil (clear to null); present → validated
// against the six stage names and returned as a pointer.
func optionalStageFlag(cmd *cobra.Command, name string) (*string, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	v, _ := cmd.Flags().GetString(name)
	if v == "" {
		return nil, nil
	}
	if !validStage(v) {
		return nil, fmt.Errorf("invalid --%s %q (valid: intake, apply, review, hydrate, ship, review-pr)", name, v)
	}
	return &v, nil
}
