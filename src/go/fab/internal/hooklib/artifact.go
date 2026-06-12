package hooklib

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
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

	// Only match known artifact files. spec.md is retired (1.10.0): a leftover
	// spec.md on disk must NOT match, so editing it cannot fire score.Compute
	// and overwrite the authoritative intake confidence.
	switch artifact {
	case "intake.md", "plan.md":
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

var checklistItemRegex = regexp.MustCompile(`^- \[(x| )\]`)
var checkedItemRegex = regexp.MustCompile(`^- \[x\]`)
var headingRegex = regexp.MustCompile(`^##\s+`)

// PlanSection enumerates the heading-keyed sections recognized in plan.md.
type PlanSection string

const (
	SectionTasks      PlanSection = "Tasks"
	SectionAcceptance PlanSection = "Acceptance"
)

// HasSectionHeading reports whether the given content contains a top-level
// (`## `) heading that exactly matches the named plan section. Used to
// guard counter updates: when a section heading is missing on an
// in-progress write, the corresponding count fields SHOULD be left
// untouched.
func HasSectionHeading(content string, section PlanSection) bool {
	target := "## " + string(section)
	for _, line := range lines.Split(content) {
		// Match exactly "## Tasks" or "## Tasks ..." (allow trailing
		// whitespace) but not "## TasksAndOther".
		if line == target || strings.HasPrefix(line, target+" ") {
			return true
		}
	}
	return false
}

// CountSectionItemsBounded counts lines matching "^- \[(x| )\]" inside the
// named heading-keyed section of content. Section bounds are: from the
// first matching `## {section}` line (exclusive) to the next `## ` heading
// (exclusive) or EOF. Returns 0 when the section heading is absent —
// callers SHOULD use HasSectionHeading first to decide whether to apply
// the count, since "section missing" and "section empty" are
// indistinguishable from the count alone.
func CountSectionItemsBounded(content string, section PlanSection) int {
	return scanSectionItems(content, section, checklistItemRegex)
}

// CountCompletedSectionItemsBounded counts lines matching "^- \[x\]"
// inside the named heading-keyed section of content. Same bounding rules
// as CountSectionItemsBounded.
func CountCompletedSectionItemsBounded(content string, section PlanSection) int {
	return scanSectionItems(content, section, checkedItemRegex)
}

func scanSectionItems(content string, section PlanSection, itemRegex *regexp.Regexp) int {
	target := "## " + string(section)
	count := 0
	inSection := false
	for _, line := range lines.Split(content) {
		if !inSection {
			if line == target || strings.HasPrefix(line, target+" ") {
				inSection = true
			}
			continue
		}
		// Stop at the next top-level heading.
		if headingRegex.MatchString(line) {
			break
		}
		if itemRegex.MatchString(line) {
			count++
		}
	}
	return count
}
