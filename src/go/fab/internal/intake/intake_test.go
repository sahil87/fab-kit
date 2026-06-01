package intake

import (
	"os"
	"path/filepath"
	"testing"
)

// writeIntake creates fab/changes/{folder}/intake.md with the given body and
// returns the fab root.
func writeIntake(t *testing.T, folder, body string) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	changeDir := filepath.Join(fabRoot, "changes", folder)
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		if err := os.WriteFile(filepath.Join(changeDir, "intake.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return fabRoot
}

func TestTitle(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "exact title",
			body: "# Intake: Make fab archive fully mechanical\n\n**Change**: x\n",
			want: "Make fab archive fully mechanical",
		},
		{
			name: "collapses internal whitespace",
			body: "#   Intake:    Fix   stale   status\n",
			want: "Fix stale status",
		},
		{
			name: "backticked title preserved verbatim",
			body: "# Intake: Fix stale `fab status` CLI\n",
			want: "Fix stale `fab status` CLI",
		},
		{
			name: "malformed heading returns empty",
			body: "# Not An Intake Heading\n\nsome text\n",
			want: "",
		},
		{
			name: "no heading at all returns empty",
			body: "just some prose with no heading\n",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "intake.md"), []byte(tt.body), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := Title(dir); got != tt.want {
				t.Errorf("Title() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTitle_MissingFile(t *testing.T) {
	dir := t.TempDir() // no intake.md written
	if got := Title(dir); got != "" {
		t.Errorf("Title() on missing file = %q, want \"\"", got)
	}
}

func TestDescriptionFor_TitlePresent(t *testing.T) {
	folder := "260601-zu41-mechanical-archive"
	fabRoot := writeIntake(t, folder, "# Intake: Make fab archive fully mechanical\n")

	got := DescriptionFor(fabRoot, folder)
	want := "Make fab archive fully mechanical"
	if got != want {
		t.Errorf("DescriptionFor() = %q, want %q", got, want)
	}
}

func TestDescriptionFor_SlugFallback(t *testing.T) {
	folder := "260601-abcd-fix-stale-status"
	// No intake.md → falls back to humanized slug.
	fabRoot := writeIntake(t, folder, "")

	got := DescriptionFor(fabRoot, folder)
	want := "fix stale status"
	if got != want {
		t.Errorf("DescriptionFor() = %q, want %q", got, want)
	}
}

func TestDescriptionFor_NoSlugSegment(t *testing.T) {
	// A two-segment name has an ID but no slug segment.
	folder := "260601-abcd"
	fabRoot := writeIntake(t, folder, "")

	got := DescriptionFor(fabRoot, folder)
	if got != "" {
		t.Errorf("DescriptionFor() = %q, want \"\"", got)
	}
}
