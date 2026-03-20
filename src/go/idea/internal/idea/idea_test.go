package idea

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Parsing Tests ---

func TestParseLine_ValidOpen(t *testing.T) {
	line := "- [ ] [a7k2] 2025-06-15: Add dark mode to settings page"
	idea, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected valid parse")
	}
	if idea.ID != "a7k2" {
		t.Errorf("ID = %q, want a7k2", idea.ID)
	}
	if idea.Date != "2025-06-15" {
		t.Errorf("Date = %q, want 2025-06-15", idea.Date)
	}
	if idea.Text != "Add dark mode to settings page" {
		t.Errorf("Text = %q, want 'Add dark mode to settings page'", idea.Text)
	}
	if idea.Done {
		t.Error("Done = true, want false")
	}
}

func TestParseLine_ValidDone(t *testing.T) {
	line := "- [x] [e5f6] 2025-06-08: Fix login redirect bug"
	idea, ok := ParseLine(line)
	if !ok {
		t.Fatal("expected valid parse")
	}
	if idea.ID != "e5f6" {
		t.Errorf("ID = %q, want e5f6", idea.ID)
	}
	if idea.Date != "2025-06-08" {
		t.Errorf("Date = %q, want 2025-06-08", idea.Date)
	}
	if idea.Text != "Fix login redirect bug" {
		t.Errorf("Text = %q, want 'Fix login redirect bug'", idea.Text)
	}
	if !idea.Done {
		t.Error("Done = false, want true")
	}
}

func TestParseLine_Invalid(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"empty", ""},
		{"random text", "some random text"},
		{"header", "# Backlog"},
		{"blank", "   "},
		{"bad checkbox", "- [y] [a7k2] 2025-06-15: Text"},
		{"missing id brackets", "- [ ] a7k2 2025-06-15: Text"},
		{"short id", "- [ ] [a7k] 2025-06-15: Text"},
		{"long id", "- [ ] [a7k2x] 2025-06-15: Text"},
		{"uppercase id", "- [ ] [A7K2] 2025-06-15: Text"},
		{"bad date format", "- [ ] [a7k2] 2025-6-15: Text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := ParseLine(tt.line)
			if ok {
				t.Errorf("ParseLine(%q) should return false", tt.line)
			}
		})
	}
}

func TestFormatLine_Open(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2025-06-15", Text: "Add dark mode", Done: false}
	got := FormatLine(i)
	want := "- [ ] [a7k2] 2025-06-15: Add dark mode"
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestFormatLine_Done(t *testing.T) {
	i := Idea{ID: "e5f6", Date: "2025-06-08", Text: "Fix bug", Done: true}
	got := FormatLine(i)
	want := "- [x] [e5f6] 2025-06-08: Fix bug"
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	lines := []string{
		"- [ ] [a7k2] 2025-06-15: Add dark mode to settings page",
		"- [x] [e5f6] 2025-06-08: Fix login redirect bug",
	}
	for _, line := range lines {
		idea, ok := ParseLine(line)
		if !ok {
			t.Fatalf("failed to parse %q", line)
		}
		got := FormatLine(idea)
		if got != line {
			t.Errorf("round-trip failed: got %q, want %q", got, line)
		}
	}
}

// --- Query Matching Tests ---

func TestMatch_ByID(t *testing.T) {
	i := Idea{ID: "a7k2", Text: "Add dark mode"}
	if !Match("a7k2", i) {
		t.Error("expected match by ID")
	}
	if Match("c3d4", i) {
		t.Error("expected no match for wrong ID")
	}
}

func TestMatch_ByText(t *testing.T) {
	i := Idea{ID: "a7k2", Text: "Add dark mode"}
	if !Match("dark", i) {
		t.Error("expected match by text substring")
	}
}

func TestMatch_CaseInsensitive(t *testing.T) {
	i := Idea{ID: "a7k2", Text: "Add dark mode"}
	if !Match("DARK", i) {
		t.Error("expected case-insensitive match on text")
	}
	if !Match("A7K2", i) {
		t.Error("expected case-insensitive match on ID")
	}
}

func TestFindAll(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode", Done: false},
		{ID: "c3d4", Text: "Add light mode", Done: false},
		{ID: "e5f6", Text: "Fix redirect", Done: true},
	}

	tests := []struct {
		name   string
		query  string
		filter FilterKind
		want   int
	}{
		{"match both by mode", "mode", FilterAll, 2},
		{"match one by dark", "dark", FilterAll, 1},
		{"no match", "nonexistent", FilterAll, 0},
		{"filter open", "dark", FilterOpen, 1},
		{"filter done", "", FilterDone, 1},
		{"all with empty query", "", FilterAll, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindAll(tt.query, ideas, tt.filter)
			if len(result) != tt.want {
				t.Errorf("FindAll(%q, filter=%d) = %d results, want %d", tt.query, tt.filter, len(result), tt.want)
			}
		})
	}
}

