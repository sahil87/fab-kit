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
	"intake", "apply", "review", "hydrate", "ship", "review-pr",
}

// change_type_source enum values. An absent/empty field is treated as
// SourceInferred (back-compat): pre-existing changes with no field behave
// exactly as before — re-inference allowed. set-change-type marks SourceExplicit
// so the intake-write hook stops overwriting a human's correction.
const (
	SourceInferred = "inferred"
	SourceExplicit = "explicit"
)

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

// Plan holds plan.md metadata. Replaces the legacy Checklist struct.
type Plan struct {
	Generated           bool `yaml:"generated"`
	TaskCount           int  `yaml:"task_count"`
	AcceptanceCount     int  `yaml:"acceptance_count"`
	AcceptanceCompleted int  `yaml:"acceptance_completed"`
}

// Dimensions holds fuzzy SRAD dimension means.
type Dimensions struct {
	Signal         float64 `yaml:"signal"`
	Reversibility  float64 `yaml:"reversibility"`
	Competence     float64 `yaml:"competence"`
	Disambiguation float64 `yaml:"disambiguation"`
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

// TrueImpactPair holds insertions/deletions/net for a single shortstat pass.
type TrueImpactPair struct {
	Added   int `yaml:"added"`
	Deleted int `yaml:"deleted"`
	Net     int `yaml:"net"`
}

// TrueImpact is the true_impact block in .status.yaml. Created lazily on
// first apply-finish (no template placeholder). Excluding is omitted when
// true_impact_exclude is absent/null/empty in fab/project/config.yaml.
type TrueImpact struct {
	Added           int             `yaml:"added"`
	Deleted         int             `yaml:"deleted"`
	Net             int             `yaml:"net"`
	Excluding       *TrueImpactPair `yaml:"excluding,omitempty"`
	Tests           *TrueImpactPair `yaml:"tests,omitempty"`
	ComputedAt      string          `yaml:"computed_at"`
	ComputedAtStage string          `yaml:"computed_at_stage"`
}

// StatusFile represents the .status.yaml structure.
type StatusFile struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Created    string `yaml:"created"`
	CreatedBy  string `yaml:"created_by"`
	ChangeType string `yaml:"change_type"`
	// ChangeTypeSource records how ChangeType was set: "inferred" (the
	// PostToolUse hook may re-infer and overwrite) or "explicit" (a human ran
	// set-change-type — the hook must not clobber it). Empty == inferred.
	ChangeTypeSource string                  `yaml:"change_type_source,omitempty"`
	Issues           []string                `yaml:"issues"`
	Progress         yaml.Node               `yaml:"-"`
	Plan             Plan                    `yaml:"plan"`
	Confidence       Confidence              `yaml:"confidence"`
	StageMetrics     map[string]*StageMetric `yaml:"-"`
	PRs              []string                `yaml:"prs"`
	TrueImpact       *TrueImpact             `yaml:"true_impact,omitempty"`
	LastUpdated      string                  `yaml:"last_updated"`

	// raw holds the full parsed document for field-preserving serialization
	raw *yaml.Node
}

// Load reads and parses a .status.yaml file. Read failures are classified
// (mz4q F06): only a genuinely missing file reports "not found" — permission
// denied, is-a-directory, and I/O errors wrap the real cause so callers (and
// agents routing recovery off error text) are not sent down a wrong
// re-create/re-switch path.
func Load(path string) (*StatusFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("status file not found: %s", path)
		}
		return nil, fmt.Errorf("read status file %s: %w", path, err)
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

	hasPlan := false
	hasChecklist := false
	var legacyChecklist Plan
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
		case "change_type_source":
			sf.ChangeTypeSource = val.Value
		case "last_updated":
			sf.LastUpdated = val.Value
		case "issues":
			sf.Issues = decodeStringSlice(val)
		case "prs":
			sf.PRs = decodeStringSlice(val)
		case "progress":
			sf.Progress = *val
		case "plan":
			hasPlan = true
			_ = val.Decode(&sf.Plan)
		case "checklist":
			hasChecklist = true
			legacyChecklist = decodeLegacyChecklist(val)
		case "confidence":
			_ = val.Decode(&sf.Confidence)
		case "stage_metrics":
			sf.StageMetrics = decodeStageMetrics(val)
		case "true_impact":
			ti := &TrueImpact{}
			if err := val.Decode(ti); err == nil {
				sf.TrueImpact = ti
			}
		}
	}

	// Legacy schema upgrade: a pre-1.9.0 .status.yaml has `checklist:` instead
	// of `plan:`. Translate the old field shape into Plan so subsequent saves
	// emit a `plan:` block and the user's mutations are not silently dropped.
	// Field mapping mirrors migration 1.8.0-to-1.9.0.md step 4.3:
	//   plan.generated            <- checklist.generated
	//   plan.acceptance_completed <- checklist.completed
	//   plan.acceptance_count     <- checklist.total
	//   plan.task_count           <- 0 (no source field; populated later by callers)
	if !hasPlan && hasChecklist {
		sf.Plan = legacyChecklist
		upgradeLegacyChecklistRaw(root)
	} else if hasPlan && hasChecklist {
		// Mixed schema: both blocks coexist (e.g., a partial migration left the
		// legacy `checklist:` key behind). The `plan:` block is authoritative —
		// drop the stale `checklist:` key from the raw mapping so it does not
		// survive Save.
		dropChecklistRaw(root)
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

// decodeLegacyChecklist parses a legacy `checklist:` mapping into the modern
// Plan shape. Unknown fields (e.g., `path`) are ignored. Missing fields default
// to zero values.
func decodeLegacyChecklist(n *yaml.Node) Plan {
	var p Plan
	if n == nil || n.Kind != yaml.MappingNode {
		return p
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		val := n.Content[i+1].Value
		switch key {
		case "generated":
			p.Generated = val == "true"
		case "completed":
			if v, err := ParseIntStrict(val); err == nil {
				p.AcceptanceCompleted = v
			}
		case "total":
			if v, err := ParseIntStrict(val); err == nil {
				p.AcceptanceCount = v
			}
		}
	}
	return p
}

// upgradeLegacyChecklistRaw rewrites the root mapping in-place: the legacy
// `checklist:` key/value pair is replaced by a `plan:` placeholder mapping so
// syncToRaw has a node to write into. The placeholder is intentionally empty;
// encodePlan repopulates it on every Save with the current Plan struct values.
func upgradeLegacyChecklistRaw(root *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "checklist" {
			root.Content[i].Value = "plan"
			root.Content[i+1] = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			return
		}
	}
}

