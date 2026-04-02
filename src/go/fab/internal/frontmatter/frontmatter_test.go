package frontmatter

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestField_QuotedValue(t *testing.T) {
	path := writeTestFile(t, `---
name: fab-new
description: "Start a new change"
---
# Content`)

	if got := Field(path, "name"); got != "fab-new" {
		t.Errorf("Field(name) = %q, want %q", got, "fab-new")
	}
	if got := Field(path, "description"); got != "Start a new change" {
		t.Errorf("Field(description) = %q, want %q", got, "Start a new change")
	}
}

func TestField_UnquotedValue(t *testing.T) {
	path := writeTestFile(t, `---
name: fab-continue
description: Advance the pipeline
---`)

	if got := Field(path, "name"); got != "fab-continue" {
		t.Errorf("Field(name) = %q, want %q", got, "fab-continue")
	}
	if got := Field(path, "description"); got != "Advance the pipeline" {
		t.Errorf("Field(description) = %q, want %q", got, "Advance the pipeline")
	}
}

func TestField_InlineComment(t *testing.T) {
	path := writeTestFile(t, `---
name: fab-test # this is a comment
---`)

	if got := Field(path, "name"); got != "fab-test" {
		t.Errorf("Field(name) = %q, want %q", got, "fab-test")
	}
}

func TestField_QuotedHashNotComment(t *testing.T) {
	path := writeTestFile(t, `---
description: "Contains # hash inside"
---`)

	if got := Field(path, "description"); got != "Contains # hash inside" {
		t.Errorf("Field(description) = %q, want %q", got, "Contains # hash inside")
	}
}

func TestField_MissingField(t *testing.T) {
	path := writeTestFile(t, `---
name: fab-test
---`)

	if got := Field(path, "description"); got != "" {
		t.Errorf("Field(description) = %q, want empty", got)
	}
}

func TestField_NoFrontmatter(t *testing.T) {
	path := writeTestFile(t, `# Just a heading
Some content`)

	if got := Field(path, "name"); got != "" {
		t.Errorf("Field(name) = %q, want empty", got)
	}
}

func TestField_MissingFile(t *testing.T) {
	if got := Field("/nonexistent/file.md", "name"); got != "" {
		t.Errorf("Field(name) = %q, want empty", got)
	}
}

func TestField_EmptyValue(t *testing.T) {
	path := writeTestFile(t, `---
name:
description: "has value"
---`)

	if got := Field(path, "name"); got != "" {
		t.Errorf("Field(name) = %q, want empty", got)
	}
}

func TestField_SingleQuotedValue(t *testing.T) {
	path := writeTestFile(t, `---
description: 'Single quoted value'
---`)

	if got := Field(path, "description"); got != "Single quoted value" {
		t.Errorf("Field(description) = %q, want %q", got, "Single quoted value")
	}
}

func TestHasFrontmatter_True(t *testing.T) {
	path := writeTestFile(t, `---
name: test
---`)

	if !HasFrontmatter(path) {
		t.Error("HasFrontmatter() = false, want true")
	}
}

func TestHasFrontmatter_False(t *testing.T) {
	path := writeTestFile(t, `# No frontmatter`)

	if HasFrontmatter(path) {
		t.Error("HasFrontmatter() = true, want false")
	}
}

func TestHasFrontmatter_MissingFile(t *testing.T) {
	if HasFrontmatter("/nonexistent/file.md") {
		t.Error("HasFrontmatter() = true, want false")
	}
}
