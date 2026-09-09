package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/memoryindex"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func docsIndexCmd() *cobra.Command   { return newDocsIndexCmd(false) }
func memoryIndexCmd() *cobra.Command { return newDocsIndexCmd(true) }

func newDocsIndexCmd(alias bool) *cobra.Command {
	var check, jsonOut, rebuild bool
	cmd := &cobra.Command{
		Use:   "docs-index [root-path]",
		Short: "Deterministically regenerate configured documentation indexes",
		Long: `Regenerates every docs_index.roots entry, or one configured positional root.
Without configuration, processes docs/memory with index.md, log:true, max_depth:3.
Each root supports path, index_file (index.md), also_accept ([]), log (false),
max_depth (3), superseded ([] glob patterns), exclude ([] glob patterns — files
or folders the walker never indexes: the index is the tree minus these), and
nav_note ('' — the root landing's navigation note, rendered verbatim when set).
Every regular file under a root is a row (only landings, log files, dotfiles,
symlinks, and exclude matches are skipped); markdown rows read description:
frontmatter, HTML rows read <title> (label) and <meta name="description">
(description — a title is never a description), and every other type lists its
filename with —. Traversal has arbitrary depth;
max_depth and missing descriptions are advisory warnings, never hard failures.
A missing description uses the file H1 and —, never invented text. The legacy
memory format retains filename-stem labels to preserve zero-config bytes.

Primary index_file landings are generated whole files carrying one hand-managed
manual block (preserved verbatim on regeneration) for rows the generator cannot
produce; first-run hand-written navigation is seed-imported into it. An existing
also_accept landing (e.g. README.md) receives only a marker-delimited generated
block; prose outside it stays human-owned and byte-preserved. Generated
files/blocks are tool-owned. Output is content-derived, byte-stable and
idempotent, with no git dates in indexes.

Superseded globs (e.g. **/archive/**) produce one parent pointer/count and one
index of child versions; no topic descriptions are read inside superseded trees.
Obsolete generated descendant indexes and logs are removed when a live subtree
becomes superseded; alternate landings retain prose outside their removed blocks.
This explicit retirement is benign drift under --check. Seeds stay unchanged.
File matches fold into their folder's superseded count. Escape literal brackets
in glob patterns (e.g. **/\[archived\]-*); Z-prefix folders can use **/Z*/**.

Only log:true roots use FKF metadata, reserved domains and freeze-on-write log.md:
existing entries stay authoritative, new (file-base, change-id) entries append,
and log.seed.md merges beneath. --rebuild discards frozen logs and re-projects
from git; it is ignored with --check and irrelevant for log:false roots.

--check writes nothing: 0 clean, 1 benign drift, 2 destructive index loss
(description wipe, tombstone drop, custom grouping flatten). Worst root
wins. Blocking malformed frontmatter floors the exit at 1; FKF change-id and
>1000-rune description escalations apply only to log:true roots. Shape, size,
missing-description, narration-density (FKF roots), and length advisories never fail a clean check. --json keeps
tier/drift/losses/malformed/warnings and adds warnings_total (all JSON advisories
before sampling). Advisory details are capped at 5 per kind across all selected
roots on stderr and in JSON; stderr states how many were omitted. Width/depth
remain stderr-only and are excluded from warnings_total. Blocking findings remain
complete. Refuse-before-regen guards key on exit 2.`,
		Example: "  fab docs-index\n  fab docs-index docs/specs --check --json\n  fab docs-index docs/memory --rebuild",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if alias {
				fmt.Fprintln(cmd.ErrOrStderr(), "fab memory-index is deprecated; use fab docs-index docs/memory (alias retained for at least one minor version).")
			}
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			cfg, err := config.Load(fabRoot)
			if err != nil {
				return err
			}
			roots, err := cfg.GetDocsIndexRoots()
			if err != nil {
				return err
			}
			selected := ""
			if len(args) > 0 {
				selected = filepath.ToSlash(filepath.Clean(args[0]))
			}
			if alias {
				selected = "docs/memory"
			}
			if selected != "" {
				var matches []config.DocsIndexRoot
				for _, r := range roots {
					if r.Path == selected {
						matches = append(matches, r)
					}
				}
				if len(matches) == 0 && alias {
					matches = config.DefaultDocsIndexRoots()
				}
				if len(matches) == 0 {
					return fmt.Errorf("root %q is not configured; add it to docs_index.roots", selected)
				}
				roots = matches
			}
			return runDocsIndex(cmd, filepath.Dir(fabRoot), fabRoot, roots, check, jsonOut, rebuild)
		},
	}
	if alias {
		cmd.Use = "memory-index"
		cmd.Short = "Deprecated alias for docs-index docs/memory"
		cmd.Args = cobra.NoArgs
	}
	cmd.Flags().BoolVar(&check, "check", false, "Write nothing; exit 0 clean / 1 benign drift or blocking findings / 2 destructive loss")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "With --check, emit tier/drift/losses/malformed/warnings as JSON")
	cmd.Flags().BoolVar(&rebuild, "rebuild", false, "DESTRUCTIVE: rebuild frozen logs for log:true roots; ignored with --check")
	return cmd
}

