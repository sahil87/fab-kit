package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- shared operator state test scaffolding -----------------------------------

// withOperatorState redirects operator state-file I/O to a temp file seeded
// with initial (empty string = missing file) and returns its path.
func withOperatorState(t *testing.T, initial string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "operator-state.yaml")
	if initial != "" {
		if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
			t.Fatalf("seed state file: %v", err)
		}
	}
	operatorStatePathOverride = path
	t.Cleanup(func() { operatorStatePathOverride = "" })
	return path
}

// runOperatorCmd executes a command with output silenced and returns its error.
func runOperatorCmd(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	return cmd.Execute()
}

// runOperatorCmdOut is runOperatorCmd but returns captured stdout.
func runOperatorCmdOut(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// readStateFile parses the state file back into a raw map.
func readStateFile(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var data map[string]interface{}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parse state file: %v", err)
	}
	return data
}

// stubQuietClock stubs every clock/reconcile seam so track-verb tests never
// touch a real rk/tmux/gh: no cron row resolves (so no mute/edit argv is ever
// issued), no panes enumerate, no operator window resolves, and the pane_pid
// fingerprint lookup fails soft (null on lookup failure, by contract).
func stubQuietClock(t *testing.T) {
	t.Helper()
	prevCron, prevPanes, prevWin, prevPID := rkCronRunner, rkPanesRunner, operatorWindowRoleRunner, trackPanePIDLookup
	rkCronRunner = func(args ...string) (string, error) { return "[]", nil }
	rkPanesRunner = func(server string) ([]byte, error) { return nil, errors.New("stubbed: no rk") }
	operatorWindowRoleRunner = func() (string, error) { return "", errors.New("stubbed: no tmux") }
	trackPanePIDLookup = func(paneID string) (int, error) { return 0, errors.New("stubbed: no tmux") }
	t.Cleanup(func() {
		rkCronRunner, rkPanesRunner, operatorWindowRoleRunner, trackPanePIDLookup = prevCron, prevPanes, prevWin, prevPID
	})
}

// stubPanePIDLookup stubs the track add/update pane_pid fingerprint seam.
func stubPanePIDLookup(t *testing.T, pid int, err error) {
	t.Helper()
	prev := trackPanePIDLookup
	trackPanePIDLookup = func(paneID string) (int, error) { return pid, err }
	t.Cleanup(func() { trackPanePIDLookup = prev })
}

// stubGHNameWithOwner stubs the gh repo-derivation seam.
func stubGHNameWithOwner(t *testing.T, nameWithOwner string, err error) {
	t.Helper()
	prev := ghNameWithOwnerRunner
	ghNameWithOwnerRunner = func(repoDir string) (string, error) { return nameWithOwner, err }
	t.Cleanup(func() { ghNameWithOwnerRunner = prev })
}

// trackAddArgs builds the common `track add` argv prefix.
func trackAddArgs(id, kind string, extra ...string) []string {
	return append([]string{id, "--kind", kind}, extra...)
}

// --- track add ---------------------------------------------------------------

func TestTrackAdd_GitHubPRDefaults(t *testing.T) {
	// R1: the github-pr kind fills its shell-probe defaults; no legacy section
	// keys are created.
	path := withOperatorState(t, "")
	stubQuietClock(t)
	stubGHNameWithOwner(t, "sahil87/run-kit", nil)

	err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("pr-913", kindGitHubPR,
		"--scope", `{"repo":"/x/hexokit","pr":913}`)...)
	if err != nil {
		t.Fatalf("track add: %v", err)
	}

	state := readStateFile(t, path)
	for _, k := range []string{"monitored", "watches", "autopilot", "notes", "notes_seq"} {
		if _, ok := state[k]; ok {
			t.Errorf("legacy key %q created by track add", k)
		}
	}
	it := readTracked(t, path)["pr-913"]
	if it.Probe.Mode != probeShell {
		t.Fatalf("probe.mode = %q, want shell", it.Probe.Mode)
	}
	wantArgv := "gh pr view 913 --repo sahil87/run-kit --json state,mergedAt,mergeable"
	if strings.Join(it.Probe.Argv, " ") != wantArgv {
		t.Errorf("argv = %v, want %q", it.Probe.Argv, wantArgv)
	}
	if strings.Join(it.Probe.Fields, ",") != "state,mergedAt,mergeable" {
		t.Errorf("fields = %v", it.Probe.Fields)
	}
	if it.DoneWhen == nil || *it.DoneWhen != `state == "MERGED"` {
		t.Errorf("done_when = %v", it.DoneWhen)
	}
	if it.CheckEvery == nil || *it.CheckEvery != "5m" {
		t.Errorf("check_every = %v, want 5m default", it.CheckEvery)
	}
	if it.Failures != 0 || it.Paused || it.Unchanged != 0 {
		t.Errorf("bookkeeping = failures %d paused %v unchanged %d, want zeroed", it.Failures, it.Paused, it.Unchanged)
	}
	for _, ts := range []string{it.AddedAt, it.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			t.Errorf("timestamp %q not RFC3339: %v", ts, err)
		}
	}
}