func TestRequireSingle_OneMatch(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode"},
		{ID: "c3d4", Text: "Fix redirect"},
	}
	i, idx, err := RequireSingle("a7k2", ideas, FilterAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.ID != "a7k2" {
		t.Errorf("ID = %q, want a7k2", i.ID)
	}
	if idx != 0 {
		t.Errorf("idx = %d, want 0", idx)
	}
}

func TestRequireSingle_NoMatch(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode"},
	}
	_, _, err := RequireSingle("nonexistent", ideas, FilterAll)
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "No idea matching") {
		t.Errorf("error = %q, want 'No idea matching'", err.Error())
	}
}

func TestRequireSingle_MultipleMatches(t *testing.T) {
	ideas := []Idea{
		{ID: "a7k2", Text: "Add dark mode"},
		{ID: "c3d4", Text: "Add light mode"},
	}
	_, _, err := RequireSingle("mode", ideas, FilterAll)
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "Multiple matches") {
		t.Errorf("error = %q, want 'Multiple matches'", err.Error())
	}
	if !strings.Contains(err.Error(), "Be more specific") {
		t.Errorf("error should contain disambiguation guidance")
	}
}

// --- File Operations Tests ---

func writeBacklog(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "backlog.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFile_MixedContent(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Add dark mode
- [x] [e5f6] 2025-06-08: Fix bug

Some footer text
`
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 2 {
		t.Fatalf("ideas count = %d, want 2", len(f.ideas))
	}
	if f.ideas[0].ID != "a7k2" {
		t.Errorf("first idea ID = %q, want a7k2", f.ideas[0].ID)
	}
	if f.ideas[1].ID != "e5f6" {
		t.Errorf("second idea ID = %q, want e5f6", f.ideas[1].ID)
	}
}

func TestLoadFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeBacklog(t, dir, "")

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if len(f.ideas) != 0 {
		t.Errorf("ideas count = %d, want 0", len(f.ideas))
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.md")
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSaveFile_PreservesNonIdeaLines(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Add dark mode

Some footer text
`
	path := writeBacklog(t, dir, content)

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	// Modify the idea
	f.ideas[0].Text = "Add dark mode with toggle"
	if err := SaveFile(f, path); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "# Backlog") {
		t.Error("header line missing after save")
	}
	if !strings.Contains(result, "Some footer text") {
		t.Error("footer line missing after save")
	}
	if !strings.Contains(result, "Add dark mode with toggle") {
		t.Error("modified idea text missing after save")
	}
}

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		env      string
		wantSfx  string
	}{
		{"default", "", "", "fab/backlog.md"},
		{"flag override", "custom/ideas.md", "", "custom/ideas.md"},
		{"env override", "", "env/ideas.md", "env/ideas.md"},
		{"flag beats env", "flag/ideas.md", "env/ideas.md", "flag/ideas.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				t.Setenv("IDEAS_FILE", tt.env)
			} else {
				t.Setenv("IDEAS_FILE", "")
			}
			got := ResolveFilePath("/repo", tt.flag)
			want := filepath.Join("/repo", tt.wantSfx)
			if got != want {
				t.Errorf("ResolveFilePath = %q, want %q", got, want)
			}
		})
	}
}

