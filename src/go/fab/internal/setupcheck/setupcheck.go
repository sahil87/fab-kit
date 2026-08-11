// Package setupcheck is the reusable probe layer behind `fab setup check` — a
// read-only environment doctor for fab's own setup state (config viability,
// provider roster, dispatch-mode viability, version/bottle skew). It answers
// the questions fab-kit's `fab doctor` deliberately does not: `doctor` covers
// system prerequisites ("is this machine good enough to use fab-kit"), while
// this package diagnoses the setup state of an installed fab ("is the effective
// config viable against what is actually on this machine").
//
// Every probe is a pure-ish function returning STRUCTURED results (Findings
// plus roster rows) with its environment seams injected — lookPath, $TMUX, the
// kit-cache dir, and the config layers are all parameters (the
// internal/dispatch.SelectMode purity precedent) — so the whole package is
// table-testable and the future interactive `fab setup` wizard (C2) can consume
// the same structs to filter its interview options without shelling out. The
// cobra command (cmd/fab/setup.go) owns only input wiring, rendering, and
// exit-code mapping.
//
// HARD INVARIANT: nothing here writes — no config mutation, no trust-store
// seeding, no agent/pane launches, no state files. Probes read config, run
// exec.LookPath, and read at most one file (the kit cache's VERSION).
package setupcheck

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// Severity is the seriousness of one probe finding. The exit-code contract
// rides on it: only Fail findings produce exit 1; Warn and Info never affect
// the exit code (a warnings-only report exits 0).
type Severity int

const (
	// OK is a passing check — rendered as a ✓ row, never counted against the run.
	OK Severity = iota
	// Info is an informational note (rk absent, unconfigured provider missing,
	// kit cache unresolvable) — advisory only.
	Info
	// Warn is a real but non-breaking condition (unviable-but-descending
	// dispatch.mode, version skew, load-bearing override, gh/yq absent).
	Warn
	// Fail is a real problem — something the effective config names cannot
	// actually run. Any Fail finding makes the command exit 1.
	Fail
)

// String renders the severity for output and tests.
func (s Severity) String() string {
	switch s {
	case OK:
		return "ok"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Fail:
		return "fail"
	default:
		return fmt.Sprintf("severity(%d)", int(s))
	}
}

// Finding is one structured probe result: which probe area, how severe, what
// it concerns, and a human-readable explanation.
type Finding struct {
	Check    string // probe area: "providers", "environment", "versions", "dispatch", "config"
	Severity Severity
	Subject  string // provider name, tool, or config key the finding is about
	Detail   string
}

// ProviderProbe is one row of the provider roster: the provider's declared
// capabilities (presence grammar — "here is how", never "select this mode"),
// the executables its commands resolve to, and whether PATH has them.
type ProviderProbe struct {
	Name        string
	Interactive bool // interactive_command declared (pane-capable grammar)
	Headless    bool // headless_command declared
	Native      bool // native capability declared
	// Executables are the distinct leading executable tokens resolved from the
	// provider's commands (nested `sh -c '...'` forms unwrap to the PROVIDER's
	// executable — agy/kimi, never sh). Missing is the subset not on PATH.
	Executables []string
	Missing     []string
	// Configured reports whether any role resolves to this provider (a depth
	// knob or an agent.profiles.<role>.provider names it) — the line between a
	// Fail (configured but unrunnable) and an Info (inert built-in) finding.
	Configured bool
}

// Found reports whether every executable the provider's commands need is on
// PATH. A provider with no commands at all (a native-only user provider) has
// nothing to look up and reads as found.
func (p ProviderProbe) Found() bool { return len(p.Missing) == 0 }

// Report is the aggregated doctor output the command renders. Producers are
// the Probe* functions and Run; consumers are the renderer (today) and the C2
// wizard's option filtering (later).
type Report struct {
	Providers []ProviderProbe
	Findings  []Finding
}

// Failures counts the failure-severity findings — the exit-1 driver.
func (r *Report) Failures() int { return r.countSeverity(Fail) }

// Warnings counts the warning-severity findings (never exit-affecting).
func (r *Report) Warnings() int { return r.countSeverity(Warn) }

// HasFailures reports whether the run found any real problem (exit 1).
func (r *Report) HasFailures() bool { return r.Failures() > 0 }

func (r *Report) countSeverity(s Severity) int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == s {
			n++
		}
	}
	return n
}

// LookPathFunc is the injectable PATH-resolution seam (production: exec.LookPath).
type LookPathFunc func(string) (string, error)

// shellWords splits a command template into words, honoring single- and
// double-quoted spans (no escape processing — the provider grammars ship
// none). Quote characters are stripped. This is deliberately NOT a full shell
// lexer: it exists to find a command's leading executable, and the nested
// `sh -c '<inner>'` forms keep their inner command as one quoted word.
func shellWords(s string) []string {
	var words []string
	var cur strings.Builder
	var quote byte // 0 = outside quotes
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur.WriteByte(c)
			}
		case c == '\'' || c == '"':
			quote = c
		case c == ' ' || c == '\t':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return words
}

