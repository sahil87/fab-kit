package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Watch verbs (fab-operator.md §7). The binary owns the schema, the
// timestamps, and the known-list 200-cap pruning; the operator states intent
// through flags and never hand-writes the YAML.

func operatorWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Manage watches in the operator state file",
	}
	cmd.AddCommand(
		operatorWatchAddCmd(),
		operatorWatchRmCmd(),
		operatorWatchToggleCmd(),
		operatorWatchUpdateCmd(),
		operatorWatchCheckedCmd(),
		operatorWatchSeenCmd(),
		operatorWatchCompleteCmd(),
	)
	return cmd
}

// mutateWatch loads the watches section, resolves name (erroring on unknown
// names), applies fn to the entry, and writes the section back from its typed
// form — invented in-section fields cannot survive.
func mutateWatch(data map[string]interface{}, name string, fn func(w *watchEntry) error) error {
	watches := map[string]watchEntry{}
	if err := operatorSection(data, "watches", &watches); err != nil {
		return err
	}
	w, ok := watches[name]
	if !ok {
		return fmt.Errorf("no watch named %s", name)
	}
	if err := fn(&w); err != nil {
		return err
	}
	watches[name] = w
	data["watches"] = watches
	return nil
}

func operatorWatchAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a watch (enabled, empty known/completed, null bookkeeping)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorWatchAdd,
	}
	cmd.Flags().String("source", "", "watch source: linear|slack (required)")
	cmd.Flags().String("target-repo", "", "absolute path of the repo spawned changes land in (required)")
	cmd.Flags().String("query", "", "query as a JSON object string (stored as the YAML query map)")
	cmd.Flags().String("stop-stage", "", "stage a spawned change parks at")
	cmd.Flags().String("instructions", "", "free-form spawn instructions")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("target-repo")
	return cmd
}

func runOperatorWatchAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	f := cmd.Flags()
	source, _ := f.GetString("source")
	if source != "linear" && source != "slack" {
		return fmt.Errorf("invalid --source %q (valid: linear, slack)", source)
	}
	targetRepo, _ := f.GetString("target-repo")
	query, err := queryFlag(cmd)
	if err != nil {
		return err
	}
	stopStage, err := optionalStageFlag(cmd, "stop-stage")
	if err != nil {
		return err
	}
	instructions, _ := f.GetString("instructions")

	entry := watchEntry{
		Enabled:      true,
		Source:       source,
		Query:        query,
		TargetRepo:   targetRepo,
		StopStage:    stopStage,
		Known:        []string{},
		Completed:    []string{},
		Instructions: instructions,
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		watches := map[string]watchEntry{}
		if err := operatorSection(data, "watches", &watches); err != nil {
			return err
		}
		if _, ok := watches[name]; ok {
			return fmt.Errorf("watch %s already exists", name)
		}
		watches[name] = entry
		data["watches"] = watches
		return nil
	})
}

func operatorWatchRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a watch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return mutateOperatorState(func(data map[string]interface{}) error {
				watches := map[string]watchEntry{}
				if err := operatorSection(data, "watches", &watches); err != nil {
					return err
				}
				if _, ok := watches[name]; !ok {
					return fmt.Errorf("no watch named %s", name)
				}
				delete(watches, name)
				data["watches"] = watches
				return nil
			})
		},
	}
}

func operatorWatchToggleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "toggle <name>",
		Short: "Flip (or force with --on/--off) a watch's enabled flag",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorWatchToggle,
	}
	cmd.Flags().Bool("on", false, "force enabled: true")
	cmd.Flags().Bool("off", false, "force enabled: false")
	return cmd
}

func runOperatorWatchToggle(cmd *cobra.Command, args []string) error {
	on, _ := cmd.Flags().GetBool("on")
	off, _ := cmd.Flags().GetBool("off")
	if on && off {
		return fmt.Errorf("--on and --off are mutually exclusive")
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		return mutateWatch(data, args[0], func(w *watchEntry) error {
			switch {
			case on:
				w.Enabled = true
			case off:
				w.Enabled = false
			default:
				w.Enabled = !w.Enabled
			}
			return nil
		})
	})
}

func operatorWatchUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Mutate a watch's target-repo, stop-stage, instructions, or query",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorWatchUpdate,
	}
	cmd.Flags().String("target-repo", "", "absolute path of the repo spawned changes land in")
	cmd.Flags().String("query", "", "query as a JSON object string (replaces the stored query)")
	cmd.Flags().String("stop-stage", "", "stage a spawned change parks at (empty string clears)")
	cmd.Flags().String("instructions", "", "free-form spawn instructions (replaces the stored text)")
	return cmd
}

func runOperatorWatchUpdate(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	targetRepo, _ := f.GetString("target-repo")
	instructions, _ := f.GetString("instructions")
	query, err := queryFlag(cmd)
	if err != nil {
		return err
	}
	stopStage, err := optionalStageFlag(cmd, "stop-stage")
	if err != nil {
		return err
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		return mutateWatch(data, args[0], func(w *watchEntry) error {
			if f.Changed("target-repo") {
				w.TargetRepo = targetRepo
			}
			if f.Changed("stop-stage") {
				w.StopStage = stopStage
			}
			if f.Changed("instructions") {
				w.Instructions = instructions
			}
			if f.Changed("query") {
				w.Query = query
			}
			return nil
		})
	})
}

func operatorWatchCheckedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checked <name>",
		Short: "Record a watch check: last_checked = now; set --error or clear it when absent",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorWatchChecked,
	}
	cmd.Flags().String("error", "", "error message from the check (absent clears last_error)")
	return cmd
}

func runOperatorWatchChecked(cmd *cobra.Command, args []string) error {
	errMsg, _ := cmd.Flags().GetString("error")
	hasErr := cmd.Flags().Changed("error")
	return mutateOperatorState(func(data map[string]interface{}) error {
		return mutateWatch(data, args[0], func(w *watchEntry) error {
			now := nowRFC3339()
			w.LastChecked = &now
			if hasErr {
				w.LastError = &errMsg
			} else {
				w.LastError = nil
			}
			return nil
		})
	})
}

func operatorWatchSeenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "seen <name> <item-id>",
		Short: "Append an item to a watch's known list (idempotent; 200-cap, oldest pruned first)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, item := args[0], args[1]
			return mutateOperatorState(func(data map[string]interface{}) error {
				return mutateWatch(data, name, func(w *watchEntry) error {
					for _, k := range w.Known {
						if k == item {
							return nil // idempotent — already tracked
						}
					}
					w.Known = append(w.Known, item)
					if len(w.Known) > knownCap {
						w.Known = w.Known[len(w.Known)-knownCap:]
					}
					return nil
				})
			})
		},
	}
}

func operatorWatchCompleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <name> <item-id>",
		Short: "Move an item from a watch's known list to completed",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, item := args[0], args[1]
			return mutateOperatorState(func(data map[string]interface{}) error {
				return mutateWatch(data, name, func(w *watchEntry) error {
					known := w.Known[:0]
					for _, k := range w.Known {
						if k != item {
							known = append(known, k)
						}
					}
					if known == nil {
						known = []string{}
					}
					w.Known = known
					// Union-dedupe invariant: an item absent from known (a late
					// completion) is still added to completed, never duplicated.
					for _, c := range w.Completed {
						if c == item {
							return nil
						}
					}
					w.Completed = append(w.Completed, item)
					return nil
				})
			})
		},
	}
}

// queryFlag parses --query as a JSON object string → the YAML query map.
// Absent → nil; invalid JSON or a non-object → error.
func queryFlag(cmd *cobra.Command) (map[string]interface{}, error) {
	if !cmd.Flags().Changed("query") {
		return nil, nil
	}
	raw, _ := cmd.Flags().GetString("query")
	if raw == "" {
		return nil, nil
	}
	var q map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		return nil, fmt.Errorf("invalid --query JSON: %v", err)
	}
	if q == nil {
		return nil, fmt.Errorf("invalid --query: must be a JSON object")
	}
	return q, nil
}
