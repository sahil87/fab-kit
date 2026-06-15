package prmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/impact"
)

// baseData returns a fully-populated, "happy path" Data the table-driven cases
// mutate. Defaults: plan complete, review done after 2 cycles, hyperlinked
// stages, an impact result with a tests pair and two excludes.
func baseData() Data {
	return Data{
		ID:               "rj31",
		Name:             "260604-rj31-mechanical-pr-meta",
		Type:             "feat",
		HasConfidence:    true,
		ConfidenceScore:  4.2,
		HasPlan:          true,
		TasksDone:        8,
		TasksTotal:       8,
		AcceptanceDone:   9,
		AcceptanceTotal:  9,
		ReviewState:      "done",
		ReviewIterations: 2,
		Progress: map[string]string{
			"intake": "done", "apply": "done", "review": "done",
			"hydrate": "pending", "ship": "pending", "review-pr": "pending",
		},
		HasIntake: true,
		HasApply:  true,
		OwnerRepo: "o/r",
		Branch:    "b",
		IntakeURL: "https://github.com/o/r/blob/b/fab/changes/260604-rj31-mechanical-pr-meta/intake.md",
		ApplyURL:  "https://github.com/o/r/blob/b/fab/changes/260604-rj31-mechanical-pr-meta/plan.md",
		HasImpact: true,
		Impact: impact.Result{
			Added: 142, Deleted: 38, Net: 104,
			Excluding: &impact.Pair{Added: 87, Deleted: 38, Net: 49},
			Tests:     &impact.Pair{Added: 40, Deleted: 0, Net: 40},
		},
		Excludes: []string{"fab/", "docs/"},
	}
}

// TestRender_Table covers the table cell-population matrix.
func TestRender_Table(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Data)
		want   string // expected table row line
	}{
		{
			name:   "complete plan, review done 2 cycles",
			mutate: func(d *Data) {},
			want:   "| rj31 | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✓ 2 cycles |",
		},
		{
			name: "no plan, pending review",
			mutate: func(d *Data) {
				d.HasPlan = false
				d.ReviewState = "pending"
				d.ReviewIterations = 0
			},
			want: "| rj31 | feat | 4.2/5.0 | — | — |",
		},
		{
			name: "missing id and confidence",
			mutate: func(d *Data) {
				d.ID = ""
				d.HasConfidence = false
			},
			want: "| — | feat | — | 8/8 tasks, 9/9 acceptance ✓ | ✓ 2 cycles |",
		},
		{
			name: "incomplete plan, no completion check",
			mutate: func(d *Data) {
				d.TasksDone = 5
				d.AcceptanceDone = 3
			},
			want: "| rj31 | feat | 4.2/5.0 | 5/8 tasks, 3/9 acceptance | ✓ 2 cycles |",
		},
		{
			name: "review failed 1 cycle (singular)",
			mutate: func(d *Data) {
				d.ReviewState = "failed"
				d.ReviewIterations = 1
			},
			want: "| rj31 | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✗ 1 cycle |",
		},
		{
			name: "review done, no iteration count → bare check",
			mutate: func(d *Data) {
				d.ReviewIterations = 0
			},
			want: "| rj31 | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✓ |",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := baseData()
			tc.mutate(&d)
			got := Render(d)
			if !strings.Contains(got, tc.want) {
				t.Errorf("missing table row %q\nfull output:\n%s", tc.want, got)
			}
		})
	}
}

func TestRender_TableHeader(t *testing.T) {
	got := Render(baseData())
	for _, want := range []string{
		"## Meta\n\n",
		"| ID | Type | Confidence | Plan | Review |\n",
		"|----|------|-----------|------|--------|\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q\nfull output:\n%s", want, got)
		}
	}
}

