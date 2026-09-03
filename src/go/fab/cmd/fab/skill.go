package main

import (
	_ "embed"
	"io"

	"github.com/spf13/cobra"
)

//go:generate ../../../../../scripts/sync-skill.sh

// skillBundle is the canonical `fab skill` bundle, copied into this package dir
// from docs/site/skill.md by scripts/sync-skill.sh and embedded at build time.
// The fab-go module root is src/go/fab/ and docs/site/ sits above it, so
// //go:embed cannot reach the canonical file directly — the sync step copies it
// here first (see scripts/sync-skill.sh). The committed copy is what a clean
// `go build ./...` compiles; TestSkillEmbedMatchesCanonical keeps it byte-honest
// against docs/site/skill.md on every `go test`. This mirrors the sync +
// drift-guard mechanism `shll standards` established, adapted to a single file.
//
//go:embed skill.md
var skillBundle []byte

// skillCmd builds the `fab skill` subcommand — the toolkit `skill`-standard
// bundle for an agent using an installed fab from any repo. It prints the
// embedded bundle as raw markdown to stdout, byte-identical to the repo's
// canonical docs/site/skill.md, with empty stderr and exit 0. Static-only: the
// bytes never vary with environment or session. The single accepted positional
// is the standard's reserved topic `topics` (a machine affordance, deliberately
// not a cobra child command so it never surfaces in --help or the help-dump
// tree): it enumerates the tool's content topics one per line, and fab ships
// zero topic pages, so it prints nothing and exits 0. Any other positional is a
// usage error classified to exit 2 by main()'s run() helper (no bespoke
// exit-code handling here).
func skillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print the fab agent skill bundle (offline, embedded, static)",
		Long: `Print the fab agent skill bundle to stdout.

A one-page, agent-first usage briefing for operating an installed fab from any
repo: when to reach for fab, a capabilities map keyed to subcommands, how fab
composes with the rest of the toolkit, the stdout/exit-code contracts, and the
non-obvious gotchas. It is the toolkit-wide 'skill' standard bundle.

The bundle is embedded into the binary at build time, so it is offline and
versioned with the release — byte-identical to the repo's canonical
docs/site/skill.md. Raw markdown, no rendering, no pager: an agent consumes the
bytes directly.

Not to be confused with fab's own kit-skills (the /fab-* markdown prompts that
'fab sync' deploys to .claude/skills/) — this command prints one static page.`,
		// Accept zero args (print the bundle) or exactly the reserved topic
		// `topics`; anything else falls through to cobra.NoArgs so the error
		// shape stays cobra's default unknown-argument usage error.
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && args[0] == "topics" {
				return nil
			}
			return cobra.NoArgs(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				// `fab skill topics`: fab ships zero topic pages, so the
				// standard's mandated output is zero bytes on stdout, exit 0.
				return nil
			}
			return runSkill(cmd.OutOrStdout())
		},
	}
}

// runSkill is the implementation seam for `fab skill`, extracted from the cobra
// factory so skill_test.go can drive it with a bytes.Buffer (mirroring shll's
// runStandards). It writes the embedded bundle bytes verbatim to stdout — no
// framing, no trailing newline added — and touches stderr not at all on success.
func runSkill(stdout io.Writer) error {
	n, err := stdout.Write(skillBundle)
	if err != nil {
		return err
	}
	// io.Writer may report a short write (n < len(p)) with a nil error; treat
	// that as a failure so a truncated bundle surfaces loudly instead of exiting 0.
	if n < len(skillBundle) {
		return io.ErrShortWrite
	}
	return nil
}
