package setupcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
)

// ProbeEnvironment reports the machine facts the config's viability depends
// on: $TMUX presence (the pane-rung viability signal, classified with
// internal/dispatch's TmuxAvailability enum — presence/absence only, the same
// resolution resolve-agent uses; a reachability upgrade is dispatch start's
// job) and the PATH presence of the pipeline's helper tools. gh/yq absence is
// Warn (pipeline tooling); rk absence is Info — rk is fail-silent optional
// tooling by the _preamble convention, so its absence must never read as a
// problem.
func ProbeEnvironment(lookPath LookPathFunc, tmuxEnv string) []Finding {
	var findings []Finding

	if tmuxEnv != "" {
		findings = append(findings, Finding{
			Check: "environment", Severity: OK, Subject: "tmux",
			Detail: fmt.Sprintf("$TMUX is set — pane rung viable (tmux %s)", dispatch.TmuxAvailable),
		})
	} else {
		findings = append(findings, Finding{
			Check: "environment", Severity: Info, Subject: "tmux",
			Detail: fmt.Sprintf("$TMUX is not set (tmux %s) — pane-mode dispatch is not viable from here", dispatch.TmuxAbsent),
		})
	}

	for _, tool := range []struct {
		name     string
		severity Severity
		note     string
	}{
		{"gh", Warn, "needed by the ship/review-pr stages"},
		{"yq", Warn, "needed by kit scripts and skills"},
		{"rk", Info, "optional tooling — every rk use is fail-silent"},
	} {
		if path, err := lookPath(tool.name); err == nil {
			findings = append(findings, Finding{
				Check: "environment", Severity: OK, Subject: tool.name,
				Detail: fmt.Sprintf("%s found (%s)", tool.name, path),
			})
		} else {
			findings = append(findings, Finding{
				Check: "environment", Severity: tool.severity, Subject: tool.name,
				Detail: fmt.Sprintf("%s not found on PATH — %s", tool.name, tool.note),
			})
		}
	}
	return findings
}

// ProbeVersions reports the three version sources that can disagree — the
// running binary's version, the kit cache's VERSION file, and the project pin
// (fab/.fab-version) — and warns on any mismatch. Mismatches are WARN, never
// Fail: a triplet disagreement is a skew SIGNAL (the #573 bottle-skew class),
// not proof of breakage. A "dev"/empty binary version is reported but never
// compared (a dev build has no release tag to skew against); an empty pin
// (outside a fab repo) and an unresolvable kit dir simply drop their side of
// the comparison, the latter with an Info finding naming the cause.
func ProbeVersions(binaryVersion, kitDir, projectPin string) []Finding {
	kitVersion := ""
	var findings []Finding
	if kitDir == "" {
		findings = append(findings, Finding{
			Check: "versions", Severity: Info, Subject: "kit cache",
			Detail: "kit cache directory could not be resolved — kit version unknown",
		})
	} else {
		data, err := os.ReadFile(filepath.Join(kitDir, "VERSION"))
		if err != nil {
			findings = append(findings, Finding{
				Check: "versions", Severity: Info, Subject: "kit cache",
				Detail: fmt.Sprintf("kit cache VERSION unreadable at %s (%v)", kitDir, err),
			})
		} else {
			kitVersion = strings.TrimSpace(string(data))
		}
	}

	display := func(v string) string {
		if v == "" {
			return "(none)"
		}
		return v
	}
	findings = append(findings, Finding{
		Check: "versions", Severity: OK, Subject: "version triplet",
		Detail: fmt.Sprintf("binary %s, kit cache %s, project pin %s",
			display(binaryVersion), display(kitVersion), display(projectPin)),
	})

	// normalize strips a leading "v" so a v-prefixed tag and a bare semver
	// compare equal.
	normalize := func(v string) string { return strings.TrimPrefix(strings.TrimSpace(v), "v") }
	comparable := binaryVersion != "" && binaryVersion != "dev"
	if comparable && kitVersion != "" && normalize(binaryVersion) != normalize(kitVersion) {
		findings = append(findings, Finding{
			Check: "versions", Severity: Warn, Subject: "version skew",
			Detail: fmt.Sprintf("binary version %s does not match the kit cache version %s — possible bottle/install skew; verify the binary's behavior, not its version string",
				binaryVersion, kitVersion),
		})
	}
	if comparable && projectPin != "" && normalize(binaryVersion) != normalize(projectPin) {
		findings = append(findings, Finding{
			Check: "versions", Severity: Warn, Subject: "version skew",
			Detail: fmt.Sprintf("binary version %s does not match the project pin %s (fab/.fab-version) — run 'fab upgrade-repo' to reconcile",
				binaryVersion, projectPin),
		})
	}
	return findings
}

