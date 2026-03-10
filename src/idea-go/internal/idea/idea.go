package idea

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Idea represents a single backlog item.
type Idea struct {
	ID   string `json:"id"`
	Date string `json:"date"`
	Text string `json:"text"`
	Done bool   `json:"-"`
}

// Status returns "open" or "done".
func (i Idea) Status() string {
	if i.Done {
		return "done"
	}
	return "open"
}

// StatusCheck returns "x" for done, " " for open.
func (i Idea) StatusCheck() string {
	if i.Done {
		return "x"
	}
	return " "
}

// MarshalJSON customizes JSON output to include status field.
func (i Idea) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID     string `json:"id"`
		Date   string `json:"date"`
		Status string `json:"status"`
		Text   string `json:"text"`
	}{
		ID:     i.ID,
		Date:   i.Date,
		Status: i.Status(),
		Text:   i.Text,
	})
}

// lineRegex matches lines like: - [ ] [a7k2] 2025-06-15: Add dark mode
var lineRegex = regexp.MustCompile(`^- \[([ x])\] \[([a-z0-9]{4})\] (\d{4}-\d{2}-\d{2}): (.+)$`)

var idChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// idRegex validates that an ID is exactly 4 lowercase alphanumeric characters.
var idRegex = regexp.MustCompile(`^[a-z0-9]{4}$`)

// dateRegex validates the YYYY-MM-DD date format.
var dateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateID checks that id matches the expected 4-char lowercase alphanumeric format.
func ValidateID(id string) error {
	if !idRegex.MatchString(id) {
		return fmt.Errorf("invalid ID %q: must be exactly 4 lowercase alphanumeric characters", id)
	}
	return nil
}

// ValidateDate checks that date matches YYYY-MM-DD format.
func ValidateDate(date string) error {
	if !dateRegex.MatchString(date) {
		return fmt.Errorf("invalid date %q: must be in YYYY-MM-DD format", date)
	}
	return nil
}

// ParseLine parses a single backlog line into an Idea.
// Returns the parsed idea and true if the line is valid, or a zero Idea and false.
func ParseLine(line string) (Idea, bool) {
	m := lineRegex.FindStringSubmatch(line)
	if m == nil {
		return Idea{}, false
	}
	return Idea{
		ID:   m[2],
		Date: m[3],
		Text: m[4],
		Done: m[1] == "x",
	}, true
}

// FormatLine serializes an Idea back to the markdown line format.
func FormatLine(i Idea) string {
	return fmt.Sprintf("- [%s] [%s] %s: %s", i.StatusCheck(), i.ID, i.Date, i.Text)
}

// GitRepoRoot returns the git repository root directory.
// It runs: git rev-parse --path-format=absolute --git-common-dir
// and returns the parent of that directory.
func GitRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository")
	}
	gitDir := strings.TrimSpace(string(out))
	return filepath.Dir(gitDir), nil
}

// File represents a loaded backlog file, preserving non-idea lines.
type File struct {
	// lines stores every line in order. Non-idea lines are stored as-is.
	// Idea lines are stored as empty strings (their content comes from ideas).
	lines []string
	// ideaIndices maps from ideas slice index to lines slice index.
	ideaIndices []int
	// ideas holds the parsed ideas in file order.
	ideas []Idea
}

// LoadFile reads and parses a backlog file.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	f := &File{}
	content := string(data)
	if content == "" {
		return f, nil
	}

	// Trim trailing newline to avoid a spurious empty last line
	content = strings.TrimRight(content, "\n")
	rawLines := strings.Split(content, "\n")

	for i, line := range rawLines {
		if idea, ok := ParseLine(line); ok {
			f.lines = append(f.lines, "") // placeholder
			f.ideaIndices = append(f.ideaIndices, i)
			f.ideas = append(f.ideas, idea)
		} else {
			f.lines = append(f.lines, line)
		}
	}

	return f, nil
}

