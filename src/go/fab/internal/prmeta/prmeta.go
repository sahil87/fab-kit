// Package prmeta renders the `## Meta` block of a fab-generated PR
// mechanically. It is the deterministic counterpart to the natural-language
// formatting prose that previously lived in the /git-pr skill (Step 3c):
// reading the same inputs (.status.yaml, plan.md, config, impact math, git/gh
// context) and emitting the exact same markdown on every run so the Meta block
// stops drifting between /git-pr invocations.
//
// Rendering is split into pure functions that take structured inputs and return
// markdown (Render and its helpers), plus a Gather orchestrator that performs
// the I/O (file reads, git/gh shelling). This mirrors internal/impact and
// internal/score and keeps the byte-for-byte render contract unit-testable
// without git/gh fixtures.
package prmeta

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/impact"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
	"gopkg.in/yaml.v3"
)

// pipelineStages is the fixed pipeline order rendered in the **Pipeline** line.
var pipelineStages = []string{"intake", "apply", "review", "hydrate", "ship", "review-pr"}

// Data holds every resolved input the Meta block renderer needs. It is produced
// by Gather (from the live repo) or constructed directly by tests. All fields
// are plain values so Render is a pure function of Data.
type Data struct {
	// Change identity / metadata
	ID   string // .status.yaml `id` (4-char change ID); "" → "—"
	Name string // change folder name (for blob URL construction)
	Type string // resolved PR type, passed via --type

	// Confidence
	HasConfidence   bool
	ConfidenceScore float64

	// Plan / task counts
	HasPlan         bool // a plan.md or legacy tasks.md was found
	TasksDone       int
	TasksTotal      int
	AcceptanceDone  int
	AcceptanceTotal int

	// Review
	ReviewState      string // progress.review (done|failed|pending|active|...)
	ReviewIterations int    // stage_metrics.review.iterations (0 → count dropped)

	// Pipeline progress: stage -> state
	Progress map[string]string

	// Artifact availability (for Pipeline hyperlinks)
	HasIntake bool
	HasApply  bool // plan.md (preferred) or legacy tasks.md present

	// Git/gh context for blob URLs
	OwnerRepo string // "owner/repo"; "" → plain-text stage labels
	Branch    string
	IntakeURL string // blob URL for intake.md (empty when unavailable)
	ApplyURL  string // blob URL for plan.md / tasks.md (empty when unavailable)

	// Issues
	Issues          []string // space-split issue IDs; empty → no Issues line
	LinearWorkspace string   // project.linear_workspace; "" → bare IDs

	// Impact
	HasImpact bool          // false → omit the Impact line entirely
	Impact    impact.Result // total = Excluding when present, else raw
	Excludes  []string      // true_impact_exclude (for the ← excludes annotation)
}

// Render assembles the complete `## Meta` block markdown for d. It is a pure
// function: identical Data always yields identical output. The block always
// contains the heading, table, and Pipeline line; the Issues and Impact lines
// are conditional.
func Render(d Data) string {
	var b strings.Builder
	b.WriteString("## Meta\n\n")
	b.WriteString(renderTable(d))
	b.WriteString("\n")
	b.WriteString(renderPipeline(d))

	if line := renderIssues(d); line != "" {
		b.WriteString("\n\n")
		b.WriteString(line)
	}
	if line := renderImpact(d); line != "" {
		b.WriteString("\n\n")
		b.WriteString(line)
	}
	return b.String()
}

// renderTable renders the 5-column Meta table (header + separator + single row).
func renderTable(d Data) string {
	id := d.ID
	if id == "" {
		id = "—"
	}

	confidence := "—"
	if d.HasConfidence {
		confidence = fmt.Sprintf("%.1f/5.0", d.ConfidenceScore)
	}

	var b strings.Builder
	b.WriteString("| ID | Type | Confidence | Plan | Review |\n")
	b.WriteString("|----|------|-----------|------|--------|\n")
	fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
		id, d.Type, confidence, planCell(d), reviewCell(d))
	return b.String()
}

// planCell renders the Plan column: "{done}/{total} tasks, {ad}/{at} acceptance"
// with a trailing " ✓" when both pairs are complete and non-zero. "—" when no
// plan/tasks artifact was found.
func planCell(d Data) string {
	if !d.HasPlan {
		return "—"
	}
	cell := fmt.Sprintf("%d/%d tasks, %d/%d acceptance",
		d.TasksDone, d.TasksTotal, d.AcceptanceDone, d.AcceptanceTotal)
	tasksComplete := d.TasksDone == d.TasksTotal && d.TasksTotal > 0
	acceptanceComplete := d.AcceptanceDone == d.AcceptanceTotal && d.AcceptanceTotal > 0
	if tasksComplete && acceptanceComplete {
		cell += " ✓"
	}
	return cell
}