// ProbeOverrideMasking runs the behavioral bottle-skew check: for each
// capability-bearing override (interactive_command / headless_command /
// native) under providers.<name> in the system or project config tier, it
// introspects the binary's OWN embedded defaults (internal/agent's
// BuiltinProvider — the go:embed'd defaults.yaml, never the kit cache) and
// flags any override on a provider+key the embedded defaults do not define.
// Such an override is LOAD-BEARING against the installed binary: unsetting it
// silently changes behavior — the exact 2026-08-10 #573 incident shape (a
// system-tier agy interactive_command masking a bottle built without it).
// Findings are Warn: the doctor reports the mask rather than calling it an
// error. Providers absent from the embedded table entirely are skipped — a
// user-defined provider is a definition, not a mask.
//
// The tiers are raw decoded layer maps (config.LoadLayers' System/Project), so
// the check sees what EACH tier sets independently of the merged cascade.
func ProbeOverrideMasking(system, project map[string]any) []Finding {
	var findings []Finding
	for _, tier := range []struct {
		label string
		m     map[string]any
	}{
		{"system", system},
		{"project", project},
	} {
		if len(tier.m) == 0 {
			continue
		}
		tierCfg, err := config.FromMap(tier.m)
		if err != nil {
			continue // the cascade loader already reported the malformed tier
		}
		names := tierCfg.ProviderNames()
		sort.Strings(names)
		for _, name := range names {
			embedded, ok := agent.BuiltinProvider(name)
			if !ok {
				continue // user-defined provider — a definition, not a mask
			}
			override, _ := tierCfg.GetProvider(name)
			masked := maskedCapabilities(override, embedded)
			for _, key := range masked {
				findings = append(findings, Finding{
					Check:    "config",
					Severity: Warn,
					Subject:  fmt.Sprintf("providers.%s.%s", name, key),
					Detail: fmt.Sprintf("%s-tier override providers.%s.%s is load-bearing against your installed binary — the embedded defaults do not define it, so unsetting the override changes behavior (bottle-skew shape)",
						tier.label, name, key),
				})
			}
		}
	}
	return findings
}

// ProbeUserSkills reports the machine-level fab-operator pointer skill that
// `fab sync` writes (`~/.agents/skills/fab-operator/SKILL.md` always;
// `~/.claude/skills/fab-operator/SKILL.md` when claude is on PATH): presence
// per tier, plus a staleness check comparing a present file's description:
// line against the kit's skills/fab-operator.md (the body is a
// version-independent pointer, so the description is the only part that can
// drift). Warn, never Fail — a missing user-level skill breaks nothing inside
// projects, so the doctor's exit code is unaffected. Read-only like every
// probe: it reads the two skill files and the kit skill's frontmatter, and
// gates the Claude tier through the lookPath seam.
func ProbeUserSkills(homeDir string, lookPath LookPathFunc, kitDir string) []Finding {
	if homeDir == "" {
		return []Finding{{
			Check: "user-skills", Severity: Info, Subject: "user-level fab-operator skill",
			Detail: "home directory unknown — user-level fab-operator skill not checked",
		}}
	}

	kitDescription := ""
	if kitDir != "" {
		if desc, err := frontmatterDescription(filepath.Join(kitDir, "skills", "fab-operator.md")); err == nil {
			kitDescription = desc
		}
	}

	var findings []Finding
	check := func(tierDir, path string, expected bool) {
		full := filepath.Join(homeDir, tierDir, "fab-operator", "SKILL.md")
		data, err := os.ReadFile(full)
		if err != nil {
			if !expected {
				findings = append(findings, Finding{
					Check: "user-skills", Severity: Info, Subject: tierDir,
					Detail: tierDir + " tier not expected (claude not on PATH)",
				})
			} else {
				findings = append(findings, Finding{
					Check: "user-skills", Severity: Warn, Subject: tierDir,
					Detail: "user-level fab-operator skill missing — run 'fab sync' in any fab project (the rk operator -L respawn window opens in $HOME)",
				})
			}
			return
		}
		findings = append(findings, Finding{
			Check: "user-skills", Severity: OK, Subject: tierDir,
			Detail: "user-level fab-operator skill: " + path,
		})
		if kitDescription != "" {
			if desc, err := frontmatterDescriptionBytes(data); err != nil || desc != kitDescription {
				findings = append(findings, Finding{
					Check: "user-skills", Severity: Warn, Subject: tierDir,
					Detail: "user-level fab-operator skill is stale — re-run 'fab sync'",
				})
			}
		}
	}

	check(filepath.Join(".agents", "skills"), "~/.agents/skills", true)
	_, claudeErr := lookPath("claude")
	check(filepath.Join(".claude", "skills"), "~/.claude/skills", claudeErr == nil)
	return findings
}