// SaveFile writes the backlog file, reconstructing from preserved lines and ideas.
func SaveFile(f *File, path string) error {
	// Rebuild lines
	result := make([]string, len(f.lines))
	copy(result, f.lines)

	for i, idx := range f.ideaIndices {
		result[idx] = FormatLine(f.ideas[i])
	}

	content := strings.Join(result, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}

// ResolveFilePath determines the backlog file path.
// Priority: flagValue > IDEAS_FILE env > default (fab/backlog.md).
// The result is relative to repoRoot.
func ResolveFilePath(repoRoot, flagValue string) string {
	if flagValue != "" {
		return filepath.Join(repoRoot, flagValue)
	}
	if env := os.Getenv("IDEAS_FILE"); env != "" {
		return filepath.Join(repoRoot, env)
	}
	return filepath.Join(repoRoot, "fab", "backlog.md")
}

// FilterKind specifies which ideas to include.
type FilterKind int

const (
	FilterOpen FilterKind = iota
	FilterDone
	FilterAll
)

// Match returns true if query is a case-insensitive substring of id or text.
func Match(query string, idea Idea) bool {
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(idea.ID), q) ||
		strings.Contains(strings.ToLower(idea.Text), q)
}

// FindAll returns all ideas matching the query and filter.
func FindAll(query string, ideas []Idea, filter FilterKind) []Idea {
	var result []Idea
	for _, idea := range ideas {
		if !matchesFilter(idea, filter) {
			continue
		}
		if query == "" || Match(query, idea) {
			result = append(result, idea)
		}
	}
	return result
}

// RequireSingle finds exactly one matching idea. Returns the idea and its
// index in the original ideas slice. Errors if 0 or >1 matches.
func RequireSingle(query string, ideas []Idea, filter FilterKind) (Idea, int, error) {
	var matches []Idea
	var indices []int
	for i, idea := range ideas {
		if !matchesFilter(idea, filter) {
			continue
		}
		if Match(query, idea) {
			matches = append(matches, idea)
			indices = append(indices, i)
		}
	}

	if len(matches) == 0 {
		return Idea{}, -1, fmt.Errorf("No idea matching '%s'", query)
	}
	if len(matches) > 1 {
		var lines []string
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  %s", FormatLine(m)))
		}
		return Idea{}, -1, fmt.Errorf("Multiple matches:\n%s\n\nBe more specific or use the exact ID.", strings.Join(lines, "\n"))
	}
	return matches[0], indices[0], nil
}

func matchesFilter(idea Idea, filter FilterKind) bool {
	switch filter {
	case FilterOpen:
		return !idea.Done
	case FilterDone:
		return idea.Done
	default:
		return true
	}
}

// generateRandomID generates a random 4-char alphanumeric ID.
func generateRandomID() (string, error) {
	b := make([]byte, 4)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(idChars))))
		if err != nil {
			return "", err
		}
		b[i] = idChars[n.Int64()]
	}
	return string(b), nil
}

// Add appends a new idea to the backlog file. Creates the file and parent
// directories if they don't exist.
func Add(path, text, customID, customDate string) (Idea, error) {
	if text == "" {
		return Idea{}, fmt.Errorf("text is required")
	}

	// Validate custom ID format if provided
	if customID != "" {
		if err := ValidateID(customID); err != nil {
			return Idea{}, err
		}
	}

	// Validate custom date format if provided
	if customDate != "" {
		if err := ValidateDate(customDate); err != nil {
			return Idea{}, err
		}
	}

	// Determine ID
	id := customID
	if id == "" {
		var err error
		id, err = generateUniqueID(path, 10)
		if err != nil {
			return Idea{}, err
		}
	} else {
		// Check for collision
		if err := checkIDCollision(path, id); err != nil {
			return Idea{}, err
		}
	}

	// Determine date
	date := customDate
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	idea := Idea{
		ID:   id,
		Date: date,
		Text: text,
		Done: false,
	}

	// Auto-create file and dirs if missing
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return Idea{}, fmt.Errorf("create directories: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return Idea{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, FormatLine(idea))
	if err != nil {
		return Idea{}, fmt.Errorf("write idea: %w", err)
	}

	return idea, nil
}

func generateUniqueID(path string, maxRetries int) (string, error) {
	for i := 0; i < maxRetries; i++ {
		id, err := generateRandomID()
		if err != nil {
			return "", err
		}
		err = checkIDCollision(path, id)
		if err == nil {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxRetries)
}

func checkIDCollision(path, id string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist yet, no collision
		return nil
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if idea, ok := ParseLine(line); ok {
			if idea.ID == id {
				return fmt.Errorf("ID '%s' already exists", id)
			}
		}
	}
	return nil
}

// List returns ideas filtered, sorted, and optionally formatted as JSON.
func List(path string, filter FilterKind, sortField string, reverse bool) ([]Idea, error) {
	f, err := LoadFile(path)
	if err != nil {
		return nil, err
	}

	var result []Idea
	for _, idea := range f.ideas {
		if matchesFilter(idea, filter) {
			result = append(result, idea)
		}
	}

	// Sort
	sort.SliceStable(result, func(i, j int) bool {
		switch sortField {
		case "id":
			return result[i].ID < result[j].ID
		default: // "date"
			return result[i].Date < result[j].Date
		}
	})

	if reverse {
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}
	}

	return result, nil
}

