// Package intake derives mechanical, agent-free descriptions for a change
// from its intake.md, used to populate the archive index.
package intake

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

// titleRe matches the intake heading line "# Intake: {title}".
var titleRe = regexp.MustCompile(`^#\s+Intake:\s*(.+)$`)

// wsRe collapses runs of internal whitespace to a single space.
var wsRe = regexp.MustCompile(`\s+`)

// Title reads changeDir/intake.md and returns the de-prefixed title from the
// first "# Intake: {title}" heading, with internal whitespace collapsed.
// Returns "" on a missing/unreadable file or when no matching heading exists.
func Title(changeDir string) string {
	// internal/lines reuse (260612-tb6f, absorbing hv7t's deletion candidate).
	// CRLF parity: lines.Split trims one trailing "\r" (vs the old TrimRight
	// trimming runs of them) — equivalent for any realistic input, and the
	// wsRe/TrimSpace normalization below absorbs pathological extras.
	body, err := lines.ReadFileLines(filepath.Join(changeDir, "intake.md"))
	if err != nil {
		return ""
	}
	for _, line := range body {
		m := titleRe.FindStringSubmatch(line)
		if m != nil {
			return strings.TrimSpace(wsRe.ReplaceAllString(m[1], " "))
		}
	}
	return ""
}

// DescriptionFor returns a one-line archive description for a change folder.
// It prefers the intake title; when that is empty it falls back to a humanized
// slug (the folder-name segment after the "YYMMDD-XXXX-" prefix, with hyphens
// replaced by spaces).
func DescriptionFor(fabRoot, folder string) string {
	changeDir := filepath.Join(fabRoot, "changes", folder)
	if title := Title(changeDir); title != "" {
		return title
	}
	return humanizeSlug(folder)
}

// humanizeSlug strips the "YYMMDD-XXXX-" prefix from a change folder name and
// replaces hyphens with spaces. Returns "" when there is no slug segment.
func humanizeSlug(folder string) string {
	id := resolve.ExtractID(folder)
	if id == "" {
		return ""
	}
	// The slug is the third segment of "{date}-{id}-{slug}" — everything after
	// the second hyphen. SplitN bounds the split so slug hyphens stay intact.
	parts := strings.SplitN(folder, "-", 3)
	if len(parts) < 3 || parts[2] == "" {
		return ""
	}
	return strings.ReplaceAll(parts[2], "-", " ")
}