// reviewCell renders the Review column from progress.review + iterations:
// done → "✓ {N} cycle{s}", failed → "✗ {N} cycle{s}", else "—". The count is
// dropped (bare ✓/✗) when iterations is 0.
func reviewCell(d Data) string {
	var mark string
	switch d.ReviewState {
	case "done":
		mark = "✓"
	case "failed":
		mark = "✗"
	default:
		return "—"
	}
	if d.ReviewIterations <= 0 {
		return mark
	}
	return fmt.Sprintf("%s %d %s", mark, d.ReviewIterations, pluralCycle(d.ReviewIterations))
}

func pluralCycle(n int) string {
	if n == 1 {
		return "cycle"
	}
	return "cycles"
}

// renderPipeline renders the **Pipeline** line: the six stages in fixed order
// joined by " → ", with " ✓" appended after each done stage. intake/apply
// labels hyperlink to their blob URLs when available; the rest are plain text.
func renderPipeline(d Data) string {
	parts := make([]string, 0, len(pipelineStages))
	for _, stage := range pipelineStages {
		label := stage
		switch stage {
		case "intake":
			if d.IntakeURL != "" {
				label = fmt.Sprintf("[intake](%s)", d.IntakeURL)
			}
		case "apply":
			if d.ApplyURL != "" {
				label = fmt.Sprintf("[apply](%s)", d.ApplyURL)
			}
		}
		if d.Progress[stage] == "done" {
			label += " ✓"
		}
		parts = append(parts, label)
	}
	return "**Pipeline**: " + strings.Join(parts, " → ")
}

// renderIssues renders the **Issues** line, or "" when there are no issues.
// Linear-linked when LinearWorkspace is set, bare comma-joined IDs otherwise.
func renderIssues(d Data) string {
	if len(d.Issues) == 0 {
		return ""
	}
	rendered := make([]string, 0, len(d.Issues))
	for _, id := range d.Issues {
		if d.LinearWorkspace != "" {
			rendered = append(rendered, fmt.Sprintf("[%s](https://linear.app/%s/issue/%s)", id, d.LinearWorkspace, id))
		} else {
			rendered = append(rendered, id)
		}
	}
	return "**Issues**: " + strings.Join(rendered, ", ")
}

// renderImpact renders the **Impact** line(s), or "" when the line must be
// omitted (no impact, or a +0/−0 total). When a tests pair is present it renders
// the three-row impl/tests/total breakdown; otherwise the single-line form.
func renderImpact(d Data) string {
	if !d.HasImpact {
		return ""
	}

	// total = the excluding pass when present, else the raw pass.
	total := impact.Pair{Added: d.Impact.Added, Deleted: d.Impact.Deleted, Net: d.Impact.Net}
	if d.Impact.Excluding != nil {
		total = *d.Impact.Excluding
	}

	// Omit entirely on a no-op total — never render +0/−0.
	if total.Added == 0 && total.Deleted == 0 {
		return ""
	}

	hasExcludes := len(d.Excludes) > 0

	if d.Impact.Tests != nil {
		tests := *d.Impact.Tests
		impl := impact.Pair{
			Added:   clampNonNeg(total.Added - tests.Added),
			Deleted: clampNonNeg(total.Deleted - tests.Deleted),
			Net:     clampNonNeg(total.Net - tests.Net),
		}
		var b strings.Builder
		b.WriteString("**Impact**:\n")
		fmt.Fprintf(&b, "  impl:  +%d/−%d  (net +%d)\n", impl.Added, impl.Deleted, impl.Net)
		fmt.Fprintf(&b, "  tests: +%d/−%d  (net +%d)\n", tests.Added, tests.Deleted, tests.Net)
		fmt.Fprintf(&b, "  total: +%d/−%d  (net +%d)", total.Added, total.Deleted, total.Net)
		if hasExcludes {
			fmt.Fprintf(&b, "  ← excludes %s", backtickList(d.Excludes))
		}
		return b.String()
	}

	// Single-line form. The trailing `total` pair is the raw measurement; the
	// leading `code` pair is the scaffolding-excluded total.
	if hasExcludes {
		return fmt.Sprintf("**Impact**: +%d/−%d code (excluding %s) · +%d/−%d total",
			total.Added, total.Deleted, backtickList(d.Excludes), d.Impact.Added, d.Impact.Deleted)
	}
	return fmt.Sprintf("**Impact**: +%d/−%d total", total.Added, total.Deleted)
}

