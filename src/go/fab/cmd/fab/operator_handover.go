package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Operator hand-over to HexoKit. When the installed rk carries the operator
// loop (`rk operator check` + `rk operator brief`), rk owns the operator: the
// `fab operator` verbs print one deprecation line and delegate to their rk
// counterparts, and fab stops writing rk's cron file. Without such an rk every
// verb behaves exactly as before — the fallback arm is untouched.

const (
	// rkBin is the rk executable name; rkOperatorVerb/rkTrackVerb/rkCronVerb
	// are the verb tokens of the delegated argv.
	rkBin          = "rk"
	rkOperatorVerb = "operator"
	rkTrackVerb    = "track"
	rkCronVerb     = "cron"
	// rkCheckVerb and rkBriefVerb are the command rows `rk operator --help`
	// must list for the hand-over to be live.
	rkCheckVerb = "check"
	rkBriefVerb = "brief"
)

// rkHelpHasHandover is the pure help-text matcher behind the probe: true only
// when the help's command listing carries a row for BOTH `check` and `brief`.
// A row is a line whose first whitespace-delimited token is the subcommand
// name — a mid-sentence mention is not a row.
func rkHelpHasHandover(help string) bool {
	var check, brief bool
	for _, line := range strings.Split(help, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case rkCheckVerb:
			check = true
		case rkBriefVerb:
			brief = true
		}
	}
	return check && brief
}

var (
	handoverOnce   sync.Once
	handoverResult bool
)

// rkOperatorHandover is the injectable hand-over capability probe (the
// rkOperatorPath precedent): side-effect-free (`--help` opens nothing),
// LookPath-gated, bounded by rkCronTimeout, fail-silent, and memoized per
// process. Package-level var so tests stub the seam (TestMain defaults it to
// false).
var rkOperatorHandover = probeRkOperatorHandover

// probeRkOperatorHandover is the real, memoized probe behind the seam.
func probeRkOperatorHandover() bool {
	handoverOnce.Do(func() {
		if _, err := exec.LookPath(rkBin); err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), rkCronTimeout)
		defer cancel()
		out, _, err := pane.RunCmdContext(ctx, rkBin, rkOperatorVerb, "--help")
		handoverResult = err == nil && rkHelpHasHandover(out)
	})
	return handoverResult
}

// exitCodeError carries a delegated rk process's exit code up to run(), which
// returns it as fab's own exit code without printing anything (rk's stdio is
// inherited, so it has already spoken).
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("rk exited with code %d", e.code) }

// rkDelegateRunner is the injectable seam for delegated rk invocations: argv
// after "rk", stdin/stdout/stderr inherited, returning rk's exit code. err is
// non-nil only when rk could not be started (code is then meaningless).
var rkDelegateRunner = func(args ...string) (int, error) {
	c := exec.Command(rkBin, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}

// runRkDelegate runs one rk invocation through the seam and maps the outcome
// to fab's: a start failure is an operational error (exit 1), a non-zero rk
// exit becomes the same fab exit code.
func runRkDelegate(args ...string) error {
	code, err := rkDelegateRunner(args...)
	if err != nil {
		return fmt.Errorf("cannot run rk %s: %w", strings.Join(args, " "), err)
	}
	if code != 0 {
		return &exitCodeError{code: code}
	}
	return nil
}

// handoverOr returns a RunE that delegates when the hand-over probe is live
// and otherwise runs the unchanged fallback — the delegation arm sits ahead of
// every fallback so the fallback body never sees the capable-rk world.
func handoverOr(delegate, fallback func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if rkOperatorHandover() {
			return delegate(cmd, args)
		}
		return fallback(cmd, args)
	}
}

// operatorReplacement names the rk command replacing a fab operator verb
// (keyed by the verb path under `fab operator`); verbs with no rk
// counterpart map to "".
func operatorReplacement(verb string) string {
	switch {
	case verb == "tick-start":
		return "rk operator check"
	case verb == "clock sync":
		return "rk cron list"
	case verb == "track clock":
		return "rk cron edit"
	case strings.HasPrefix(verb, "track "):
		return "rk operator " + verb
	}
	return ""
}

// printOperatorDeprecation writes the one-line deprecation notice for a verb.
func printOperatorDeprecation(w io.Writer, verb string) {
	if repl := operatorReplacement(verb); repl != "" {
		fmt.Fprintf(w, "fab operator %s is deprecated — delegating to '%s' (HexoKit now owns the operator)\n", verb, repl)
		return
	}
	fmt.Fprintf(w, "fab operator %s is deprecated — HexoKit now owns the operator ('rk operator'); no rk counterpart\n", verb)
}

