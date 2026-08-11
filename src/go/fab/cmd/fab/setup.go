package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/setupcheck"
	"github.com/spf13/cobra"
)

// setupCmd is the `fab setup` family seat: the bare command runs the
// interactive setup wizard (see setup_wizard.go), and `check` is the read-only
// doctor. The router needs no allowlist change: `setup` is not a lifecycle
// command, so the shim's default route forwards it to fab-go.
func setupCmd() *cobra.Command {
	var opts wizardOptions
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up and diagnose the fab environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetupWizard(cmd, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.defaults, "defaults", false, "Accept every question's default (the current effective value) non-interactively")
	cmd.Flags().BoolVar(&opts.project, "project", false, "Write to fab/project/config.yaml instead of the system tier (~/.fab-kit/config.yaml)")
	cmd.AddCommand(setupCheckCmd())
	return cmd
}

// setupCheckCmd implements `fab setup check` — the read-only environment
// doctor. HARD INVARIANT: it writes nothing (no config mutation, no trust
// stores, no launches, no prompts), so repeated runs are byte-identical
// (Constitution III). All probing lives in internal/setupcheck; this file owns
// input wiring, rendering, and the exit-code mapping only: any
// failure-severity finding returns an operational error (exit 1 via run()'s
// classification); warnings-only exits 0; usage errors exit 2 at the cobra
// layer before RunE begins.
//
// It coexists with fab-kit's `fab doctor` by distinct job: doctor checks
// system prerequisites (git/bash/yq/…, "good enough to use fab-kit"), while
// this checks fab's own setup state (config viability, provider roster,
// dispatch mode, version skew).
func setupCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Read-only environment doctor — probe providers, config viability, and version skew",
		Args:  cobra.NoArgs,
		Example: "  fab setup check\n" +
			"  fab setup check || echo 'real problems found'  # CI gate: exit 1 only on failures",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := setupcheck.Run(setupCheckInput())
			renderSetupCheck(cmd.OutOrStdout(), report)
			if report.HasFailures() {
				return fmt.Errorf("setup check found %d problem(s) — see findings above", report.Failures())
			}
			return nil
		},
	}
}

// setupCheckInput wires the live environment into the probe layer. A missing
// fab/ directory is NOT an error — the doctor degrades to the system+env
// config tiers (a pre-init machine is exactly where a doctor must still run).
func setupCheckInput() setupcheck.Input {
	in := setupcheck.Input{
		BinaryVersion: version,
		TmuxEnv:       os.Getenv("TMUX"),
	}
	if fabRoot, err := resolve.FabRoot(); err == nil {
		in.ProjectConfigPath = fabRoot + "/project/config.yaml"
		if cfg, err := readProjectPin(fabRoot); err == nil {
			in.ProjectPin = cfg
		}
	}
	return in
}

// readProjectPin reads fab/.fab-version (the sole project version source,
// 260708-j0qm) without pulling in the full config load — the probe layer
// already loads the config itself.
func readProjectPin(fabRoot string) (string, error) {
	data, err := os.ReadFile(fabRoot + "/.fab-version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// renderSetupCheck writes the report: the provider roster as a table, then the
// findings in probe order, each prefixed with its severity marker (✓ ok /
// ! warning / i info / ✗ failure), then a summary line. All of it goes to stdout — it IS the data the caller
// asked for (toolkit principle №2); stderr carries only the run() ERROR line
// when the exit is non-zero.
func renderSetupCheck(w io.Writer, report *setupcheck.Report) {
	fmt.Fprintln(w, "fab setup check — environment doctor (read-only)")
	fmt.Fprintln(w)

	if len(report.Providers) > 0 {
		fmt.Fprintln(w, "Providers:")
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "  provider\tbinary\tinteractive\theadless\tnative")
		for _, p := range report.Providers {
			fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\t%s\n",
				p.Name, renderBinary(p), yesNo(p.Interactive), yesNo(p.Headless), yesNo(p.Native))
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Checks:")
	for _, f := range report.Findings {
		fmt.Fprintf(w, "  %s %s\n", severityMarker(f.Severity), f.Detail)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Summary: %d failure(s), %d warning(s), %d finding(s) total.\n",
		report.Failures(), report.Warnings(), len(report.Findings))
}

// renderBinary renders the roster's binary cell: the resolved path summary,
// or the missing executables.
func renderBinary(p setupcheck.ProviderProbe) string {
	if len(p.Executables) == 0 {
		return "(no commands)"
	}
	if p.Found() {
		return strings.Join(p.Executables, ", ")
	}
	return strings.Join(p.Missing, ", ") + " — not found"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// severityMarker maps a finding severity to its output marker, mirroring
// fab-kit doctor's ✓/✗ vocabulary extended with warning and info tiers.
func severityMarker(s setupcheck.Severity) string {
	switch s {
	case setupcheck.OK:
		return "✓"
	case setupcheck.Info:
		return "i"
	case setupcheck.Warn:
		return "!"
	case setupcheck.Fail:
		return "✗"
	default:
		return "?"
	}
}