func TestTrackAdd_GitHubPRDerivationFailureIsAnAddTimeError(t *testing.T) {
	// A failed gh derivation with scope.repo set is an actionable add-time
	// error, never a silent --repo drop (a probe run from the operator's cwd
	// could query the wrong repository in the cross-repo model).
	path := withOperatorState(t, "")
	stubQuietClock(t)
	stubGHNameWithOwner(t, "", errors.New("gh: not logged in"))

	err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("pr-913", kindGitHubPR,
		"--scope", `{"repo":"/x/hexokit","pr":913}`)...)
	if err == nil {
		t.Fatal("track add succeeded, want an add-time error on derivation failure")
	}
	for _, want := range []string{"/x/hexokit", "gh: not logged in", "--argv"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	// Nothing written: the add errored before its save, so the (previously
	// missing) state file must still be absent.
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("state file exists after a failed add (stat err = %v); want no write", statErr)
	}
}

func TestTrackAdd_GitHubPRWithoutScopeRepoNeedsNoDerivation(t *testing.T) {
	// No scope.repo → no derivation attempted; argv carries no --repo and gh
	// infers the repository from the probe's cwd.
	path := withOperatorState(t, "")
	stubQuietClock(t)
	stubGHNameWithOwner(t, "", errors.New("must not be called"))

	err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("pr-913", kindGitHubPR,
		"--scope", `{"pr":913}`)...)
	if err != nil {
		t.Fatalf("track add: %v", err)
	}
	it := readTracked(t, path)["pr-913"]
	wantArgv := "gh pr view 913 --json state,mergedAt,mergeable"
	if strings.Join(it.Probe.Argv, " ") != wantArgv {
		t.Errorf("argv = %v, want %q", it.Probe.Argv, wantArgv)
	}
}

func TestTrackAdd_PaneSugarAndBranchMap(t *testing.T) {
	// R2/R3: the flag sugar lands in scope (all eleven keys present),
	// --change pre-declares the expected change, --pane records the pane_pid
	// fingerprint, and branch_map keys on --change — all in one mutation.
	path := withOperatorState(t, "")
	stubQuietClock(t)
	stubPanePIDLookup(t, 48213, nil)

	err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("r3m7", kindPane,
		"--pane", "%3", "--repo", "/home/u/foo", "--session", "work",
		"--branch", "260324-r3m7-add-retry-logic", "--stage", "apply", "--agent", "active",
		"--stop-stage", "review", "--spawned-by", "linear-bugs", "--change", "r3m7")...)
	if err != nil {
		t.Fatalf("track add: %v", err)
	}

	it := readTracked(t, path)["r3m7"]
	if it.Kind != kindPane || it.Probe.Mode != probePane {
		t.Fatalf("kind/probe = %s/%s", it.Kind, it.Probe.Mode)
	}
	if it.CheckEvery != nil {
		t.Errorf("check_every = %v, want null for pane items", *it.CheckEvery)
	}
	for k, want := range map[string]string{
		"pane": "%3", "change": "r3m7", "repo": "/home/u/foo", "session": "work",
		"branch": "260324-r3m7-add-retry-logic", "stage": "apply", "agent": "active",
		"stop_stage": "review", "spawned_by": "linear-bugs",
	} {
		if got := scopeString(it.Scope, k); got != want {
			t.Errorf("scope.%s = %q, want %q", k, got, want)
		}
	}
	for _, k := range []string{"pane", "pane_pid", "change", "repo", "session", "branch", "stage", "agent", "stop_stage", "spawned_by", "merge_mode"} {
		if _, ok := it.Scope[k]; !ok {
			t.Errorf("scope.%s key missing (eleven pinned keys, null until set)", k)
		}
	}
	if pid, ok := scopeInt(it.Scope, "pane_pid"); !ok || pid != 48213 {
		t.Errorf("scope.pane_pid = %v, want 48213 (recorded at add)", it.Scope["pane_pid"])
	}
	if it.Scope["merge_mode"] != nil {
		t.Errorf("scope.merge_mode = %v, want null (null until chained)", it.Scope["merge_mode"])
	}

	// branch_map keys on --change when given.
	bm := map[string]branchMapEntry{}
	if err := operatorSection(readStateFile(t, path), "branch_map", &bm); err != nil {
		t.Fatalf("decode branch_map: %v", err)
	}
	if bm["r3m7"] != (branchMapEntry{Branch: "260324-r3m7-add-retry-logic", Repo: "/home/u/foo"}) {
		t.Errorf("branch_map.r3m7 = %+v", bm["r3m7"])
	}
}