// dropChecklistRaw removes the legacy `checklist:` key/value pair from the
// root mapping. Used when both `plan:` and `checklist:` coexist — the `plan:`
// block is authoritative and the stale `checklist:` block must not survive
// Save.
func dropChecklistRaw(root *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "checklist" {
			root.Content = append(root.Content[:i], root.Content[i+2:]...)
			return
		}
	}
}

// ParseIntStrict parses a non-negative integer string (digits only). Empty /
// non-numeric inputs return an error and the caller should leave the
// destination at zero. Exported as the package's single digit-only parser —
// internal/status reuses it instead of carrying a duplicate (260612-tb6f, F47).
func ParseIntStrict(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("non-digit")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
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
	// Sync before rename: .status.yaml is the pipeline state machine's source
	// of truth — a crash must never leave an empty/torn file behind the
	// rename (mz4q F03). Contrast runtime.SaveFile, which skips fsync for the
	// ephemeral, re-derivable runtime file.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("sync temp file: %w", err)
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

// SetProgress sets the state of a stage in the progress map, creating the
// stage key when absent (appended at the end of the mapping). It returns an
// error when the document shape is malformed — `progress:` missing or not a
// mapping — so callers cannot report success on a write that would be
// silently dropped (mz4q F07).
func (sf *StatusFile) SetProgress(stage, state string) error {
	if sf.Progress.Kind != yaml.MappingNode {
		return fmt.Errorf("malformed .status.yaml: progress is missing or not a mapping")
	}
	for i := 0; i+1 < len(sf.Progress.Content); i += 2 {
		if sf.Progress.Content[i].Value == stage {
			sf.Progress.Content[i+1].Value = state
			return nil
		}
	}
	sf.Progress.Content = append(sf.Progress.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: stage},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: state},
	)
	return nil
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
//
// Write-time key insertion (mz4q F07): struct fields whose key is absent from
// the parsed document are inserted instead of silently dropped. Sparse
// documents are real — restored pre-0.24.0 archives (migrations never touch
// fab/changes/archive/**) and hand-edited files — and exit-0 commands whose
// write vanished feed wrong state into autonomous pipelines. Command-mutated
// keys (issues, prs, plan, confidence, stage_metrics, last_updated) are
// always ensured; identity scalars (name, change_type) only when non-empty,
// to avoid materializing empty strings.
func (sf *StatusFile) syncToRaw() {
	root := sf.raw.Content[0]

	seen := make(map[string]bool)

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		val := root.Content[i+1]
		seen[key] = true

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
		case "change_type_source":
			// Empty == inferred default: drop the key rather than emit an empty
			// scalar, so an absent field stays absent (back-compat round-trip).
			if sf.ChangeTypeSource == "" {
				dropKeyAt(root, i)
				i -= 2
			} else {
				val.Value = sf.ChangeTypeSource
			}
		case "last_updated":
			val.Value = sf.LastUpdated
		case "issues":
			encodeStringSlice(val, sf.Issues)
		case "prs":
			encodeStringSlice(val, sf.PRs)
		case "progress":
			*val = sf.Progress
		case "plan":
			encodePlan(val, &sf.Plan)
		case "confidence":
			encodeConfidence(val, &sf.Confidence)
		case "stage_metrics":
			encodeStageMetrics(val, sf.StageMetrics)
		case "true_impact":
			if sf.TrueImpact == nil {
				dropKeyAt(root, i)
				i -= 2
			} else {
				encodeTrueImpact(val, sf.TrueImpact)
			}
		}
	}

	if !seen["name"] && sf.Name != "" {
		insertKey(root, "name", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sf.Name})
	}
	if !seen["change_type"] && sf.ChangeType != "" {
		insertKey(root, "change_type", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sf.ChangeType})
	}
	if !seen["change_type_source"] && sf.ChangeTypeSource != "" {
		insertKey(root, "change_type_source", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sf.ChangeTypeSource})
	}
	if !seen["issues"] {
		node := &yaml.Node{}
		encodeStringSlice(node, sf.Issues)
		insertKey(root, "issues", node)
	}
	if !seen["plan"] {
		node := &yaml.Node{}
		encodePlan(node, &sf.Plan)
		insertKey(root, "plan", node)
	}
	if !seen["confidence"] {
		node := &yaml.Node{}
		encodeConfidence(node, &sf.Confidence)
		insertKey(root, "confidence", node)
	}
	if !seen["stage_metrics"] {
		node := &yaml.Node{}
		encodeStageMetrics(node, sf.StageMetrics)
		insertKey(root, "stage_metrics", node)
	}
	if !seen["prs"] {
		node := &yaml.Node{}
		encodeStringSlice(node, sf.PRs)
		insertKey(root, "prs", node)
	}
	if !seen["true_impact"] && sf.TrueImpact != nil {
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		encodeTrueImpact(node, sf.TrueImpact)
		insertKey(root, "true_impact", node)
	}
	// last_updated goes last so the keys above land before it.
	if !seen["last_updated"] {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "last_updated"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: sf.LastUpdated, Style: yaml.DoubleQuotedStyle},
		)
	}
}

