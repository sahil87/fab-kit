package main

// --json output tests for the read-only `fab status` query subcommands
// (260717-jx4w). The --json branches emit via cmd.OutOrStdout() (the text
// paths still write to os.Stdout via fmt.Print*), so these tests exercise the
// JSON path: they capture through cmd.SetOut(buf) and assert the decoded JSON
// shape — the intake's stable per-subcommand contract (snake_case keys, ordered
// arrays for list subcommands, [] not null for empty lists, {"summary":""} for
// empty summary, and text/JSON parity on plan's live-acceptance read path).

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// statusJSONTestYAML is a change with populated confidence, one issue, one PR,
// and a summary — so the get-issues/get-prs arrays and get-summary object are
// exercised with real content (the empty cases get their own repo below).
const statusJSONTestYAML = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: feat
issues:
  - DEV-988
progress:
  intake: done
  apply: active
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: true
  task_count: 12
  acceptance_count: 3
  acceptance_completed: 1
confidence:
  certain: 2
  confident: 3
  tentative: 1
  unresolved: 0
  score: 4.2
summary: "a one-line log summary"
stage_metrics: {}
prs:
  - https://github.com/owner/repo/pull/42
last_updated: "2026-03-10T12:00:00Z"
`

// planMDWithAcceptance is a plan.md whose ## Acceptance section has 5 items,
// 2 checked — deliberately different from the cached counter (3/1) so the
// live-acceptance read path is observable.
const planMDWithAcceptance = `# Plan

## Requirements

## Tasks

- [x] T001 done <!-- R1 -->

## Acceptance

- [x] A-001 R1: alpha
- [x] A-002 R1: beta
- [ ] A-003 R1: gamma
- [ ] A-004 R1: delta
- [ ] A-005 R1: epsilon
`

// setupStatusJSONRepo builds a repo with one change (active via symlink),
// optionally writing a plan.md, and chdirs into it so resolve.FabRoot()
// resolves. Returns the change folder name.
func setupStatusJSONRepo(t *testing.T, statusYAML, planMD string) string {
	t.Helper()
	repoRoot := t.TempDir()
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(repoRoot, "fab", "changes", folder)
	mustMkdir(t, changeDir)
	mustWrite(t, filepath.Join(changeDir, ".status.yaml"), statusYAML)
	mustWrite(t, filepath.Join(changeDir, "intake.md"), "# Intake\n")
	if planMD != "" {
		mustWrite(t, filepath.Join(changeDir, "plan.md"), planMD)
	}
	if err := os.Symlink("fab/changes/"+folder+"/.status.yaml", filepath.Join(repoRoot, ".fab-status.yaml")); err != nil {
		t.Fatal(err)
	}
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	return folder
}

// runJSON executes cmd with args, capturing cmd.OutOrStdout() into a buffer and
// decoding the JSON into dst.
func runJSON(t *testing.T, cmd *cobra.Command, dst any, args ...string) {
	t.Helper()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	if err := json.Unmarshal(buf.Bytes(), dst); err != nil {
		t.Fatalf("decode %v output %q: %v", args, buf.String(), err)
	}
}

// runRaw executes cmd with args, returning the captured cmd.OutOrStdout() bytes.
func runRaw(t *testing.T, cmd *cobra.Command, args ...string) []byte {
	t.Helper()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return buf.Bytes()
}

func TestStatusConfidenceJSON(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got confidenceJSON
	runJSON(t, statusConfidenceCmd(), &got, "abcd", "--json")
	want := confidenceJSON{Certain: 2, Confident: 3, Tentative: 1, Unresolved: 0, Score: 4.2}
	if got != want {
		t.Errorf("confidence --json = %+v, want %+v", got, want)
	}
}

func TestStatusPlanJSON_LiveAcceptanceParity(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, planMDWithAcceptance)

	var got planJSON
	runJSON(t, statusPlanCmd(), &got, "abcd", "--json")

	// generated + task_count come from the cache; acceptance_* is the LIVE
	// count from plan.md (5 items, 2 checked) — NOT the cached 3/1.
	want := planJSON{Generated: true, TaskCount: 12, AcceptanceCount: 5, AcceptanceCompleted: 2}
	if got != want {
		t.Errorf("plan --json = %+v, want %+v (acceptance must be the live count)", got, want)
	}

	// Text path must report the SAME live-derived acceptance values. The text
	// branch prints via fmt.Printf (os.Stdout), so capture with execCapture
	// (the shared os.Stdout-capturing helper) rather than cmd.OutOrStdout().
	text, err := execCapture(t, statusPlanCmd(), "abcd")
	if err != nil {
		t.Fatalf("plan (text): %v", err)
	}
	for _, want := range []string{
		"generated:true\n",
		"task_count:12\n",
		"acceptance_count:5\n",
		"acceptance_completed:2\n",
	} {
		if !bytes.Contains([]byte(text), []byte(want)) {
			t.Errorf("plan (text) = %q, missing line %q", text, want)
		}
	}
}

func TestStatusProgressMapJSON_PreservesStageOrder(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got []stageStateJSON
	runJSON(t, statusProgressMapCmd(), &got, "abcd", "--json")

	want := []stageStateJSON{
		{Stage: "intake", State: "done"},
		{Stage: "apply", State: "active"},
		{Stage: "review", State: "pending"},
		{Stage: "hydrate", State: "pending"},
		{Stage: "ship", State: "pending"},
		{Stage: "review-pr", State: "pending"},
	}
	if len(got) != len(want) {
		t.Fatalf("progress-map --json len = %d, want %d (%+v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("progress-map --json[%d] = %+v, want %+v (stage order must be preserved)", i, got[i], want[i])
		}
	}
}

func TestStatusDisplayStageJSON(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got stageStateJSON
	runJSON(t, statusDisplayStageCmd(), &got, "abcd", "--json")
	want := stageStateJSON{Stage: "apply", State: "active"}
	if got != want {
		t.Errorf("display-stage --json = %+v, want %+v", got, want)
	}
}

func TestStatusCurrentStageJSON(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got currentStageJSON
	runJSON(t, statusCurrentStageCmd(), &got, "abcd", "--json")
	if got.Stage != "apply" {
		t.Errorf("current-stage --json = %+v, want {Stage:apply}", got)
	}
}

func TestStatusAllStagesJSON(t *testing.T) {
	// all-stages takes no <change> argument.
	var got []string
	runJSON(t, statusAllStagesCmd(), &got, "--json")
	want := []string{"intake", "apply", "review", "hydrate", "ship", "review-pr"}
	if len(got) != len(want) {
		t.Fatalf("all-stages --json = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("all-stages --json[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStatusGetIssuesJSON(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got []string
	runJSON(t, statusGetIssuesCmd(), &got, "abcd", "--json")
	if len(got) != 1 || got[0] != "DEV-988" {
		t.Errorf("get-issues --json = %v, want [DEV-988]", got)
	}
}

func TestStatusGetPRsJSON(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got []string
	runJSON(t, statusGetPRsCmd(), &got, "abcd", "--json")
	if len(got) != 1 || got[0] != "https://github.com/owner/repo/pull/42" {
		t.Errorf("get-prs --json = %v, want [https://github.com/owner/repo/pull/42]", got)
	}
}

func TestStatusGetSummaryJSON(t *testing.T) {
	setupStatusJSONRepo(t, statusJSONTestYAML, "")
	var got summaryJSON
	runJSON(t, statusGetSummaryCmd(), &got, "abcd", "--json")
	if got.Summary != "a one-line log summary" {
		t.Errorf("get-summary --json = %+v, want {Summary:%q}", got, "a one-line log summary")
	}
}

// emptyStatusJSONYAML has empty issues/prs and no summary — the empty-list []
// and empty-summary {"summary":""} cases.
const emptyStatusJSONYAML = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: active
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: false
  task_count: 0
  acceptance_count: 0
  acceptance_completed: 0
confidence:
  certain: 0
  confident: 0
  tentative: 0
  unresolved: 0
  score: 0.0
stage_metrics: {}
prs: []
last_updated: "2026-03-10T12:00:00Z"
`