func TestTrackAdd_PanePIDLookupFailureStoresNull(t *testing.T) {
	// R3: ANY lookup failure (tmux unqueryable, pane absent, unparseable pid)
	// stores pane_pid null and the add proceeds — a null fingerprint means
	// the tick joins on the pane id alone. --scope '{"pane":…}' records too.
	path := withOperatorState(t, "")
	stubQuietClock(t) // the pid seam fails soft here
	stubPanePIDLookup(t, 0, errors.New("tmux: no server"))

	if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("a1", kindPane, "--pane", "%3")...); err != nil {
		t.Fatalf("track add: %v", err)
	}
	if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("a2", kindPane, "--scope", `{"pane":"%9"}`)...); err != nil {
		t.Fatalf("track add --scope: %v", err)
	}
	for _, id := range []string{"a1", "a2"} {
		it := readTracked(t, path)[id]
		if v, present := it.Scope["pane_pid"]; !present || v != nil {
			t.Errorf("%s scope.pane_pid = %v (present %v), want key present with null on lookup failure", id, v, present)
		}
	}

	// A successful lookup records the pid for the --scope form too.
	stubPanePIDLookup(t, 501, nil)
	if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("a3", kindPane, "--scope", `{"pane":"%9"}`)...); err != nil {
		t.Fatalf("track add --scope: %v", err)
	}
	if pid, ok := scopeInt(readTracked(t, path)["a3"].Scope, "pane_pid"); !ok || pid != 501 {
		t.Errorf("a3 scope.pane_pid = %v, want 501", readTracked(t, path)["a3"].Scope["pane_pid"])
	}
}

func TestTrackUpdate_PaneScopeRecordsPID(t *testing.T) {
	// R3: `track update --scope` carrying pane re-records pane_pid; clearing
	// the pane nulls it.
	path := withOperatorState(t, "")
	stubQuietClock(t)
	stubPanePIDLookup(t, 48213, nil)
	if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("a1", kindPane)...); err != nil {
		t.Fatalf("track add: %v", err)
	}
	if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "a1", "--scope", `{"pane":"%7"}`); err != nil {
		t.Fatalf("track update: %v", err)
	}
	it := readTracked(t, path)["a1"]
	if got := scopeString(it.Scope, "pane"); got != "%7" {
		t.Fatalf("scope.pane = %q, want %%7", got)
	}
	if pid, ok := scopeInt(it.Scope, "pane_pid"); !ok || pid != 48213 {
		t.Errorf("scope.pane_pid = %v, want 48213 after the update set the pane", it.Scope["pane_pid"])
	}

	// Clearing the pane nulls the fingerprint.
	if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "a1", "--scope", `{"pane":""}`); err != nil {
		t.Fatalf("track update (clear): %v", err)
	}
	it = readTracked(t, path)["a1"]
	if v := it.Scope["pane_pid"]; v != nil {
		t.Errorf("scope.pane_pid = %v, want null after the pane cleared", v)
	}
}

func TestTrackAdd_ChainResolvesAndPrintsMergeMode(t *testing.T) {
	// A chain add (--depends-on) resolves merge_mode by the ladder and prints
	// `mode: <name> (<source>)`.
	chdirTestEnv(t, t.TempDir(), nil) // no fab project up-tree → built-in default
	path := withOperatorState(t, "")
	stubQuietClock(t)

	if err := runOperatorCmd(t, operatorTrackAddCmd(), trackAddArgs("k8ds", kindPane,
		"--repo", "/r/a", "--branch", "b-k8ds")...); err != nil {
		t.Fatalf("track add first: %v", err)
	}
	out, err := runOperatorCmdOut(t, operatorTrackAddCmd(), trackAddArgs("ef56", kindPane,
		"--repo", "/r/a", "--branch", "b-ef56", "--depends-on", "k8ds")...)
	if err != nil {
		t.Fatalf("track add chained: %v", err)
	}
	if out != "mode: cherry-pick-ladder (default)\n" {
		t.Errorf("stdout = %q, want the mode line", out)
	}
	it := readTracked(t, path)["ef56"]
	if got := scopeString(it.Scope, "merge_mode"); got != "cherry-pick-ladder" {
		t.Errorf("scope.merge_mode = %q", got)
	}
	if strings.Join(it.DependsOn, ",") != "k8ds" {
		t.Errorf("depends_on = %v", it.DependsOn)
	}

	// Explicit --mode wins the ladder and reports the flag source.
	out, err = runOperatorCmdOut(t, operatorTrackAddCmd(), trackAddArgs("gh42", kindPane,
		"--repo", "/r/a", "--branch", "b-gh42", "--mode", "merge-auto")...)
	if err != nil {
		t.Fatalf("track add --mode: %v", err)
	}
	if out != "mode: merge-auto (flag)\n" {
		t.Errorf("stdout = %q, want mode: merge-auto (flag)", out)
	}
}

