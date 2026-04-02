package frontmatter

import (
	"bufio"
	"os"
	"strings"
)

// Field extracts a named field from YAML frontmatter (between --- markers)
// at the start of a file. It handles quoted and unquoted values and strips
// inline comments. Returns empty string if the field is not found or the
// file has no frontmatter.
func Field(filePath, fieldName string) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// First line must be "---"
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return ""
	}

	// Scan until closing "---"
	for scanner.Scan() {
		line := scanner.Text()
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
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return false
	}
	return strings.TrimSpace(scanner.Text()) == "---"
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
