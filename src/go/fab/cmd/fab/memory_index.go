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

	cmd := &cobra.Command{
		Use:   "memory-index",
		Short: "Deterministically (re)generate docs/memory index files",
		Long: "Regenerates the root docs/memory/index.md (domains-only), " +
			"every docs/memory/{domain}/index.md (file rows + a Sub-Domains " +
			"reference table when sub-domains exist), and every " +
			"docs/memory/{domain}/{sub-domain}/index.md (file rows) from folder " +
			"contents, reading each file's H1 + `description:` frontmatter and " +
			"stamping \"Last Updated\" from git. Output is byte-stable / " +
			"idempotent across runs, so the indexes stop drifting and stop " +
			"generating merge conflicts. Also emits non-fatal stderr warnings " +
			"when a folder exceeds the soft width bound (~12 files) or depth 3 " +
			"(reserved domains _shared/ and _unsorted/ are width-exempt). With " +
			"--check, writes nothing and classifies drift by severity in the " +
			"exit code: 0 = clean, 1 = benign drift (regen changes content but " +
			"destroys nothing), 2 = destructive loss (regen would wipe a curated " +
			"description, drop a tombstone row, or flatten a custom grouping). " +
			"--json emits the loss report machine-readably (with --check).",
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
						LinkBase: linkBase,
					})
				}
				memExists := func(relPath string) bool {
					_, statErr := os.Stat(filepath.Join(memRoot, filepath.FromSlash(relPath)))
					return statErr == nil
				}
				report := memoryindex.Classify(checkTargets, memExists)
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

	cmd.Flags().BoolVar(&check, "check", false, "Write nothing; encode drift severity in the exit code (0 clean / 1 benign drift / 2 destructive loss)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "With --check, emit the loss report as JSON on stdout (suppresses human-readable text)")
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

// emitCheckReport renders the --check report and maps the severity tier onto
// the process exit code. Tier 0 returns nil (cobra → exit 0); tier 1 returns a
// drift error (cobra → exit 1); tier 2 prints the loss enumeration + pointer
// and os.Exit(2) — main() exits 1 on any returned error, so a non-1 code must
// be set in-handler (the established pane_capture / pane_send pattern for
// genuinely-needed non-1 codes). With --json the report is emitted as a single
// object on stdout and human-readable text is suppressed; the exit dispatch is
// identical so machine consumers branch on the code, not the text.
func emitCheckReport(cmd *cobra.Command, report memoryindex.LossReport, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
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
		return nil // unreachable
	case memoryindex.TierBenignDrift:
		if jsonOut {
			// JSON already emitted to stdout above. Exit 1 directly (mirroring the
			// tier-2 os.Exit pattern) so stdout stays the only output: returning an
			// error here would make main() print "ERROR: ..." to stderr — main()'s
			// unconditional print is not governed by cobra's SilenceErrors.
			os.Exit(1)
			return nil // unreachable
		}
		return fmt.Errorf("memory index out of date — run `fab memory-index`")
	default:
		return nil
	}
}

type indexTarget struct {
	path    string
	content string
}

func rel(repoRoot, path string) string {
	if r, err := filepath.Rel(repoRoot, path); err == nil {
		return filepath.ToSlash(r)
	}
	return path
}
