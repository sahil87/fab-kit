// Package backlog parses and mutates the project backlog file (fab/backlog.md).
// It is shared by the batch-new flow (parsing pending items) and the archive
// flow (marking the originating item done).
package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
)

// Item holds a parsed pending backlog entry.
type Item struct {
	ID   string
	Desc string
}

// backlogItemRe matches a pending backlog line: - [ ] [xxxx] ...
var backlogItemRe = regexp.MustCompile(`^- \[ \] \[([a-z0-9]{4})\]`)

// backlogPrefixRe matches and strips the prefix to extract the description.
var backlogPrefixRe = regexp.MustCompile(`^- \[[x ]\] \[[a-z0-9]{4}\] (\[[A-Z]+-[0-9]+\] )?(\(BUG\) )?[0-9]{4}-[0-9]{2}-[0-9]{2}: `)

// Path returns the backlog.md path for a fab root.
func Path(fabRoot string) string {
	return filepath.Join(fabRoot, "backlog.md")
}

// ParsePending reads the backlog file and returns its pending items. Read
// failures (including a missing file) are returned as errors rather than an
// empty list, so callers can distinguish "no pending items" from "could not
// read the backlog".
func ParsePending(backlogPath string) ([]Item, error) {
	fileLines, err := lines.ReadFileLines(backlogPath)
	if err != nil {
		return nil, err
	}

	var items []Item
	for _, line := range fileLines {
		m := backlogItemRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		id := m[1]
		desc := backlogPrefixRe.ReplaceAllString(line, "")
		items = append(items, Item{ID: id, Desc: desc})
	}
	return items, nil
}

// ExtractContent extracts the full description for a backlog ID, including
// continuation lines. Read failures return the real error; a genuinely
// missing ID returns "not found in backlog".
func ExtractContent(backlogPath, id string) (string, error) {
	fileLines, err := lines.ReadFileLines(backlogPath)
	if err != nil {
		return "", err
	}

	// itemLineRe matches a line whose ID field is [<id>]
	itemLineRe := regexp.MustCompile(`^- \[[x ]\] \[` + regexp.QuoteMeta(id) + `\]`)
	// newItemRe matches a new list item (used to detect end of continuation)
	newItemRe := regexp.MustCompile(`^\s*- \[`)

	found := false
	var content string

	for _, line := range fileLines {
		if !found {
			if itemLineRe.MatchString(line) {
				content = backlogPrefixRe.ReplaceAllString(line, "")
				found = true
			}
			continue
		}

		// Continuation: starts with whitespace, not a new list item
		trimmed := strings.TrimSpace(line)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && !newItemRe.MatchString(line) && trimmed != "" {
			content += " " + trimmed
		} else {
			break
		}
	}

	if !found {
		return "", fmt.Errorf("not found in backlog")
	}
	return content, nil
}

// MarkDone flips the backlog line `- [ ] [<id>]` to `- [x] [<id>]` in place.
// It returns:
//   - "marked"    — found an unchecked matching item, flipped it, wrote the file
//   - "already"   — found a matching item already checked (no write)
//   - "not_found" — no matching ID, or backlog.md missing (nil error — silent no-op)
//
// The item is flipped where it sits; it is never moved to another section.
// MarkDone is idempotent.
func MarkDone(backlogPath, id string) (string, error) {
	data, err := os.ReadFile(backlogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "not_found", nil
		}
		return "not_found", err
	}

	checkedRe := regexp.MustCompile(`^- \[x\] \[` + regexp.QuoteMeta(id) + `\]`)
	uncheckedRe := regexp.MustCompile(`^- \[ \] \[` + regexp.QuoteMeta(id) + `\]`)

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if checkedRe.MatchString(line) {
			return "already", nil
		}
		if uncheckedRe.MatchString(line) {
			lines[i] = strings.Replace(line, "- [ ]", "- [x]", 1)
			if err := os.WriteFile(backlogPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return "not_found", err
			}
			return "marked", nil
		}
	}
	return "not_found", nil
}