func runDocsIndex(cmd *cobra.Command, repo, fabRoot string, roots []config.DocsIndexRoot, check, jsonOut, rebuild bool) error {
	report := memoryindex.Classify(nil, nil)
	var all []memoryindex.Target
	var allWarnings []memoryindex.Warning
	for _, root := range roots {
		targets, warnings, err := memoryindex.GatherRoot(repo, fabRoot, root, rebuild && !check)
		if err != nil {
			return err
		}
		all = append(all, targets...)
		allWarnings = append(allWarnings, warnings...)
		r := classifyDocsRoot(repo, root, targets)
		if r.Tier > report.Tier {
			report.Tier = r.Tier
		}
		report.Drift = report.Drift || r.Drift
		report.Losses = append(report.Losses, r.Losses...)
	}
	reportDocsWarnings(cmd, &report, allWarnings)
	if check {
		return emitCheckReport(cmd, report, jsonOut)
	}
	written := 0
	for _, t := range all {
		existing, err := os.ReadFile(t.Path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if t.Remove {
			if err := os.Remove(t.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing %s: %w", t.Path, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed: %s\n", rel(repo, t.Path))
			written++
			continue
		}
		if string(existing) == t.Content {
			continue
		}
		if err := os.WriteFile(t.Path, []byte(t.Content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", t.Path, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Updated: %s\n", rel(repo, t.Path))
		written++
	}
	if written == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Documentation indexes already up to date.")
	}
	return nil
}

// Advisory samples are capped per kind across the entire invocation, rather
// than resetting at each root. Gathering stays complete for rendering and checks.
const docsWarningLimit = 5

func reportDocsWarnings(cmd *cobra.Command, report *memoryindex.LossReport, warnings []memoryindex.Warning) {
	counts := map[string]int{}
	for _, w := range warnings {
		if w.IsBlocking() {
			fmt.Fprintln(cmd.ErrOrStderr(), w.String())
			report.Malformed = append(report.Malformed, memoryindex.MalformedFinding{Kind: w.Kind, Path: w.Path, Detail: w.Detail})
			continue
		}
		counts[w.Kind]++
		if counts[w.Kind] <= docsWarningLimit {
			fmt.Fprintln(cmd.ErrOrStderr(), w.String())
		}
		if w.Kind == memoryindex.KindWidth || w.Kind == memoryindex.KindDepth {
			continue // Shape warnings retain their stderr-only contract.
		}
		report.WarningsTotal++
		if counts[w.Kind] <= docsWarningLimit {
			report.Warnings = append(report.Warnings, memoryindex.WarningFinding{Kind: w.Kind, Path: w.Path, Count: w.Count, Bytes: w.Bytes, Detail: w.Detail})
		}
	}
	var truncated []string
	for kind, count := range counts {
		if count > docsWarningLimit {
			truncated = append(truncated, kind)
		}
	}
	sort.Strings(truncated)
	for _, kind := range truncated {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: [%s] … and %d more (%d total)\n", kind, counts[kind]-docsWarningLimit, counts[kind])
	}
}

func classifyDocsRoot(repo string, root config.DocsIndexRoot, targets []memoryindex.Target) memoryindex.LossReport {
	base := filepath.Join(repo, filepath.FromSlash(root.Path))
	inputs := make([]memoryindex.CheckTarget, 0, len(targets))
	cleanupDrift := false
	for _, t := range targets {
		existing, _ := os.ReadFile(t.Path)
		if t.SupersededCleanup {
			cleanupDrift = cleanupDrift || t.Remove || string(existing) != t.Content
			continue // Explicit superseded configuration retires only tool-owned output.
		}
		linkBase := filepath.ToSlash(filepath.Dir(rel(base, t.Path)))
		if linkBase == "." {
			linkBase = ""
		}
		inputs = append(inputs, memoryindex.CheckTarget{Path: rel(repo, t.Path), Existing: string(existing), Rendered: t.Content, IsRoot: t.IsRoot, IsLog: t.IsLog, LinkBase: linkBase})
	}
	report := memoryindex.Classify(inputs, func(p string) bool { _, err := os.Stat(filepath.Join(base, filepath.FromSlash(p))); return err == nil })
	if cleanupDrift {
		report.Drift = true
		if report.Tier < memoryindex.TierBenignDrift {
			report.Tier = memoryindex.TierBenignDrift
		}
	}
	return report
}

// remediationPointer is the remediation pointer appended to the tier-2 human
// report — the refuse-before-regen escape hatch for a pre-fab-kit tree. It
// names /docs-reorg-memory, the orchestrator that handles all three tier-2
// categories: it relocates removal-history (tombstone) rows itself and
// dispatches /docs-hydrate-memory backfill mode for description: frontmatter
// (backfill alone does NOT relocate tombstones — that is reorg's job).
const remediationPointer = "→ run /docs-reorg-memory to remediate (it relocates removal-history rows " +
	"to _shared/removed-domains.md and backfills description: frontmatter via " +
	"/docs-hydrate-memory) before regenerating."

// malformedRemediation is the fix-the-file pointer appended to the blocking-
// finding enumeration. Distinct from remediationPointer (the destructive-loss
// /docs-reorg-memory pointer): the blocking findings are SOURCE-file problems,
// not index-target losses, fixed by editing the offending file — restoring the
// closing `---` / matching quotes (malformed frontmatter), or trimming the
// `description:` and moving change-id citations to the body (the §3.2
// escalations) — not by a reorg.
const malformedRemediation = "→ fix the file(s) above — restore the closing `---` and matching " +
	"quotes on a malformed `description:`, or trim an over-cap `description:` and move change-id " +
	"citations to the body (FKF §3.2) — before regenerating."

// emitCheckReport renders the --check report and maps its findings onto the
// process exit code. INDEX-DRIFT tiers: tier 0 → exit 0; tier 1 → exit 1 (drift
// error); tier 2 → exit 2 (loss enumeration + /docs-reorg-memory pointer).
// MALFORMED frontmatter (source corruption, report.Malformed) is a SEPARATE
// blocking signal orthogonal to the drift tier: any malformed finding FLOORS the
// exit at 1 even when the tier is 0 (the loom case: byte-clean drift, corrupt
// source), enumerating the offending file(s) to stderr with a fix-the-file
// pointer. Exit precedence: tier 2 (exit 2) still wins over a malformed floor,
// but the malformed files are enumerated in either case so they are never
// silently swallowed by a co-occurring loss. main() exits 1 on any returned
// error, so a non-1 code must be set in-handler via os.Exit (the established
// pane_capture pattern). With --json the report is emitted as a
// single object on stdout and human-readable text is suppressed; the exit
// dispatch is identical so machine consumers branch on the code, not the text.
func emitCheckReport(cmd *cobra.Command, report memoryindex.LossReport, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	}

	hasMalformed := len(report.Malformed) > 0
	// blockingLabel names the blocking class in stderr/error text. Malformed
	// frontmatter is still a member, so when any malformed-frontmatter kind is
	// present the label leads with "malformed frontmatter" (keeping that literal
	// phrase in the output). When the only blocking findings are the description
	// escalations (registry-gated change-id / gross over-cap), the label reads
	// "blocking `description:` findings" — accurate for those kinds, which are
	// not frontmatter corruption.
	blockingLabel := "blocking `description:` findings"
	if reportHasMalformedFrontmatter(report) {
		blockingLabel = "malformed frontmatter"
	}
	if hasMalformed && !jsonOut {
		err := cmd.ErrOrStderr()
		fmt.Fprintf(err, "%s — regeneration would propagate the offending value(s) into the index:\n", blockingLabel)
		for _, m := range report.Malformed {
			fmt.Fprintf(err, "  [%s] %s", m.Kind, m.Path)
			if m.Detail != "" {
				fmt.Fprintf(err, ": %s", m.Detail)
			}
			fmt.Fprintln(err)
		}
		fmt.Fprintln(err, malformedRemediation)
	}

	switch report.Tier {
	case memoryindex.TierDestructiveLoss:
		if !jsonOut {
			err := cmd.ErrOrStderr()
			fmt.Fprintln(err, "destructive loss — regenerating would wipe hand-managed/historical content:")
			for _, l := range report.Losses {
				fmt.Fprintf(err, "  [%s] %s: %s\n", l.Category, l.Path, l.Detail)
			}
			fmt.Fprintln(err, lossRemediation(report))
		}
		os.Exit(2)
		return nil // unreachable — tier 2 wins over the malformed floor
	case memoryindex.TierBenignDrift:
		if jsonOut {
			// JSON already emitted to stdout above. Exit 1 directly (mirroring the
			// tier-2 os.Exit pattern) so stdout stays the only output: returning an
			// error here would make main() print "ERROR: ..." to stderr — main()'s
			// unconditional print is not governed by cobra's SilenceErrors.
			os.Exit(1)
			return nil // unreachable
		}
		// A blocking floor can co-occur with benign drift; the enumeration is
		// already on stderr above, but the RETURNED error must also name the
		// blocking finding so callers surfacing only the error text are not misled
		// into treating it as mere staleness (mirrors the tier-0 blocking branch).
		if hasMalformed {
			return fmt.Errorf("documentation index out of date and %s — regenerate, then fix the file(s) above and re-run `fab docs-index`", blockingLabel)
		}
		return fmt.Errorf("documentation index out of date — run `fab docs-index`")
	default:
		// Tier 0 (no index drift). If there is a blocking finding, block anyway —
		// the whole point is that source problems must FAIL --check independent of
		// drift (the loom case: committed garbage == regenerated garbage, tier 0).
		if hasMalformed {
			if jsonOut {
				os.Exit(1)
				return nil // unreachable
			}
			return fmt.Errorf("%s — fix the file(s) above and re-run `fab docs-index`", blockingLabel)
		}
		return nil
	}
}

// reportHasMalformedFrontmatter reports whether any of the report's blocking
// findings is a malformed-frontmatter kind (as opposed to a description
// escalation). It selects the stderr/error label so genuine corruption still
// leads with the "malformed frontmatter" phrase while an escalation-only run
// reads accurately.
func reportHasMalformedFrontmatter(report memoryindex.LossReport) bool {
	for _, m := range report.Malformed {
		if m.Kind == memoryindex.KindMalformedFence || m.Kind == memoryindex.KindMalformedDescription {
			return true
		}
	}
	return false
}

func rel(repoRoot, path string) string {
	if r, err := filepath.Rel(repoRoot, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}

// Keep the memory remediation contract, while pointing generic-root callers at
// their own navigation instead of unrelated memory files.
func lossRemediation(report memoryindex.LossReport) string {
	for _, loss := range report.Losses {
		if !strings.HasPrefix(loss.Path, "docs/memory/") {
			return "→ preserve the reported descriptions, historical rows, and grouping in their configured root; re-run `fab docs-index <root-path> --check` before regenerating."
		}
	}
	return remediationPointer
}