// TestRender_Pipeline covers the Pipeline line — done markers and hyperlinking.
func TestRender_Pipeline(t *testing.T) {
	t.Run("hyperlinked with done markers", func(t *testing.T) {
		got := renderPipeline(baseData())
		want := "**Pipeline**: [intake](https://github.com/o/r/blob/b/fab/changes/260604-rj31-mechanical-pr-meta/intake.md) ✓ → " +
			"[apply](https://github.com/o/r/blob/b/fab/changes/260604-rj31-mechanical-pr-meta/plan.md) ✓ → " +
			"review ✓ → hydrate → ship → review-pr"
		if got != want {
			t.Errorf("pipeline =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("plain-text labels when no owner/repo (gh failure)", func(t *testing.T) {
		d := baseData()
		d.IntakeURL = ""
		d.ApplyURL = ""
		got := renderPipeline(d)
		want := "**Pipeline**: intake ✓ → apply ✓ → review ✓ → hydrate → ship → review-pr"
		if got != want {
			t.Errorf("pipeline =\n%q\nwant\n%q", got, want)
		}
	})
}

// TestRender_Issues covers the Issues line: Linear-linked, bare, and omitted.
func TestRender_Issues(t *testing.T) {
	t.Run("linear-linked", func(t *testing.T) {
		d := baseData()
		d.Issues = []string{"DEV-1", "DEV-2"}
		d.LinearWorkspace = "acme"
		got := renderIssues(d)
		want := "**Issues**: [DEV-1](https://linear.app/acme/issue/DEV-1), [DEV-2](https://linear.app/acme/issue/DEV-2)"
		if got != want {
			t.Errorf("issues = %q, want %q", got, want)
		}
	})

	t.Run("bare ids when no workspace", func(t *testing.T) {
		d := baseData()
		d.Issues = []string{"DEV-1", "DEV-2"}
		got := renderIssues(d)
		if got != "**Issues**: DEV-1, DEV-2" {
			t.Errorf("issues = %q", got)
		}
	})

	t.Run("omitted when empty", func(t *testing.T) {
		if got := renderIssues(baseData()); got != "" {
			t.Errorf("expected no Issues line, got %q", got)
		}
	})

	t.Run("issues line positioned between pipeline and impact", func(t *testing.T) {
		d := baseData()
		d.Issues = []string{"DEV-1"}
		got := Render(d)
		pipeIdx := strings.Index(got, "**Pipeline**:")
		issIdx := strings.Index(got, "**Issues**:")
		impIdx := strings.Index(got, "**Impact**:")
		if !(pipeIdx < issIdx && issIdx < impIdx) {
			t.Errorf("ordering wrong: pipeline=%d issues=%d impact=%d\n%s", pipeIdx, issIdx, impIdx, got)
		}
	})
}

// TestRender_Impact covers the three-row form, single-line form, omission, and
// the per-component clamp / Unicode minus / backtick excludes.
func TestRender_Impact(t *testing.T) {
	t.Run("three-row form with excludes", func(t *testing.T) {
		got := renderImpact(baseData())
		want := "**Impact**:\n" +
			"  impl:  +47/−38  (net +9)\n" +
			"  tests: +40/−0  (net +40)\n" +
			"  total: +87/−38  (net +49)  ← excludes `fab/`, `docs/`"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("three-row form clamps negative impl components to zero", func(t *testing.T) {
		d := baseData()
		// tests exceeds total in every component → impl clamps to 0/0/0.
		d.Impact.Excluding = &impact.Pair{Added: 10, Deleted: 2, Net: 8}
		d.Impact.Tests = &impact.Pair{Added: 40, Deleted: 5, Net: 35}
		got := renderImpact(d)
		if !strings.Contains(got, "impl:  +0/−0  (net +0)") {
			t.Errorf("expected clamped impl row, got:\n%s", got)
		}
	})

	t.Run("three-row form omits annotation when no excludes", func(t *testing.T) {
		d := baseData()
		d.Excludes = nil
		d.Impact.Excluding = nil // no excluding pass → total = raw
		got := renderImpact(d)
		// total degenerates to raw (142/38); impl = 142-40 / 38-0.
		want := "**Impact**:\n" +
			"  impl:  +102/−38  (net +64)\n" +
			"  tests: +40/−0  (net +40)\n" +
			"  total: +142/−38  (net +104)"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
		if strings.Contains(got, "← excludes") {
			t.Errorf("expected no excludes annotation, got:\n%s", got)
		}
	})

	t.Run("single-line form with excludes", func(t *testing.T) {
		d := baseData()
		d.Impact.Tests = nil
		got := renderImpact(d)
		want := "**Impact**: +87/−38 code (excluding `fab/`, `docs/`) · +142/−38 total"
		if got != want {
			t.Errorf("impact = %q\nwant %q", got, want)
		}
	})

	t.Run("single-line form without excludes", func(t *testing.T) {
		d := baseData()
		d.Impact.Tests = nil
		d.Impact.Excluding = nil
		d.Excludes = nil
		got := renderImpact(d)
		want := "**Impact**: +142/−38 total"
		if got != want {
			t.Errorf("impact = %q\nwant %q", got, want)
		}
	})

	t.Run("omitted when HasImpact false", func(t *testing.T) {
		d := baseData()
		d.HasImpact = false
		if got := renderImpact(d); got != "" {
			t.Errorf("expected no Impact line, got %q", got)
		}
	})

	t.Run("omitted on +0/−0 total", func(t *testing.T) {
		d := baseData()
		d.Impact = impact.Result{Added: 0, Deleted: 0, Net: 0, Excluding: &impact.Pair{}}
		if got := renderImpact(d); got != "" {
			t.Errorf("expected no Impact line for +0/−0, got %q", got)
		}
	})

	// jznd (e): a test-heavy PR where tests exceed the total drives impl net
	// negative. The displayed net stays clamped at +0 (downstream consumers may
	// assume non-negative), but the impl row MUST annotate the true negative
	// value rather than silently hiding the net-deletion-in-production.
	t.Run("annotates true value when impl net clamps", func(t *testing.T) {
		d := baseData()
		// total = excluding pass = 10/2 (net 8); tests = 50/3 (net 47).
		// impl raw = -40/-1 (net -39) → all clamp to 0.
		d.Impact.Excluding = &impact.Pair{Added: 10, Deleted: 2, Net: 8}
		d.Impact.Tests = &impact.Pair{Added: 50, Deleted: 3, Net: 47}
		got := renderImpact(d)
		if !strings.Contains(got, "impl:  +0/−0  (net +0)") {
			t.Errorf("expected clamped impl display value, got:\n%s", got)
		}
		if !strings.Contains(got, "clamped from net -39") {
			t.Errorf("expected clamp annotation naming the true negative net, got:\n%s", got)
		}
		if !strings.Contains(got, "added -40") || !strings.Contains(got, "deleted -1") {
			t.Errorf("expected clamp annotation naming clamped added/deleted, got:\n%s", got)
		}
	})

	t.Run("no clamp annotation when impl net is non-negative", func(t *testing.T) {
		got := renderImpact(baseData())
		if strings.Contains(got, "clamped from") {
			t.Errorf("did not expect a clamp annotation for non-negative impl, got:\n%s", got)
		}
	})
}

// TestRender_FullBlock asserts overall block structure (blank-line separation
// between sections) for the full happy path with issues.
func TestRender_FullBlock(t *testing.T) {
	d := baseData()
	d.Issues = []string{"DEV-1"}
	d.LinearWorkspace = "acme"
	got := Render(d)

	for _, want := range []string{
		"## Meta\n\n| ID | Type",
		"|\n\n**Pipeline**:", // table ends, blank line, pipeline
		"\n\n**Issues**:",
		"\n\n**Impact**:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing structural marker %q\nfull output:\n%s", want, got)
		}
	}
}

// TestHasConfidenceBlock covers the presence check that restores the old Step 3c
// "—" parity: a status file without a `confidence:` mapping must report false so
// the table renders "—" rather than "0.0/5.0".
func TestHasConfidenceBlock(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "present block",
			content: "id: rj31\nconfidence:\n  score: 3.5\n",
			want:    true,
		},
		{
			name:    "present but zero-valued block",
			content: "id: rj31\nconfidence:\n  score: 0.0\n",
			want:    true,
		},
		{
			name:    "absent block",
			content: "id: rj31\nprogress:\n  intake: done\n",
			want:    false,
		},
		{
			// Regression: an explicit null must NOT count as present (it would
			// otherwise render 0.0/5.0 instead of —). The old `!= nil` check
			// risked this; the mapping-node check rejects it.
			name:    "explicit null block",
			content: "id: rj31\nconfidence: null\n",
			want:    false,
		},
		{
			name:    "bare key with no value",
			content: "id: rj31\nconfidence:\n",
			want:    false,
		},
		{
			name:    "empty mapping",
			content: "id: rj31\nconfidence: {}\n",
			want:    false,
		},
		{
			name:    "non-mapping scalar",
			content: "id: rj31\nconfidence: 5\n",
			want:    false,
		},
		{
			name:    "malformed yaml",
			content: "id: rj31\n  : : :\n",
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".status.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write status file: %v", err)
			}
			if got := hasConfidenceBlock(path); got != tc.want {
				t.Errorf("hasConfidenceBlock = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("missing file", func(t *testing.T) {
		if hasConfidenceBlock(filepath.Join(t.TempDir(), "nope.yaml")) {
			t.Error("hasConfidenceBlock on missing file = true, want false")
		}
	})
}

