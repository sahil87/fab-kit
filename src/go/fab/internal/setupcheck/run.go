package setupcheck

import (
	"os/exec"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/kitpath"
)

// Input carries every environment seam Run needs, so the aggregation is as
// testable as the individual probes. The zero value is not useful — production
// wiring lives in cmd/fab/setup.go.
type Input struct {
	// BinaryVersion is the running fab-go binary's version stamp (cmd/fab's
	// `version` var). "dev"/"" is reported but never compared.
	BinaryVersion string
	// ProjectConfigPath is fab/project/config.yaml's path, or "" outside a fab
	// repo — the doctor degrades to the system+env tiers instead of erroring
	// (a pre-init machine is exactly where a doctor must still run).
	ProjectConfigPath string
	// ProjectPin is fab/.fab-version's content, "" when unknown/absent.
	ProjectPin string
	// TmuxEnv is $TMUX verbatim (presence is the pane-viability signal).
	TmuxEnv string
	// KitDir is the resolved kit cache directory; "" asks Run to resolve it
	// via internal/kitpath (resolution failure degrades to an Info finding).
	KitDir string
	// LookPath resolves an executable on PATH; nil means exec.LookPath.
	LookPath LookPathFunc
}

// Run executes every probe against the given environment and aggregates the
// structured report. It is the C2 wizard's future entry point as much as the
// command's. Config load failures surface as a Fail finding (a malformed
// project config IS a real problem) with the probes continuing against the
// empty config — the doctor reports as much as it can rather than dying on
// the first bad read.
func Run(in Input) *Report {
	lookPath := in.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	report := &Report{}

	cfg := &config.Config{}
	var systemLayer, projectLayer map[string]any
	if in.ProjectConfigPath != "" {
		loaded, err := config.LoadPath(in.ProjectConfigPath)
		if err != nil {
			report.Findings = append(report.Findings, Finding{
				Check: "config", Severity: Fail, Subject: in.ProjectConfigPath,
				Detail: "project config is unreadable — fix or remove it: " + err.Error(),
			})
		} else {
			cfg = loaded
		}
		if layers, err := config.LoadLayers(in.ProjectConfigPath); err == nil {
			systemLayer, projectLayer = layers.System, layers.Project
		}
	} else {
		// No repo: still honor the system+env tiers so the doctor diagnoses a
		// machine-wide misconfiguration before any repo exists.
		if layers, err := config.LoadLayers(""); err == nil {
			systemLayer = layers.System
			if merged, err := config.FromMap(layers.Effective); err == nil {
				cfg = merged
			}
		}
	}

	probes, findings := ProbeProviders(cfg, lookPath)
	report.Providers = probes
	report.Findings = append(report.Findings, findings...)

	report.Findings = append(report.Findings, ProbeEnvironment(lookPath, in.TmuxEnv)...)

	kitDir := in.KitDir
	if kitDir == "" {
		if dir, err := kitpath.KitDir(); err == nil {
			kitDir = dir
		}
	}
	report.Findings = append(report.Findings, ProbeVersions(in.BinaryVersion, kitDir, in.ProjectPin)...)

	report.Findings = append(report.Findings, ProbeDispatchMode(cfg, in.TmuxEnv)...)
	report.Findings = append(report.Findings, ProbeOverrideMasking(systemLayer, projectLayer)...)

	return report
}