func TestTrackAdd_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		seed    string
		args    []string
		wantErr string
	}{
		{"unknown kind", "", trackAddArgs("x", "email"), "unknown --kind"},
		{"kind-forbidden probe", "", trackAddArgs("x", kindNote, "--probe", "shell", "--text", "hi"), "does not allow probe mode"},
		{"check-every below floor", "", trackAddArgs("x", kindShell, "--argv", "true", "--fields", "a", "--done-when", `a == 1`, "--check-every", "30s"), "below the 1m0s floor"},
		{"R3 scenario: floor error wins over the done-when requirement", "", trackAddArgs("x", kindShell, "--argv", "true", "--fields", "a", "--check-every", "30s"), "below the 1m0s floor"},
		{"malformed done-when", "", trackAddArgs("x", kindShell, "--argv", "true", "--fields", "a", "--done-when", `a == MERGED`), `invalid --done-when: invalid clause "a == MERGED"`},
		{"shell probe without fields", "", trackAddArgs("x", kindShell, "--argv", "true", "--done-when", `a == 1`), "requires --fields"},
		{"shell probe without argv", "", trackAddArgs("x", kindShell, "--fields", "a", "--done-when", `a == 1`), "requires --argv"},
		{"shell kind without done-when", "", trackAddArgs("x", kindShell, "--argv", "true", "--fields", "a"), "kind shell requires --done-when"},
		{"unknown depends-on id", "", trackAddArgs("x", kindTask, "--depends-on", "zz"), `--depends-on "zz": no tracked item with that id`},
		{"note without text", "", trackAddArgs("x", kindNote), "kind note requires --text"},
		{"text on non-note kind", "", trackAddArgs("x", kindTask, "--text", "hi"), "--text applies only to kind note"},
		{"pane sugar on non-pane kind", "", trackAddArgs("x", kindTask, "--pane", "%3"), "--pane applies only to kind pane"},
		{"change sugar on non-pane kind", "", trackAddArgs("x", kindTask, "--change", "4a8m"), "--change applies only to kind pane"},
		{"empty change sugar rejected", "", trackAddArgs("x", kindPane, "--pane", "%3", "--change", ""), "--change requires a non-empty change id"},
		{"mode on non-pane kind", "", trackAddArgs("x", kindTask, "--mode", "merge-auto"), "--mode applies only to kind pane"},
		{"removed kind fab-change is unknown at add", "", trackAddArgs("x", "fab-change", "--pane", "%3"), `unknown --kind "fab-change" (valid: pane, github-pr, linear, slack, shell, task, note)`},
		{"invalid stage sugar", "", trackAddArgs("x", kindPane, "--stage", "deploy"), "invalid --stage"},
		{"github-pr without scope.pr", "", trackAddArgs("x", kindGitHubPR, "--scope", `{"repo":"/x"}`), "requires scope.pr"},
		{"duplicate id", seedTrackedShell, trackAddArgs("pr-1", kindTask), "tracked item pr-1 already exists"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := withOperatorState(t, tc.seed)
			stubQuietClock(t)
			stubGHNameWithOwner(t, "o/r", nil)
			var before []byte
			if tc.seed != "" {
				before, _ = os.ReadFile(path)
			}
			err := runOperatorCmd(t, operatorTrackAddCmd(), tc.args...)
			if err == nil {
				t.Fatalf("track add %v succeeded, want error %q", tc.args, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantErr)
			}
			if tc.seed == "" {
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Error("state file written on a rejected add")
				}
			} else {
				after, _ := os.ReadFile(path)
				if string(before) != string(after) {
					t.Error("state file changed on a rejected add")
				}
			}
		})
	}
}

// seedTrackedShell is a state file carrying one shell item (id pr-1) and one
// done github-pr item (id pr-done) — the shared seed for update/observe/rm
// tests.
const seedTrackedShell = `tracked:
  - id: pr-1
    kind: shell
    probe:
      mode: shell
      argv: [myprobe, --flag]
      fields: [status]
    check_every: 2m
    done_when: 'status == "green"'
    then: null
    depends_on: []
    scope: {}
    last: {status: red}
    checked_at: "2026-09-10T00:00:00Z"
    unchanged: 1
    failures: 0
    paused: false
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-10T00:00:00Z"
  - id: pr-done
    kind: github-pr
    probe:
      mode: shell
      argv: [gh, pr, view, "5"]
      fields: [state]
    check_every: 5m
    done_when: 'state == "MERGED"'
    then: null
    depends_on: []
    scope: {repo: /x, pr: 5}
    last: {state: MERGED}
    checked_at: "2026-09-10T00:00:00Z"
    unchanged: 0
    failures: 0
    paused: false
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-10T00:00:00Z"
branch_map: {}
`

// --- track update --------------------------------------------------------------

func TestTrackUpdate(t *testing.T) {
	path := withOperatorState(t, seedTrackedShell)
	stubQuietClock(t)

	err := runOperatorCmd(t, operatorTrackUpdateCmd(), "pr-1",
		"--check-every", "10m", "--done-when", `status == "green" and extra == null`,
		"--then", "deploy next", "--scope", `{"env":"prod"}`)
	if err != nil {
		t.Fatalf("track update: %v", err)
	}
	it := readTracked(t, path)["pr-1"]
	if it.CheckEvery == nil || *it.CheckEvery != "10m" {
		t.Errorf("check_every = %v", it.CheckEvery)
	}
	if it.DoneWhen == nil || *it.DoneWhen != `status == "green" and extra == null` {
		t.Errorf("done_when = %v", it.DoneWhen)
	}
	if it.Then == nil || *it.Then != "deploy next" {
		t.Errorf("then = %v", it.Then)
	}
	if it.Scope["env"] != "prod" {
		t.Errorf("scope = %v, want env merged in", it.Scope)
	}
	if it.UpdatedAt == "2026-09-10T00:00:00Z" {
		t.Error("updated_at not bumped")
	}

	// --done-when "" clears to null.
	if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "pr-1", "--done-when", ""); err != nil {
		t.Fatalf("clear done-when: %v", err)
	}
	if it := readTracked(t, path)["pr-1"]; it.DoneWhen != nil {
		t.Errorf("done_when = %v, want cleared", *it.DoneWhen)
	}
}