func clampNonNeg(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// backtickList joins values with ", ", each wrapped in single backticks
// (e.g. "`fab/`, `docs/`"). Built from the actual config values — never
// hardcoded.
func backtickList(items []string) string {
	wrapped := make([]string, 0, len(items))
	for _, it := range items {
		wrapped = append(wrapped, "`"+it+"`")
	}
	return strings.Join(wrapped, ", ")
}

// Gather resolves changeArg and reads every input the Meta block needs from the
// live repo rooted at fabRoot. It returns (data, ok, err): ok is false (with a
// nil err) when there is no usable fab context — the caller should then exit
// non-zero and emit nothing, exactly as the skill omitted the Meta block when
// {has_fab} was false. A non-nil err is reserved for unexpected failures.
//
// Degradation is built in: a gh/owner-repo failure leaves OwnerRepo empty
// (plain-text Pipeline labels) and a missing/failed merge-base or impact
// computation leaves HasImpact false (no Impact line) — neither is an error.
func Gather(fabRoot, changeArg, prType, issues string) (Data, bool, error) {
	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		return Data{}, false, nil // no fab context → caller exits non-zero
	}

	changeDir := filepath.Join(fabRoot, "changes", folder)
	statusPath := filepath.Join(changeDir, ".status.yaml")
	status, err := sf.Load(statusPath)
	if err != nil {
		return Data{}, false, nil // absent/invalid .status.yaml → no Meta block
	}

	d := Data{
		ID:              status.ID,
		Name:            folder,
		Type:            prType,
		HasConfidence:   hasConfidenceBlock(statusPath),
		ConfidenceScore: status.Confidence.Score,
		ReviewState:     status.GetProgress("review"),
		Progress:        progressMap(status),
		Issues:          splitIssues(issues),
	}

	if rm, ok := status.StageMetrics["review"]; ok && rm != nil {
		d.ReviewIterations = rm.Iterations
	}

	// Plan / tasks counts. applyFile records which apply artifact was found
	// (plan.md preferred, legacy tasks.md fallback) so the blob URL below can
	// point at the right file without re-running fileExists.
	planPath := filepath.Join(changeDir, "plan.md")
	tasksPath := filepath.Join(changeDir, "tasks.md")
	var applyFile string
	if fileExists(planPath) {
		d.HasPlan = true
		d.HasApply = true
		applyFile = "plan.md"
		d.TasksDone, d.TasksTotal = countCheckboxesInTasksSection(planPath)
		d.AcceptanceDone = status.Plan.AcceptanceCompleted
		d.AcceptanceTotal = status.Plan.AcceptanceCount
	} else if fileExists(tasksPath) {
		// Legacy fallback for pre-1.9.0 changes (one-release back-compat).
		d.HasPlan = true
		d.HasApply = true
		applyFile = "tasks.md"
		d.TasksDone, d.TasksTotal = countCheckboxes(tasksPath)
		d.AcceptanceDone = status.Plan.AcceptanceCompleted
		d.AcceptanceTotal = status.Plan.AcceptanceCount
	}

	d.HasIntake = fileExists(filepath.Join(changeDir, "intake.md"))

	// Config: impact excludes + test paths + linear workspace.
	cfg, _ := config.Load(fabRoot)
	if cfg != nil {
		d.Excludes = cfg.TrueImpactExclude
	}
	d.LinearWorkspace = readLinearWorkspace(fabRoot)

	// Git/gh context (degrades gracefully).
	repoDir := filepath.Dir(fabRoot)
	d.Branch = gitBranch(repoDir)
	d.OwnerRepo = ghOwnerRepo(repoDir)
	if d.OwnerRepo != "" && d.Branch != "" {
		if d.HasIntake {
			d.IntakeURL = blobURL(d.OwnerRepo, d.Branch, folder, "intake.md")
		}
		if d.HasApply {
			d.ApplyURL = blobURL(d.OwnerRepo, d.Branch, folder, applyFile)
		}
	}

	// Impact (degrades gracefully — missing merge-base / failure → no line).
	if base := mergeBase(repoDir); base != "" {
		if res, err := impact.ComputeForRepo(fabRoot, base, "HEAD"); err == nil {
			d.HasImpact = true
			d.Impact = res
		}
	}

	return d, true, nil
}

