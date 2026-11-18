// Package agentmodels discovers the models offered by agent CLI tools and merges
// them with fab-kit's provider fills.
package agentmodels

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

const discoveryTimeout = 3 * time.Second

const (
	// SourceTool marks a model reported by the tool or its models_command.
	SourceTool = "tool"
	// SourceAlias marks a built-in Claude family alias.
	SourceAlias = "alias"
	// SourceFill marks a model supplied by the resolved provider's role fills.
	SourceFill = "fill"
	// SourceReserved marks the reserved tool-default value, not a discovered model.
	SourceReserved = "reserved"
)

// Row is one model offered by a tool or supplied by a provider fill. Efforts is
// normalized to an empty array when the tool reports no per-model set.
type Row struct {
	CLI     string   `json:"cli"`
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Efforts []string `json:"efforts"`
	Default bool     `json:"default"`
	Source  string   `json:"source"`
}

type environment struct {
	homeDir   func() (string, error)
	lookupEnv func(string) (string, bool)
	run       func(context.Context, string, ...string) ([]byte, error)
	timeout   time.Duration
}

type adapter func(environment) ([]Row, string, error)

// Discover returns one provider's discovered rows, resolved fills, and any
// fail-open discovery warnings.
func Discover(cfg *config.Config, name string) ([]Row, []string) {
	return discoverWith(cfg, name, defaultEnvironment())
}

func defaultEnvironment() environment {
	return environment{
		homeDir:   os.UserHomeDir,
		lookupEnv: os.LookupEnv,
		run:       runProcess,
		timeout:   discoveryTimeout,
	}
}

func discoverWith(cfg *config.Config, name string, env environment) ([]Row, []string) {
	provider, _ := agent.ResolveProvider(cfg, name)
	var discover adapter
	if provider.ModelsCommand != "" {
		discover = commandAdapter(provider.ModelsCommand)
	} else {
		switch name {
		case "codex":
			discover = discoverCodex
		case "claude":
			discover = discoverClaude
		case "kimi":
			discover = discoverKimi
		}
	}

	var rows []Row
	var defaultID string
	var warnings []string
	if discover != nil {
		var err error
		rows, defaultID, err = discover(env)
		if err != nil {
			warnings = append(warnings, err.Error())
		}
	}
	rows = append(rows, providerFills(provider)...)
	rows = mergeRows(rows, defaultID)
	if name == "claude" {
		for i := range rows {
			if len(rows[i].Efforts) == 0 {
				rows[i].Efforts = append([]string(nil), claudeEfforts...)
			}
		}
	}
	normalized := rows[:0]
	for i := range rows {
		if rows[i].ID == agent.ToolDefaultModel {
			continue
		}
		rows[i].CLI = name
		if rows[i].Label == "" {
			rows[i].Label = rows[i].ID
		}
		if rows[i].Efforts == nil {
			rows[i].Efforts = []string{}
		}
		normalized = append(normalized, rows[i])
	}
	return append([]Row{toolDefaultRow(name, normalized)}, normalized...), warnings
}

// toolDefaultRow builds the reserved tool-default row: labelled with the tool's
// reported default model when one exists, carrying that model's efforts, else
// claude's known effort set, else none.
func toolDefaultRow(name string, rows []Row) Row {
	row := Row{
		CLI:     name,
		ID:      agent.ToolDefaultModel,
		Label:   "tool default",
		Efforts: []string{},
		Source:  SourceReserved,
	}
	for _, candidate := range rows {
		if candidate.Default {
			row.Label = "tool default (" + candidate.ID + ")"
			row.Efforts = append(row.Efforts, candidate.Efforts...)
			break
		}
	}
	if len(row.Efforts) == 0 && name == "claude" {
		row.Efforts = append(row.Efforts, claudeEfforts...)
	}
	return row
}

func providerFills(provider config.ProviderConfig) []Row {
	unique := make(map[string]struct{})
	for _, profile := range provider.Profiles {
		if profile.Model != "" {
			unique[profile.Model] = struct{}{}
		}
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	rows := make([]Row, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, Row{ID: id, Efforts: []string{}, Source: SourceFill})
	}
	return rows
}

func mergeRows(input []Row, defaultID string) []Row {
	if defaultID != "" {
		input = append(input, Row{ID: defaultID, Default: true, Source: SourceTool})
	}
	rows := make([]Row, 0, len(input))
	positions := make(map[string]int, len(input))
	for _, row := range input {
		if row.ID == "" {
			continue
		}
		index, exists := positions[row.ID]
		if !exists {
			positions[row.ID] = len(rows)
			rows = append(rows, row)
			continue
		}
		current := rows[index]
		current.Default = current.Default || row.Default
		if sourceRank(row.Source) > sourceRank(current.Source) {
			if row.Label == "" {
				row.Label = current.Label
			}
			if len(row.Efforts) == 0 {
				row.Efforts = current.Efforts
			}
			row.Default = current.Default
			rows[index] = row
			continue
		}
		if current.Label == "" {
			current.Label = row.Label
		}
		if len(current.Efforts) == 0 {
			current.Efforts = row.Efforts
		}
		rows[index] = current
	}
	return rows
}

func sourceRank(source string) int {
	switch source {
	case SourceTool:
		return 3
	case SourceAlias:
		return 2
	case SourceFill:
		return 1
	default:
		return 0
	}
}

func runProcess(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = 100 * time.Millisecond
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return nil, fmt.Errorf("%w: %s", err, detail)
		}
		return nil, err
	}
	return stdout, nil
}

func runBounded(env environment, name string, args ...string) ([]byte, error) {
	timeout := env.timeout
	if timeout == 0 {
		timeout = discoveryTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	stdout, err := env.run(ctx, name, args...)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("timed out after %s", timeout)
	}
	return stdout, err
}

func configPath(env environment, override, directory, filename string) (string, error) {
	if override != "" {
		if value, ok := env.lookupEnv(override); ok && value != "" {
			return filepath.Join(value, filename), nil
		}
	}
	home, err := env.homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, directory, filename), nil
}
