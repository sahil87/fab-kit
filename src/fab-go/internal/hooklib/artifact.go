package hooklib

import (
	"bufio"
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

// postToolUsePayload represents the relevant fields of a Claude Code PostToolUse JSON payload.
type postToolUsePayload struct {
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

// ParsePayload reads a PostToolUse JSON payload from stdin and extracts the file_path.
func ParsePayload(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	var payload postToolUsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}

	return payload.ToolInput.FilePath, nil
}

// ArtifactMatch holds the result of matching a file path against fab artifact patterns.
type ArtifactMatch struct {
	ChangeFolder string
	Artifact     string
}

// MatchArtifactPath checks if a file path matches a fab artifact pattern.
// Returns the change folder and artifact name, or ok=false if no match.
// Matches patterns: fab/changes/*/artifact.md or */fab/changes/*/artifact.md
// The "fab/" must appear at start of path or after a "/" separator.
func MatchArtifactPath(filePath string) (ArtifactMatch, bool) {
	// Normalize path separators
	normalized := strings.ReplaceAll(filePath, "\\", "/")

	// Find "fab/changes/" in the path, ensuring it's preceded by "/" or at start
	const marker = "fab/changes/"
	idx := -1
	for i := len(normalized) - len(marker); i >= 0; i-- {
		if normalized[i:i+len(marker)] == marker {
			if i == 0 || normalized[i-1] == '/' {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return ArtifactMatch{}, false
	}

	// Extract everything after "fab/changes/"
	rest := normalized[idx+len(marker):]

	// Expect "folder/artifact.md"
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		return ArtifactMatch{}, false
	}

	folder := rest[:slashIdx]
	artifact := rest[slashIdx+1:]

	if folder == "" || artifact == "" {
		return ArtifactMatch{}, false
	}

	// Only match known artifact files
	switch artifact {
	case "intake.md", "spec.md", "tasks.md", "checklist.md":
		return ArtifactMatch{ChangeFolder: folder, Artifact: artifact}, true
	default:
		return ArtifactMatch{}, false
	}
}

// changeTypePatterns defines keyword patterns for inferring change type.
// Order matters: first match wins.
var changeTypePatterns = []struct {
	Type    string
	Pattern *regexp.Regexp
}{
	{"fix", regexp.MustCompile(`(?i)\b(fix|bug|broken|regression)\b`)},
	{"refactor", regexp.MustCompile(`(?i)\b(refactor|restructure|consolidate|split|rename|redesign)\b`)},
	{"docs", regexp.MustCompile(`(?i)\b(docs|document|readme|guide)\b`)},
	{"test", regexp.MustCompile(`(?i)\b(test|spec|coverage)\b`)},
	{"ci", regexp.MustCompile(`(?i)\b(ci|pipeline|deploy|build)\b`)},
	{"chore", regexp.MustCompile(`(?i)\b(chore|cleanup|maintenance|housekeeping)\b`)},
}

// InferChangeType determines the change type from intake content via keyword matching.
// Returns "feat" as the default if no keywords match.
func InferChangeType(content string) string {
	for _, p := range changeTypePatterns {
		if p.Pattern.MatchString(content) {
			return p.Type
		}
	}
	return "feat"
}

var uncheckedTaskRegex = regexp.MustCompile(`^- \[ \]`)
var checklistItemRegex = regexp.MustCompile(`^- \[(x| )\]`)

// CountUncheckedTasks counts lines matching "^- \[ \]" in content.
func CountUncheckedTasks(content string) int {
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if uncheckedTaskRegex.MatchString(scanner.Text()) {
			count++
		}
	}
	return count
}

// CountChecklistItems counts lines matching "^- \[(x| )\]" in content.
func CountChecklistItems(content string) int {
	count := 0
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if checklistItemRegex.MatchString(scanner.Text()) {
			count++
		}
	}
	return count
}