func TestTrackUpdate_PauseResume(t *testing.T) {
	path := withOperatorState(t, seedTrackedShell)
	stubQuietClock(t)

	if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "pr-1", "--pause"); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if it := readTracked(t, path)["pr-1"]; !it.Paused {
		t.Error("paused = false after --pause")
	}

	// Simulate tripped failures, then resume: failures zero.
	if err := runOperatorCmd(t, operatorTrackObserveCmd(), "pr-1", "--error", "boom"); err != nil {
		t.Fatalf("observe --error: %v", err)
	}
	if err := runOperatorCmd(t, operatorTrackUpdateCmd(), "pr-1", "--resume"); err != nil {
		t.Fatalf("resume: %v", err)
	}
	it := readTracked(t, path)["pr-1"]
	if it.Paused || it.Failures != 0 {
		t.Errorf("after --resume: paused=%v failures=%d, want false/0", it.Paused, it.Failures)
	}

	err := runOperatorCmd(t, operatorTrackUpdateCmd(), "pr-1", "--pause", "--resume")
	if err == nil || !strings.Contains(err.Error(), "flags in the group") {
		t.Errorf("--pause --resume = %v, want mutually-exclusive error", err)
	}
}

func TestTrackUpdate_Errors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown id", []string{"zz", "--pause"}, "no tracked item zz"},
		{"check-every below floor", []string{"pr-1", "--check-every", "10s"}, "below the 1m0s floor"},
		{"malformed done-when", []string{"pr-1", "--done-when", "a > 1"}, "invalid --done-when"},
		{"unknown depends-on", []string{"pr-1", "--depends-on", "zz"}, "no tracked item with that id"},
		{"text on non-note", []string{"pr-1", "--text", "hi"}, "--text applies only to kind note"},
		{"bad scope JSON", []string{"pr-1", "--scope", "{nope"}, "invalid --scope JSON"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := withOperatorState(t, seedTrackedShell)
			stubQuietClock(t)
			before, _ := os.ReadFile(path)
			err := runOperatorCmd(t, operatorTrackUpdateCmd(), tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("update %v = %v, want %q", tc.args, err, tc.wantErr)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Error("state file changed on a rejected update")
			}
		})
	}
}

// --- track rm ------------------------------------------------------------------