func progressMap(status *sf.StatusFile) map[string]string {
	m := make(map[string]string, len(pipelineStages))
	for _, stage := range pipelineStages {
		m[stage] = status.GetProgress(stage)
	}
	return m
}

func splitIssues(issues string) []string {
	fields := strings.Fields(issues)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func blobURL(ownerRepo, branch, folder, file string) string {
	return fmt.Sprintf("https://github.com/%s/blob/%s/fab/changes/%s/%s", ownerRepo, branch, folder, file)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// countCheckboxesInTasksSection counts "- [x]" vs "- [ ]" lines within the
// `## Tasks` section of plan.md (content between `## Tasks` and the next `## `
// heading). Returns (done, total).
func countCheckboxesInTasksSection(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	done, total := 0, 0
	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Match the heading exactly — "## Tasks" or "## Tasks ..." (trailing
		// text/whitespace) — but never "## TasksAndStuff". Mirrors the
		// canonical bound in hooklib.HasSectionHeading / scanSectionItems.
		if line == tasksHeading || strings.HasPrefix(line, tasksHeading+" ") {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(line, "## ") {
			break
		}
		if !inSection {
			continue
		}
		d, t := tallyCheckbox(line)
		done += d
		total += t
	}
	return done, total
}

// tasksHeading is the exact `## Tasks` section heading parsed from plan.md.
const tasksHeading = "## Tasks"

// countCheckboxes counts "- [x]" vs "- [ ]" across an entire file (legacy
// tasks.md, which has no `## Tasks` section wrapper). Returns (done, total).
func countCheckboxes(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	done, total := 0, 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		d, t := tallyCheckbox(scanner.Text())
		done += d
		total += t
	}
	return done, total
}

// tallyCheckbox classifies a single line as a checked task, an unchecked task,
// or neither, returning (done, total) contributions (0/0, 0/1, or 1/1).
func tallyCheckbox(line string) (int, int) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]"):
		return 1, 1
	case strings.HasPrefix(trimmed, "- [ ]"):
		return 0, 1
	}
	return 0, 0
}

// readLinearWorkspace reads project.linear_workspace from
// fab/project/config.yaml. The shared internal/config.Config struct does not
// model this nested field (it is prmeta-specific), so it is parsed locally to
// avoid widening the shared config surface. Returns "" when absent.
func readLinearWorkspace(fabRoot string) string {
	data, err := os.ReadFile(filepath.Join(fabRoot, "project", "config.yaml"))
	if err != nil {
		return ""
	}
	var doc struct {
		Project struct {
			LinearWorkspace string `yaml:"linear_workspace"`
		} `yaml:"project"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	return doc.Project.LinearWorkspace
}

// hasConfidenceBlock reports whether .status.yaml actually carries a populated
// `confidence:` mapping. The shared statusfile.Confidence struct uses value
// types, so after Load an absent block is indistinguishable from an all-zero
// one — but the old Step 3c prose renders "—" for an absent block, not
// "0.0/5.0". This local presence check restores that parity without widening
// the shared struct.
//
// Only a non-empty mapping node counts as present: an absent key, an explicit
// `confidence: null`, a bare `confidence:` (both decode to a !!null scalar), an
// empty `confidence: {}`, or a non-mapping scalar all mean "no usable
// confidence data" and render "—". Returns false on any read/parse failure.
func hasConfidenceBlock(statusPath string) bool {
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return false
	}
	var doc struct {
		Confidence yaml.Node `yaml:"confidence"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false
	}
	return doc.Confidence.Kind == yaml.MappingNode && len(doc.Confidence.Content) > 0
}

func gitBranch(repoDir string) string {
	return strings.TrimSpace(runGit(repoDir, "branch", "--show-current"))
}

// mergeBase resolves the merge-base of HEAD against origin/main (falling back to
// origin/master). Returns "" when neither resolves.
func mergeBase(repoDir string) string {
	for _, ref := range []string{"origin/main", "origin/master"} {
		if base := strings.TrimSpace(runGit(repoDir, "merge-base", ref, "HEAD")); base != "" {
			return base
		}
	}
	return ""
}

// ghOwnerRepo returns "owner/repo" via `gh repo view`, or "" on any failure
// (gh missing, not authed, no network) so blob URLs degrade to plain labels.
func ghOwnerRepo(repoDir string) string {
	cmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runGit(repoDir string, args ...string) string {
	cmd := exec.Command("git", args...)
	if repoDir != "" {
		cmd.Dir = repoDir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