func TestStatusGetIssuesJSON_EmptyIsBracketsNotNull(t *testing.T) {
	setupStatusJSONRepo(t, emptyStatusJSONYAML, "")
	raw := bytes.TrimSpace(runRaw(t, statusGetIssuesCmd(), "abcd", "--json"))
	if string(raw) != "[]" {
		t.Errorf("get-issues --json (empty) = %q, want %q (never null)", raw, "[]")
	}
}

func TestStatusGetPRsJSON_EmptyIsBracketsNotNull(t *testing.T) {
	setupStatusJSONRepo(t, emptyStatusJSONYAML, "")
	raw := bytes.TrimSpace(runRaw(t, statusGetPRsCmd(), "abcd", "--json"))
	if string(raw) != "[]" {
		t.Errorf("get-prs --json (empty) = %q, want %q (never null)", raw, "[]")
	}
}

func TestStatusGetSummaryJSON_EmptyIsEmptyString(t *testing.T) {
	setupStatusJSONRepo(t, emptyStatusJSONYAML, "")
	var got summaryJSON
	runJSON(t, statusGetSummaryCmd(), &got, "abcd", "--json")
	if got.Summary != "" {
		t.Errorf("get-summary --json (empty) = %+v, want {Summary:\"\"}", got)
	}
	// And confirm the object wrapper (not a bare string) is emitted.
	raw := bytes.TrimSpace(runRaw(t, statusGetSummaryCmd(), "abcd", "--json"))
	if !bytes.Contains(raw, []byte(`"summary"`)) {
		t.Errorf("get-summary --json (empty) = %q, want an object with a \"summary\" key", raw)
	}
}

// TestStatusQueryCmds_HaveJSONFlag guards the flag registration across the
// whole query surface — every one of the nine subcommands must expose --json.
func TestStatusQueryCmds_HaveJSONFlag(t *testing.T) {
	cmds := map[string]*cobra.Command{
		"confidence":    statusConfidenceCmd(),
		"plan":          statusPlanCmd(),
		"progress-map":  statusProgressMapCmd(),
		"get-issues":    statusGetIssuesCmd(),
		"get-prs":       statusGetPRsCmd(),
		"get-summary":   statusGetSummaryCmd(),
		"current-stage": statusCurrentStageCmd(),
		"display-stage": statusDisplayStageCmd(),
		"all-stages":    statusAllStagesCmd(),
	}
	for name, cmd := range cmds {
		f := cmd.Flags().Lookup("json")
		if f == nil {
			t.Errorf("%s: missing --json flag", name)
			continue
		}
		if f.Usage != "Output as JSON" {
			t.Errorf("%s: --json usage = %q, want %q", name, f.Usage, "Output as JSON")
		}
	}
}