func TestTrackRm(t *testing.T) {
	seed := `tracked:
  - id: ab12
    kind: pane
    probe: {mode: pane}
    depends_on: []
    scope: {pane: "%3", repo: /home/u/foo, branch: 260909-ab12-x}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
branch_map:
  ab12: {branch: 260909-ab12-x, repo: /home/u/foo}
`
	path := withOperatorState(t, seed)
	stubQuietClock(t)

	if err := runOperatorCmd(t, operatorTrackRmCmd(), "zz"); err == nil {
		t.Error("rm unknown id succeeded, want error")
	}
	if err := runOperatorCmd(t, operatorTrackRmCmd(), "ab12"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	state := readStateFile(t, path)
	if items := readTracked(t, path); len(items) != 0 {
		t.Errorf("tracked = %v, want empty after rm", items)
	}
	// R6: branch_map survives track rm.
	bm := map[string]branchMapEntry{}
	if err := operatorSection(state, "branch_map", &bm); err != nil {
		t.Fatal(err)
	}
	if _, ok := bm["ab12"]; !ok {
		t.Error("branch_map entry must survive track rm")
	}
}

// --- track observe ---------------------------------------------------------------

func TestTrackObserve_StoresFieldsAndSeen(t *testing.T) {
	// R4 GIVEN/WHEN/THEN: observe stores the observed object, sets checked_at,
	// resets failures, appends --seen ids.
	seed := `tracked:
  - id: linear-bugs
    kind: linear
    probe:
      mode: agent
      instruction: list issues
    check_every: 5m
    done_when: null
    then: null
    depends_on: []
    scope: {repo: /home/u/foo}
    last: {}
    seen: [A]
    checked_at: null
    unchanged: 0
    failures: 2
    paused: false
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`
	path := withOperatorState(t, seed)
	stubQuietClock(t)

	err := runOperatorCmd(t, operatorTrackObserveCmd(), "linear-bugs", "--json", `{"new":["B"]}`, "--seen", "B")
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	it := readTracked(t, path)["linear-bugs"]
	if !reflect.DeepEqual(it.Last, map[string]interface{}{"new": []interface{}{"B"}}) {
		t.Errorf("last = %v, want {new: [B]}", it.Last)
	}
	if strings.Join(it.Seen, ",") != "A,B" {
		t.Errorf("seen = %v, want [A B]", it.Seen)
	}
	if it.CheckedAt == nil {
		t.Error("checked_at not set")
	}
	if it.Failures != 0 {
		t.Errorf("failures = %d, want reset to 0", it.Failures)
	}

	// A repeated observation of the same fields bumps unchanged.
	if err := runOperatorCmd(t, operatorTrackObserveCmd(), "linear-bugs", "--json", `{"new":["B"]}`); err != nil {
		t.Fatalf("observe (repeat): %v", err)
	}
	if it := readTracked(t, path)["linear-bugs"]; it.Unchanged != 1 {
		t.Errorf("unchanged = %d after repeat, want 1", it.Unchanged)
	}
}

func TestTrackObserve_ExtractsDeclaredFieldsOnly(t *testing.T) {
	path := withOperatorState(t, seedTrackedShell)
	stubQuietClock(t)

	err := runOperatorCmd(t, operatorTrackObserveCmd(), "pr-1", "--json", `{"status":"green","extra":1}`)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	it := readTracked(t, path)["pr-1"]
	if !reflect.DeepEqual(it.Last, map[string]interface{}{"status": "green"}) {
		t.Errorf("last = %v, want only the declared field", it.Last)
	}
	if it.Unchanged != 0 {
		t.Errorf("unchanged = %d on a field delta, want 0", it.Unchanged)
	}
	if !trackedItemDone(it) {
		t.Error("done_when must fire after the observed flip (status == green)")
	}
}

func TestTrackObserve_ErrorFailuresPause(t *testing.T) {
	path := withOperatorState(t, seedTrackedShell)
	stubQuietClock(t)

	for i := 1; i <= trackFailurePauseCap; i++ {
		if err := runOperatorCmd(t, operatorTrackObserveCmd(), "pr-1", "--error", "timeout"); err != nil {
			t.Fatalf("observe --error %d: %v", i, err)
		}
		it := readTracked(t, path)["pr-1"]
		if it.Failures != i {
			t.Errorf("failures = %d after %d errors", it.Failures, i)
		}
		wantPaused := i == trackFailurePauseCap
		if it.Paused != wantPaused {
			t.Errorf("paused = %v after %d errors, want %v", it.Paused, i, wantPaused)
		}
	}
}

func TestTrackObserve_Validation(t *testing.T) {
	stubQuietClock(t)
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown id", []string{"zz", "--json", `{}`}, "no tracked item zz"},
		{"neither flag", []string{"pr-1"}, "exactly one of --json or --error"},
		{"both flags", []string{"pr-1", "--json", `{}`, "--error", "x"}, "exactly one of --json or --error"},
		{"non-object json", []string{"pr-1", "--json", `[1]`}, "must be a JSON object"},
		{"invalid json", []string{"pr-1", "--json", `{nope`}, "invalid --json"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := withOperatorState(t, seedTrackedShell)
			before, _ := os.ReadFile(path)
			err := runOperatorCmd(t, operatorTrackObserveCmd(), tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("observe %v = %v, want %q", tc.args, err, tc.wantErr)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Error("state file changed on a rejected observe")
			}
		})
	}
}

func TestTrackObserve_SeenCapPrunesOldest(t *testing.T) {
	seed := `tracked:
  - id: w
    kind: slack
    probe: {mode: agent, instruction: check}
    check_every: 5m
    depends_on: []
    scope: {}
    last: {}
    seen: []
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`
	path := withOperatorState(t, seed)
	stubQuietClock(t)
	// Seed seen with 200 entries, then observe one more: the oldest prunes.
	items := readTracked(t, path)
	_ = items
	seen := make([]string, 0, trackSeenCap)
	for i := 0; i < trackSeenCap; i++ {
		seen = append(seen, fmt.Sprintf("S%03d", i))
	}
	err := mutateOperatorState(func(data map[string]interface{}) error {
		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		items[0].Seen = seen
		data["tracked"] = items
		return nil
	})
	if err != nil {
		t.Fatalf("seed seen: %v", err)
	}
	if err := runOperatorCmd(t, operatorTrackObserveCmd(), "w", "--json", `{}`, "--seen", "NEW"); err != nil {
		t.Fatalf("observe: %v", err)
	}
	it := readTracked(t, path)["w"]
	if len(it.Seen) != trackSeenCap {
		t.Fatalf("seen = %d entries, want the %d cap", len(it.Seen), trackSeenCap)
	}
	if it.Seen[0] != "S001" || it.Seen[trackSeenCap-1] != "NEW" {
		t.Errorf("seen = [%s … %s], want oldest pruned and NEW appended", it.Seen[0], it.Seen[trackSeenCap-1])
	}
}

// --- track list ------------------------------------------------------------------

