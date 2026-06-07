package memoryindex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Pure renderer fixtures (byte-for-byte) -------------------------------

func TestRenderRoot_DomainsOnly(t *testing.T) {
	got := RenderRoot(RootData{
		Domains: []DomainRow{
			{Name: "auth", Description: "Authentication and authorization"},
			{Name: "payments", Description: "Payment processing and billing"},
		},
	})

	// Domains-only table — no inlined per-file column.
	wantRows := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Authentication and authorization |\n" +
		"| [payments](payments/index.md) | Payment processing and billing |\n"
	if !strings.HasSuffix(got, wantRows) {
		t.Fatalf("RenderRoot table mismatch.\n--- got ---\n%s\n--- want suffix ---\n%s", got, wantRows)
	}
	if !strings.HasPrefix(got, "# Memory Index\n") {
		t.Errorf("RenderRoot should start with the Memory Index H1, got:\n%s", got)
	}
	if strings.Contains(got, "Memory Files") {
		t.Error("RenderRoot must NOT contain the legacy 'Memory Files' per-file column")
	}
}

func TestRenderRoot_MissingDescriptionDegrades(t *testing.T) {
	got := RenderRoot(RootData{Domains: []DomainRow{{Name: "auth", Description: ""}}})
	want := "| [auth](auth/index.md) | " + missingCell + " |\n"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("missing description should degrade to %q.\ngot:\n%s", missingCell, got)
	}
}

func TestRenderDomain_FileRows(t *testing.T) {
	got := RenderDomain(DomainData{
		Name:  "auth",
		Title: "Auth Documentation",
		Files: []FileEntry{
			{Base: "authentication", Description: "Login & sessions", LastUpdated: "2026-05-08"},
			{Base: "authorization", Description: "Roles & permissions", LastUpdated: "2026-04-02"},
		},
	})
	wantRows := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [authentication](authentication.md) | Login & sessions | 2026-05-08 |\n" +
		"| [authorization](authorization.md) | Roles & permissions | 2026-04-02 |\n"
	if !strings.HasSuffix(got, wantRows) {
		t.Fatalf("RenderDomain table mismatch.\n--- got ---\n%s\n--- want suffix ---\n%s", got, wantRows)
	}
	if !strings.HasPrefix(got, "# Auth Documentation\n") {
		t.Errorf("RenderDomain should start with the domain Title H1, got:\n%s", got)
	}
}

func TestRenderDomain_MissingDescriptionAndDateDegrade(t *testing.T) {
	got := RenderDomain(DomainData{
		Name:  "auth",
		Title: "Auth Documentation",
		Files: []FileEntry{{Base: "orphan", Description: "", LastUpdated: ""}},
	})
	want := "| [orphan](orphan.md) | " + missingCell + " | " + missingCell + " |\n"
	if !strings.HasSuffix(got, want) {
		t.Fatalf("missing description+date should both degrade.\ngot:\n%s", got)
	}
}

func TestRender_Idempotent(t *testing.T) {
	root := RootData{Domains: []DomainRow{{Name: "auth", Description: "x"}}}
	if RenderRoot(root) != RenderRoot(root) {
		t.Error("RenderRoot is not byte-stable for identical input")
	}
	dom := DomainData{Name: "auth", Title: "Auth", Files: []FileEntry{{Base: "a", Description: "d", LastUpdated: "2026-01-01"}}}
	if RenderDomain(dom) != RenderDomain(dom) {
		t.Error("RenderDomain is not byte-stable for identical input")
	}
}

// --- Warning rendering -----------------------------------------------------

func TestWarning_String(t *testing.T) {
	w := Warning{Path: "docs/memory/fab-workflow", Kind: "width", Count: 20}
	got := w.String()
	if !strings.Contains(got, "20 topic files") || !strings.Contains(got, "~12") {
		t.Errorf("width warning should name count and bound, got: %q", got)
	}
	d := Warning{Path: "docs/memory/a/b/c", Kind: "depth", Depth: 4}
	if !strings.Contains(d.String(), "exceeds depth 3") {
		t.Errorf("depth warning should name max depth, got: %q", d.String())
	}
}

// --- Gather integration on a synthetic tree --------------------------------

