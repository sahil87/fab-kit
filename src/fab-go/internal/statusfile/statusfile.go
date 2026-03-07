package statusfile

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Ordered stage list — pipeline order.
var StageOrder = []string{
	"intake", "spec", "tasks", "apply", "review", "hydrate", "ship", "review-pr",
}

// StageNumber returns the 1-indexed position of a stage.
func StageNumber(stage string) int {
	for i, s := range StageOrder {
		if s == stage {
			return i + 1
		}
	}
	return 0
}

// NextStage returns the next stage in the pipeline, or "" if at the end.
func NextStage(stage string) string {
	for i, s := range StageOrder {
		if s == stage && i+1 < len(StageOrder) {
			return StageOrder[i+1]
		}
	}
	return ""
}

// Checklist holds checklist metadata.
type Checklist struct {
	Generated bool   `yaml:"generated"`
	Path      string `yaml:"path,omitempty"`
	Completed int    `yaml:"completed"`
	Total     int    `yaml:"total"`
}

// Dimensions holds fuzzy SRAD dimension means.
type Dimensions struct {
	Signal          float64 `yaml:"signal"`
	Reversibility   float64 `yaml:"reversibility"`
	Competence      float64 `yaml:"competence"`
	Disambiguation  float64 `yaml:"disambiguation"`
}

// Confidence holds the confidence scoring block.
type Confidence struct {
	Certain    int         `yaml:"certain"`
	Confident  int         `yaml:"confident"`
	Tentative  int         `yaml:"tentative"`
	Unresolved int         `yaml:"unresolved"`
	Score      float64     `yaml:"score"`
	Indicative *bool       `yaml:"indicative,omitempty"`
	Fuzzy      *bool       `yaml:"fuzzy,omitempty"`
	Dimensions *Dimensions `yaml:"dimensions,omitempty"`
}

// StageMetric holds timing/driver metadata for a stage.
type StageMetric struct {
	StartedAt   string `yaml:"started_at,omitempty"`
	Driver      string `yaml:"driver,omitempty"`
	Iterations  int    `yaml:"iterations,omitempty"`
	CompletedAt string `yaml:"completed_at,omitempty"`
}

// StatusFile represents the .status.yaml structure.
type StatusFile struct {
	ID          string                  `yaml:"id"`
	Name        string                  `yaml:"name"`
	Created     string                  `yaml:"created"`
	CreatedBy   string                  `yaml:"created_by"`
	ChangeType  string                  `yaml:"change_type"`
	Issues      []string                `yaml:"issues"`
	Progress    yaml.Node               `yaml:"-"`
	Checklist   Checklist               `yaml:"checklist"`
	Confidence  Confidence              `yaml:"confidence"`
	StageMetrics map[string]*StageMetric `yaml:"-"`
	PRs         []string                `yaml:"prs"`
	LastUpdated string                  `yaml:"last_updated"`

	// raw holds the full parsed document for field-preserving serialization
	raw *yaml.Node
}

// Load reads and parses a .status.yaml file.
func Load(path string) (*StatusFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("status file not found: %s", path)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("empty or invalid YAML document: %s", path)
	}

	sf := &StatusFile{
		raw:          &doc,
		StageMetrics: make(map[string]*StageMetric),
	}

	// Parse top-level fields from the mapping node
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping at root of %s", path)
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		val := root.Content[i+1]

		switch key {
		case "id":
			sf.ID = val.Value
		case "name":
			sf.Name = val.Value
		case "created":
			sf.Created = val.Value
		case "created_by":
			sf.CreatedBy = val.Value
		case "change_type":
			sf.ChangeType = val.Value
		case "last_updated":
			sf.LastUpdated = val.Value
		case "issues":
			sf.Issues = decodeStringSlice(val)
		case "prs":
			sf.PRs = decodeStringSlice(val)
		case "progress":
			sf.Progress = *val
		case "checklist":
			_ = val.Decode(&sf.Checklist)
		case "confidence":
			_ = val.Decode(&sf.Confidence)
		case "stage_metrics":
			sf.StageMetrics = decodeStageMetrics(val)
		}
	}

	if sf.Issues == nil {
		sf.Issues = []string{}
	}
	if sf.PRs == nil {
		sf.PRs = []string{}
	}
	if sf.StageMetrics == nil {
		sf.StageMetrics = make(map[string]*StageMetric)
	}

	return sf, nil
}

// Save writes the StatusFile back to disk atomically (temp + rename).
func (sf *StatusFile) Save(path string) error {
	sf.LastUpdated = nowISO()
	sf.syncToRaw()

	data, err := yaml.Marshal(sf.raw)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".status.yaml.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// GetProgress returns the state of a stage from the progress map.
func (sf *StatusFile) GetProgress(stage string) string {
	if sf.Progress.Kind != yaml.MappingNode {
		return "pending"
	}
	for i := 0; i+1 < len(sf.Progress.Content); i += 2 {
		if sf.Progress.Content[i].Value == stage {
			return sf.Progress.Content[i+1].Value
		}
	}
	return "pending"
}

// SetProgress sets the state of a stage in the progress map.
func (sf *StatusFile) SetProgress(stage, state string) {
	if sf.Progress.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(sf.Progress.Content); i += 2 {
		if sf.Progress.Content[i].Value == stage {
			sf.Progress.Content[i+1].Value = state
			return
		}
	}
}

// GetProgressMap returns an ordered slice of stage:state pairs.
func (sf *StatusFile) GetProgressMap() []StageState {
	result := make([]StageState, 0, len(StageOrder))
	for _, stage := range StageOrder {
		result = append(result, StageState{Stage: stage, State: sf.GetProgress(stage)})
	}
	return result
}

// StageState is a stage name and its state.
type StageState struct {
	Stage string
	State string
}

// syncToRaw updates the raw yaml.Node from the struct fields.
func (sf *StatusFile) syncToRaw() {
	root := sf.raw.Content[0]

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		val := root.Content[i+1]

		switch key {
		case "id":
			val.Value = sf.ID
		case "name":
			val.Value = sf.Name
		case "created":
			val.Value = sf.Created
		case "created_by":
			val.Value = sf.CreatedBy
		case "change_type":
			val.Value = sf.ChangeType
		case "last_updated":
			val.Value = sf.LastUpdated
		case "issues":
			encodeStringSlice(val, sf.Issues)
		case "prs":
			encodeStringSlice(val, sf.PRs)
		case "progress":
			*val = sf.Progress
		case "checklist":
			encodeChecklist(val, &sf.Checklist)
		case "confidence":
			encodeConfidence(val, &sf.Confidence)
		case "stage_metrics":
			encodeStageMetrics(val, sf.StageMetrics)
		}
	}
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}

