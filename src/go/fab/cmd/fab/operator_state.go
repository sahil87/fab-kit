package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/atomicfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/status"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Shared operator state-file IO. Every `fab operator` state subcommand reads
// the whole file tolerantly (unknown TOP-LEVEL keys survive a read-modify-write
// — the tick-start posture, so a legacy hand-drifted file never wedges the
// operator) and re-marshals the five OWNED sections (monitored, autopilot,
// branch_map, watches, notes) from the typed structs below on mutation — an
// invented field inside an owned section can neither be introduced nor survive
// a mutation of that section. All writes go through atomicfile.WriteFile; all
// timestamps are computed here (RFC3339 UTC) — no subcommand accepts one.

// monitoredEntry is one `monitored` entry — the fab-operator.md §4 schema,
// byte-compatibly.
type monitoredEntry struct {
	Pane           string   `yaml:"pane"`
	Repo           string   `yaml:"repo"`
	Session        string   `yaml:"session"`
	Stage          string   `yaml:"stage,omitempty"`
	Agent          string   `yaml:"agent,omitempty"`
	StopStage      *string  `yaml:"stop_stage"`
	SpawnedBy      *string  `yaml:"spawned_by"`
	DependsOn      []string `yaml:"depends_on"`
	Branch         string   `yaml:"branch"`
	EnrolledAt     string   `yaml:"enrolled_at"`
	LastTransition string   `yaml:"last_transition"`
}

// autopilotState is the top-level `autopilot` block (absent → `autopilot: null`).
type autopilotState struct {
	Queue     []string `yaml:"queue"`
	Current   *string  `yaml:"current"`
	Completed []string `yaml:"completed"`
	State     *string  `yaml:"state"`
	Mode      string   `yaml:"mode"`
}

// branchMapEntry is one `branch_map` value ({ branch, repo }).
type branchMapEntry struct {
	Branch string `yaml:"branch"`
	Repo   string `yaml:"repo"`
}

// watchEntry is one `watches` entry — the fab-operator.md §7 schema,
// byte-compatibly.
type watchEntry struct {
	Enabled      bool                   `yaml:"enabled"`
	Source       string                 `yaml:"source"`
	Query        map[string]interface{} `yaml:"query,omitempty"`
	TargetRepo   string                 `yaml:"target_repo"`
	StopStage    *string                `yaml:"stop_stage"`
	Known        []string               `yaml:"known"`
	Completed    []string               `yaml:"completed"`
	LastChecked  *string                `yaml:"last_checked"`
	LastError    *string                `yaml:"last_error"`
	Instructions string                 `yaml:"instructions,omitempty"`
}

// knownCap is the binary-enforced cap on a watch's `known` list (oldest pruned
// first) — previously an agent-counted prose rule.
const knownCap = 200

// nowRFC3339 is the single timestamp source for every state mutation.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// operatorStatePath resolves the state-file path, honoring the test seam.
func operatorStatePath() (string, error) {
	if operatorStatePathOverride != "" {
		return operatorStatePathOverride, nil
	}
	// server "" → query the operator's own (current) tmux server socket.
	p, err := StatePath("")
	if err != nil {
		return "", fmt.Errorf("cannot determine operator state path: %w", err)
	}
	return p, nil
}

// loadOperatorState reads the whole state file tolerantly. A missing file is
// an empty map (not an error); an unparsable file is a hard error.
func loadOperatorState(path string) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &data); err != nil {
			return nil, fmt.Errorf("cannot parse %s: %w", path, err)
		}
	}
	return data, nil
}

// saveOperatorState writes the whole state file atomically (temp+rename — the
// operator state file is a cold path, so the always-fsync variant is fine).
func saveOperatorState(path string, data map[string]interface{}) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("cannot marshal %s: %w", path, err)
	}
	if err := atomicfile.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

// mutateOperatorState is the read-modify-write skeleton every mutation verb
// runs: load (tolerant) → fn applies typed edits → save (atomic). fn errors
// abort without writing.
func mutateOperatorState(fn func(data map[string]interface{}) error) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	data, err := loadOperatorState(path)
	if err != nil {
		return err
	}
	if err := fn(data); err != nil {
		return err
	}
	return saveOperatorState(path, data)
}

