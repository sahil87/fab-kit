package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// autopilotSubOut executes an autopilot subcommand, capturing stdout (the
// `mode: <name> (<source>)` line start prints) alongside the error.
func autopilotSubOut(t *testing.T, name string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := operatorAutopilotCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(append([]string{name}, args...))
	err := cmd.Execute()
	return out.String(), err
}

// readAutopilot decodes the autopilot section (nil when the block is absent).
func readAutopilot(t *testing.T, path string) *autopilotState {
	t.Helper()
	data := readStateFile(t, path)
	if data["autopilot"] == nil {
		return nil
	}
	ap := &autopilotState{}
	if err := operatorSection(data, "autopilot", ap); err != nil {
		t.Fatalf("decode autopilot: %v", err)
	}
	return ap
}

func autopilotSub(t *testing.T, name string, args ...string) error {
	t.Helper()
	cmd := operatorAutopilotCmd()
	return runOperatorCmd(t, cmd, append([]string{name}, args...)...)
}

func TestOperatorAutopilot_StartLifecycle(t *testing.T) {
	path := withOperatorState(t, "")

	if err := autopilotSub(t, "start", "--queue", "ab12,cd34"); err != nil {
		t.Fatalf("autopilot start: %v", err)
	}
	ap := readAutopilot(t, path)
	if ap == nil {
		t.Fatal("autopilot block missing after start")
	}
	if len(ap.Queue) != 2 || ap.Queue[0] != "ab12" || ap.Queue[1] != "cd34" {
		t.Errorf("queue = %v", ap.Queue)
	}
	if ap.Current == nil || *ap.Current != "ab12" {
		t.Errorf("current = %v, want ab12", ap.Current)
	}
	if ap.State == nil || *ap.State != "running" {
		t.Errorf("state = %v, want running", ap.State)
	}
	if ap.Completed == nil || len(ap.Completed) != 0 {
		t.Errorf("completed = %v, want empty non-nil", ap.Completed)
	}
}

