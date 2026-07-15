package frontmatter

import (
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
)

// Field extracts a named field from YAML frontmatter (between --- markers)
// at the start of a file. It handles quoted and unquoted values and strips
// inline comments. Returns empty string if the field is not found, the file
// has no frontmatter, or the file cannot be read.
func Field(filePath, fieldName string) string {
	fileLines, err := lines.ReadFileLines(filePath)
	if err != nil {
		return ""
	}

	// First line must be "---"
	if len(fileLines) == 0 || strings.TrimSpace(fileLines[0]) != "---" {
		return ""
	}

	// Scan until closing "---"
	for _, line := range fileLines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}

		// Match "fieldName:" at start of line
		prefix := fieldName + ":"
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		value := strings.TrimSpace(line[len(prefix):])

		// Strip inline comments (not inside quotes)
		value = stripInlineComment(value)

		// Strip surrounding quotes
		value = stripQuotes(value)

		return value
	}

	return ""
}

// HasFrontmatter checks whether a file starts with a "---" frontmatter marker.
func HasFrontmatter(filePath string) bool {
	fileLines, err := lines.ReadFileLines(filePath)
	if err != nil || len(fileLines) == 0 {
		return false
	}
	return strings.TrimSpace(fileLines[0]) == "---"
}

// Finding kinds returned by Validate. They name the two malformed-frontmatter
// signatures the loom corruption exposed (see docs/specs/fkf.md and the change
// 260715-xu0k intake). Both are structural corruption of the frontmatter block
// itself — not a missing field (that degrades gracefully and is not a finding).
const (
	// KindUnclosedFence: the file opens with "---" (line 1) but has no
	// subsequent standalone "---" line — the block never closes. The loom
	// glued-fence corruption is ALSO an instance of this: gluing the fence onto
	// the description line removes the closing fence entirely.
	KindUnclosedFence = "unclosed-fence"
	// KindQuoteStripFailure: a `description:` value begins with a quote
	// character ('"' or '\'') but does not end with the matching quote, so
	// stripQuotes returns it verbatim (leading quote + trailing garbage). This
	// is the specific glued-fence diagnostic — e.g. a value ending in `"---`.
	KindQuoteStripFailure = "quote-strip-failure"
)

// Finding is one malformed-frontmatter diagnostic returned by Validate. Detail
// carries the offending value (for KindQuoteStripFailure) or is empty.
type Finding struct {
	// Kind is one of the Kind* constants above.
	Kind string
	// Detail is the offending frontmatter value (KindQuoteStripFailure), or "".
	Detail string
}

// Validate reads filePath's leading frontmatter with the SAME line-based grammar
// as Field/HasFrontmatter and returns structured findings for the two malformed
// signatures. It performs a read-only diagnostic pass — it never mutates the
// file and never changes what Field extracts (a malformed value keeps rendering
// exactly as it does now; validation is stderr/exit-code only). A file with no
// frontmatter at all (line 1 is not "---") is NOT a finding: it is simply a
// non-frontmatter file, and Field already returns "" for it. Findings are
// returned in a deterministic order (fence before description).
func Validate(filePath string) []Finding {
	fileLines, err := lines.ReadFileLines(filePath)
	if err != nil {
		return nil
	}
	// A file that does not open with "---" has no frontmatter block to be
	// malformed — this mirrors Field's own precondition. Return no findings.
	if len(fileLines) == 0 || strings.TrimSpace(fileLines[0]) != "---" {
		return nil
	}

	var findings []Finding

	// Scan for the closing "---" while inspecting `description:` values, so a
	// single pass mirrors Field's scan. closed records whether the block ends.
	closed := false
	descQuoteFailure := ""
	for _, line := range fileLines[1:] {
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		prefix := "description:"
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimSpace(line[len(prefix):])
		value = stripInlineComment(value)
		// A value that starts with a quote but is not stripped by stripQuotes
		// (no matching closing quote) is the glued-fence signature.
		if startsWithQuote(value) && stripQuotes(value) == value {
			descQuoteFailure = value
		}
	}

	// Emit the fence finding first (it subsumes the loom case), then the
	// description-specific diagnostic when present.
	if !closed {
		findings = append(findings, Finding{Kind: KindUnclosedFence})
	}
	if descQuoteFailure != "" {
		findings = append(findings, Finding{Kind: KindQuoteStripFailure, Detail: descQuoteFailure})
	}
	return findings
}

// startsWithQuote reports whether s begins with a single or double quote.
func startsWithQuote(s string) bool {
	return len(s) > 0 && (s[0] == '"' || s[0] == '\'')
}

// stripInlineComment removes a trailing # comment from a value string.
// Respects quoted strings: # inside quotes is not treated as a comment.
func stripInlineComment(s string) string {
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			}
			continue
		}
		if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
			continue
		}
		if c == '#' {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

// stripQuotes removes surrounding double or single quotes from a value.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
