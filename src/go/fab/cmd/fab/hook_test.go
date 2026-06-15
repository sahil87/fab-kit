package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/runtime"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

func TestParseTmuxServer(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{"empty", "", ""},
		{"named socket fabKit", "/tmp/tmux-1001/fabKit,8671,0", "fabKit"},
		{"default socket", "/tmp/tmux-1001/default,8671,0", "default"},
		{"only path", "/tmp/tmux-1001/foo", "foo"},
		{"trailing comma after empty path", ",8671,0", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseTmuxServer(tc.env)
			if got != tc.want {
				t.Errorf("parseTmuxServer(%q) = %q, want %q", tc.env, got, tc.want)
			}
		})
	}
}

// hookTestEnv isolates env vars and cwd for a hook test and restores them on
// t.Cleanup.
func hookTestEnv(t *testing.T, cwd string, envOverrides map[string]string) {
	t.Helper()
	origEnv := map[string]string{}
	for k := range envOverrides {
		origEnv[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})
	for k, v := range envOverrides {
		if v == "" {
			os.Unsetenv(k)
		} else {
			os.Setenv(k, v)
		}
	}
	// Chdir for resolve.FabRoot()
	origWd, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWd) })
}

// setupHookRepoRoot creates a temp repo with a fab/ subdir and returns
// (repoRoot, fabRoot).
func setupHookRepoRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	fabRoot := filepath.Join(root, "fab")
	if err := os.MkdirAll(fabRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, fabRoot
}

func TestHookStop_WellFormedPayload(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "/tmp/tmux-1001/fabKit,8671,0",
		envTmuxPane: "%15",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-stop-1","transcript_path":"/tmp/t.jsonl","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook stop returned error: %v", err)
	}

	m, err := runtime.LoadFile(runtime.FilePath(fabRoot))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	agents, ok := m["_agents"].(map[string]interface{})
	if !ok {
		t.Fatal("expected _agents map")
	}
	got, ok := agents["uuid-stop-1"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry for uuid-stop-1")
	}
	if got["idle_since"] == nil {
		t.Error("expected idle_since to be set")
	}
	if got["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane = %v, want \"%%15\"", got["tmux_pane"])
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server = %v, want fabKit", got["tmux_server"])
	}
	if got["transcript_path"] != "/tmp/t.jsonl" {
		t.Errorf("transcript_path = %v", got["transcript_path"])
	}
}

func TestHookStop_MissingSessionID(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook stop returned error: %v", err)
	}

	// File should NOT have been created.
	if _, err := os.Stat(runtime.FilePath(fabRoot)); !os.IsNotExist(err) {
		t.Errorf("expected no runtime file; got stat err=%v", err)
	}
}

func TestHookStop_MalformedPayload(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader("not-json"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook stop should swallow errors, got: %v", err)
	}
	if _, err := os.Stat(runtime.FilePath(fabRoot)); !os.IsNotExist(err) {
		t.Errorf("expected no runtime file on malformed input")
	}
}

func TestHookStop_NoTmuxOmitsFields(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-nt","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-nt"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry")
	}
	if _, present := got["tmux_pane"]; present {
		t.Errorf("tmux_pane should be absent, got %v", got["tmux_pane"])
	}
	if _, present := got["tmux_server"]; present {
		t.Errorf("tmux_server should be absent, got %v", got["tmux_server"])
	}
}

func TestHookStop_TmuxPaneOnly(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "%15",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-pane","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-pane"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry")
	}
	if got["tmux_pane"] != "%15" {
		t.Errorf("tmux_pane = %v, want \"%%15\"", got["tmux_pane"])
	}
	if _, present := got["tmux_server"]; present {
		t.Errorf("tmux_server should be omitted when $TMUX unset, got %v", got["tmux_server"])
	}
}

func TestHookStop_TmuxServerOnly(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "/tmp/tmux-1001/fabKit,8671,0",
		envTmuxPane: "",
	})

	cmd := hookStopCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-srv","hook_event_name":"Stop"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-srv"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry")
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server = %v, want fabKit", got["tmux_server"])
	}
	if _, present := got["tmux_pane"]; present {
		t.Errorf("tmux_pane should be omitted, got %v", got["tmux_pane"])
	}
}

