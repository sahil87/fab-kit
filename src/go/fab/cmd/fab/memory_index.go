package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sahil87/fab-kit/src/go/fab/internal/memoryindex"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func memoryIndexCmd() *cobra.Command {
	var check bool

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
			"--check, writes nothing and exits non-zero if regeneration would " +
			"change any index file.",
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
				drift := 0
				for _, t := range targets {
					existing, _ := os.ReadFile(t.path)
					if string(existing) != t.content {
						fmt.Fprintf(cmd.ErrOrStderr(), "out of date: %s\n", rel(repoRoot, t.path))
						drift++
					}
				}
				if drift > 0 {
					return fmt.Errorf("%d memory index file(s) out of date — run `fab memory-index`", drift)
				}
				return nil
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

	cmd.Flags().BoolVar(&check, "check", false, "Exit non-zero (writing nothing) if any index file is out of date")
	return cmd
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