// --- CRUD Operations Tests ---

func TestAdd_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	i, err := Add(path, "Build search feature", "", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(i.ID) != 4 {
		t.Errorf("ID length = %d, want 4", len(i.ID))
	}
	if i.Date == "" {
		t.Error("Date should not be empty")
	}
	if i.Text != "Build search feature" {
		t.Errorf("Text = %q, want 'Build search feature'", i.Text)
	}
	if i.Done {
		t.Error("new idea should not be done")
	}

	// Verify file content
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Build search feature") {
		t.Error("file should contain the idea text")
	}
}

func TestAdd_CustomIDAndDate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	i, err := Add(path, "My idea", "ab12", "2025-01-01")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if i.ID != "ab12" {
		t.Errorf("ID = %q, want ab12", i.ID)
	}
	if i.Date != "2025-01-01" {
		t.Errorf("Date = %q, want 2025-01-01", i.Date)
	}
}

func TestAdd_IDCollision(t *testing.T) {
	dir := t.TempDir()
	path := writeBacklog(t, dir, "- [ ] [ab12] 2025-06-15: Existing idea\n")

	_, err := Add(path, "New idea", "ab12", "")
	if err == nil {
		t.Fatal("expected error for ID collision")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, want 'already exists'", err.Error())
	}
}

func TestAdd_AutoCreateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "backlog.md")

	_, err := Add(path, "New idea in new file", "", "2025-06-15")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should have been auto-created")
	}
}

func TestAdd_EmptyText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backlog.md")

	_, err := Add(path, "", "", "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestList_OpenOnly(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
- [ ] [c3d4] 2025-06-10: Open two
- [x] [e5f6] 2025-06-08: Done one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterOpen, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 2 {
		t.Fatalf("count = %d, want 2", len(ideas))
	}
	for _, i := range ideas {
		if i.Done {
			t.Error("open filter returned a done idea")
		}
	}
}

func TestList_DoneOnly(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
- [x] [e5f6] 2025-06-08: Done one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterDone, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 1 {
		t.Fatalf("count = %d, want 1", len(ideas))
	}
	if !ideas[0].Done {
		t.Error("done filter returned an open idea")
	}
}

func TestList_All(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
- [x] [e5f6] 2025-06-08: Done one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 2 {
		t.Fatalf("count = %d, want 2", len(ideas))
	}
}

func TestList_SortByID(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [c3d4] 2025-06-10: Second
- [ ] [a1b2] 2025-06-15: First
- [ ] [e5f6] 2025-06-08: Third
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "id", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ideas[0].ID != "a1b2" || ideas[1].ID != "c3d4" || ideas[2].ID != "e5f6" {
		t.Errorf("sort by id: got %s, %s, %s", ideas[0].ID, ideas[1].ID, ideas[2].ID)
	}
}

func TestList_SortByDate(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [c3d4] 2025-06-15: Third date
- [ ] [a1b2] 2025-06-08: First date
- [ ] [e5f6] 2025-06-10: Second date
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ideas[0].Date != "2025-06-08" || ideas[1].Date != "2025-06-10" || ideas[2].Date != "2025-06-15" {
		t.Errorf("sort by date: got %s, %s, %s", ideas[0].Date, ideas[1].Date, ideas[2].Date)
	}
}

func TestList_Reverse(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a1b2] 2025-06-08: First
- [ ] [c3d4] 2025-06-10: Second
- [ ] [e5f6] 2025-06-15: Third
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "id", true)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if ideas[0].ID != "e5f6" || ideas[1].ID != "c3d4" || ideas[2].ID != "a1b2" {
		t.Errorf("reverse sort by id: got %s, %s, %s", ideas[0].ID, ideas[1].ID, ideas[2].ID)
	}
}

