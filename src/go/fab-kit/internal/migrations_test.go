package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOK   bool
		wantFrom string
		wantTo   string
	}{
		{"valid", "1.9.7-to-1.10.0.md", true, "1.9.7", "1.10.0"},
		{"valid wide range", "0.2.0-to-0.4.0.md", true, "0.2.0", "0.4.0"},
		{"gitkeep", ".gitkeep", false, "", ""},
		{"readme", "README.md", false, "", ""},
		{"missing to-sep", "1.0.0.md", false, "", ""},
		{"non-md", "1.0.0-to-2.0.0.txt", false, "", ""},
		{"non-semver from", "abc-to-1.0.0.md", false, "", ""},
		{"non-semver to", "1.0.0-to-xyz.md", false, "", ""},
		{"two-part version", "1.0-to-2.0.md", false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMigrationFilename(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("parseMigrationFilename(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.From != tt.wantFrom || got.To != tt.wantTo || got.File != tt.input {
				t.Errorf("parseMigrationFilename(%q) = %+v, want From=%s To=%s File=%s",
					tt.input, got, tt.wantFrom, tt.wantTo, tt.input)
			}
		})
	}
}

// writeMigrationFiles creates an empty migration file per name under a temp dir
// and returns the dir path.
func writeMigrationFiles(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("# migration\n"), 0644); err != nil {
			t.Fatalf("cannot write migration file %s: %v", n, err)
		}
	}
	return dir
}

func applicableFiles(result DiscoverResult) []string {
	var files []string
	for _, r := range result.Applicable {
		files = append(files, r.File)
	}
	return files
}

func TestDiscoverMigrations(t *testing.T) {
	tests := []struct {
		name           string
		files          []string
		local          string
		engine         string
		wantApplicable []string
		wantGapSkips   int
		wantOverlaps   int
	}{
		{
			name:           "applicable chain across multiple files",
			files:          []string{"0.2.0-to-0.3.0.md", "0.3.0-to-0.4.0.md"},
			local:          "0.2.0",
			engine:         "0.4.0",
			wantApplicable: []string{"0.2.0-to-0.3.0.md", "0.3.0-to-0.4.0.md"},
		},
		{
			name:           "gap skip then apply",
			files:          []string{"0.2.0-to-0.3.0.md", "0.5.0-to-0.6.0.md"},
			local:          "0.2.0",
			engine:         "0.6.0",
			wantApplicable: []string{"0.2.0-to-0.3.0.md", "0.5.0-to-0.6.0.md"},
			wantGapSkips:   1,
		},
		{
			name:           "leading gap skip",
			files:          []string{"0.5.0-to-0.6.0.md"},
			local:          "0.2.0",
			engine:         "0.6.0",
			wantApplicable: []string{"0.5.0-to-0.6.0.md"},
			wantGapSkips:   1,
		},
		{
			name:         "overlap detected",
			files:        []string{"1.0.0-to-1.2.0.md", "1.1.0-to-1.3.0.md"},
			local:        "1.0.0",
			engine:       "1.3.0",
			wantOverlaps: 1,
			// Discovery still walks the sorted ranges into a chain; the overlap is
			// surfaced separately via Overlaps for the caller (upgrade-repo / the
			// skill) to refuse on. With local 1.0.0 the walk applies 1.0.0-to-1.2.0
			// (current -> 1.2.0), then 1.1.0-to-1.3.0 matches (1.1.0 <= 1.2.0 < 1.3.0).
			wantApplicable: []string{"1.0.0-to-1.2.0.md", "1.1.0-to-1.3.0.md"},
		},
		{
			name:           "no-op when nothing applies (local ahead of newest)",
			files:          []string{"1.9.7-to-1.10.0.md"},
			local:          "2.1.0",
			engine:         "2.1.2",
			wantApplicable: nil,
		},
		{
			name:           "no-op when local equals engine",
			files:          []string{"0.2.0-to-0.3.0.md"},
			local:          "0.3.0",
			engine:         "0.3.0",
			wantApplicable: nil,
		},
		{
			name:           "ignores non-migration files",
			files:          []string{".gitkeep", "README.md", "0.2.0-to-0.3.0.md"},
			local:          "0.2.0",
			engine:         "0.3.0",
			wantApplicable: []string{"0.2.0-to-0.3.0.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeMigrationFiles(t, tt.files...)
			result, err := DiscoverMigrations(dir, tt.local, tt.engine)
			if err != nil {
				t.Fatalf("DiscoverMigrations error: %v", err)
			}
			if result.Local != tt.local || result.Engine != tt.engine {
				t.Errorf("Local/Engine = %s/%s, want %s/%s", result.Local, result.Engine, tt.local, tt.engine)
			}
			got := applicableFiles(result)
			if len(got) != len(tt.wantApplicable) {
				t.Fatalf("Applicable = %v, want %v", got, tt.wantApplicable)
			}
			for i := range got {
				if got[i] != tt.wantApplicable[i] {
					t.Errorf("Applicable[%d] = %s, want %s", i, got[i], tt.wantApplicable[i])
				}
			}
			if len(result.GapSkips) != tt.wantGapSkips {
				t.Errorf("GapSkips count = %d, want %d (%v)", len(result.GapSkips), tt.wantGapSkips, result.GapSkips)
			}
			if len(result.Overlaps) != tt.wantOverlaps {
				t.Errorf("Overlaps count = %d, want %d (%v)", len(result.Overlaps), tt.wantOverlaps, result.Overlaps)
			}
		})
	}
}

func TestDiscoverMigrations_UnreadableDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := DiscoverMigrations(missing, "1.0.0", "1.1.0"); err == nil {
		t.Error("expected error for unreadable migrations dir, got nil")
	}
}