// TestCountCheckboxesInTasksSection covers the section-bounded checkbox tally,
// including the exact-heading guard: a heading like "## TasksAndStuff" must not
// open the Tasks section, and counting must stop at the next "## " heading.
func TestCountCheckboxesInTasksSection(t *testing.T) {
	cases := []struct {
		name              string
		content           string
		wantDone, wantTot int
	}{
		{
			name:     "basic mixed checkboxes",
			content:  "## Tasks\n- [x] T001 done\n- [ ] T002 todo\n- [X] T003 done\n",
			wantDone: 2, wantTot: 3,
		},
		{
			name:     "stops at next heading",
			content:  "## Tasks\n- [x] T001\n## Acceptance\n- [ ] A-001 must not count\n",
			wantDone: 1, wantTot: 1,
		},
		{
			name: "heading with trailing text still opens section",
			// "## Tasks (13)" — trailing text after a space is allowed.
			content:  "## Tasks (13)\n- [x] T001\n- [ ] T002\n",
			wantDone: 1, wantTot: 2,
		},
		{
			// Regression for the Copilot finding: "## TasksAndStuff" must NOT be
			// treated as the Tasks section, so its checkboxes are not counted.
			name:     "prefix-only heading does not open section",
			content:  "## TasksAndStuff\n- [x] not a task\n- [ ] also not\n",
			wantDone: 0, wantTot: 0,
		},
		{
			name:     "no tasks section",
			content:  "## Requirements\n- [x] R1\n",
			wantDone: 0, wantTot: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "plan.md")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write plan: %v", err)
			}
			done, total := countCheckboxesInTasksSection(path)
			if done != tc.wantDone || total != tc.wantTot {
				t.Errorf("countCheckboxesInTasksSection = (%d, %d), want (%d, %d)",
					done, total, tc.wantDone, tc.wantTot)
			}
		})
	}
}