// frontmatterDescription reads a file and returns its frontmatter description.
func frontmatterDescription(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return frontmatterDescriptionBytes(data)
}

// frontmatterDescriptionBytes extracts the description: value from a leading
// --- frontmatter block (line-prefix scan; the value is compared verbatim,
// quotes included — `fab sync` writes it byte-identical to the kit skill's).
func frontmatterDescriptionBytes(data []byte) (string, error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", fmt.Errorf("no frontmatter block")
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if value, ok := strings.CutPrefix(line, "description:"); ok {
			if value = strings.TrimSpace(value); value != "" {
				return value, nil
			}
			break
		}
	}
	return "", fmt.Errorf("no description: frontmatter line")
}

// maskedCapabilities returns the capability keys the override SETS that the
// embedded defaults leave undefined — the mask condition. The deprecated
// spellings (session_command / dispatch_command) count as their modern
// equivalents, matching ResolveProvider's read-time aliases.
func maskedCapabilities(override, embedded config.ProviderConfig) []string {
	var masked []string
	interactive := override.InteractiveCommand
	if interactive == "" {
		interactive = override.SessionCommand
	}
	if interactive != "" && embedded.InteractiveCommand == "" {
		masked = append(masked, "interactive_command")
	}
	headless := override.HeadlessCommand
	if headless == "" {
		headless = override.DispatchCommand
	}
	if headless != "" && embedded.HeadlessCommand == "" {
		masked = append(masked, "headless_command")
	}
	if override.NativeSet && override.Native && !embedded.Native {
		masked = append(masked, "native")
	}
	return masked
}

// ProbeDispatchMode reports whether the configured dispatch.mode preference is
// viable against the environment and the resolved workers-depth provider's
// capabilities, reusing internal/dispatch.SelectMode itself so the descent
// reasons are the EXACT strings a real dispatch would produce ("pane
// unavailable: no tmux", "pane unavailable: no interactive_command", "native
// unavailable"). A working descent is Warn (the config will silently not do
// what was asked); no reachable rung at all is Fail. The workers-depth
// (agent.workers) provider is the probe subject because stage dispatch is
// where the ladder runs.
func ProbeDispatchMode(cfg *config.Config, tmuxEnv string) []Finding {
	preference := cfg.GetDispatchMode()

	profile, err := agent.ResolveRole(cfg, agent.RoleDoing)
	if err != nil {
		return []Finding{{
			Check: "dispatch", Severity: Warn, Subject: "dispatch.mode",
			Detail: fmt.Sprintf("could not resolve the workers-depth provider: %v", err),
		}}
	}
	prov, known := agent.ResolveProvider(cfg, profile.Provider)
	if !known {
		return []Finding{{
			Check: "dispatch", Severity: Fail, Subject: profile.Provider,
			Detail: fmt.Sprintf("provider %q (resolved for workers-depth roles) is not defined — no built-in or configured providers entry matches", profile.Provider),
		}}
	}

	tmux := dispatch.TmuxAbsent
	if tmuxEnv != "" {
		tmux = dispatch.TmuxAvailable
	}
	mode, reason, err := dispatch.SelectMode(false, false, false, false, preference,
		prov.Native, prov.InteractiveCommand != "", prov.HeadlessCommand != "", tmux)
	switch {
	case err != nil:
		return []Finding{{
			Check: "dispatch", Severity: Fail, Subject: "dispatch.mode",
			Detail: fmt.Sprintf("dispatch.mode %q has no reachable rung for provider %q: %v", preference, profile.Provider, err),
		}}
	case string(mode) != preference:
		return []Finding{{
			Check: "dispatch", Severity: Warn, Subject: "dispatch.mode",
			Detail: fmt.Sprintf("dispatch.mode %q would descend at dispatch time: %s", preference, reason),
		}}
	default:
		return []Finding{{
			Check: "dispatch", Severity: OK, Subject: "dispatch.mode",
			Detail: fmt.Sprintf("dispatch.mode %q is viable (%s)", preference, reason),
		}}
	}
}