func TestOperatorAutopilot_PauseResume(t *testing.T) {
	path := withOperatorState(t, "")
	if err := autopilotSub(t, "start", "--queue", "ab12,cd34"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := autopilotSub(t, "pause"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if ap := readAutopilot(t, path); ap.State == nil || *ap.State != "paused" {
		t.Errorf("state after pause = %v", ap.State)
	}
	if err := autopilotSub(t, "resume"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if ap := readAutopilot(t, path); ap.State == nil || *ap.State != "running" {
		t.Errorf("state after resume = %v", ap.State)
	}
}

func TestOperatorAutopilot_AdvanceAndExhaustion(t *testing.T) {
	path := withOperatorState(t, "")
	if err := autopilotSub(t, "start", "--queue", "ab12,cd34"); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := autopilotSub(t, "advance"); err != nil {
		t.Fatalf("advance 1: %v", err)
	}
	ap := readAutopilot(t, path)
	if ap.Current == nil || *ap.Current != "cd34" {
		t.Errorf("after advance 1 current = %v, want cd34", ap.Current)
	}
	if len(ap.Completed) != 1 || ap.Completed[0] != "ab12" {
		t.Errorf("after advance 1 completed = %v", ap.Completed)
	}

	// Exhaustion: current/state null, queue/completed retained for the summary.
	if err := autopilotSub(t, "advance"); err != nil {
		t.Fatalf("advance 2: %v", err)
	}
	ap = readAutopilot(t, path)
	if ap == nil {
		t.Fatal("autopilot block must survive exhaustion (queue/completed retained)")
	}
	if ap.Current != nil || ap.State != nil {
		t.Errorf("exhausted: current/state = %v/%v, want null", ap.Current, ap.State)
	}
	if len(ap.Queue) != 2 || len(ap.Completed) != 2 || ap.Completed[1] != "cd34" {
		t.Errorf("exhausted: queue/completed = %v/%v", ap.Queue, ap.Completed)
	}

	// stop clears the retained block.
	if err := autopilotSub(t, "stop"); err != nil {
		t.Fatalf("stop after exhaustion: %v", err)
	}
	if ap := readAutopilot(t, path); ap != nil {
		t.Errorf("autopilot = %+v after stop, want nil", ap)
	}
}

func TestOperatorAutopilot_AdvanceSkip(t *testing.T) {
	path := withOperatorState(t, "")
	if err := autopilotSub(t, "start", "--queue", "ab12,cd34"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := autopilotSub(t, "advance", "--skip"); err != nil {
		t.Fatalf("advance --skip: %v", err)
	}
	ap := readAutopilot(t, path)
	if len(ap.Completed) != 0 {
		t.Errorf("completed = %v, want empty after --skip", ap.Completed)
	}
	if ap.Current == nil || *ap.Current != "cd34" {
		t.Errorf("current = %v, want cd34", ap.Current)
	}
}

func TestOperatorAutopilot_NoActiveQueueErrors(t *testing.T) {
	withOperatorState(t, "")
	for _, sub := range []string{"pause", "resume", "advance", "stop"} {
		if err := autopilotSub(t, sub); err == nil {
			t.Errorf("autopilot %s with no active queue must error", sub)
		}
	}
}

func TestOperatorBranchMapRm(t *testing.T) {
	path := withOperatorState(t, "")
	for _, id := range []string{"ab12", "cd34"} {
		if err := runOperatorCmd(t, operatorEnrollCmd(), enrollArgs(id)...); err != nil {
			t.Fatalf("enroll %s: %v", id, err)
		}
	}

	rm := func(args ...string) error {
		t.Helper()
		return runOperatorCmd(t, operatorBranchMapCmd(), append([]string{"rm"}, args...)...)
	}

	if err := rm("ab12"); err != nil {
		t.Fatalf("branch-map rm ab12: %v", err)
	}
	bm := map[string]branchMapEntry{}
	if err := operatorSection(readStateFile(t, path), "branch_map", &bm); err != nil {
		t.Fatalf("decode branch_map: %v", err)
	}
	if _, ok := bm["ab12"]; ok {
		t.Error("branch_map.ab12 still present after rm")
	}
	if _, ok := bm["cd34"]; !ok {
		t.Error("branch_map.cd34 must survive rm ab12")
	}

	if err := rm("ghost"); err == nil {
		t.Error("rm of unknown id must error")
	}
	if err := rm("--all"); err != nil {
		t.Fatalf("branch-map rm --all: %v", err)
	}
	bm = map[string]branchMapEntry{}
	if err := operatorSection(readStateFile(t, path), "branch_map", &bm); err != nil {
		t.Fatalf("decode branch_map: %v", err)
	}
	if len(bm) != 0 {
		t.Errorf("branch_map = %v after --all, want empty", bm)
	}
	if err := rm(); err == nil {
		t.Error("rm with no id and no --all must error")
	}
	if err := rm("ab12", "--all"); err == nil {
		t.Error("rm with both id and --all must error")
	}
}

func TestOperatorState_SkeletonOnMissingAndJSON(t *testing.T) {
	path := withOperatorState(t, "")

	var out bytes.Buffer
	cmd := operatorStateCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("state --json: %v", err)
	}

	// Skeleton persisted to disk.
	data := readStateFile(t, path)
	for _, key := range []string{"monitored", "autopilot", "branch_map", "watches"} {
		if _, ok := data[key]; !ok {
			t.Errorf("persisted skeleton missing key %q: %v", key, data)
		}
	}
	if data["autopilot"] != nil {
		t.Errorf("skeleton autopilot = %v, want null", data["autopilot"])
	}

	got := out.String()
	if !strings.Contains(got, `"monitored"`) || !strings.Contains(got, `"autopilot": null`) || !strings.Contains(got, `"watches"`) {
		t.Errorf("--json output missing skeleton keys:\n%s", got)
	}
}

func TestOperatorState_PrintsExistingVerbatim(t *testing.T) {
	content := "tick_count: 9\nmonitored: {}\nbranch_map: {}\nwatches: {}\ncustom_top: keep-me\n"
	path := withOperatorState(t, content)

	var out bytes.Buffer
	cmd := operatorStateCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("state: %v", err)
	}
	if out.String() != content {
		t.Errorf("state output =\n%s\nwant verbatim:\n%s", out.String(), content)
	}
	// A pure read must not rewrite the file.
	raw, _ := readFileBytes(t, path)
	if string(raw) != content {
		t.Error("state verb must not modify an existing file")
	}
}

func readFileBytes(t *testing.T, path string) ([]byte, error) {
	t.Helper()
	return os.ReadFile(path)
}

func TestOperatorAutopilot_ModeFlagParsing(t *testing.T) {
	path := withOperatorState(t, "")
	if err := autopilotSub(t, "start", "--queue", "ab12", "--mode", "stacked-prs"); err != nil {
		t.Fatalf("start --mode stacked-prs: %v", err)
	}
	if ap := readAutopilot(t, path); ap.Mode != "stacked-prs" {
		t.Errorf("mode = %q, want stacked-prs", ap.Mode)
	}
}

func TestOperatorAutopilot_ModeDefaultFill(t *testing.T) {
	setupConfigRepo(t, "project:\n    name: t\n")
	path := withOperatorState(t, "")
	out, err := autopilotSubOut(t, "start", "--queue", "ab12")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if ap := readAutopilot(t, path); ap.Mode != "cherry-pick-ladder" {
		t.Errorf("mode = %q, want default cherry-pick-ladder", ap.Mode)
	}
	if !strings.Contains(out, "mode: cherry-pick-ladder (default)") {
		t.Errorf("stdout must print the resolved source line, got %q", out)
	}
}

func TestOperatorAutopilot_ModeValidation(t *testing.T) {
	path := withOperatorState(t, "")
	err := autopilotSub(t, "start", "--queue", "ab12", "--mode", "merge-everything")
	if err == nil {
		t.Fatal("unknown --mode must error")
	}
	for _, want := range []string{"merge-everything", "cherry-pick-ladder", "merge-auto", "stacked-prs"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("validation error must name %q: %v", want, err)
		}
	}
	// Validation precedes the mutate: no state file is written.
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("state file must not exist after a rejected --mode")
	}
}