func TestList_JSON(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterAll, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	data, err := json.Marshal(ideas)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var parsed []map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("JSON array len = %d, want 1", len(parsed))
	}
	obj := parsed[0]
	if obj["id"] != "a7k2" {
		t.Errorf("JSON id = %v, want a7k2", obj["id"])
	}
	if obj["status"] != "open" {
		t.Errorf("JSON status = %v, want open", obj["status"])
	}
	if obj["date"] != "2025-06-15" {
		t.Errorf("JSON date = %v, want 2025-06-15", obj["date"])
	}
	if obj["text"] != "Add dark mode" {
		t.Errorf("JSON text = %v, want 'Add dark mode'", obj["text"])
	}
}

func TestShow_SingleMatch(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Fix redirect
`
	path := writeBacklog(t, dir, content)

	i, err := Show(path, "a7k2")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if i.ID != "a7k2" {
		t.Errorf("ID = %q, want a7k2", i.ID)
	}
}

func TestShow_NoMatch(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, err := Show(path, "nonexistent")
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "No idea matching 'nonexistent'") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestShow_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Add light mode
`
	path := writeBacklog(t, dir, content)

	_, err := Show(path, "mode")
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "Multiple matches") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestDone_MarkOpen(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, err := Done(path, "a7k2")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}
	if !i.Done {
		t.Error("idea should be done after Done()")
	}

	// Verify file was updated
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] [a7k2]") {
		t.Error("file should contain done marker")
	}
}

func TestDone_AlreadyDone(t *testing.T) {
	dir := t.TempDir()
	content := `- [x] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, err := Done(path, "a7k2")
	if err == nil {
		t.Fatal("expected error when marking already-done idea as done")
	}
	if !strings.Contains(err.Error(), "No idea matching") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestReopen_MarkDone(t *testing.T) {
	dir := t.TempDir()
	content := `- [x] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, err := Reopen(path, "a7k2")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if i.Done {
		t.Error("idea should be open after Reopen()")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [ ] [a7k2]") {
		t.Error("file should contain open marker")
	}
}

