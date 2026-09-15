package prmeta

import (
	"os"
	"os/exec"
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
		Version:  "2.6.6", // fixed so the provenance caption is byte-stable
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
			want:   "| `rj31` | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✓ 2 cycles |",
		},
		{
			name: "no plan, pending review",
			mutate: func(d *Data) {
				d.HasPlan = false
				d.ReviewState = "pending"
				d.ReviewIterations = 0
			},
			want: "| `rj31` | feat | 4.2/5.0 | — | — |",
		},
		{
			name: "missing id and confidence (bare em-dash, no backticks)",
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
			want: "| `rj31` | feat | 4.2/5.0 | 5/8 tasks, 3/9 acceptance | ✓ 2 cycles |",
		},
		{
			name: "review failed 1 cycle (singular)",
			mutate: func(d *Data) {
				d.ReviewState = "failed"
				d.ReviewIterations = 1
			},
			want: "| `rj31` | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✗ 1 cycle |",
		},
		{
			name: "review done, no iteration count → bare check",
			mutate: func(d *Data) {
				d.ReviewIterations = 0
			},
			want: "| `rj31` | feat | 4.2/5.0 | 8/8 tasks, 9/9 acceptance ✓ | ✓ |",
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
		"| Change ID | Type | Confidence | Plan | Review |\n",
		"|-----------|------|------------|------|--------|\n",
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
		want := "**Pipeline:** [intake](https://github.com/o/r/blob/b/fab/changes/260604-rj31-mechanical-pr-meta/intake.md) ✓ → " +
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
		want := "**Pipeline:** intake ✓ → apply ✓ → review ✓ → hydrate → ship → review-pr"
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

	t.Run("issues line positioned between impact and pipeline", func(t *testing.T) {
		d := baseData()
		d.Issues = []string{"DEV-1"}
		got := Render(d)
		// 260625 layout: table → Impact → Issues → Pipeline (last). The Impact
		// table self-labels via its `| Impact |` header (no `**Impact**:` lead-in).
		impIdx := strings.Index(got, "| Impact | +/− | Net |")
		issIdx := strings.Index(got, "**Issues**:")
		pipeIdx := strings.Index(got, "**Pipeline:**")
		if !(impIdx < issIdx && issIdx < pipeIdx) {
			t.Errorf("ordering wrong: impact=%d issues=%d pipeline=%d\n%s", impIdx, issIdx, pipeIdx, got)
		}
	})
}