// operatorDeprecationHook is the PersistentPreRunE on `fab operator`: under a
// capable rk every subcommand prints the deprecation line to stderr before
// doing anything else. The bare launcher (already delegating via
// rkOperatorPath) is exempt.
func operatorDeprecationHook(cmd *cobra.Command, args []string) error {
	if cmd.Name() == rkOperatorVerb || !rkOperatorHandover() {
		return nil
	}
	printOperatorDeprecation(cmd.ErrOrStderr(), strings.TrimPrefix(cmd.CommandPath(), "fab operator "))
	return nil
}

// rkArgvFromChangedFlags rebuilds an rk flag argv from the flags the user
// actually set (cobra-parsed, so fab's own validation already ran): repeatable
// stringArray flags repeat, stringSlice flags join with commas. Flags named in
// skip are omitted.
func rkArgvFromChangedFlags(cmd *cobra.Command, skip ...string) []string {
	var argv []string
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name == "help" || stringSliceContains(skip, f.Name) {
			return
		}
		flag := "--" + f.Name
		switch f.Value.Type() {
		case "stringArray":
			for _, v := range f.Value.(pflag.SliceValue).GetSlice() {
				argv = append(argv, flag, v)
			}
		case "stringSlice":
			argv = append(argv, flag, strings.Join(f.Value.(pflag.SliceValue).GetSlice(), ","))
		case "bool":
			if f.Value.String() == "true" {
				argv = append(argv, flag)
			} else {
				argv = append(argv, flag+"=false")
			}
		default:
			argv = append(argv, flag, f.Value.String())
		}
	})
	return argv
}

// delegateTrackVerb builds the delegating RunE for a track verb whose rk
// counterpart takes the same positional args and flags (add, list, rm,
// observe).
func delegateTrackVerb(verb string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		argv := append([]string{rkOperatorVerb, rkTrackVerb, verb}, args...)
		return runRkDelegate(append(argv, rkArgvFromChangedFlags(cmd)...)...)
	}
}

// delegateTrackUpdate splits `track update` across rk's verbs: update (any
// flag other than pause/resume/text), then pause, resume, and note, stopping
// at the first non-zero exit. `--pause` with `--resume` is already rejected by
// cobra's flag group before this runs.
func delegateTrackUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]
	f := cmd.Flags()
	steps := [][]string{}
	if rest := rkArgvFromChangedFlags(cmd, "pause", "resume", "text"); len(rest) > 0 || !(f.Changed("pause") || f.Changed("resume") || f.Changed("text")) {
		steps = append(steps, append([]string{rkOperatorVerb, rkTrackVerb, "update", id}, rest...))
	}
	if pause, _ := f.GetBool("pause"); f.Changed("pause") && pause {
		steps = append(steps, []string{rkOperatorVerb, rkTrackVerb, "pause", id})
	}
	if resume, _ := f.GetBool("resume"); f.Changed("resume") && resume {
		steps = append(steps, []string{rkOperatorVerb, rkTrackVerb, "resume", id})
	}
	if f.Changed("text") {
		text, _ := f.GetString("text")
		steps = append(steps, []string{rkOperatorVerb, rkTrackVerb, "note", id, text})
	}
	for _, argv := range steps {
		if err := runRkDelegate(argv...); err != nil {
			return err
		}
	}
	return nil
}

// delegateTickStart replaces tick-start with `rk operator check`; --diff and
// --quiet are accepted and dropped, and no fab state is touched.
func delegateTickStart(cmd *cobra.Command, args []string) error {
	return runRkDelegate(rkOperatorVerb, rkCheckVerb)
}

// delegateClockSync shows rk's cron entries in place of seeding or reconciling
// anything.
func delegateClockSync(cmd *cobra.Command, args []string) error {
	return runRkDelegate(rkCronVerb, "list")
}

// refuseTrackClock is `track clock` under a capable rk: it writes nothing and
// points at the rk cron verbs that now own the operator loop's cadence.
func refuseTrackClock(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(cmd.ErrOrStderr(), "fab operator track clock no longer sets the clock: tune the 'operator loop' entry with 'rk cron edit <id>' or 'rk cron mute <id> --for <dur>' (find the id with 'rk cron list')")
	return &exitCodeError{code: 1}
}
