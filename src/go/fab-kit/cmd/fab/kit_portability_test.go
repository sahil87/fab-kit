package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// hostConventionPaths are the documented host-project convention paths that
// deployed kit content may cite (they describe the customer's docs layout,
// which fab docs-index generates in the host project).
var hostConventionPaths = map[string]bool{
	"docs/memory/index.md":                   true,
	"docs/specs/index.md":                    true,
	"docs/memory/_shared/removed-domains.md": true,
	"docs/memory/_shared/utilities.md":       true,
}

var (
	kitDocPathRe = regexp.MustCompile(`\bdocs/(specs|memory|site)/[A-Za-z0-9_./-]+\.md\b`)
	kitGoPathRe  = regexp.MustCompile(`\bsrc/go/[A-Za-z0-9_./-]+\b`)
)

// kitGuardExcludes are paths under src/kit/ skipped by the walk: VERSION is
// content-free, and reference/fkf.md is a drift-guarded byte-copy of the
// published standard that cannot be rewritten in place.
var kitGuardExcludes = map[string]bool{
	"VERSION":          true,
	"reference/fkf.md": true,
}

type kitPathViolation struct {
	file string
	line int
	path string
}

func (v kitPathViolation) String() string {
	return fmt.Sprintf("%s:%d — cites a repo-local doc path %s; restate the rule, or name the external repo/standard in prose", v.file, v.line, v.path)
}

// isExemptKitCitation reports whether a matched path is a legal citation from
// deployed kit content: a host-convention path, a generated tier index at any
// depth, or a placeholder-shaped illustrative example (a segment containing
// {…} or a bare x.md leaf). There is no marker or attribution-token escape
// hatch.
func isExemptKitCitation(p string) bool {
	if hostConventionPaths[p] {
		return true
	}
	if strings.HasSuffix(p, "/index.md") &&
		(strings.HasPrefix(p, "docs/memory/") || strings.HasPrefix(p, "docs/specs/")) {
		return true
	}
	for _, seg := range strings.Split(p, "/") {
		if strings.Contains(seg, "{") && strings.Contains(seg, "}") {
			return true
		}
	}
	return filepath.Base(p) == "x.md"
}

// scanKitTree walks every regular file under root (skipping kitGuardExcludes)
// and returns one violation per non-exempt repo-local doc path citation.
func scanKitTree(root string) ([]kitPathViolation, error) {
	var violations []kitPathViolation
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if kitGuardExcludes[filepath.ToSlash(rel)] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			matches := append(kitDocPathRe.FindAllString(line, -1), kitGoPathRe.FindAllString(line, -1)...)
			for _, m := range matches {
				if isExemptKitCitation(m) {
					continue
				}
				violations = append(violations, kitPathViolation{file: filepath.ToSlash(rel), line: i + 1, path: m})
			}
		}
		return nil
	})
	return violations, err
}

// TestKitContentCitesNoRepoLocalPaths guards Constitution V's deployed-content
// citation rule: everything under src/kit/ deploys into customer repos via
// fab sync, where fab-kit's docs/specs/*, docs/memory/* (outside the host
// convention paths), docs/site/*, and src/go/* do not exist — a citation of
// one is a dead pointer, or worse, resolves to the customer's unrelated doc.
func TestKitContentCitesNoRepoLocalPaths(t *testing.T) {
	kitRoot := findRepoFile(t, "src/kit")

	violations, err := scanKitTree(kitRoot)
	if err != nil {
		t.Fatalf("walk %s: %v", kitRoot, err)
	}
	for _, v := range violations {
		t.Error(v.String())
	}
}

// TestKitPortabilityMatcher exercises the matcher itself over a fixture tree:
// exactly the violating file must be reported; the allowlisted, tier-index,
// and placeholder-shaped citations must not be.
func TestKitPortabilityMatcher(t *testing.T) {
	root := t.TempDir()
	fixtures := map[string]string{
		"skills/violating.md":   "See docs/specs/change-types.md for the full taxonomy.\nAlso src/go/fab/cmd/fab/main.go.\n",
		"skills/allowlisted.md": "Read docs/memory/index.md and docs/memory/_shared/utilities.md first.\n",
		"skills/tier-index.md":  "Probe docs/memory/probe/index.md and docs/specs/probe/index.md.\n",
		"skills/placeholder.md": "Write docs/memory/{domain}/{file}.md; see docs/memory/pipeline/runtime/x.md.\n",
		"VERSION":               "docs/specs/not-checked.md\n",
		"reference/fkf.md":      "docs/specs/not-checked.md\n",
	}
	for rel, content := range fixtures {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	violations, err := scanKitTree(root)
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	var got []string
	for _, v := range violations {
		got = append(got, v.String())
	}
	want := []string{
		"skills/violating.md:1 — cites a repo-local doc path docs/specs/change-types.md; restate the rule, or name the external repo/standard in prose",
		"skills/violating.md:2 — cites a repo-local doc path src/go/fab/cmd/fab/main.go; restate the rule, or name the external repo/standard in prose",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("violations mismatch:\n  got:  %q\n  want: %q", got, want)
	}
}