// Show finds a single idea matching the query.
func Show(path, query string) (Idea, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, err
	}

	idea, _, err := RequireSingle(query, f.ideas, FilterAll)
	if err != nil {
		return Idea{}, err
	}
	return idea, nil
}

// Done marks a single matching open idea as done.
func Done(path, query string) (Idea, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterOpen)
	if err != nil {
		return Idea{}, err
	}

	f.ideas[idx].Done = true
	if err := SaveFile(f, path); err != nil {
		return Idea{}, err
	}
	return f.ideas[idx], nil
}

// Reopen marks a single matching done idea as open.
func Reopen(path, query string) (Idea, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterDone)
	if err != nil {
		return Idea{}, err
	}

	f.ideas[idx].Done = false
	if err := SaveFile(f, path); err != nil {
		return Idea{}, err
	}
	return f.ideas[idx], nil
}

// Edit modifies a single matching idea's text, and optionally its ID and date.
func Edit(path, query, newText, newID, newDate string) (Idea, error) {
	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterAll)
	if err != nil {
		return Idea{}, err
	}

	// Validate new ID format if provided
	if newID != "" {
		if err := ValidateID(newID); err != nil {
			return Idea{}, err
		}
	}

	// Validate new date format if provided
	if newDate != "" {
		if err := ValidateDate(newDate); err != nil {
			return Idea{}, err
		}
	}

	// Check ID collision if changing
	if newID != "" && newID != f.ideas[idx].ID {
		for i, idea := range f.ideas {
			if i != idx && idea.ID == newID {
				return Idea{}, fmt.Errorf("ID '%s' already exists", newID)
			}
		}
		f.ideas[idx].ID = newID
	}

	if newDate != "" {
		f.ideas[idx].Date = newDate
	}

	f.ideas[idx].Text = newText

	if err := SaveFile(f, path); err != nil {
		return Idea{}, err
	}
	return f.ideas[idx], nil
}

// Rm removes a single matching idea from the file.
func Rm(path, query string, force bool) (Idea, error) {
	if !force {
		return Idea{}, fmt.Errorf("Use --force to confirm deletion")
	}

	f, err := LoadFile(path)
	if err != nil {
		return Idea{}, err
	}

	_, idx, err := RequireSingle(query, f.ideas, FilterAll)
	if err != nil {
		return Idea{}, err
	}

	removed := f.ideas[idx]

	// Remove the idea line from lines and update indices
	lineIdx := f.ideaIndices[idx]
	f.lines = append(f.lines[:lineIdx], f.lines[lineIdx+1:]...)

	// Remove from ideas and ideaIndices
	f.ideas = append(f.ideas[:idx], f.ideas[idx+1:]...)
	f.ideaIndices = append(f.ideaIndices[:idx], f.ideaIndices[idx+1:]...)

	// Adjust line indices for ideas that come after the removed line
	for i := idx; i < len(f.ideaIndices); i++ {
		f.ideaIndices[i]--
	}

	if err := SaveFile(f, path); err != nil {
		return Idea{}, err
	}
	return removed, nil
}