// dropKeyAt removes the key/value pair at index i from a mapping node.
func dropKeyAt(root *yaml.Node, i int) {
	root.Content = append(root.Content[:i], root.Content[i+2:]...)
}

// insertKey inserts a key/value pair immediately before the last_updated key
// (or at the end when last_updated is absent). Generalized from the original
// true_impact-only insertion helper so any mutated key absent from a sparse
// legacy document is created on write instead of silently dropped (mz4q F07).
func insertKey(root *yaml.Node, key string, valNode *yaml.Node) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}

	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "last_updated" {
			before := root.Content[:i]
			after := root.Content[i:]
			merged := make([]*yaml.Node, 0, len(root.Content)+2)
			merged = append(merged, before...)
			merged = append(merged, keyNode, valNode)
			merged = append(merged, after...)
			root.Content = merged
			return
		}
	}
	root.Content = append(root.Content, keyNode, valNode)
}

func encodeTrueImpact(n *yaml.Node, ti *TrueImpact) {
	n.Kind = yaml.MappingNode
	n.Tag = "!!map"
	n.Style = 0
	content := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "added"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", ti.Added), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "deleted"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", ti.Deleted), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "net"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", ti.Net), Tag: "!!int"},
	}
	if ti.Excluding != nil {
		content = append(content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "excluding"},
			pairNode(ti.Excluding),
		)
	}
	// `tests` is emitted after `excluding` and before `computed_at`, lazily
	// omitted when nil (matching the `excluding` posture).
	if ti.Tests != nil {
		content = append(content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "tests"},
			pairNode(ti.Tests),
		)
	}
	content = append(content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "computed_at"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: ti.ComputedAt, Tag: "!!str", Style: yaml.DoubleQuotedStyle},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "computed_at_stage"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: ti.ComputedAtStage, Tag: "!!str"},
	)
	n.Content = content
}

// pairNode encodes a TrueImpactPair as an added/deleted/net mapping node.
// Shared by the `excluding` and `tests` sub-blocks of encodeTrueImpact.
func pairNode(p *TrueImpactPair) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "added"},
			{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", p.Added), Tag: "!!int"},
			{Kind: yaml.ScalarNode, Value: "deleted"},
			{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", p.Deleted), Tag: "!!int"},
			{Kind: yaml.ScalarNode, Value: "net"},
			{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", p.Net), Tag: "!!int"},
		},
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
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

func encodePlan(n *yaml.Node, p *Plan) {
	n.Kind = yaml.MappingNode
	n.Tag = "!!map"
	n.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "generated"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%v", p.Generated), Tag: "!!bool"},
		{Kind: yaml.ScalarNode, Value: "task_count"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", p.TaskCount), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "acceptance_count"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", p.AcceptanceCount), Tag: "!!int"},
		{Kind: yaml.ScalarNode, Value: "acceptance_completed"},
		{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", p.AcceptanceCompleted), Tag: "!!int"},
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

	// confidence.indicative is retired (1.10.0): no longer written. The
	// Confidence.Indicative field is kept as a decode-tolerant target so a
	// legacy `indicative: true` key on an un-migrated/archived file round-trips
	// without error and is simply dropped on the next Save.

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
