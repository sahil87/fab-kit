package agentmodels

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCodexRecordedFixture(t *testing.T) {
	codexHome := t.TempDir()
	writeTestFile(t, filepath.Join(codexHome, "config.toml"), `model = "gpt-6.1-sol"`)
	t.Setenv("CODEX_HOME", codexHome)
	env := defaultEnvironment()
	env.run = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "codex" || !reflect.DeepEqual(args, []string{"debug", "models"}) {
			t.Fatalf("command = %s %v", name, args)
		}
		return os.ReadFile(filepath.Join("testdata", "codex-models.json"))
	}
	rows, warnings := discoverWith(nil, "codex", env)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if findRow(rows, "gpt-reserve") != nil {
		t.Fatal("hidden codex model was returned")
	}
	sol := findRow(rows, "gpt-6.1-sol")
	if sol == nil || sol.Label != "GPT-6.1 Sol" || !sol.Default || sol.Source != SourceTool {
		t.Fatalf("sol row = %+v", sol)
	}
	astra := findRow(rows, "gpt-6-astra")
	if astra == nil || astra.Efforts[len(astra.Efforts)-1] != "ultra" {
		t.Fatalf("astra row = %+v", astra)
	}
}

func TestCodexFailuresReturnFillsAndWarning(t *testing.T) {
	for _, test := range []struct {
		name   string
		stdout string
	}{
		{name: "malformed JSON", stdout: "not-json"},
		{name: "missing models", stdout: `{"other":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CODEX_HOME", t.TempDir())
			env := defaultEnvironment()
			env.run = func(context.Context, string, ...string) ([]byte, error) {
				return []byte(test.stdout), nil
			}
			rows, warnings := discoverWith(nil, "codex", env)
			assertFailureFills(t, rows, warnings)
		})
	}
}

func TestCodexProcessFailures(t *testing.T) {
	t.Run("non-zero exit", func(t *testing.T) {
		bin := t.TempDir()
		writeExecutable(t, filepath.Join(bin, "codex"), "#!/bin/sh\nexit 7\n")
		t.Setenv("PATH", bin)
		t.Setenv("CODEX_HOME", t.TempDir())
		rows, warnings := discoverWith(nil, "codex", defaultEnvironment())
		assertFailureFills(t, rows, warnings)
		if !strings.Contains(warnings[0], "exit status 7") {
			t.Errorf("warning = %q", warnings[0])
		}
	})
	t.Run("timeout", func(t *testing.T) {
		bin := t.TempDir()
		writeExecutable(t, filepath.Join(bin, "codex"), "#!/bin/sh\n/bin/sleep 1\n")
		t.Setenv("PATH", bin)
		t.Setenv("CODEX_HOME", t.TempDir())
		env := defaultEnvironment()
		env.timeout = 20 * time.Millisecond
		rows, warnings := discoverWith(nil, "codex", env)
		assertFailureFills(t, rows, warnings)
		if !strings.Contains(warnings[0], "timed out after 20ms") {
			t.Errorf("warning = %q", warnings[0])
		}
	})
	t.Run("binary absent", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())
		t.Setenv("CODEX_HOME", t.TempDir())
		rows, warnings := discoverWith(nil, "codex", defaultEnvironment())
		assertFailureFills(t, rows, warnings)
		if !strings.Contains(warnings[0], "executable file not found") {
			t.Errorf("warning = %q", warnings[0])
		}
	})
}

func TestCodexMalformedDefaultWarnsWithoutLosingModels(t *testing.T) {
	codexHome := t.TempDir()
	writeTestFile(t, filepath.Join(codexHome, "config.toml"), `model = "unterminated`)
	t.Setenv("CODEX_HOME", codexHome)
	env := defaultEnvironment()
	env.run = func(context.Context, string, ...string) ([]byte, error) {
		return os.ReadFile(filepath.Join("testdata", "codex-models.json"))
	}
	rows, warnings := discoverWith(nil, "codex", env)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "parse codex config") {
		t.Fatalf("warnings = %v", warnings)
	}
	if findRow(rows, "gpt-6-astra") == nil {
		t.Fatal("valid process rows were lost after a config error")
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatal(err)
	}
}

func assertFailureFills(t *testing.T, rows []Row, warnings []string) {
	t.Helper()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one", warnings)
	}
	if len(rows) == 0 {
		t.Fatal("kit fills disappeared")
	}
	for _, row := range rows {
		if row.Source != SourceFill {
			t.Errorf("row = %+v, want fill", row)
		}
	}
}