func decodeStringSlice(n *yaml.Node) []string {
	if n.Kind != yaml.SequenceNode {
		return []string{}
	}
	result := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		result = append(result, c.Value)
	}
	return result
}

func encodeStringSlice(n *yaml.Node, items []string) {
	n.Kind = yaml.SequenceNode
	n.Tag = "!!seq"
	n.Value = ""
	if len(items) == 0 {
		n.Content = nil
		n.Style = yaml.FlowStyle
		return
	}
	n.Style = 0
	content := make([]*yaml.Node, 0, len(items))
	for _, item := range items {
		content = append(content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: item,
		})
	}
	n.Content = content
}

func encodeChecklist(n *yaml.Node, c *Checklist) {
	n.Kind = yaml.MappingNode
	n.Tag = "!!map"
	n.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "generated"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", c.Generated), Tag: "!!bool"},
		{Kind: yaml.ScalarNode, Value: "path"},
		{Kind: yaml.ScalarNode, Value: c.Path, Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: "completed"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", c.Completed), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "total"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", c.Total), Tag: "!!int"},
	}
}

func encodeConfidence(n *yaml.Node, c *Confidence) {
	n.Kind = yaml.MappingNode
	n.Tag = "!!map"
	content := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "certain"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", c.Certain), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "confident"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", c.Confident), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "tentative"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", c.Tentative), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "unresolved"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", c.Unresolved), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "score"},
		{Kind: yaml.ScalarNode, Value: formatFloat(c.Score), Tag: "!!float"},
	}

	if c.Indicative != nil && *c.Indicative {
		content = append(content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "indicative"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
		)
	}

	if c.Fuzzy != nil && *c.Fuzzy {
		content = append(content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "fuzzy"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: "true", Tag: "!!bool"},
		)
		if c.Dimensions != nil {
			dimNode := &yaml.Node{
				Kind: yaml.MappingNode,
				Tag:  "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "signal"},
					{Kind: yaml.ScalarNode, Value: formatFloat(c.Dimensions.Signal), Tag: "!!float"},
					{Kind: yaml.ScalarNode, Value: "reversibility"},
					{Kind: yaml.ScalarNode, Value: formatFloat(c.Dimensions.Reversibility), Tag: "!!float"},
					{Kind: yaml.ScalarNode, Value: "competence"},
					{Kind: yaml.ScalarNode, Value: formatFloat(c.Dimensions.Competence), Tag: "!!float"},
					{Kind: yaml.ScalarNode, Value: "disambiguation"},
					{Kind: yaml.ScalarNode, Value: formatFloat(c.Dimensions.Disambiguation), Tag: "!!float"},
				},
			}
			content = append(content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "dimensions"},
				dimNode,
			)
		}
	}

	n.Content = content
}

func decodeStageMetrics(n *yaml.Node) map[string]*StageMetric {
	result := make(map[string]*StageMetric)
	if n.Kind != yaml.MappingNode {
		return result
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		stage := n.Content[i].Value
		sm := &StageMetric{}
		_ = n.Content[i+1].Decode(sm)
		result[stage] = sm
	}
	return result
}

func encodeStageMetrics(n *yaml.Node, metrics map[string]*StageMetric) {
	n.Kind = yaml.MappingNode
	n.Tag = "!!map"

	if len(metrics) == 0 {
		n.Content = nil
		n.Style = yaml.FlowStyle
		return
	}

	n.Style = 0
	content := make([]*yaml.Node, 0)

	// Preserve stage order
	for _, stage := range StageOrder {
		sm, ok := metrics[stage]
		if !ok {
			continue
		}
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: stage}

		valNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
		valContent := make([]*yaml.Node, 0)

		if sm.StartedAt != "" {
			valContent = append(valContent,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "started_at"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: sm.StartedAt, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
			)
		}
		if sm.Driver != "" {
			valContent = append(valContent,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "driver"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: sm.Driver},
			)
		}
		if sm.Iterations > 0 {
			valContent = append(valContent,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "iterations"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", sm.Iterations), Tag: "!!int"},
			)
		}
		if sm.CompletedAt != "" {
			valContent = append(valContent,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "completed_at"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: sm.CompletedAt, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
			)
		}

		valNode.Content = valContent
		content = append(content, keyNode, valNode)
	}

	n.Content = content
}

func formatFloat(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	return s
}

// BoolPtr returns a pointer to a bool value.
func BoolPtr(b bool) *bool {
	return &b
}
