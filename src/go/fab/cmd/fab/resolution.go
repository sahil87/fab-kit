package main

import (
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
)

// resolutionRequest is the normalized addressing/output intent shared by the
// two CLI surfaces. Cobra owns environment reads and flag supplied-ness; this
// composer owns every profile-to-resolution assembly step.
type resolutionRequest struct {
	Selector     string
	Kind         string
	Role         string
	ProviderOnly bool
	Overrides    agent.Overrides
	Headless     bool
	Passthrough  []string
	Dispatch     bool
	TmuxEnv      string
}

// composeAgentResolution is the single assembly site for agent.Resolution.
// Callers normalize addressing and render a projection; they do not compose
// profiles, commands, aliases, fill modes, provenance, or dispatch themselves.
func composeAgentResolution(cfg *config.Config, request resolutionRequest) (agent.Resolution, error) {
	var (
		profile agent.Profile
		source  agent.Source
		err     error
	)
	if request.ProviderOnly {
		profile = agent.Profile{
			Provider: request.Overrides.Provider,
			Model:    request.Overrides.Model,
			Effort:   request.Overrides.Effort,
		}
		source.Provider = "flag"
		if request.Overrides.ModelSet {
			source.Model = "flag"
		}
		if request.Overrides.EffortSet {
			source.Effort = "flag"
		}
	} else {
		profile, source, err = agent.ResolveRoleWithSource(cfg, request.Role, request.Overrides)
		if err != nil {
			return agent.Resolution{}, err
		}
	}

	prov, known := agent.ResolveProvider(cfg, profile.Provider)
	if request.ProviderOnly && !known {
		return agent.Resolution{}, unknownProviderError(cfg, profile.Provider)
	}
	if !request.ProviderOnly && request.Overrides.ProviderSet && !known {
		return agent.Resolution{}, unknownProviderError(cfg, profile.Provider)
	}

	template := prov.InteractiveCommand
	if request.Headless {
		template = prov.HeadlessCommand
	}
	fillMode := agent.FillModeAppend
	if spawn.IsTemplate(template) {
		fillMode = agent.FillModeTemplate
	}
	command := ""
	if template != "" {
		command = appendPassthrough(spawn.WithProfile(template, profile.Model, profile.Effort), request.Passthrough)
	}

	resolution := agent.Resolution{
		Selector: request.Selector,
		Kind:     request.Kind,
		Role:     request.Role,
		Provider: profile.Provider,
		Model:    profile.Model,
		Effort:   profile.Effort,
		Command:  command,

		Template: template,
		FillMode: fillMode,
		Source:   source,
	}
	if alias := agent.ModelAlias(profile.Model); alias != profile.Model {
		resolution.ModelAlias = alias
	}

	if request.Dispatch {
		resolution.Dispatch, err = dispatchResolutionFor(prov, profile, cfg.GetDispatchMode(), request.TmuxEnv)
		if err != nil {
			return agent.Resolution{}, noDispatchCapabilityError(profile.Provider, cfg.GetDispatchMode(), err)
		}
	}
	return resolution, nil
}

// dispatchResolutionFor selects and composes the non-native dispatch projection.
// A nil result is native. tmuxEnv is supplied by the cobra layer, keeping the
// selector pure and deterministic in tests.
func dispatchResolutionFor(prov config.ProviderConfig, profile agent.Profile, preference, tmuxEnv string) (*agent.DispatchResolution, error) {
	tmux := dispatch.TmuxAbsent
	if tmuxEnv != "" {
		tmux = dispatch.TmuxAvailable
	}
	mode, _, err := dispatch.SelectMode(false, false, false, false, preference,
		prov.Native, prov.InteractiveCommand != "", prov.HeadlessCommand != "", tmux)
	if err != nil {
		return nil, err
	}

	var template string
	switch mode {
	case dispatch.ModeNative:
		return nil, nil
	case dispatch.ModePane:
		template = prov.InteractiveCommand
	case dispatch.ModeHeadless:
		template = prov.HeadlessCommand
	default:
		return nil, fmt.Errorf("unexpected dispatch mode %q", mode)
	}
	return &agent.DispatchResolution{
		Rung:    string(mode),
		Command: spawn.WithProfile(template, profile.Model, profile.Effort),
	}, nil
}