// writeFile writes content to repoRoot/relpath, creating parent dirs.
func writeFile(t *testing.T, repoRoot, relpath, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, filepath.FromSlash(relpath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGather_ReadsFrontmatterAndH1(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "docs/memory/auth/authentication.md",
		"---\ndescription: \"Login & sessions\"\n---\n# Authentication\n\n## Overview\n")
	writeFile(t, repo, "docs/memory/auth/index.md",
		"---\ndescription: \"Auth domain\"\n---\n# Auth Documentation\n")

	root, domains, _, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 || domains[0].Name != "auth" {
		t.Fatalf("expected one auth domain, got %+v", domains)
	}
	if len(domains[0].Files) != 1 {
		t.Fatalf("expected one topic file (index.md excluded), got %d", len(domains[0].Files))
	}
	f := domains[0].Files[0]
	if f.Base != "authentication" || f.Description != "Login & sessions" || f.Title != "Authentication" {
		t.Errorf("file entry mismatch: %+v", f)
	}
	// Domain index.md H1 + description feed Title and root row.
	if domains[0].Title != "Auth Documentation" {
		t.Errorf("domain title should come from index.md H1, got %q", domains[0].Title)
	}
	if len(root.Domains) != 1 || root.Domains[0].Description != "Auth domain" {
		t.Errorf("root row should use domain index.md description, got %+v", root.Domains)
	}
}

func TestGather_UncommittedDateDegrades(t *testing.T) {
	// A temp dir is not a git repo → git log produces no output → "" date,
	// which the renderer degrades to the missing-cell fallback.
	repo := t.TempDir()
	writeFile(t, repo, "docs/memory/auth/x.md", "# X\n")
	_, domains, _, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	if domains[0].Files[0].LastUpdated != "" {
		t.Errorf("uncommitted file should have empty LastUpdated, got %q", domains[0].Files[0].LastUpdated)
	}
	out := RenderDomain(domains[0])
	if !strings.Contains(out, "| "+missingCell+" |") {
		t.Errorf("uncommitted date should render as %q in domain index, got:\n%s", missingCell, out)
	}
}

func TestGather_WidthWarningAndReservedExemption(t *testing.T) {
	repo := t.TempDir()
	// fab-workflow: 13 files → over the 12 bound → warns.
	for i := 0; i < WidthWarnThreshold+1; i++ {
		writeFile(t, repo, "docs/memory/fab-workflow/f"+string(rune('a'+i))+".md", "# F\n")
	}
	// _unsorted: also over-wide, but reserved → exempt.
	for i := 0; i < WidthWarnThreshold+1; i++ {
		writeFile(t, repo, "docs/memory/_unsorted/g"+string(rune('a'+i))+".md", "# G\n")
	}
	// small domain: under bound → no warning.
	writeFile(t, repo, "docs/memory/small/a.md", "# A\n")

	_, _, warnings, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	var widthPaths []string
	for _, w := range warnings {
		if w.Kind == "width" {
			widthPaths = append(widthPaths, w.Path)
		}
	}
	if len(widthPaths) != 1 || widthPaths[0] != "docs/memory/fab-workflow" {
		t.Fatalf("expected exactly one width warning for fab-workflow, got %v", widthPaths)
	}
}

func TestGather_DepthWarning(t *testing.T) {
	repo := t.TempDir()
	// docs/memory/d/sub/deep/topic.md = depth 4 under docs/memory/ → warns.
	writeFile(t, repo, "docs/memory/d/sub/deep/topic.md", "# T\n")
	_, _, warnings, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range warnings {
		if w.Kind == "depth" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a depth warning for a depth-4 file, got %+v", warnings)
	}
}

func TestGather_DeterministicOrdering(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "docs/memory/zeta/c.md", "# C\n")
	writeFile(t, repo, "docs/memory/zeta/a.md", "# A\n")
	writeFile(t, repo, "docs/memory/alpha/b.md", "# B\n")

	_, domains, _, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	if domains[0].Name != "alpha" || domains[1].Name != "zeta" {
		t.Errorf("domains should be lexicographic, got %s,%s", domains[0].Name, domains[1].Name)
	}
	zeta := domains[1]
	if zeta.Files[0].Base != "a" || zeta.Files[1].Base != "c" {
		t.Errorf("files should be lexicographic by base, got %+v", zeta.Files)
	}
}

func TestGather_MissingMemoryDirErrors(t *testing.T) {
	repo := t.TempDir() // no docs/memory/
	if _, _, _, err := Gather(repo); err == nil {
		t.Error("expected an error when docs/memory/ is absent")
	}
}

// TestGather_SelfHealsStaleRoster is the loom-derived regression scenario
// (intake Assumption #11): a domain folder whose actual file count differs
// from a stale hand-maintained roster. The generated index reflects the real
// on-disk count, by construction — there is no roster to drift.
func TestGather_SelfHealsStaleRoster(t *testing.T) {
	repo := t.TempDir()
	// Simulate loom's wd-web/canvas: roster once claimed "(12)" but 20 on disk.
	for i := 0; i < 20; i++ {
		writeFile(t, repo, "docs/memory/canvas/file"+string(rune('a'+i))+".md",
			"---\ndescription: \"d\"\n---\n# F\n")
	}
	_, domains, warnings, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains[0].Files) != 20 {
		t.Errorf("generated index must reflect the true 20-file count, got %d", len(domains[0].Files))
	}
	// And it should warn that 20 > 12.
	sawWidth := false
	for _, w := range warnings {
		if w.Kind == "width" && w.Count == 20 {
			sawWidth = true
		}
	}
	if !sawWidth {
		t.Error("expected a width warning for the 20-file canvas domain")
	}
}
