package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/memoryindex"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func memoryIndexCmd() *cobra.Command {
	var check bool
	var jsonOut bool
	var rebuild bool

	cmd := &cobra.Command{
		Use:   "memory-index",
		Short: "Deterministically (re)generate docs/memory index files",
		Long: "Regenerates the root docs/memory/index.md (domains-only, with the " +
			"FKF fkf_version: \"0.1\" frontmatter), every docs/memory/{domain}/index.md " +
			"(file rows + a Sub-Domains reference table when sub-domains exist), and " +
			"every docs/memory/{domain}/{sub-domain}/index.md (file rows) from folder " +
			"contents, reading each file's H1 + `description:` frontmatter. The index " +
			"is a pure function of content (no git dates), so its output is " +
			"branch-independent and idempotent. It also emits a per-folder FKF " +
			"log.md (C-lite change history: one batched git-log pass joined with each " +
			"change's .status.yaml summary, change-id recovered from the git history " +
			"and gated against the fab/changes registry; unattributable commits " +
			"degrade gracefully). log.md uses FREEZE-ON-WRITE generation: the existing " +
			"log.md is authoritative and write-once — regeneration reads it back and " +
			"APPENDS only new entries keyed on (file-base, change-id); existing entries " +
			"are never reworded or re-dated, and a NEW unattributable commit (no " +
			"registry change-id — a migration, a direct-main edit) is NOT projected " +
			"after first write (frozen, not re-projected), so a squash + branch-delete " +
			"that rewrites history no longer churns the log. Output is byte-stable / " +
			"idempotent across runs, so the indexes and logs stop drifting and stop " +
			"generating merge conflicts. Also emits non-fatal stderr warnings when a " +
			"folder exceeds the soft width bound (~12 files) or depth 3 (reserved " +
			"domains _shared/ and _unsorted/ are width-exempt). --rebuild is the " +
			"destructive escape hatch: it discards the frozen state and re-projects " +
			"every log.md from current git (the pre-freeze behavior, opt-in) — for a " +
			"corrupted log or a deliberate re-baseline. Also emits non-fatal stderr " +
			"warnings, split into a BLOCKING class (fails --check) and an ADVISORY " +
			"class (never affects the exit code). BLOCKING: malformed frontmatter (an " +
			"unclosed `---` block or a `description:` value that fails quote-stripping " +
			"— e.g. a glued closing fence), a `description:` carrying a registry-gated " +
			"change-id (the FKF §3.2 ban, enforced), and a `description:` over 1000 " +
			"runes (2× the 500 soft cap — gross over-cap). ADVISORY: an over-long " +
			"`description:` in the 501–1000 range (trim nag), per-topic-file " +
			"narration-marker density (transition stems + registry-gated change-id tokens, at ≥5 " +
			"— the distillation-debt meter), per-topic-file size (>400 lines or >15KB), " +
			"a non-empty _unsorted/ staging folder, and broken bundle-relative " +
			"memory↔memory links. With --check, writes nothing and classifies index " +
			"drift by severity in the exit code: 0 = clean, 1 = benign drift (regen " +
			"changes content but destroys nothing — e.g. an improved `description:`, or " +
			"any log.md / FKF frontmatter drift; for log.md a benign FAIL means the " +
			"committed log is missing a projected attributable (file-base, change-id) " +
			"entry, or a frozen line was hand-edited render-unstably — a committed log " +
			"that is a valid SUPERSET of the freeze-on-write merge PASSES), 2 = " +
			"destructive loss (regen would wipe a curated description, drop a tombstone " +
			"row, or flatten a custom grouping — index-only categories). The BLOCKING " +
			"class is a separate signal from the drift tier: any blocking finding " +
			"FLOORS the --check exit at 1 (enumerating the offending file(s) with a " +
			"fix-the-file pointer) even when index drift is clean (tier 0). It is NOT a " +
			"tier-2 destructive-loss category, so the hydrate/reorg refuse-before-regen " +
			"guards (which fire only on exit 2) are unaffected; exit 2 still wins when a " +
			"tier-2 loss co-occurs. The ADVISORY warnings never fail --check " +
			"(blocking blocks, advisory nags). --json emits the loss report " +
			"machine-readably (with --check), including an additive `malformed` array " +
			"(blocking findings) and an additive `warnings` array (advisory findings) " +
			"alongside the unchanged `tier`/`drift`/`losses` keys.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fabRoot, err := resolve.FabRoot()
			if err != nil {
				return err
			}
			repoRoot := filepath.Dir(fabRoot)

			root, domains, warnings, err := memoryindex.Gather(repoRoot)
			if err != nil {
				return err
			}

			// Warnings are advisory — always to stderr, never fatal, never
			// affecting the written index output.
			for _, w := range warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), w.String())
			}

			memRoot := filepath.Join(repoRoot, "docs", "memory")
			targets := make([]indexTarget, 0, len(domains)+1)
			targets = append(targets, indexTarget{
				path:    filepath.Join(memRoot, "index.md"),
				content: memoryindex.RenderRoot(root),
			})
			for _, d := range domains {
				targets = append(targets, indexTarget{
					path:    filepath.Join(memRoot, d.Name, "index.md"),
					content: memoryindex.RenderDomain(d),
				})
				// Each sub-domain gets its own generated index one level down.
				for _, sd := range d.SubDomains {
					targets = append(targets, indexTarget{
						path:    filepath.Join(memRoot, d.Name, sd.Name, "index.md"),
						content: memoryindex.RenderDomain(sd),
					})
				}
			}

			// FKF per-folder log.md targets (C-lite — git history + per-change
			// summaries). Gathered from the SAME batched git pass + the change
			// registry. When git history is unavailable the git-projection surface
			// degrades to empty, but existing frozen log.md and/or log.seed.md
			// entries still emit targets (freeze-on-write — GatherLogs nets to no
			// targets only when no folder has any frozen/seed/git entry).
			// They flow through the same byte-stable write / --check loops as the
			// indexes, but are classified as benign-drift-only (isLog) so the
			// index-row loss detectors never false-positive on log list content.
			//
			// Freeze-on-write (R6): a write run honors --rebuild (re-project
			// destructively when set). A --check run always uses rebuild=false so the
			// rendered content is the freeze-on-write merge the classifier compares
			// against (R7–R9) — --check never re-projects, so --check --rebuild would
			// be meaningless and is treated as a plain --check.
			logTargets, err := memoryindex.GatherLogs(repoRoot, fabRoot, rebuild && !check)
			if err != nil {
				return err
			}
			for _, lt := range logTargets {
				targets = append(targets, indexTarget{path: lt.Path, content: lt.Content, isLog: true})
			}

			if check {
				// Build the classifier inputs from the same targets the write
				// path uses — reusing the rendered-vs-existing comparison, never
				// duplicating it. LinkBase is the index file's directory relative
				// to docs/memory/ (""/<domain>/<domain>/<sub>); memExists checks
				// a docs/memory/-relative path on disk for tombstone detection.
				checkTargets := make([]memoryindex.CheckTarget, 0, len(targets))
				for _, t := range targets {
					existing, _ := os.ReadFile(t.path)
					linkBase := filepath.ToSlash(filepath.Dir(rel(memRoot, t.path)))
					if linkBase == "." {
						linkBase = ""
					}
					checkTargets = append(checkTargets, memoryindex.CheckTarget{
						Path:     rel(repoRoot, t.path),
						Existing: string(existing),
						Rendered: t.content,
						IsRoot:   t.path == filepath.Join(memRoot, "index.md"),
						IsLog:    t.isLog,
						LinkBase: linkBase,
					})
				}
				memExists := func(relPath string) bool {
					_, statErr := os.Stat(filepath.Join(memRoot, filepath.FromSlash(relPath)))
					return statErr == nil
				}
				report := memoryindex.Classify(checkTargets, memExists)
				// Feed the gathered warnings into the report, split by class:
				//   - BLOCKING findings (malformed frontmatter + the two
				//     description escalations) → report.Malformed, so `--check`
				//     blocks on them independent of index drift (the loom case is
				//     byte-clean drift but corrupt/over-cap/change-id-laden source).
				//   - ADVISORY findings (density / size / _unsorted / broken links)
				//     → report.Warnings, the additive machine surface — never
				//     affecting the exit code.
				for _, w := range warnings {
					if w.IsBlocking() {
						report.Malformed = append(report.Malformed, memoryindex.MalformedFinding{
							Kind:   w.Kind,
							Path:   w.Path,
							Detail: w.Detail,
						})
						continue
					}
					// Advisory kinds carried on the JSON `warnings` array. Width
					// and depth are advisory shape bounds that predate the machine
					// surface and are stderr-only, so they are NOT emitted here —
					// only the mxgu debt-meter kinds join the array.
					switch w.Kind {
					case memoryindex.KindNarrationDensity, memoryindex.KindFileSize,
						memoryindex.KindUnsorted, memoryindex.KindBrokenLink:
						report.Warnings = append(report.Warnings, memoryindex.WarningFinding{
							Kind:   w.Kind,
							Path:   w.Path,
							Count:  w.Count,
							Bytes:  w.Bytes,
							Detail: w.Detail,
						})
					}
				}
				return emitCheckReport(cmd, report, jsonOut)
			}

			written := 0
			for _, t := range targets {
				existing, _ := os.ReadFile(t.path)
				if string(existing) == t.content {
					continue // byte-stable — skip the write so mtime/no-op diff stays clean
				}
				if err := os.WriteFile(t.path, []byte(t.content), 0o644); err != nil {
					return fmt.Errorf("writing %s: %w", t.path, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Updated: %s\n", rel(repoRoot, t.path))
				written++
			}
			if written == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Memory indexes already up to date.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "Write nothing; encode index-drift severity in the exit code (0 clean / 1 benign drift / 2 destructive loss). Blocking source findings (malformed frontmatter, a change-id in `description:`, or a `description:` over 1000 runes) floor the exit at 1 independent of drift")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "With --check, emit the loss report as JSON on stdout (suppresses human-readable text); includes an additive `malformed` array (blocking findings) and an additive `warnings` array carrying the four advisory kinds — narration-density, file-size, unsorted-nonempty, broken-link (the 501–1000 description-length nag is deliberately excluded from the array)")
	cmd.Flags().BoolVar(&rebuild, "rebuild", false, "DESTRUCTIVE: discard the frozen log.md state and re-project every log.md from current git (the pre-freeze behavior, opt-in). Ignored with --check (which never writes)")
	return cmd
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
// pane_capture / pane_send pattern). With --json the report is emitted as a
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
			fmt.Fprintln(err, "destructive loss — regenerating would wipe curated/historical content:")
			for _, l := range report.Losses {
				fmt.Fprintf(err, "  [%s] %s: %s\n", l.Category, l.Path, l.Detail)
			}
			fmt.Fprintln(err, remediationPointer)
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
			return fmt.Errorf("memory index out of date and %s — regenerate, then fix the file(s) above and re-run `fab memory-index`", blockingLabel)
		}
		return fmt.Errorf("memory index out of date — run `fab memory-index`")
	default:
		// Tier 0 (no index drift). If there is a blocking finding, block anyway —
		// the whole point is that source problems must FAIL --check independent of
		// drift (the loom case: committed garbage == regenerated garbage, tier 0).
		if hasMalformed {
			if jsonOut {
				os.Exit(1)
				return nil // unreachable
			}
			return fmt.Errorf("%s — fix the file(s) above and re-run `fab memory-index`", blockingLabel)
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

type indexTarget struct {
	path    string
	content string
	isLog   bool // a log.md target — benign-drift-only (no destructive-loss detectors)
}

func rel(repoRoot, path string) string {
	if r, err := filepath.Rel(repoRoot, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}