func TestTrackList(t *testing.T) {
	seed := `tracked:
  - id: k8ds
    kind: pane
    probe: {mode: pane}
    depends_on: []
    scope: {pane: "%7", repo: /r/a}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: ef56
    kind: pane
    probe: {mode: pane}
    depends_on: [k8ds]
    scope: {pane: null, repo: /r/a}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: pr-9
    kind: github-pr
    probe: {mode: shell, argv: [gh, pr, view, "9"], fields: [state]}
    check_every: 2m
    done_when: 'state == "MERGED"'
    then: arm next
    depends_on: []
    scope: {repo: /r/a, pr: 9}
    last: {state: MERGED}
    checked_at: "2026-09-09T00:00:00Z"
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: n1
    kind: note
    probe: {mode: none}
    depends_on: []
    scope: {}
    last: {}
    text: phase 2 of 4
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`
	withOperatorState(t, seed)

	out, err := runOperatorCmdOut(t, operatorTrackListCmd())
	if err != nil {
		t.Fatalf("track list: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("track list = %q, want 4 lines", out)
	}
	if !strings.HasPrefix(lines[0], "k8ds · pane · live · live · —") {
		t.Errorf("line 0 = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "ef56 · pane · held · — · held: k8ds") {
		t.Errorf("line 1 = %q", lines[1])
	}
	if !strings.HasPrefix(lines[2], "pr-9 · github-pr · done · ") || !strings.HasSuffix(lines[2], " · arm next") {
		t.Errorf("line 2 = %q", lines[2])
	}
	if !strings.HasPrefix(lines[3], "n1 · note · watching · ") {
		t.Errorf("line 3 = %q", lines[3])
	}

	// --kind filter.
	out, err = runOperatorCmdOut(t, operatorTrackListCmd(), "--kind", "note")
	if err != nil {
		t.Fatalf("track list --kind note: %v", err)
	}
	if strings.Count(strings.TrimRight(out, "\n"), "\n") != 0 || !strings.HasPrefix(out, "n1 ·") {
		t.Errorf("--kind note = %q, want only n1", out)
	}

	// The retired kind is unknown at track list too (no alias).
	_, err = runOperatorCmdOut(t, operatorTrackListCmd(), "--kind", "fab-change")
	if err == nil || !strings.Contains(err.Error(), `unknown --kind "fab-change" (valid: pane,`) {
		t.Errorf("--kind fab-change = %v, want the unknown-kind error naming pane first", err)
	}

	// --json emits the items array.
	out, err = runOperatorCmdOut(t, operatorTrackListCmd(), "--json")
	if err != nil {
		t.Fatalf("track list --json: %v", err)
	}
	var items []trackedItem
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("--json output: %v", err)
	}
	if len(items) != 4 || items[0].ID != "k8ds" {
		t.Errorf("--json items = %v", items)
	}
}

// --- track clock (R13) -----------------------------------------------------------

func TestTrackClock(t *testing.T) {
	t.Run("writes a bounded override", func(t *testing.T) {
		path := withOperatorState(t, seedTrackedShell)
		stubQuietClock(t)
		before := time.Now().UTC()
		if err := runOperatorCmd(t, operatorTrackClockCmd(), "--every", "10m", "--for", "2h"); err != nil {
			t.Fatalf("track clock: %v", err)
		}
		o := readClockOverride(readStateFile(t, path))
		if o == nil {
			t.Fatal("clock_override missing after track clock")
		}
		if o.Schedule.Kind != "every" || o.Schedule.Every != "10m" || o.Deliver != "skip-if-busy" {
			t.Errorf("override = %+v", o)
		}
		until, err := time.Parse(time.RFC3339, o.Until)
		if err != nil {
			t.Fatalf("until %q: %v", o.Until, err)
		}
		// until is RFC3339 (second-granularity, truncated) — allow slack.
		if until.Before(before.Add(2*time.Hour-5*time.Second)) || until.After(time.Now().UTC().Add(2*time.Hour+time.Minute)) {
			t.Errorf("until = %v, want ≈ now+2h", until)
		}
	})

	t.Run("--idle-every writes the idle-every kind", func(t *testing.T) {
		path := withOperatorState(t, seedTrackedShell)
		stubQuietClock(t)
		if err := runOperatorCmd(t, operatorTrackClockCmd(), "--idle-every", "3m", "--for", "30m"); err != nil {
			t.Fatalf("track clock: %v", err)
		}
		if o := readClockOverride(readStateFile(t, path)); o == nil || o.Schedule.Kind != "idle-every" {
			t.Errorf("override = %+v, want idle-every", o)
		}
	})

	t.Run("--off clears the override", func(t *testing.T) {
		path := withOperatorState(t, seedTrackedShell)
		stubQuietClock(t)
		if err := runOperatorCmd(t, operatorTrackClockCmd(), "--every", "10m", "--for", "2h"); err != nil {
			t.Fatal(err)
		}
		if err := runOperatorCmd(t, operatorTrackClockCmd(), "--off"); err != nil {
			t.Fatalf("clock --off: %v", err)
		}
		if _, ok := readStateFile(t, path)["clock_override"]; ok {
			t.Error("clock_override survived --off")
		}
	})

	t.Run("validation writes nothing", func(t *testing.T) {
		tests := []struct {
			name    string
			args    []string
			wantErr string
		}{
			{"missing --for", []string{"--every", "10m"}, "requires --for"},
			{"neither cadence", []string{"--for", "1h"}, "requires --every or --idle-every"},
			{"both cadences", []string{"--every", "10m", "--idle-every", "5m", "--for", "1h"}, "exactly one"},
			{"below floor", []string{"--every", "30s", "--for", "1h"}, "below the 1m0s floor"},
			{"bad --for", []string{"--every", "10m", "--for", "later"}, "invalid --for"},
			{"--off with flags", []string{"--off", "--every", "10m"}, "takes no other flags"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				path := withOperatorState(t, seedTrackedShell)
				stubQuietClock(t)
				before, _ := os.ReadFile(path)
				err := runOperatorCmd(t, operatorTrackClockCmd(), tc.args...)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("clock %v = %v, want %q", tc.args, err, tc.wantErr)
				}
				after, _ := os.ReadFile(path)
				if string(before) != string(after) {
					t.Error("state file changed on a rejected clock")
				}
			})
		}
	})
}