func TestReopen_AlreadyOpen(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, err := Reopen(path, "a7k2")
	if err == nil {
		t.Fatal("expected error when reopening already-open idea")
	}
	if !strings.Contains(err.Error(), "No idea matching") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestEdit_TextOnly(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, err := Edit(path, "a7k2", "Add dark mode with toggle", "", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.Text != "Add dark mode with toggle" {
		t.Errorf("Text = %q", i.Text)
	}
	if i.ID != "a7k2" {
		t.Error("ID should be preserved")
	}
	if i.Date != "2025-06-15" {
		t.Error("Date should be preserved")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "Add dark mode with toggle") {
		t.Error("file should contain updated text")
	}
}

func TestEdit_WithID_NoCollision(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, err := Edit(path, "a7k2", "Same text", "z9y8", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.ID != "z9y8" {
		t.Errorf("ID = %q, want z9y8", i.ID)
	}
}

func TestEdit_WithID_Collision(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [z9y8] 2025-06-10: Other idea
`
	path := writeBacklog(t, dir, content)

	_, err := Edit(path, "a7k2", "Text", "z9y8", "")
	if err == nil {
		t.Fatal("expected error for ID collision")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestEdit_WithDate(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	i, err := Edit(path, "a7k2", "Same text", "", "2025-12-01")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if i.Date != "2025-12-01" {
		t.Errorf("Date = %q, want 2025-12-01", i.Date)
	}
}

func TestRm_WithForce(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Fix redirect
`
	path := writeBacklog(t, dir, content)

	removed, err := Rm(path, "a7k2", true)
	if err != nil {
		t.Fatalf("Rm: %v", err)
	}
	if removed.ID != "a7k2" {
		t.Errorf("removed ID = %q, want a7k2", removed.ID)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "a7k2") {
		t.Error("file should not contain removed idea")
	}
	if !strings.Contains(string(data), "c3d4") {
		t.Error("file should still contain other idea")
	}
}

func TestRm_WithoutForce(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Add dark mode
`
	path := writeBacklog(t, dir, content)

	_, err := Rm(path, "a7k2", false)
	if err == nil {
		t.Fatal("expected error without --force")
	}
	if !strings.Contains(err.Error(), "Use --force") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestRm_PreservesNonIdeaLines(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Add dark mode
- [ ] [c3d4] 2025-06-10: Fix redirect

Footer
`
	path := writeBacklog(t, dir, content)

	_, err := Rm(path, "a7k2", true)
	if err != nil {
		t.Fatalf("Rm: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "# Backlog") {
		t.Error("header should be preserved")
	}
	if !strings.Contains(result, "Footer") {
		t.Error("footer should be preserved")
	}
	if !strings.Contains(result, "c3d4") {
		t.Error("other idea should be preserved")
	}
}

// --- JSON Output Tests ---

func TestIdeaJSON(t *testing.T) {
	i := Idea{ID: "a7k2", Date: "2025-06-15", Text: "Add dark mode", Done: false}
	data, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if obj["id"] != "a7k2" {
		t.Errorf("id = %v", obj["id"])
	}
	if obj["date"] != "2025-06-15" {
		t.Errorf("date = %v", obj["date"])
	}
	if obj["status"] != "open" {
		t.Errorf("status = %v", obj["status"])
	}
	if obj["text"] != "Add dark mode" {
		t.Errorf("text = %v", obj["text"])
	}
}

func TestIdeaJSON_Done(t *testing.T) {
	i := Idea{ID: "e5f6", Date: "2025-06-08", Text: "Fix bug", Done: true}
	data, err := json.Marshal(i)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var obj map[string]interface{}
	json.Unmarshal(data, &obj)
	if obj["status"] != "done" {
		t.Errorf("status = %v, want done", obj["status"])
	}
}

// --- Repo Root Tests ---

func TestMainRepoRoot_ReturnsPath(t *testing.T) {
	root, err := MainRepoRoot()
	if err != nil {
		t.Fatalf("MainRepoRoot: %v", err)
	}
	if root == "" {
		t.Error("MainRepoRoot returned empty string")
	}
}

func TestWorktreeRoot_ReturnsPath(t *testing.T) {
	root, err := WorktreeRoot()
	if err != nil {
		t.Fatalf("WorktreeRoot: %v", err)
	}
	if root == "" {
		t.Error("WorktreeRoot returned empty string")
	}
}

// --- Edge Case Tests ---

func TestAdd_AppendToExisting(t *testing.T) {
	dir := t.TempDir()
	content := `# Backlog

- [ ] [a7k2] 2025-06-15: Existing idea
`
	path := writeBacklog(t, dir, content)

	_, err := Add(path, "New idea", "b1c2", "2025-07-01")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "# Backlog") {
		t.Error("header should be preserved")
	}
	if !strings.Contains(result, "a7k2") {
		t.Error("existing idea should be preserved")
	}
	if !strings.Contains(result, "b1c2") {
		t.Error("new idea should be present")
	}
}

func TestDone_PreservesOtherIdeas(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: First
- [ ] [c3d4] 2025-06-10: Second
`
	path := writeBacklog(t, dir, content)

	_, err := Done(path, "a7k2")
	if err != nil {
		t.Fatalf("Done: %v", err)
	}

	data, _ := os.ReadFile(path)
	result := string(data)
	if !strings.Contains(result, "- [x] [a7k2]") {
		t.Error("first idea should be marked done")
	}
	if !strings.Contains(result, "- [ ] [c3d4]") {
		t.Error("second idea should remain open")
	}
}

func TestEdit_PreservesStatus(t *testing.T) {
	dir := t.TempDir()
	content := `- [x] [a7k2] 2025-06-15: Done idea
`
	path := writeBacklog(t, dir, content)

	i, err := Edit(path, "a7k2", "Updated done idea", "", "")
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if !i.Done {
		t.Error("status should be preserved as done")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "- [x] [a7k2]") {
		t.Error("done marker should be preserved in file")
	}
}

func TestList_EmptyResult(t *testing.T) {
	dir := t.TempDir()
	content := `- [ ] [a7k2] 2025-06-15: Open one
`
	path := writeBacklog(t, dir, content)

	ideas, err := List(path, FilterDone, "date", false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ideas) != 0 {
		t.Errorf("count = %d, want 0", len(ideas))
	}
}
