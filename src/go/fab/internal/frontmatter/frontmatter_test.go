package frontmatter

import (
	"os"
	"path/filepath"
	"strings"
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

func TestField_AfterOversizedLineFound(t *testing.T) {
	// The old scanner aborted on a >64KB frontmatter line, reporting every
	// later field as silently absent — dropping skills from fab help
	// listings and descriptions from fab memory-index.
	long := strings.Repeat("x", 70*1024)
	path := writeTestFile(t, "---\nname: fab-test\nnotes: \""+long+"\"\ndescription: \"Found me\"\n---\n# Content")

	if got := Field(path, "description"); got != "Found me" {
		t.Errorf("Field(description) = %q, want %q (field after oversized line)", got, "Found me")
	}
}

func TestField_EmptyFile(t *testing.T) {
	path := writeTestFile(t, "")

	if got := Field(path, "name"); got != "" {
		t.Errorf("Field(name) = %q, want empty for empty file", got)
	}
	if HasFrontmatter(path) {
		t.Error("HasFrontmatter() = true, want false for empty file")
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

// hasKind reports whether findings contain one with the given kind.
func hasKind(findings []Finding, kind string) bool {
	for _, f := range findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}

// TestValidate_GluedFence pins the loom corruption verbatim: the description
// line has the closing fence glued onto it (`description: "text"---`) with no
// standalone closing `---` and no trailing newline. Both signatures must fire:
// the fence is unclosed AND the description value fails quote-stripping. And
// Field must return exactly what it returns today (unchanged extraction).
func TestValidate_GluedFence(t *testing.T) {
	// No trailing newline — the loom file lost it when the fence was glued on.
	path := writeTestFile(t, "---\ndescription: \"a curated one-liner\"---")

	findings := Validate(path)
	if !hasKind(findings, KindUnclosedFence) {
		t.Errorf("glued-fence must report an unclosed fence, got %+v", findings)
	}
	if !hasKind(findings, KindQuoteStripFailure) {
		t.Errorf("glued-fence must report a quote-strip failure, got %+v", findings)
	}
	// Field's extraction is unchanged: the raw value (leading quote + trailing
	// `"---`) is returned verbatim exactly as it renders into the index today.
	if got := Field(path, "description"); got != "\"a curated one-liner\"---" {
		t.Errorf("Field extraction must be unchanged, got %q", got)
	}
}

// TestValidate_UnclosedFence: opens with `---` on line 1 but never closes.
func TestValidate_UnclosedFence(t *testing.T) {
	path := writeTestFile(t, "---\ndescription: \"clean value\"\nname: x\n# Body starts, no closing fence\n")

	findings := Validate(path)
	if !hasKind(findings, KindUnclosedFence) {
		t.Errorf("unclosed fence must be reported, got %+v", findings)
	}
	// The description here is well-formed (matching quotes), so no quote-strip
	// failure even though the fence is unclosed.
	if hasKind(findings, KindQuoteStripFailure) {
		t.Errorf("well-formed description must not report a quote-strip failure, got %+v", findings)
	}
}

// TestValidate_CleanValues: well-formed frontmatter (quoted, unquoted, single-
// quoted, empty) produces no findings — no false positives.
func TestValidate_CleanValues(t *testing.T) {
	cases := map[string]string{
		"double-quoted": "---\ndescription: \"Clean summary\"\n---\n# Body",
		"unquoted":      "---\ndescription: Clean summary\n---\n# Body",
		"single-quoted": "---\ndescription: 'Clean summary'\n---\n# Body",
		"empty":         "---\ndescription:\n---\n# Body",
		"missing":       "---\nname: x\n---\n# Body",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTestFile(t, content)
			if findings := Validate(path); len(findings) != 0 {
				t.Errorf("clean %s frontmatter must produce no findings, got %+v", name, findings)
			}
		})
	}
}

// TestValidate_NoFrontmatter: a file that does not open with `---` is not a
// malformed-frontmatter file — it is simply a non-frontmatter file (Field
// returns "" for it), so Validate reports nothing.
func TestValidate_NoFrontmatter(t *testing.T) {
	path := writeTestFile(t, "# Just a heading\nSome content\n")
	if findings := Validate(path); len(findings) != 0 {
		t.Errorf("a non-frontmatter file must produce no findings, got %+v", findings)
	}
}

// TestValidate_MissingFile: an unreadable/absent file produces no findings.
func TestValidate_MissingFile(t *testing.T) {
	if findings := Validate("/nonexistent/file.md"); len(findings) != 0 {
		t.Errorf("missing file must produce no findings, got %+v", findings)
	}
}