// --- state OPEN NOTES header -------------------------------------------------------

func TestStateOpenNotesHeader(t *testing.T) {
	seed := `tracked:
  - id: n1
    kind: note
    probe: {mode: none}
    depends_on: []
    scope: {}
    last: {}
    text: phase 2 of 4 — hexokit after #913
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: n2
    kind: note
    probe: {mode: none}
    depends_on: []
    scope: {}
    last: {}
    text: second note
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`
	withOperatorState(t, seed)
	out, err := runOperatorCmdOut(t, operatorStateCmd())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if !strings.HasPrefix(out, "# OPEN NOTES (2)\n# n1 · note · ") {
		t.Errorf("state stdout missing the OPEN NOTES header:\n%s", out)
	}
	if !strings.Contains(out, "# n1 · note · ") || !strings.Contains(out, "phase 2 of 4 — hexokit after #913") {
		t.Errorf("header line wrong:\n%s", out)
	}

	// --json carries no comment header.
	out, err = runOperatorCmdOut(t, operatorStateCmd(), "--json")
	if err != nil {
		t.Fatalf("state --json: %v", err)
	}
	if strings.Contains(out, "OPEN NOTES") {
		t.Errorf("--json output must not carry the comment header:\n%s", out)
	}

	// No note items → no header.
	withOperatorState(t, seedTrackedShell)
	out, err = runOperatorCmdOut(t, operatorStateCmd())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if strings.Contains(out, "OPEN NOTES") {
		t.Errorf("header emitted with no note items:\n%s", out)
	}
}

func TestStateSkeletonOnMissing(t *testing.T) {
	path := withOperatorState(t, "")
	out, err := runOperatorCmdOut(t, operatorStateCmd())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	state := readStateFile(t, path)
	if _, ok := state["tracked"]; !ok {
		t.Errorf("skeleton missing tracked: %v", state)
	}
	if _, ok := state["branch_map"]; !ok {
		t.Errorf("skeleton missing branch_map: %v", state)
	}
	for _, k := range []string{"monitored", "watches", "autopilot", "notes"} {
		if _, ok := state[k]; ok {
			t.Errorf("skeleton carries legacy key %q", k)
		}
	}
	if !strings.Contains(out, "tracked: []") {
		t.Errorf("stdout = %q, want tracked: []", out)
	}
}

// --- PR #663 review fixes ------------------------------------------------------

func TestTrackRm_DoneDependencyDropsSatisfiedEdges(t *testing.T) {
	// Removing a DONE dependency drops it from every dependent's depends_on
	// (a satisfied edge is inert), so the chain advances after the ack.
	seed := `tracked:
  - id: pr-1
    kind: shell
    probe: {mode: shell, argv: [true], fields: [status]}
    check_every: 5m
    done_when: 'status == "green"'
    depends_on: []
    scope: {}
    last: {status: green}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: n34
    kind: pane
    probe: {mode: pane}
    depends_on: [pr-1, other]
    scope: {repo: /home/u/foo, branch: 260909-n34-x}
    last: {}
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
  - id: other
    kind: note
    probe: {mode: none}
    depends_on: []
    scope: {}
    last: {}
    text: standing note
    added_at: "2026-09-09T00:00:00Z"
    updated_at: "2026-09-09T00:00:00Z"
`
	path := withOperatorState(t, seed)
	stubQuietClock(t)

	if err := runOperatorCmd(t, operatorTrackRmCmd(), "pr-1"); err != nil {
		t.Fatalf("rm done dep: %v", err)
	}
	items := readTracked(t, path)
	if got := items["n34"].DependsOn; !reflect.DeepEqual(got, []string{"other"}) {
		t.Errorf("n34.depends_on = %v after removing the done dep, want [other]", got)
	}

	// Removing a NOT-done dependency keeps the edge — dependents stay held
	// and the tick names the missing id.
	if err := runOperatorCmd(t, operatorTrackRmCmd(), "other"); err != nil {
		t.Fatalf("rm not-done dep: %v", err)
	}
	items = readTracked(t, path)
	if got := items["n34"].DependsOn; !reflect.DeepEqual(got, []string{"other"}) {
		t.Errorf("n34.depends_on = %v after removing a not-done dep, want [other] kept", got)
	}
}

func TestExtractProbeFields_MergesSharedDottedPrefix(t *testing.T) {
	obj := map[string]interface{}{
		"a": map[string]interface{}{"b": 1, "c": 2, "d": 3},
		"x": "y",
	}
	got := extractProbeFields(obj, []string{"a.b", "a.c", "missing.k"})
	want := map[string]interface{}{
		"a":       map[string]interface{}{"b": 1, "c": 2},
		"missing": map[string]interface{}{"k": nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractProbeFields = %#v, want %#v", got, want)
	}
}
