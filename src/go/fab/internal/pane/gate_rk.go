package pane

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// This file holds the rk-DELEGATED arm of the pane readiness gate: when a
// sentinel-capable run-kit is on PATH, the mechanical classification
// (state-present / sentinel echo / parked) is run by `rk mux await --ready`
// instead of fab's own sentinel/echo/stability choreography — "fab consumes,
// never reimplements" (docs/specs/agent-messaging.md). The raw-tmux
// classifier in gate.go stays as the rk-less fallback, byte-identical, and the
// takeover precondition stays FAB-SIDE ahead of both arms: rk's AwaitReady
// deliberately classifies a cooked-shell echo as `ready (echo)`
// (terminals-are-one-standard), which for a dispatch pane is exactly the 57mp
// false-ready — so rk is never invoked while a shell owns the pane.

// rkReadyTimeout bounds one delegated probe, in seconds (rk's --timeout unit).
// It is an unexported internal constant with NO flag and NO config field — the
// settleDelay/stabilityDelay gate-timings precedent: mechanics, not policy.
// Twenty seconds lets one rk await absorb ordinary boot churn while keeping
// `fab … ready` a probe (a bound expiry maps to `booting`, so the skill-side
// re-probe loop and its consecutive-booting allowance stay unrewired).
const rkReadyTimeout = 20

// rkSentinelToken is the capability discriminant in `rk mux await --help`:
// `--ready` predates run-kit's sentinel classifier (capture-settle semantics,
// with the `ready (settled)` false-fire hazard), so the flag's PRESENCE is not
// capability — the sentinel contract's report word `parked` is. The binary is
// probed, never the version string (a bottle can predate a same-version
// source change).
const rkSentinelToken = "parked"

// rkAwaitArgs builds the rk argv for the delegated classification probe:
// `mux await --ready <pane> [-L <server>] --timeout <secs>`. When server is
// non-empty it appends rk's own `-L <server>` (the mux family takes the same
// socket flag as fab pane — the rkPanesArgs precedent in cmd/fab/pane_map.go).
// Extracted for unit-testability of argv construction.
func rkAwaitArgs(paneID, server string) []string {
	args := []string{"mux", "await", "--ready", paneID}
	if server != "" {
		args = append(args, "-L", server)
	}
	return append(args, "--timeout", strconv.Itoa(rkReadyTimeout))
}

// rkAwaitRunner locates rk on PATH and runs `rk mux await --ready`, returning
// its raw stdout, raw stderr, and the exec error. Package-level var so the
// rk-arm mapping tests stub the seam without a live rk or tmux (the
// rkPanesRunner / setupcheck LookPathFunc precedent).
var rkAwaitRunner = func(paneID, server string) (string, []byte, error) {
	if _, err := exec.LookPath("rk"); err != nil {
		return "", nil, err
	}
	return RunCmd("rk", rkAwaitArgs(paneID, server)...)
}

// rkSentinelProbe is the injectable capability-probe seam: it answers whether
// the installed rk's `--ready` is the sentinel classifier by running
// `rk mux await --help` and requiring the sentinel contract's `parked` token.
// rk absent (LookPath) or a failed help run both answer false — the raw-tmux
// arm is the degradation, never an error.
var rkSentinelProbe = func() bool {
	if _, err := exec.LookPath("rk"); err != nil {
		return false
	}
	out, _, err := RunCmd("rk", "mux", "await", "--help")
	return err == nil && rkHelpSentinelCapable(out)
}

// rkHelpSentinelCapable is the pure decision half of the capability probe: the
// help output is sentinel-capable iff it mentions the `parked` report (present
// in run-kit ≥ the sentinel `--ready` release, absent from the pre-sentinel
// capture-settle help).
func rkHelpSentinelCapable(helpOut string) bool {
	return strings.Contains(helpOut, rkSentinelToken)
}

// The capability answer is computed at most ONCE per process: the gate loop
// re-probes, and a per-probe `rk mux await --help` subprocess would tax every
// iteration. Guarded by a mutex (not sync.Once) so tests can reset it.
var (
	rkCapableMu    sync.Mutex
	rkCapableKnown bool
	rkCapableValue bool
)

// rkSentinelCapable is the cached capability answer: on the first call it runs
// the probe seam and memoizes the result for the rest of the process.
func rkSentinelCapable() bool {
	rkCapableMu.Lock()
	defer rkCapableMu.Unlock()
	if !rkCapableKnown {
		rkCapableValue = rkSentinelProbe()
		rkCapableKnown = true
	}
	return rkCapableValue
}

// rkReportToken extracts the classification token from rk's report — the first
// field of stdout's first line ("ready %5 (echo)" → "ready", "parked %5" →
// "parked"). Empty or whitespace-only stdout yields "".
func rkReportToken(stdout string) string {
	line, _, _ := strings.Cut(stdout, "\n")
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// The fail-open warning fires at most ONCE per process: a per-probe warning
// would spam the gate loop's re-probes, but the gate is a correctness path, so
// one warning beats the map-enumeration delegation's silence. Guarded by a
// mutex (not sync.Once) so tests can reset it.
var (
	rkWarnMu sync.Mutex
	rkWarned bool
)

// rkWarn is the fail-open warning sink, a var so tests can capture it.
var rkWarn = func(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
}

// warnRKFallback reports an unexpected rk-arm failure before the raw-tmux arm
// classifies the probe instead. rk's stderr is the child's own diagnostic (the
// StderrError convention); it is surfaced ONLY here — classification snippets
// stay fab-captured on both arms so the report contract never couples to rk's
// stderr formatting.
func warnRKFallback(paneID string, err error, stderr []byte) {
	rkWarnMu.Lock()
	already := rkWarned
	rkWarned = true
	rkWarnMu.Unlock()
	if already {
		return
	}
	msg := strings.TrimSpace(string(stderr))
	if err != nil && msg != "" {
		rkWarn("rk gate delegation failed for %s (%v: %s); falling back to the raw-tmux classifier", paneID, err, msg)
	} else if err != nil {
		rkWarn("rk gate delegation failed for %s (%v); falling back to the raw-tmux classifier", paneID, err)
	} else {
		rkWarn("rk gate delegation returned an unparsable report for %s; falling back to the raw-tmux classifier", paneID)
	}
}