func TestOperatorAutopilot_ModeLifecycleRetention(t *testing.T) {
	path := withOperatorState(t, "")
	if err := autopilotSub(t, "start", "--queue", "ab12,cd34", "--mode", "merge-auto"); err != nil {
		t.Fatalf("start: %v", err)
	}
	wantMode := func(step string) {
		t.Helper()
		if ap := readAutopilot(t, path); ap == nil || ap.Mode != "merge-auto" {
			t.Errorf("mode after %s = %v, want merge-auto", step, ap)
		}
	}
	if err := autopilotSub(t, "pause"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	wantMode("pause")
	if err := autopilotSub(t, "resume"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	wantMode("resume")
	if err := autopilotSub(t, "advance"); err != nil {
		t.Fatalf("advance 1: %v", err)
	}
	wantMode("advance 1")
	// Exhaustion retains mode alongside queue/completed for the summary.
	if err := autopilotSub(t, "advance"); err != nil {
		t.Fatalf("advance 2: %v", err)
	}
	ap := readAutopilot(t, path)
	if ap == nil || ap.Mode != "merge-auto" {
		t.Errorf("mode after exhaustion = %v, want merge-auto retained", ap)
	}
	if ap.Current != nil || ap.State != nil {
		t.Errorf("exhausted: current/state = %v/%v, want null", ap.Current, ap.State)
	}
	if err := autopilotSub(t, "stop"); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if ap := readAutopilot(t, path); ap != nil {
		t.Errorf("autopilot = %+v after stop, want nil", ap)
	}
}

func TestOperatorAutopilot_ModeAbsentBackCompat(t *testing.T) {
	// A pre-existing state file whose autopilot block lacks `mode` reads as
	// cherry-pick-ladder — the BUILT-IN default, never config-resolved, even when
	// a config preference names another mode (the mode was fixed at queue start;
	// re-resolving could silently retopologize a running queue). The next
	// mutation re-marshals the field.
	setupConfigRepo(t, "project:\n    name: t\nautopilot:\n  merge_mode: stacked-prs\n")
	legacy := "autopilot:\n  queue: [ab12]\n  current: ab12\n  completed: []\n  state: running\n"
	path := withOperatorState(t, legacy)

	if err := autopilotSub(t, "pause"); err != nil {
		t.Fatalf("pause on legacy block: %v", err)
	}
	ap := readAutopilot(t, path)
	if ap == nil {
		t.Fatal("autopilot block missing after pause")
	}
	if ap.Mode != "cherry-pick-ladder" {
		t.Errorf("mode = %q after legacy pause, want cherry-pick-ladder (built-in, not config-resolved)", ap.Mode)
	}
	if ap.State == nil || *ap.State != "paused" {
		t.Errorf("state = %v, want paused", ap.State)
	}
}

func TestOperatorAutopilot_ModeResolution(t *testing.T) {
	defaultYAML := "project:\n    name: t\n"
	cases := []struct {
		name       string
		projectCfg string // project config.yaml; "" = same as defaultYAML
		systemCfg  string // ~/.fab-kit/config.yaml; "" = absent
		neutralDir bool   // chdir to a fab-less parent (only system/env tiers compose)
		args       []string
		wantMode   string
		wantSource string // the printed `mode: <name> (<source>)` source
	}{
		{name: "flag absent, no config", args: []string{"--queue", "ab12"}, wantMode: config.DefaultAutopilotMergeMode, wantSource: "default"},
		{name: "config resolves when flag absent", projectCfg: "project:\n    name: t\nautopilot:\n  merge_mode: stacked-prs\n", args: []string{"--queue", "ab12"}, wantMode: "stacked-prs", wantSource: "config"},
		{name: "flag beats config", projectCfg: "project:\n    name: t\nautopilot:\n  merge_mode: stacked-prs\n", args: []string{"--queue", "ab12", "--mode", "merge-auto"}, wantMode: "merge-auto", wantSource: "flag"},
		{name: "explicit default flag reports flag", args: []string{"--queue", "ab12", "--mode", "cherry-pick-ladder"}, wantMode: "cherry-pick-ladder", wantSource: "flag"},
		{name: "system tier reaches a fab-less cwd", systemCfg: "autopilot:\n  merge_mode: stacked-prs\n", neutralDir: true, args: []string{"--queue", "ab12"}, wantMode: "stacked-prs", wantSource: "config"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projectCfg := tc.projectCfg
			if projectCfg == "" {
				projectCfg = defaultYAML
			}
			_, home := setupConfigRepo(t, projectCfg)
			if tc.systemCfg != "" {
				writeSystemConfig(t, home, tc.systemCfg)
			}
			if tc.neutralDir {
				// A fab-less parent directory: resolve.FabRoot fails, and the
				// cwd-relative fallback finds no project file — only the system
				// (and env) tiers compose.
				neutral := t.TempDir()
				if err := os.Chdir(neutral); err != nil {
					t.Fatal(err)
				}
			}
			path := withOperatorState(t, "")
			out, err := autopilotSubOut(t, "start", tc.args...)
			if err != nil {
				t.Fatalf("start: %v", err)
			}
			if ap := readAutopilot(t, path); ap == nil || ap.Mode != tc.wantMode {
				t.Errorf("mode = %v, want %q", ap, tc.wantMode)
			}
			wantLine := fmt.Sprintf("mode: %s (%s)", tc.wantMode, tc.wantSource)
			if !strings.Contains(out, wantLine) {
				t.Errorf("stdout = %q, want it to contain %q", out, wantLine)
			}
		})
	}
}

func TestOperatorAutopilot_ModeConfigInvalid(t *testing.T) {
	setupConfigRepo(t, "project:\n    name: t\nautopilot:\n  merge_mode: merge-everything\n")
	path := withOperatorState(t, "")
	_, err := autopilotSubOut(t, "start", "--queue", "ab12")
	if err == nil {
		t.Fatal("an invalid autopilot.merge_mode must error")
	}
	for _, want := range []string{"autopilot.merge_mode", "merge-everything", "cherry-pick-ladder", "merge-auto", "stacked-prs"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("config validation error must name %q: %v", want, err)
		}
	}
	// Validation precedes the mutate: no state file is written.
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("state file must not exist after a rejected config value")
	}

	// An explicit flag still wins over — and sidesteps — the invalid config value.
	out, err := autopilotSubOut(t, "start", "--queue", "ab12", "--mode", "merge-auto")
	if err != nil {
		t.Fatalf("start with explicit flag over an invalid config: %v", err)
	}
	if !strings.Contains(out, "mode: merge-auto (flag)") {
		t.Errorf("stdout = %q, want the flag-source line", out)
	}
}