// TestRender_Impact covers the single-table Option A render (pnao): the four
// (excludes ± tests) combinations, the omission cases, the dev-version caption,
// the row-drop rules, and the per-component impl clamp / annotation.
func TestRender_Impact(t *testing.T) {
	// Header + right-aligned separator shared by every non-omitted render. The
	// 260625 layout drops the `**Impact**:` lead-in (the `| Impact |` header
	// self-labels) and compacts the `+/−` header.
	const head = "| Impact | +/− | Net |\n" +
		"|--------|----:|----:|\n"

	t.Run("excludes + tests: raw, bold true, nested impl/tests, caption", func(t *testing.T) {
		got := renderImpact(baseData())
		// excludes configured (Excluding != nil) → raw row shown.
		// raw=142/38 (104); true=87/38 (49); impl = true−tests = 47/38 (9); tests = 40/0 (40).
		want := head +
			"| raw | +142 / −38 | +104 |\n" +
			"| **true** | **+87 / −38** | **+49** |\n" +
			"| └ impl | +47 / −38 | +9 |\n" +
			"| └ tests | +40 / −0 | +40 |\n" +
			"\n<sub>excludes `fab/`, `docs/` · generated by fab-kit v2.6.6</sub>"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("no excludes + tests: no duplicate raw row, no-excludes caption", func(t *testing.T) {
		d := baseData()
		d.Excludes = nil
		d.Impact.Excluding = nil // no excluding pass → true == raw → no separate raw row
		got := renderImpact(d)
		// true = raw = 142/38 (104); impl = 102/38 (64); tests = 40/0 (40).
		want := head +
			"| **true** | **+142 / −38** | **+104** |\n" +
			"| └ impl | +102 / −38 | +64 |\n" +
			"| └ tests | +40 / −0 | +40 |\n" +
			"\n<sub>generated by fab-kit v2.6.6</sub>"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
		if strings.Contains(got, "| raw |") {
			t.Errorf("expected no duplicate raw row when Excluding == nil, got:\n%s", got)
		}
		if strings.Contains(got, "excludes") {
			t.Errorf("expected no excludes clause in caption, got:\n%s", got)
		}
	})

	t.Run("excludes configured but no-op (raw == true): raw row still shown", func(t *testing.T) {
		// New rule (pnao follow-up): the raw row is shown whenever excludes are
		// configured (Excluding != nil), EVEN IF the excluding pass touched no
		// lines so raw == true. A configured-but-no-op exclude still earns its
		// own raw row — the reader always sees raw alongside true when excludes
		// apply. (This is distinct from the no-excludes case above, where
		// Excluding == nil collapses raw into true and the row is dropped.)
		d := baseData()
		d.Impact.Tests = nil
		// Excluding equals the raw triple: excludes configured but matched nothing.
		d.Impact.Excluding = &impact.Pair{Added: 142, Deleted: 38, Net: 104}
		got := renderImpact(d)
		want := head +
			"| raw | +142 / −38 | +104 |\n" +
			"| **true** | **+142 / −38** | **+104** |\n" +
			"\n<sub>excludes `fab/`, `docs/` · generated by fab-kit v2.6.6</sub>"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("excludes + no tests: raw + bold true only", func(t *testing.T) {
		d := baseData()
		d.Impact.Tests = nil
		got := renderImpact(d)
		want := head +
			"| raw | +142 / −38 | +104 |\n" +
			"| **true** | **+87 / −38** | **+49** |\n" +
			"\n<sub>excludes `fab/`, `docs/` · generated by fab-kit v2.6.6</sub>"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
		if strings.Contains(got, "└ impl") || strings.Contains(got, "└ tests") {
			t.Errorf("expected no nested rows without a tests pair, got:\n%s", got)
		}
	})

	t.Run("no excludes + no tests: bold true row only", func(t *testing.T) {
		d := baseData()
		d.Impact.Tests = nil
		d.Impact.Excluding = nil
		d.Excludes = nil
		got := renderImpact(d)
		want := head +
			"| **true** | **+142 / −38** | **+104** |\n" +
			"\n<sub>generated by fab-kit v2.6.6</sub>"
		if got != want {
			t.Errorf("impact =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("dev build renders fab-kit vdev honestly", func(t *testing.T) {
		d := baseData()
		d.Version = "dev"
		got := renderImpact(d)
		if !strings.HasSuffix(got, "generated by fab-kit vdev</sub>") {
			t.Errorf("expected honest dev version stamp, got:\n%s", got)
		}
	})

	t.Run("omitted when HasImpact false", func(t *testing.T) {
		d := baseData()
		d.HasImpact = false
		if got := renderImpact(d); got != "" {
			t.Errorf("expected no Impact block, got %q", got)
		}
	})

	t.Run("omitted on +0/−0 true", func(t *testing.T) {
		d := baseData()
		d.Impact = impact.Result{Added: 0, Deleted: 0, Net: 0, Excluding: &impact.Pair{}}
		if got := renderImpact(d); got != "" {
			t.Errorf("expected no Impact block for +0/−0, got %q", got)
		}
	})

	// jznd (e): a test-heavy PR where tests exceed `true` drives impl net
	// negative. The displayed figures stay clamped at +0 (downstream consumers
	// may assume non-negative), but the └ impl row MUST annotate the true
	// negative value rather than silently hiding net-deletion-in-production.
	t.Run("clamps negative impl and annotates the true value", func(t *testing.T) {
		d := baseData()
		// true = excluding = 10/2 (net 8); tests = 50/3 (net 47).
		// impl pre = -40/-1 (net -39) → all clamp to 0.
		d.Impact.Excluding = &impact.Pair{Added: 10, Deleted: 2, Net: 8}
		d.Impact.Tests = &impact.Pair{Added: 50, Deleted: 3, Net: 47}
		got := renderImpact(d)
		if !strings.Contains(got, "| └ impl  (clamped from net -39, added -40, deleted -1) | +0 / −0 | +0 |") {
			t.Errorf("expected clamped impl row with annotation, got:\n%s", got)
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

	// 260625 layout order: heading → table → Impact table + <sub> caption →
	// Issues → Pipeline (last), each block blank-line separated.
	for _, want := range []string{
		"## Meta\n\n| Change ID | Type",
		"|\n\n| Impact | +/− | Net |", // table ends, blank line, Impact table
		"</sub>\n\n**Issues**:",       // caption ends, blank line, issues
		"\n\n**Pipeline:**",           // issues end, blank line, pipeline (last)
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing structural marker %q\nfull output:\n%s", want, got)
		}
	}

	// Hard ordering assertion: Impact precedes Issues precedes Pipeline.
	impIdx := strings.Index(got, "| Impact | +/− | Net |")
	issIdx := strings.Index(got, "**Issues**:")
	pipeIdx := strings.Index(got, "**Pipeline:**")
	if !(impIdx < issIdx && issIdx < pipeIdx) {
		t.Errorf("ordering wrong: impact=%d issues=%d pipeline=%d\n%s", impIdx, issIdx, pipeIdx, got)
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

// --- Gather base resolution (vy27) ---

const gatherStatusYAML = `id: t3st
name: 260915-t3st-stacked-gather
created: "2026-09-15T12:00:00Z"
created_by: test
change_type: feat
issues: []
progress:
  intake: done
  apply: done
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
plan:
  generated: true
  task_count: 1
  acceptance_count: 1
  acceptance_completed: 0
confidence:
  certain: 1
  confident: 0
  tentative: 0
  unresolved: 0
  score: 5.0
stage_metrics: {}
prs: []
last_updated: "2026-09-15T12:00:00Z"
`

// setupGatherRepo builds a git repo with a fab/ tree (config + one change
// folder) and returns (repoRoot, fabRoot). gitSteps runs after the initial
// commit so each test shapes its own refs/commits.
func setupGatherRepo(t *testing.T, baseBranch string, gitSteps func(repoRoot string, run func(args ...string))) (string, string) {
	t.Helper()
	repoRoot := t.TempDir()
	fabRoot := filepath.Join(repoRoot, "fab")

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}

	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")
	run("git", "config", "commit.gpgsign", "false")

	changeDir := filepath.Join(fabRoot, "changes", "260915-t3st-stacked-gather")
	if err := os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte("stage_hooks: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	statusYAML := gatherStatusYAML
	if baseBranch != "" {
		statusYAML = strings.Replace(statusYAML, "change_type: feat\n", "change_type: feat\nbase_branch: "+baseBranch+"\n", 1)
	}
	if err := os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(statusYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "initial")

	gitSteps(repoRoot, run)

	return repoRoot, fabRoot
}

// TestGather_StackedBaseExcludesParent covers vy27: with base_branch: parent
// (parent carrying its own commit past main), Gather's Impact counts exclude
// the parent's lines; without the field they measure against the default
// branch exactly as before.
func TestGather_StackedBaseExcludesParent(t *testing.T) {
	stackedSteps := func(repoRoot string, run func(args ...string)) {
		run("git", "update-ref", "refs/remotes/origin/main", "HEAD")
		// parent branch: its own commit (3 added lines) past main.
		run("git", "checkout", "-q", "-b", "parent")
		if err := os.WriteFile(filepath.Join(repoRoot, "parent.txt"), []byte("p1\np2\np3\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("git", "add", "-A")
		run("git", "commit", "-q", "-m", "parent work")
		run("git", "update-ref", "refs/remotes/origin/parent", "HEAD")
		// child branch off parent: its own commit (2 added lines), now HEAD.
		run("git", "checkout", "-q", "-b", "child")
		if err := os.WriteFile(filepath.Join(repoRoot, "child.txt"), []byte("c1\nc2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("git", "add", "-A")
		run("git", "commit", "-q", "-m", "child work")
	}

	_, fabRoot := setupGatherRepo(t, "parent", stackedSteps)
	d, ok, err := Gather(fabRoot, "260915-t3st-stacked-gather", "feat", "")
	if err != nil || !ok {
		t.Fatalf("Gather: ok=%v err=%v", ok, err)
	}
	if !d.HasImpact {
		t.Fatal("expected Impact block (stacked base)")
	}
	if d.Impact.Added != 2 {
		t.Errorf("stacked impact.added = %d, want 2 (child commit only; parent's 3 lines excluded)", d.Impact.Added)
	}

	_, fabRoot2 := setupGatherRepo(t, "", stackedSteps)
	d2, ok, err := Gather(fabRoot2, "260915-t3st-stacked-gather", "feat", "")
	if err != nil || !ok {
		t.Fatalf("Gather (plain): ok=%v err=%v", ok, err)
	}
	if !d2.HasImpact {
		t.Fatal("expected Impact block (default base)")
	}
	if d2.Impact.Added != 5 {
		t.Errorf("plain impact.added = %d, want 5 (parent + child against origin/main)", d2.Impact.Added)
	}
}

// TestGather_OriginHeadOnlyHasImpact covers vy27: a repo whose only base ref
// is origin/develop (via origin/HEAD → origin/develop, no origin/main or
// origin/master) still produces an Impact block — before the shared helper the
// hardcoded main/master probe silently emitted none.
func TestGather_OriginHeadOnlyHasImpact(t *testing.T) {
	_, fabRoot := setupGatherRepo(t, "", func(repoRoot string, run func(args ...string)) {
		run("git", "update-ref", "refs/remotes/origin/develop", "HEAD")
		run("git", "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/develop")
		if err := os.WriteFile(filepath.Join(repoRoot, "a.txt"), []byte("hello\nworld\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("git", "add", "-A")
		run("git", "commit", "-q", "-m", "feature work")
	})

	d, ok, err := Gather(fabRoot, "260915-t3st-stacked-gather", "feat", "")
	if err != nil || !ok {
		t.Fatalf("Gather: ok=%v err=%v", ok, err)
	}
	if !d.HasImpact {
		t.Fatal("expected Impact block with origin/HEAD → origin/develop")
	}
	if d.Impact.Added <= 0 {
		t.Errorf("expected non-zero impact.added, got %d", d.Impact.Added)
	}
}