func TestHookSessionStart_DeletesEntry(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	// Seed an entry first.
	is := int64(1000)
	if err := runtime.WriteAgent(fabRoot, "uuid-ss", runtime.AgentEntry{IdleSince: &is, TmuxPane: "%5"}, runtime.NoGC); err != nil {
		t.Fatal(err)
	}

	cmd := hookSessionStartCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-ss","hook_event_name":"SessionStart"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	if _, present := agents["uuid-ss"]; present {
		t.Error("expected entry to be deleted")
	}
}

func TestHookUserPrompt_ClearsIdleOnly(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{
		envTmux:     "",
		envTmuxPane: "",
	})

	// Seed with idle and tmux info.
	is := int64(1000)
	if err := runtime.WriteAgent(fabRoot, "uuid-up", runtime.AgentEntry{
		IdleSince:  &is,
		TmuxPane:   "%5",
		TmuxServer: "fabKit",
		Change:     "260417-2fbb",
	}, runtime.NoGC); err != nil {
		t.Fatal(err)
	}

	cmd := hookUserPromptCmd()
	cmd.SetIn(strings.NewReader(`{"session_id":"uuid-up","hook_event_name":"UserPromptSubmit"}`))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	m, _ := runtime.LoadFile(runtime.FilePath(fabRoot))
	agents, _ := m["_agents"].(map[string]interface{})
	got, ok := agents["uuid-up"].(map[string]interface{})
	if !ok {
		t.Fatal("expected entry to remain")
	}
	if _, present := got["idle_since"]; present {
		t.Error("expected idle_since to be cleared")
	}
	if got["tmux_pane"] != "%5" {
		t.Errorf("tmux_pane lost: %v", got["tmux_pane"])
	}
	if got["tmux_server"] != "fabKit" {
		t.Errorf("tmux_server lost: %v", got["tmux_server"])
	}
	if got["change"] != "260417-2fbb" {
		t.Errorf("change lost: %v", got["change"])
	}
}

// --- fab hook sync exit contract (k4ge): all hook subcommands exit 0 ---

func TestHookSync_FailureSurfacedButExitsZero(t *testing.T) {
	repoRoot, _ := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{})

	// Block the settings write: a directory at the settings.local.json path
	// makes hooklib.Sync fail.
	settingsPath := filepath.Join(repoRoot, ".claude", "settings.local.json")
	if err := os.MkdirAll(settingsPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cmd := hookSyncCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook sync must exit 0 on failure, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "hook sync:") {
		t.Errorf("failure should be surfaced on stderr, got: %q", errOut.String())
	}
}

func TestHookSync_NoFabRootExitsZero(t *testing.T) {
	// A directory tree with no fab/ anywhere up to root would be needed to
	// make resolve.FabRoot fail; t.TempDir() is safe as long as no ancestor
	// has a fab/ dir — guard by skipping if resolution unexpectedly succeeds.
	dir := t.TempDir()
	hookTestEnv(t, dir, map[string]string{})

	cmd := hookSyncCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook sync must exit 0 even without a fab root, got: %v", err)
	}
}

func TestHookSync_SuccessOutputUnchanged(t *testing.T) {
	repoRoot, _ := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{})

	cmd := hookSyncCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook sync failed: %v", err)
	}
	if !strings.Contains(out.String(), "Created") {
		t.Errorf("expected Created message on first sync, got: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("no stderr expected on success, got: %q", errOut.String())
	}
}

// --- artifact-write bookkeeping (mz4q F02): in-memory mutation + exactly one
// Save under the status lock; external contract (additionalContext shape,
// final .status.yaml state, exit-0) unchanged. ---