// operatorSection decodes an owned top-level section into its typed form
// (dropping any in-section drift). A missing/null section leaves out zero.
func operatorSection(data map[string]interface{}, key string, out interface{}) error {
	raw, ok := data[key]
	if !ok || raw == nil {
		return nil
	}
	b, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("cannot decode %s section: %w", key, err)
	}
	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("cannot decode %s section: %w", key, err)
	}
	return nil
}

// validStage gates stage-valued flags against the six pipeline stage names.
func validStage(s string) bool {
	for _, st := range status.AllStages() {
		if st == s {
			return true
		}
	}
	return false
}

// emptyOperatorState is the skeleton `state` persists when the file is missing.
func emptyOperatorState() map[string]interface{} {
	return map[string]interface{}{
		"monitored":  map[string]interface{}{},
		"autopilot":  nil,
		"branch_map": map[string]interface{}{},
		"watches":    map[string]interface{}{},
		"notes":      []interface{}{},
	}
}

func operatorStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Print the operator state file (creates the empty skeleton when missing)",
		Args:  cobra.NoArgs,
		RunE:  runOperatorState,
	}
	cmd.Flags().Bool("all", false, "include resolved notes in the notes list (excluded by default)")
	cmd.Flags().Bool("json", false, "print the state as JSON instead of YAML")
	return cmd
}

func runOperatorState(cmd *cobra.Command, args []string) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}
		// Skeleton-on-missing: persist the empty state, then print it — the
		// binary owns the "create if missing" init step.
		skeleton := emptyOperatorState()
		if err := saveOperatorState(path, skeleton); err != nil {
			return err
		}
		raw, err = yaml.Marshal(skeleton)
		if err != nil {
			return fmt.Errorf("cannot marshal %s: %w", path, err)
		}
	}

	w := cmd.OutOrStdout()
	asJSON, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")

	var data map[string]interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("cannot parse %s: %w", path, err)
	}
	notes, err := readNotes(data)
	if err != nil {
		return err
	}
	open := []noteEntry{}
	hasResolved := false
	for _, n := range notes {
		if n.Resolved {
			hasResolved = true
		} else {
			open = append(open, n)
		}
	}
	// Resolved notes are excluded unless --all. Filtering re-marshals the
	// parsed state; with nothing to filter the raw bytes print verbatim (the
	// read never rewrites the file, and untouched files stay byte-stable).
	filtering := hasResolved && !showAll
	if filtering {
		data["notes"] = open
		if raw, err = yaml.Marshal(data); err != nil {
			return fmt.Errorf("cannot marshal %s: %w", path, err)
		}
	}

	if !asJSON {
		// OPEN NOTES header — human output only: comment-prefixed so stdout
		// stays parseable YAML for yq consumers. Omitted when nothing is open.
		if len(open) > 0 {
			now := time.Now().UTC()
			fmt.Fprintf(w, "# OPEN NOTES (%d)\n", len(open))
			for _, n := range open {
				fmt.Fprintf(w, "# %s\n", formatNoteLine(n, now))
			}
		}
		_, err = w.Write(raw)
		return err
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot convert %s to JSON: %w", path, err)
	}
	fmt.Fprintln(w, string(out))
	return nil
}

func operatorBranchMapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch-map",
		Short: "Manage the operator state file's branch_map",
	}
	cmd.AddCommand(operatorBranchMapRmCmd())
	return cmd
}

func operatorBranchMapRmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm [<change-id>]",
		Short: "Remove a branch_map entry (or all with --all)",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runOperatorBranchMapRm,
	}
	cmd.Flags().Bool("all", false, "clear the entire branch_map")
	return cmd
}

func runOperatorBranchMapRm(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")
	if all && len(args) > 0 {
		return fmt.Errorf("branch-map rm takes either a change-id or --all, not both")
	}
	if !all && len(args) == 0 {
		return fmt.Errorf("branch-map rm requires a change-id or --all")
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		if all {
			data["branch_map"] = map[string]interface{}{}
			return nil
		}
		bm := map[string]branchMapEntry{}
		if err := operatorSection(data, "branch_map", &bm); err != nil {
			return err
		}
		id := args[0]
		if _, ok := bm[id]; !ok {
			return fmt.Errorf("no branch_map entry for %s", id)
		}
		delete(bm, id)
		data["branch_map"] = bm
		return nil
	})
}
