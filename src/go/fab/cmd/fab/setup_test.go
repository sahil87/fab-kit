package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/configupgrade"
)

// setupCheckFixture builds a minimal fab repo (fab/project/config.yaml +
// fab/.fab-version), a stub PATH dir with the given fake executables, and a
// stub kit cache (VERSION file) wired through FAB_KIT_PATH — then chdirs into
// the repo with TMUX scrubbed and HOME isolated (the system config tier must
// not leak the developer machine's own ~/.fab-kit/config.yaml into the run).
// version is "dev" under test, and dev is never compared, so no fixture
// VERSION value can produce a skew warning.
func setupCheckFixture(t *testing.T, configYAML string, pathStubs ...string) string {
	t.Helper()

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "fab", "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "fab", "project", "config.yaml"), []byte(configYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "fab", ".fab-version"), []byte("2.19.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bin := t.TempDir()
	for _, name := range pathStubs {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	kit := t.TempDir()
	if err := os.WriteFile(filepath.Join(kit, "VERSION"), []byte("2.19.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", bin)
	t.Setenv("FAB_KIT_PATH", kit)
	chdirTestEnv(t, repo, map[string]string{"TMUX": ""})
	return repo
}

// runFab executes the assembled root command through the real run() exit-code
// mapping, capturing both streams.
func runFab(args ...string) (int, string, string) {
	var out, errBuf bytes.Buffer
	code := run(args, &out, &errBuf)
	return code, out.String(), errBuf.String()
}

// runSetupWizard executes the bare setup command directly (bypassing run()'s
// exit-code mapping) with injected stdin, capturing both streams. cobra's
// SetIn buffer is never a TTY, so interactive-path tests pair it with
// forceTTY(true) — the same seam batch_archive's prompt tests use.
func runSetupWizardCmd(t *testing.T, stdin string, args ...string) (error, string, string) {
	t.Helper()
	return runSetupWizardCmdReader(t, strings.NewReader(stdin), args...)
}

// runSetupWizardCmdReader is runSetupWizardCmd with an arbitrary stdin reader,
// for tests that need a failing (non-EOF) stdin.
func runSetupWizardCmdReader(t *testing.T, stdin io.Reader, args ...string) (error, string, string) {
	t.Helper()
	cmd := setupCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(stdin)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return err, out.String(), errBuf.String()
}

// errAfterReader yields r's bytes, then a non-EOF error where EOF would be.
type errAfterReader struct {
	r   io.Reader
	err error
}

func (e *errAfterReader) Read(p []byte) (int, error) {
	n, err := e.r.Read(p)
	if errors.Is(err, io.EOF) {
		return n, e.err
	}
	return n, err
}

func TestSetupWizard_AllEnterRunIsZeroWrite(t *testing.T) {
	setupCheckFixture(t, "", "claude")
	forceTTY(t, true)

	err, out, _ := runSetupWizardCmd(t, "\n\n\n\n\n")
	if err != nil {
		t.Fatalf("all-Enter run error = %v; output:\n%s", err, out)
	}
	for _, want := range []string{"Configuring the system tier", "agent.session", "agent.workers", "dispatch.mode", "nothing to change"} {
		if !strings.Contains(out, want) {
			t.Errorf("all-Enter run output missing %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Yet to be implemented") {
		t.Errorf("the placeholder string must be gone, got:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("all-Enter run must write NO file, stat err = %v", statErr)
	}
}

func TestSetupWizard_SystemScaffoldWarmup(t *testing.T) {
	writeSystem := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("clean is silent", func(t *testing.T) {
		setupCheckFixture(t, "", "claude")
		forceTTY(t, false)
		clean, err := configupgrade.RenderSystemScaffold(version)
		if err != nil {
			t.Fatal(err)
		}
		path := writeSystem(t, clean)
		before, _ := os.ReadFile(path)
		err, out, errOut := runSetupWizardCmd(t, "", "--defaults")
		if err != nil {
			t.Fatalf("setup with clean system scaffold: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
		}
		if strings.Contains(out, "Refreshed ") {
			t.Fatalf("clean warm-up printed a refresh advisory: %q", out)
		}
		after, _ := os.ReadFile(path)
		if string(after) != string(before) {
			t.Fatal("clean warm-up changed the system config")
		}
	})

	t.Run("drift prints one advisory", func(t *testing.T) {
		setupCheckFixture(t, "", "claude")
		forceTTY(t, false)
		path := writeSystem(t, "# work laptop only\n")
		err, out, errOut := runSetupWizardCmd(t, "", "--defaults")
		if err != nil {
			t.Fatalf("setup with drifted scaffold: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
		}
		if count := strings.Count(out, "Refreshed "); count != 1 {
			t.Fatalf("warm-up printed %d refresh advisories, want 1: %q", count, out)
		}
		got, _ := os.ReadFile(path)
		if !strings.Contains(string(got), "# >>> fab reference") || !strings.Contains(string(got), "# work laptop only") {
			t.Fatalf("warm-up did not refresh while preserving the user comment:\n%s", got)
		}
	})

	t.Run("failure warns and continues", func(t *testing.T) {
		setupCheckFixture(t, "", "claude")
		forceTTY(t, false)
		path := writeSystem(t, "agent: [\n")
		err, out, errOut := runSetupWizardCmd(t, "", "--defaults")
		if err != nil {
			t.Fatalf("warm-up error aborted the setup interview: %v\nstdout:\n%s\nstderr:\n%s", err, out, errOut)
		}
		if !strings.Contains(errOut, "warning: could not refresh") {
			t.Fatalf("warm-up failure emitted no warning: %q", errOut)
		}
		if !strings.Contains(out, "Configuring the system tier") {
			t.Fatalf("interview did not continue after warm-up failure: %q", out)
		}
		if got, _ := os.ReadFile(path); string(got) != "agent: [\n" {
			t.Fatalf("failed warm-up changed malformed config: %q", got)
		}
	})
}

func TestSetupWizard_ChangedAnswerDiffsAndWritesSystemTier(t *testing.T) {
	setupCheckFixture(t, "", "claude", "codex")
	forceTTY(t, true)

	// Q1 Enter, Q2 codex, Q3 Enter, Q4 n, confirm y.
	err, out, _ := runSetupWizardCmd(t, "\ncodex\n\nn\ny\n")
	if err != nil {
		t.Fatalf("changed-answer run error = %v; output:\n%s", err, out)
	}
	if !strings.Contains(out, "agent.workers: claude → codex") {
		t.Errorf("diff summary missing the agent.workers change, got:\n%s", out)
	}
	data, readErr := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml"))
	if readErr != nil {
		t.Fatalf("confirmed write must land in the system tier: %v\noutput:\n%s", readErr, out)
	}
	if !strings.Contains(string(data), "workers: codex") {
		t.Errorf("system config must carry the surgical write, got:\n%s", string(data))
	}
}

func TestSetupWizard_DeclinedConfirmationWritesNothing(t *testing.T) {
	setupCheckFixture(t, "", "claude", "codex")
	forceTTY(t, true)

	// Q2 changed to codex, but the write confirmation is declined.
	err, out, _ := runSetupWizardCmd(t, "\ncodex\n\nn\nn\n")
	if err != nil {
		t.Fatalf("declined-confirmation run error = %v; output:\n%s", err, out)
	}
	if !strings.Contains(out, "No changes written.") {
		t.Errorf("declined confirmation must say so, got:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("a declined confirmation must write NO file, stat err = %v", statErr)
	}
}

func TestSetupWizard_ProjectFlagWritesProjectConfig(t *testing.T) {
	repo := setupCheckFixture(t, "", "claude", "codex")
	forceTTY(t, true)

	err, out, _ := runSetupWizardCmd(t, "\ncodex\n\nn\ny\n", "--project")
	if err != nil {
		t.Fatalf("--project run error = %v; output:\n%s", err, out)
	}
	if !strings.Contains(out, "Configuring the project tier") {
		t.Errorf("--project banner must name the project tier, got:\n%s", out)
	}
	data, readErr := os.ReadFile(filepath.Join(repo, "fab", "project", "config.yaml"))
	if readErr != nil {
		t.Fatalf("project config unreadable after --project write: %v", readErr)
	}
	if !strings.Contains(string(data), "workers: codex") {
		t.Errorf("--project write must land in fab/project/config.yaml, got:\n%s", string(data))
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("--project must not touch the system tier, stat err = %v", statErr)
	}
}

func TestSetupWizard_ProjectFlagOutsideRepoErrors(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", bin)
	t.Setenv("FAB_KIT_PATH", t.TempDir())
	chdirTestEnv(t, t.TempDir(), map[string]string{"TMUX": ""})
	forceTTY(t, true)

	err, _, _ := runSetupWizardCmd(t, "", "--project")
	if err == nil {
		t.Fatal("--project outside a fab repo must fail")
	}
	if !strings.Contains(err.Error(), "--project") {
		t.Errorf("error should name --project, got: %v", err)
	}
}

func TestSetupWizard_DefaultsRunWithMissingConfigIsNonInteractiveAndZeroWrite(t *testing.T) {
	setupCheckFixture(t, "", "claude")
	forceTTY(t, false) // a non-TTY run MUST complete under --defaults without reading stdin

	err, out, _ := runSetupWizardCmd(t, "", "--defaults")
	if err != nil {
		t.Fatalf("--defaults run error = %v; output:\n%s", err, out)
	}
	for _, want := range []string{"Configuring the system tier", "accepted by --defaults", "nothing to change"} {
		if !strings.Contains(out, want) {
			t.Errorf("--defaults output missing %q, got:\n%s", want, out)
		}
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("--defaults with a missing system config must be a zero-write run, stat err = %v", statErr)
	}
}

func TestSetupWizard_DefaultsComposesWithProject(t *testing.T) {
	setupCheckFixture(t, "", "claude")
	forceTTY(t, false)

	err, out, _ := runSetupWizardCmd(t, "", "--defaults", "--project")
	if err != nil {
		t.Fatalf("--defaults --project run error = %v; output:\n%s", err, out)
	}
	if !strings.Contains(out, "Configuring the project tier") {
		t.Errorf("composed run must target the project tier, got:\n%s", out)
	}
	if !strings.Contains(out, "nothing to change") {
		t.Errorf("composed run must stay zero-write, got:\n%s", out)
	}
}

func TestSetupWizard_NonTTYWithoutDefaultsFails(t *testing.T) {
	setupCheckFixture(t, "", "claude")
	forceTTY(t, false)

	// Direct: the RunE error names the non-interactive escape hatch.
	err, _, _ := runSetupWizardCmd(t, "")
	if err == nil {
		t.Fatal("non-TTY without --defaults must fail")
	}
	if !strings.Contains(err.Error(), "--defaults") {
		t.Errorf("error must name --defaults, got: %v", err)
	}

	// Through the real run() seam: an in-RunE failure is operational (exit 1).
	code, _, errOut := runFab("setup")
	if code != 1 {
		t.Errorf("non-TTY bare `fab setup` exit = %d, want 1 (operational)", code)
	}
	if !strings.Contains(errOut, "--defaults") {
		t.Errorf("stderr must carry the --defaults hint, got %q", errOut)
	}
}

func TestSetupWizard_ProviderOptionsFilterToDetected(t *testing.T) {
	// Only claude's binary is on PATH: codex/agy/kimi are dropped outright,
	// and claude carries its capability annotation.
	setupCheckFixture(t, "", "claude")
	forceTTY(t, true)

	err, out, _ := runSetupWizardCmd(t, "\n\n\n\n\n")
	if err != nil {
		t.Fatalf("run error = %v; output:\n%s", err, out)
	}
	q1 := out[strings.Index(out, "Q1:"):strings.Index(out, "Q2:")]
	if !strings.Contains(q1, "claude (interactive, headless, native)") {
		t.Errorf("Q1 must annotate claude's capabilities, got:\n%s", q1)
	}
	for _, absent := range []string{"codex", "agy", "kimi"} {
		if strings.Contains(q1, absent) {
			t.Errorf("Q1 must drop undetected provider %q, got:\n%s", absent, q1)
		}
	}
}

func TestSetupWizard_DispatchModeFiltersWithoutTmux(t *testing.T) {
	// The fixture scrubs $TMUX: pane is not viable, and the ladder semantics
	// must be stated in the question text.
	setupCheckFixture(t, "", "claude")
	forceTTY(t, true)

	err, out, _ := runSetupWizardCmd(t, "\n\n\n\n\n")
	if err != nil {
		t.Fatalf("run error = %v; output:\n%s", err, out)
	}
	q3 := out[strings.Index(out, "Q3:"):strings.Index(out, "Q4:")]
	if strings.Contains(q3, "pane") && !strings.Contains(q3, "pane →") {
		t.Errorf("Q3 must not OFFER pane without tmux, got:\n%s", q3)
	}
	for _, want := range []string{"native", "headless", "never ascends"} {
		if !strings.Contains(q3, want) {
			t.Errorf("Q3 missing %q, got:\n%s", want, q3)
		}
	}
}

func TestSetupWizard_AdvancedOptInAsksAllKeys(t *testing.T) {
	// Fresh machine (empty config): opting in asks all four advanced questions
	// anyway — the sparse profile keys render their depth-correct inherit
	// indication — and an all-Enter pass through them still writes nothing.
	setupCheckFixture(t, "", "claude")
	forceTTY(t, true)

	err, out, _ := runSetupWizardCmd(t, "\n\n\ny\n\n\n\n\n")
	if err != nil {
		t.Fatalf("run error = %v; output:\n%s", err, out)
	}
	for _, prompt := range []string{
		"agent.profiles.operator.provider [(inherit agent.session)]:",
		"agent.profiles.review.provider [(inherit agent.workers)]:",
		"dispatch.column_width [35]:",
		"dispatch.reap_done [true]:",
	} {
		if !strings.Contains(out, prompt) {
			t.Errorf("opted-in advanced section must ask %q, got:\n%s", prompt, out)
		}
	}
	if !strings.Contains(out, "nothing to change") {
		t.Errorf("an all-Enter advanced pass must record zero changes, got:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("an all-Enter advanced pass must write NO file, stat err = %v", statErr)
	}
}

func TestSetupWizard_StdinReadErrorAbortsWrite(t *testing.T) {
	// The interview produces a change, then stdin fails with a non-EOF error
	// at the write confirmation (whose default is Yes): the run must abort
	// without writing — a failing stdin is never an implicit confirmation.
	setupCheckFixture(t, "", "claude", "codex")
	forceTTY(t, true)

	stdin := &errAfterReader{r: strings.NewReader("\ncodex\n\nn\n"), err: errors.New("input/output error")}
	err, out, _ := runSetupWizardCmdReader(t, stdin)
	if err == nil {
		t.Fatalf("a stdin read error must abort the run, output:\n%s", out)
	}
	if !strings.Contains(err.Error(), "stdin read failed") || !strings.Contains(err.Error(), "input/output error") {
		t.Errorf("error must name the stdin failure and wrap its cause, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml")); !os.IsNotExist(statErr) {
		t.Errorf("a read-error run must write NO file, stat err = %v", statErr)
	}
}

func TestSetupWizard_NoDetectedProvidersFailsFast(t *testing.T) {
	// No provider executable on PATH: Q1/Q2's option set would be empty and
	// the questions would degrade to unvalidated free-form input — the wizard
	// must refuse up front instead, pointing at the read-only doctor.
	setupCheckFixture(t, "")
	forceTTY(t, true)

	err, _, _ := runSetupWizardCmd(t, "")
	if err == nil {
		t.Fatal("zero detected providers must fail fast")
	}
	if !strings.Contains(err.Error(), "no agent providers detected") || !strings.Contains(err.Error(), "fab setup check") {
		t.Errorf("error must state the cause and point at the doctor, got: %v", err)
	}
}

func TestSetupWizard_NoViableDispatchModeFailsFast(t *testing.T) {
	// Only an interactive-only custom provider is detected and the fixture
	// scrubs $TMUX: pane, native, and headless are all unviable, so Q3's
	// option set would be empty — the wizard must refuse up front.
	setupCheckFixture(t, "providers:\n  soloui:\n    interactive_command: soloui\n", "soloui")
	forceTTY(t, true)

	err, _, _ := runSetupWizardCmd(t, "")
	if err == nil {
		t.Fatal("zero viable dispatch modes must fail fast")
	}
	if !strings.Contains(err.Error(), "no viable dispatch mode") || !strings.Contains(err.Error(), "fab setup check") {
		t.Errorf("error must name the dispatch-mode gap and point at the doctor, got: %v", err)
	}
}

func TestSetupWizard_AdvancedOverriddenKeyIsAsked(t *testing.T) {
	// dispatch.column_width overridden at the project tier: its question
	// defaults to the override with its origin, and the other three keys are
	// asked too (at their built-in default / inherit indication).
	setupCheckFixture(t, "dispatch:\n  column_width: 42\n", "claude")
	forceTTY(t, true)

	err, out, _ := runSetupWizardCmd(t, "\n\n\ny\n\n\n\n\n")
	if err != nil {
		t.Fatalf("run error = %v; output:\n%s", err, out)
	}
	if !strings.Contains(out, "dispatch.column_width [42]:") {
		t.Errorf("the overridden key must be asked with its current value as default, got:\n%s", out)
	}
	for _, prompt := range []string{
		"agent.profiles.operator.provider [(inherit agent.session)]:",
		"agent.profiles.review.provider [(inherit agent.workers)]:",
		"dispatch.reap_done [true]:",
	} {
		if !strings.Contains(out, prompt) {
			t.Errorf("at-default keys must be asked alongside the overridden one, want %q, got:\n%s", prompt, out)
		}
	}
}

func TestSetupWizard_AdvancedFirstTimeProfileWriteLandsSystemTier(t *testing.T) {
	// A never-set profile key answered with a detected provider: the diff
	// summary renders the inherit indication as the old side, and the
	// confirmed write lands exactly that key in the system tier.
	setupCheckFixture(t, "", "claude", "codex")
	forceTTY(t, true)

	// Q1-Q3 Enter, Q4 y, operator=codex, review/width/reap Enter, confirm y.
	err, out, _ := runSetupWizardCmd(t, "\n\n\ny\ncodex\n\n\n\ny\n")
	if err != nil {
		t.Fatalf("run error = %v; output:\n%s", err, out)
	}
	if !strings.Contains(out, "agent.profiles.operator.provider: (inherit agent.session) → codex") {
		t.Errorf("diff summary must render the inherit indication as the old side, got:\n%s", out)
	}
	data, readErr := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".fab-kit", "config.yaml"))
	if readErr != nil {
		t.Fatalf("confirmed write must land in the system tier: %v\noutput:\n%s", readErr, out)
	}
	if !strings.Contains(string(data), "operator") || !strings.Contains(string(data), "provider: codex") {
		t.Errorf("system config must carry the operator profile write, got:\n%s", string(data))
	}
	live := strings.SplitN(string(data), "# >>> fab reference", 2)[0]
	if strings.Contains(live, "review") {
		t.Errorf("only the answered key may be written — review profile must be absent, got:\n%s", string(data))
	}
}

func TestSetupCmd_UnknownSubcommandIsUsageError(t *testing.T) {
	setupCheckFixture(t, "", "claude")

	code, _, _ := runFab("setup", "bogus")
	if code != 2 {
		t.Errorf("`fab setup bogus` exit = %d, want 2 (usage error via the run() seam)", code)
	}

	code, _, _ = runFab("setup", "check", "extra")
	if code != 2 {
		t.Errorf("`fab setup check extra` exit = %d, want 2 (arg-count violation)", code)
	}
}

func TestSetupCheck_WarningsOnlyExits0(t *testing.T) {
	// claude present (so the default knobs resolve runnable), gh/yq absent
	// (warnings), tmux absent (info). Warnings-only MUST exit 0.
	setupCheckFixture(t, "", "claude")

	code, out, _ := runFab("setup", "check")
	if code != 0 {
		t.Errorf("warnings-only report exit = %d, want 0; output:\n%s", code, out)
	}
	for _, want := range []string{"Providers:", "claude", "Checks:", "Summary:", "project pin"} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q, got:\n%s", want, out)
		}
	}
}

func TestSetupCheck_FailureFindingExits1(t *testing.T) {
	// agent.workers names agy, whose binary is absent — a real problem.
	setupCheckFixture(t, "agent:\n  workers: agy\n", "claude")

	code, out, errOut := runFab("setup", "check")
	if code != 1 {
		t.Errorf("failure-finding report exit = %d, want 1; output:\n%s", code, out)
	}
	if !strings.Contains(out, "agy") {
		t.Errorf("report should name the missing provider, got:\n%s", out)
	}
	if !strings.Contains(errOut, "ERROR:") {
		t.Errorf("exit-1 path should carry the run() ERROR line on stderr, got %q", errOut)
	}
}

func TestSetupCheck_IsReadOnly(t *testing.T) {
	repo := setupCheckFixture(t, "", "claude")

	snapshot := func() map[string]int64 {
		sizes := map[string]int64{}
		filepath.Walk(repo, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				rel, _ := filepath.Rel(repo, path)
				sizes[rel] = info.Size()
			}
			return nil
		})
		return sizes
	}

	before := snapshot()
	if code, _, _ := runFab("setup", "check"); code != 0 {
		t.Fatalf("setup check failed on the clean fixture (exit %d)", code)
	}
	after := snapshot()

	if len(before) != len(after) {
		t.Errorf("file count changed: before %d, after %d — the doctor must write nothing", len(before), len(after))
	}
	for path, size := range before {
		if after[path] != size {
			t.Errorf("file %s changed size (%d → %d) — the doctor must write nothing", path, size, after[path])
		}
	}
}

func TestSetupCheck_RunsOutsideRepo(t *testing.T) {
	// No fab/ anywhere up from cwd: the doctor degrades to system+env tiers
	// rather than erroring (a pre-init machine is exactly where it must run).
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", bin)
	t.Setenv("FAB_KIT_PATH", t.TempDir()) // no VERSION file → Info, not failure
	chdirTestEnv(t, t.TempDir(), map[string]string{"TMUX": ""})

	code, out, _ := runFab("setup", "check")
	if code != 0 {
		t.Errorf("outside-repo run exit = %d, want 0 (degraded, not failed); output:\n%s", code, out)
	}
	if !strings.Contains(out, "project pin (none)") {
		t.Errorf("outside-repo run should report no project pin, got:\n%s", out)
	}
}