const hookStatusFixture = `id: abcd
name: %s
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

// setupHookChange creates a change folder with a .status.yaml fixture under
// the given fabRoot and returns the folder name.
func setupHookChange(t *testing.T, fabRoot string) string {
	t.Helper()
	folder := "260310-abcd-my-change"
	changeDir := filepath.Join(fabRoot, "changes", folder)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	statusYAML := strings.Replace(hookStatusFixture, "%s", folder, 1)
	if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("project:\n  name: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return folder
}

func runArtifactWriteHook(t *testing.T, relPath string) string {
	t.Helper()
	cmd := hookArtifactWriteCmd()
	payload := fmt.Sprintf(`{"tool_name":"Write","tool_input":{"file_path":"%s"}}`, relPath)
	cmd.SetIn(strings.NewReader(payload))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("artifact-write hook returned error: %v", err)
	}
	return out.String()
}

// TestHookArtifactWrite_RespectsExplicitChangeType covers jznd (2/a): when a
// human has set change_type_source: explicit, the intake-write hook must NOT
// re-infer/overwrite change_type — even though the intake prose ("fix a bug")
// would infer "fix". This is the F02 re-clobber bug fixed.
func TestHookArtifactWrite_RespectsExplicitChangeType(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{envTmux: "", envTmuxPane: ""})
	folder := setupHookChange(t, fabRoot)

	// Mark the change_type explicit (as `fab status set-change-type` would).
	statusPath := filepath.Join(fabRoot, "changes", folder, ".status.yaml")
	st, err := sf.Load(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	st.ChangeType = "feat"
	st.ChangeTypeSource = sf.SourceExplicit
	if err := st.Save(statusPath); err != nil {
		t.Fatal(err)
	}

	// Intake prose that WOULD infer "fix".
	intakeMD := `# Intake: A new feature

This change fixes a bug while adding the widget.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | D1 | R1 | |
`
	intakePath := filepath.Join(fabRoot, "changes", folder, "intake.md")
	if err := os.WriteFile(intakePath, []byte(intakeMD), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runArtifactWriteHook(t, "fab/changes/"+folder+"/intake.md")
	var ctx map[string]string
	if err := json.Unmarshal([]byte(out), &ctx); err != nil {
		t.Fatalf("expected additionalContext JSON, got %q: %v", out, err)
	}
	if !strings.Contains(ctx["additionalContext"], "explicit, kept") {
		t.Errorf("additionalContext = %q, want explicit-kept note", ctx["additionalContext"])
	}

	reloaded, err := sf.Load(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ChangeType != "feat" {
		t.Errorf("change_type = %q, want feat (explicit type must survive a re-infer write)", reloaded.ChangeType)
	}
	if reloaded.ChangeTypeSource != sf.SourceExplicit {
		t.Errorf("change_type_source = %q, want explicit", reloaded.ChangeTypeSource)
	}
}

// TestHookArtifactWrite_InferredChangeTypeStillReinfers is the back-compat
// complement: a change with no/inferred source still gets re-inferred by the
// hook (the default behavior is unchanged).
func TestHookArtifactWrite_InferredChangeTypeStillReinfers(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{envTmux: "", envTmuxPane: ""})
	folder := setupHookChange(t, fabRoot)

	intakeMD := `# Intake: Fix the broken widget

This is a fix for a bug.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | D1 | R1 | |
`
	intakePath := filepath.Join(fabRoot, "changes", folder, "intake.md")
	if err := os.WriteFile(intakePath, []byte(intakeMD), 0o644); err != nil {
		t.Fatal(err)
	}

	runArtifactWriteHook(t, "fab/changes/"+folder+"/intake.md")

	statusPath := filepath.Join(fabRoot, "changes", folder, ".status.yaml")
	reloaded, err := sf.Load(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ChangeType != "fix" {
		t.Errorf("change_type = %q, want fix (inferred source must re-infer)", reloaded.ChangeType)
	}
}

func TestHookArtifactWrite_PlanSingleSave(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{envTmux: "", envTmuxPane: ""})
	folder := setupHookChange(t, fabRoot)

	planMD := `# Plan

## Tasks

- [ ] T001 first
- [x] T002 second
- [ ] T003 third

## Acceptance

