package configupgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 261120-f2ly: the providers → clis rename. A live top-level `providers:` key is
// carried to `clis:` by the registry's renamed_from, and a fence rendered by 2.30.1
// (the last release spelled "provider") re-renders with the new wording.

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpgrade_CarriesProvidersTableToCLIs(t *testing.T) {
	path := writeFixture(t, "providers:\n  mytool:\n    headless_command: mytool run\n")
	result, err := Upgrade(ProjectTarget(path), "2.31.0")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	live := string(got)
	if i := strings.Index(live, "# >>> fab reference"); i >= 0 {
		live = live[:i]
	}
	if !strings.HasPrefix(live, "clis:\n  mytool:\n    headless_command: mytool run\n") || strings.Contains(live, "providers") {
		t.Errorf("live block after upgrade:\n%s", live)
	}
	if !strings.Contains(strings.Join(result.Report, "\n"), `carried rename "providers" → "clis"`) {
		t.Errorf("report = %v, want the carried-rename line", result.Report)
	}
}

func TestUpgrade_SkipsProvidersCarryWhenCLIsAlreadyLive(t *testing.T) {
	// A carry would emit a duplicate top-level key; the user's explicit clis: wins
	// and the old block is left for the unknown-key parker.
	path := writeFixture(t, "clis:\n  a: {headless_command: x}\nproviders:\n  b: {headless_command: y}\n")
	result, err := Upgrade(ProjectTarget(path), "2.31.0")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if n := strings.Count(string(got), "\nclis:"); n+boolToInt(strings.HasPrefix(string(got), "clis:")) != 1 {
		t.Errorf("exactly one live clis: key expected, file:\n%s", got)
	}
	if !strings.Contains(strings.Join(result.Report, "\n"), "skipped rename") {
		t.Errorf("report = %v, want the skipped-rename line", result.Report)
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestUpgrade_Rerenders2301FenceWithoutUserEditWarning(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "config-2.30.1-providers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "agent.session / agent.workers — provider") {
		t.Fatal("fixture must be a fence rendered with the pre-rename wording")
	}
	path := writeFixture(t, string(data))

	result, err := Upgrade(ProjectTarget(path), "2.31.0")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed {
		t.Error("the 2.30.1 fence must be rewritten")
	}
	for _, line := range result.Report {
		if strings.Contains(line, "edit") || strings.Contains(line, "parked") {
			t.Errorf("unexpected user-edit/parking notice for an untouched fence: %q", line)
		}
	}
	got, _ := os.ReadFile(path)
	fence := string(got)[strings.Index(string(got), "# >>> fab reference"):]
	if strings.Contains(fence, "agent.session / agent.workers — provider") || strings.Contains(fence, "{ provider: codex }") {
		t.Errorf("the fence must no longer spell provider:\n%s", fence)
	}
	if !strings.Contains(fence, "agent.session / agent.workers — CLIs") || !strings.Contains(fence, "{ cli: codex }") {
		t.Errorf("the fence must carry the new wording:\n%s", fence)
	}

	// Idempotent: a second run is a no-op.
	again, err := Check(ProjectTarget(path), "2.31.0")
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed {
		t.Errorf("a second upgrade must be a no-op, report = %v", again.Report)
	}
}

func TestSetUnset_DeprecatedKeysMapAndTwinHandling(t *testing.T) {
	t.Run("Set maps a providers key and drops the twin", func(t *testing.T) {
		path := writeFixture(t, "providers:\n  codex:\n    headless_command: old\n")
		if _, err := Set(path, "providers.codex.headless_command", "new", "2.31.0"); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		live := string(got)[:strings.Index(string(got), "# >>> fab reference")]
		if strings.Contains(live, "providers") || !strings.Contains(live, "clis:") || !strings.Contains(live, "headless_command: new") {
			t.Errorf("live block:\n%s", live)
		}
	})

	t.Run("Unset removes both spellings and prunes the emptied parent", func(t *testing.T) {
		path := writeFixture(t, "agent:\n  profiles:\n    review:\n      cli: claude\n      provider: codex\n")
		result, err := Unset(path, "agent.profiles.review.provider", "2.31.0")
		if err != nil {
			t.Fatal(err)
		}
		if !result.Changed {
			t.Error("unset must report a change")
		}
		got, _ := os.ReadFile(path)
		live := string(got)[:strings.Index(string(got), "# >>> fab reference")]
		if strings.Contains(live, "claude") || strings.Contains(live, "codex") || strings.Contains(live, "profiles") {
			t.Errorf("both spellings and the empty parents must go:\n%s", live)
		}
	})

	t.Run("an unset with neither live reports not set", func(t *testing.T) {
		path := writeFixture(t, "project:\n  name: x\n")
		result, err := Unset(path, "agent.profiles.review.cli", "2.31.0")
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed || !strings.Contains(strings.Join(result.Report, "\n"), "is not set") {
			t.Errorf("result = %+v", result)
		}
	})
}