// LeadingExecutable resolves the executable a provider command actually runs.
// The plain form's first word is the answer (`claude --model ...` → `claude`);
// a nested-shell wrapper (`sh -c 'agy -p "$(cat)"'`) recurses into the -c
// argument so the PROVIDER's executable (`agy`) is resolved, never `sh` — the
// wrapper is fab's own stdin idiom, not the binary the user must install.
// Returns "" for an empty command (capability not declared).
func LeadingExecutable(command string) string {
	words := shellWords(command)
	if len(words) == 0 {
		return ""
	}
	switch words[0] {
	case "sh", "bash", "env":
		// sh -c '<inner>' / bash -c '<inner>': the inner command's leading
		// executable is the real one. `env` gets the same treatment for the
		// user-defined command it prefixes.
		for i := 1; i < len(words); i++ {
			if words[i] == "-c" && i+1 < len(words) {
				return LeadingExecutable(words[i+1])
			}
			// env VAR=value prefixes carry no executable; skip assignments.
			if words[0] == "env" && strings.Contains(words[i], "=") && !strings.HasPrefix(words[i], "-") {
				continue
			}
			if !strings.HasPrefix(words[i], "-") {
				return words[i]
			}
		}
		return words[0]
	default:
		return words[0]
	}
}

// ProbeProviders builds the provider roster for the effective config: every
// resolvable provider (built-ins ∪ project/system providers) with its declared
// capabilities and binary presence, plus a finding per provider whose
// executables are missing — Fail when a role resolves to it (dispatch would
// break mid-pipeline), Info when it is an inert built-in nobody named. A
// configured-and-present provider earns an OK finding, so the report shows the
// positive check alongside the roster table.
func ProbeProviders(cfg *config.Config, lookPath LookPathFunc) ([]ProviderProbe, []Finding) {
	configured := configuredProviderRoles(cfg)

	var probes []ProviderProbe
	var findings []Finding
	for _, name := range agent.ProviderNames(cfg) {
		prov, _ := agent.ResolveProvider(cfg, name)
		p := ProviderProbe{
			Name:        name,
			Interactive: prov.InteractiveCommand != "",
			Headless:    prov.HeadlessCommand != "",
			Native:      prov.Native,
			Configured:  configured[name] != "",
		}
		seen := make(map[string]bool)
		for _, cmd := range []string{prov.InteractiveCommand, prov.HeadlessCommand} {
			exe := LeadingExecutable(cmd)
			if exe == "" || seen[exe] {
				continue
			}
			seen[exe] = true
			p.Executables = append(p.Executables, exe)
			if _, err := lookPath(exe); err != nil {
				p.Missing = append(p.Missing, exe)
			}
		}
		probes = append(probes, p)

		switch {
		case len(p.Missing) > 0 && p.Configured:
			findings = append(findings, Finding{
				Check:    "providers",
				Severity: Fail,
				Subject:  name,
				Detail: fmt.Sprintf("provider %q (named by %s) is not runnable: executable %s not found on PATH",
					name, configured[name], strings.Join(p.Missing, ", ")),
			})
		case len(p.Missing) > 0:
			findings = append(findings, Finding{
				Check:    "providers",
				Severity: Info,
				Subject:  name,
				Detail: fmt.Sprintf("provider %q executable %s not found on PATH (not configured — inert until a knob or profile names it)",
					name, strings.Join(p.Missing, ", ")),
			})
		case p.Configured:
			findings = append(findings, Finding{
				Check:    "providers",
				Severity: OK,
				Subject:  name,
				Detail: fmt.Sprintf("provider %q (named by %s): %s on PATH",
					name, configured[name], strings.Join(p.Executables, ", ")),
			})
		}
	}
	return probes, findings
}

// configuredProviderRoles maps each provider a role actually resolves to → the
// list of role names resolving to it (rendered as "agent role(s): doing,
// review" in findings). This is the RESOLVED view — depth knobs and
// agent.profiles overrides included — so a provider named only via
// agent.profiles.review.provider is still "configured".
func configuredProviderRoles(cfg *config.Config) map[string]string {
	roles := make(map[string][]string)
	for _, role := range agent.RoleNames() {
		profile, err := agent.ResolveRole(cfg, role)
		if err != nil || profile.Provider == "" {
			continue
		}
		roles[profile.Provider] = append(roles[profile.Provider], role)
	}
	out := make(map[string]string, len(roles))
	for name, rs := range roles {
		sort.Strings(rs)
		out[name] = "agent role(s): " + strings.Join(rs, ", ")
	}
	return out
}