- [x] A-001 done thing
- [ ] A-002 open thing
`
	planPath := filepath.Join(fabRoot, "changes", folder, "plan.md")
	if err := os.WriteFile(planPath, []byte(planMD), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runArtifactWriteHook(t, "fab/changes/"+folder+"/plan.md")

	// additionalContext JSON shape preserved.
	var ctx map[string]string
	if err := json.Unmarshal([]byte(out), &ctx); err != nil {
		t.Fatalf("expected additionalContext JSON on stdout, got %q: %v", out, err)
	}
	if !strings.Contains(ctx["additionalContext"], "plan tasks: 3") {
		t.Errorf("additionalContext = %q, want plan tasks: 3", ctx["additionalContext"])
	}
	if !strings.Contains(ctx["additionalContext"], "plan acceptance: 1/2") {
		t.Errorf("additionalContext = %q, want plan acceptance: 1/2", ctx["additionalContext"])
	}

	// All four plan fields persisted by the single Save.
	statusPath := filepath.Join(fabRoot, "changes", folder, ".status.yaml")
	reloaded, err := sf.Load(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Plan.Generated {
		t.Error("plan.generated not persisted")
	}
	if reloaded.Plan.TaskCount != 3 {
		t.Errorf("task_count = %d, want 3", reloaded.Plan.TaskCount)
	}
	if reloaded.Plan.AcceptanceCount != 2 {
		t.Errorf("acceptance_count = %d, want 2", reloaded.Plan.AcceptanceCount)
	}
	if reloaded.Plan.AcceptanceCompleted != 1 {
		t.Errorf("acceptance_completed = %d, want 1", reloaded.Plan.AcceptanceCompleted)
	}
}

func TestHookArtifactWrite_PlanNoSectionsNoWrite(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{envTmux: "", envTmuxPane: ""})
	folder := setupHookChange(t, fabRoot)

	planPath := filepath.Join(fabRoot, "changes", folder, "plan.md")
	if err := os.WriteFile(planPath, []byte("# Plan without sections\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	statusPath := filepath.Join(fabRoot, "changes", folder, ".status.yaml")
	before, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}

	out := runArtifactWriteHook(t, "fab/changes/"+folder+"/plan.md")
	if out != "" {
		t.Errorf("expected no additionalContext output, got %q", out)
	}

	after, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("expected no .status.yaml write when nothing was mutated")
	}
}

func TestHookArtifactWrite_IntakeSingleLoadAndSave(t *testing.T) {
	repoRoot, fabRoot := setupHookRepoRoot(t)
	hookTestEnv(t, repoRoot, map[string]string{envTmux: "", envTmuxPane: ""})
	folder := setupHookChange(t, fabRoot)

	intakeMD := `# Intake: Fix the broken widget

This is a fix for a bug.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | D1 | R1 | S:100 R:100 A:100 D:100 |
| 2 | Certain | D2 | R2 | S:100 R:100 A:100 D:100 |
| 3 | Certain | D3 | R3 | S:100 R:100 A:100 D:100 |
| 4 | Certain | D4 | R4 | S:100 R:100 A:100 D:100 |
| 5 | Certain | D5 | R5 | S:100 R:100 A:100 D:100 |
`
	intakePath := filepath.Join(fabRoot, "changes", folder, "intake.md")
	if err := os.WriteFile(intakePath, []byte(intakeMD), 0o644); err != nil {
		t.Fatal(err)
	}

	out := runArtifactWriteHook(t, "fab/changes/"+folder+"/intake.md")

	var ctx map[string]string
	if err := json.Unmarshal([]byte(out), &ctx); err != nil {
		t.Fatalf("expected additionalContext JSON on stdout, got %q: %v", out, err)
	}
	if !strings.Contains(ctx["additionalContext"], "type: fix") {
		t.Errorf("additionalContext = %q, want type: fix", ctx["additionalContext"])
	}
	if !strings.Contains(ctx["additionalContext"], "score: 5.0") {
		t.Errorf("additionalContext = %q, want score: 5.0", ctx["additionalContext"])
	}

	// Change type and confidence both landed in the single Save.
	statusPath := filepath.Join(fabRoot, "changes", folder, ".status.yaml")
	reloaded, err := sf.Load(statusPath)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ChangeType != "fix" {
		t.Errorf("change_type = %q, want fix", reloaded.ChangeType)
	}
	if reloaded.Confidence.Score != 5.0 {
		t.Errorf("confidence.score = %v, want 5.0", reloaded.Confidence.Score)
	}
	if reloaded.Confidence.Certain != 5 {
		t.Errorf("confidence.certain = %d, want 5", reloaded.Confidence.Certain)
	}
}
