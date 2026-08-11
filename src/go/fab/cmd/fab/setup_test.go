package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestSetupCmd_BarePrintsPlaceholder(t *testing.T) {
	setupCheckFixture(t, "", "claude")

	code, out, _ := runFab("setup")
	if code != 0 {
		t.Errorf("bare `fab setup` exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Yet to be implemented") {
		t.Errorf("bare `fab setup` must print the wizard placeholder, got:\n%s", out)
	}
	if strings.Contains(out, "Checks:") {
		t.Errorf("bare `fab setup` must NOT run the doctor, got:\n%s", out)
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
