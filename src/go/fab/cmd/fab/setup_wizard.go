package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configupgrade"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/setupcheck"
	"github.com/spf13/cobra"
)

// setup_wizard.go is the bare-`fab setup` interactive wizard: an interview
// layer over the setupcheck probe. All detection comes from ONE
// setupcheck.Run call (the same call `fab setup check` makes) — the wizard
// adds no probing of its own and never asks a capability question; it asks
// preference questions over options pre-filtered to what the probe found.
// Writes go only through the existing surgical `fab config set` code path
// (configupgrade.SetSystem / configupgrade.Set via configMutationPath), so the
// wizard adds no config-write machinery of its own.
//
// The interview loop lives in package main (not an internal package) because
// the seams it reuses — configMutationPath, cascadeTier/effectiveTierFor-style
// origin rendering, warnIfShadowed, isStdinTTY, version — are unexported
// members of package main; C1's split (cmd owns wiring/rendering, internal
// owns probing) is preserved: the wizard IS wiring/rendering.

// wizardOptions carries the bare-`fab setup` flags.
type wizardOptions struct {
	defaults bool // --defaults: accept every default, never read stdin
	project  bool // --project: write the project tier instead of the system tier
}

// setupWizard holds one wizard run: the probe report, the config read model
// (loaded once for every question's current-value rendering), and the pending
// answers.
type setupWizard struct {
	cmd      *cobra.Command
	opts     wizardOptions
	path     string // write target (system or project config path)
	tmux     bool   // pane-rung viability, from the probe input's $TMUX signal
	report   *setupcheck.Report
	layers   *config.Layers
	defaults map[string]any
	reader   *bufio.Reader
	readErr  error // first non-EOF stdin read error — poisons the write step
	answers  []wizardAnswer
}

// wizardAnswer is one asked question: the key it writes, the current effective
// value (the comparison baseline for diffAndWrite), its display form (differs
// only when an empty current renders as an inherit indication), and the chosen
// answer.
type wizardAnswer struct {
	key            string
	current        string
	currentDisplay string
	answer         string
}

// wizardQuestion is one interview prompt. A nil options list means a
// free-form scalar answer validated by validate (nil = no validation).
// inheritAs, when set, marks a key whose built-in-defaults row is DERIVED
// (the sparse agent.profiles keys — readModelDefaults resolves them from the
// depth knobs, they are never stored): a value still at the derived default is
// presented as this inherit indication over an empty baseline, so Enter keeps
// the inherit and even typing the currently-inherited provider is an explicit
// pin that writes.
type wizardQuestion struct {
	key       string
	text      string
	options   []wizardOption
	validate  func(string) bool
	inheritAs string
}

// wizardOption is one selectable answer, with an optional trailing annotation
// (e.g. a provider's capability list).
type wizardOption struct {
	value      string
	annotation string
}

// runSetupWizard is the bare-`fab setup` entry point. Flow: non-TTY guard →
// write-target resolution → existing-system-scaffold refresh → probe/read model →
// scope banner → default-path questions → opt-in advanced section →
// diff-before-write summary → surgical writes.
func runSetupWizard(cmd *cobra.Command, opts wizardOptions) error {
	if !opts.defaults && !isStdinTTY(cmd.InOrStdin()) {
		return fmt.Errorf("stdin is not a TTY — use --defaults for non-interactive runs (or `fab setup check` for the read-only doctor)")
	}
	path, err := configMutationPath(!opts.project)
	if err != nil {
		if opts.project {
			return fmt.Errorf("--project targets this repo's fab/project/config.yaml: %w", err)
		}
		return err
	}
	refreshSystemScaffold(cmd)

	in := setupCheckInput()
	layers, defaults, err := wizardReadModel()
	if err != nil {
		return err
	}
	w := &setupWizard{
		cmd:      cmd,
		opts:     opts,
		path:     path,
		tmux:     in.TmuxEnv != "",
		report:   setupcheck.Run(in),
		layers:   layers,
		defaults: defaults,
		reader:   bufio.NewReader(cmd.InOrStdin()),
	}

	// The interview asks preference questions over probe-filtered options; an
	// empty option set would silently degrade a question to unvalidated
	// free-form input, contradicting "capability is detected, never asked".
	// Refuse up front with the read-only doctor as the remediation.
	if len(w.providerOptions()) == 0 {
		return fmt.Errorf("no agent providers detected on PATH — install a provider CLI first, then re-run (see `fab setup check` for the roster)")
	}
	if len(w.dispatchModeOptions()) == 0 {
		return fmt.Errorf("no viable dispatch mode detected — not inside tmux, and no detected provider is native- or headless-capable (see `fab setup check`)")
	}

	w.printBanner()
	w.askDefaultPath()
	w.askAdvanced()
	return w.diffAndWrite()
}

// refreshSystemScaffold self-heals an existing machine-level scaffold before
// the interview starts. A missing file remains absent so the wizard's all-Enter
// path keeps its zero-write invariant. Refresh failures are advisory: setup is
// still useful for repairing or replacing a bad value.
func refreshSystemScaffold(cmd *cobra.Command) {
	path, err := configMutationPath(true)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "fab: warning: could not refresh the system config scaffold: %v\n", err)
		return
	}
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(cmd.ErrOrStderr(), "fab: warning: could not inspect the system config scaffold: %v\n", err)
		}
		return
	}
	result, err := configupgrade.Upgrade(configupgrade.SystemTarget(path), version)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "fab: warning: could not refresh %s: %v\n", path, err)
		return
	}
	if result.Changed {
		fmt.Fprintf(cmd.OutOrStdout(), "Refreshed %s reference block (kit %s)\n", path, version)
	}
}

// wizardReadModel loads the config cascade once for the whole interview, so
// every question's current-value rendering reads one consistent snapshot.
// Outside a fab repo the project tier is absent ("" project path — the same
// degradation setupcheck.Run applies), leaving system+env+defaults.
func wizardReadModel() (*config.Layers, map[string]any, error) {
	projectPath := ""
	if fabRoot, err := resolve.FabRoot(); err == nil {
		projectPath = filepath.Join(fabRoot, "project", "config.yaml")
	}
	layers, err := config.LoadLayers(projectPath)
	if err != nil {
		return nil, nil, err
	}
	defaults, err := readModelDefaults(layers)
	if err != nil {
		return nil, nil, err
	}
	return layers, defaults, nil
}

// printBanner states the write target up front, before any question.
func (w *setupWizard) printBanner() {
	out := w.cmd.OutOrStdout()
	if w.opts.project {
		fmt.Fprintf(out, "Configuring the project tier (%s) — this repository only.\n", w.path)
	} else {
		fmt.Fprintf(out, "Configuring the system tier (%s) — machine-wide preferences. Use --project to target this repo instead.\n", w.path)
	}
	fmt.Fprintln(out)
}

// effectiveValue renders a key's current effective value with its provenance,
// descending the tier stack exactly as the merge does (the same shared descent
// the keyed --origin listing and warnIfShadowed use).
func (w *setupWizard) effectiveValue(key string) (value, origin, tier string) {
	descent := descendPath(cascadeTiers(w.defaults, w.layers, topLevelKey(key)), key)
	switch {
	case !descent.found:
		return "", "", ""
	case descent.ancestor != nil:
		t := *descent.ancestor
		return renderCompactValue(t.value), t.origin(descent.ancestorKey, w.layers.EnvOrigins), t.kind
	}
	defining := definingTiers(descent.tiers)
	if len(defining) == 0 {
		return "", "", ""
	}
	winner := defining[0]
	return renderCompactValue(winner.value), winner.origin(key, w.layers.EnvOrigins), winner.kind
}

// ask renders one question — text, numbered options, the current effective
// value with its origin as the default, and the `fab config explain` pointer —
// then reads the answer. Enter (or EOF) keeps the default, so an all-Enter run
// records zero changes. An option-listed question accepts the option's name or
// its number and re-asks on anything else; --defaults never reads stdin.
func (w *setupWizard) ask(q wizardQuestion) string {
	out := w.cmd.OutOrStdout()
	current, origin, tier := w.effectiveValue(q.key)
	display := current
	if q.inheritAs != "" && (tier == tierDefault || current == "") {
		// Not explicitly set anywhere — the winning row is the derived
		// built-in default, or the key resolves to no value at all (a knob
		// the derivation could not fill). Present the inherit indication over
		// an empty baseline: Enter records "" (== current, no write), while
		// any typed provider — including the currently-inherited one — is a
		// real change.
		current = ""
		origin = ""
		display = q.inheritAs
	}

	fmt.Fprintln(out, q.text)
	for i, opt := range q.options {
		annotation := ""
		if opt.annotation != "" {
			annotation = " " + opt.annotation
		}
		fmt.Fprintf(out, "  %d) %s%s\n", i+1, opt.value, annotation)
	}
	if origin == "" {
		fmt.Fprintf(out, "Default: %s\n", display)
	} else {
		fmt.Fprintf(out, "Default: %s (origin: %s)\n", display, origin)
	}
	fmt.Fprintf(out, "More: fab config explain %s\n", q.key)
	if w.opts.defaults {
		fmt.Fprintf(out, "%s: %s (accepted by --defaults)\n\n", q.key, display)
		w.answers = append(w.answers, wizardAnswer{key: q.key, current: current, currentDisplay: display, answer: current})
		return current
	}

	for {
		fmt.Fprintf(out, "%s [%s]: ", q.key, display)
		answer, ok := w.readLine()
		if !ok || answer == "" {
			answer = current // Enter or EOF keeps the current effective value
		} else if len(q.options) > 0 {
			resolved := ""
			for i, opt := range q.options {
				if answer == opt.value || answer == strconv.Itoa(i+1) {
					resolved = opt.value
					break
				}
			}
			if resolved == "" {
				fmt.Fprintf(out, "Unknown value %q — choose from: %s\n", answer, optionNames(q.options))
				continue
			}
			answer = resolved
		} else if q.validate != nil && !q.validate(answer) {
			fmt.Fprintf(out, "Invalid value %q for %s\n", answer, q.key)
			continue
		}
		fmt.Fprintln(out)
		w.answers = append(w.answers, wizardAnswer{key: q.key, current: current, currentDisplay: display, answer: answer})
		return answer
	}
}

// readLine reads one answer line; ok=false on EOF (the caller falls back to
// the default, so an exhausted stdin can never hang the interview). A non-EOF
// read error also reports ok=false, but is additionally recorded on w.readErr
// so diffAndWrite can refuse: a failing stdin degrades to a read-only run,
// never to an implicitly confirmed write.
func (w *setupWizard) readLine() (string, bool) {
	line, err := w.reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		if w.readErr == nil {
			w.readErr = err
		}
		return "", false
	}
	answer := strings.TrimSpace(line)
	return answer, err == nil || answer != ""
}

// confirm asks a yes/no question with the given default. EOF answers the
// default, matching ask's never-hang rule.
func (w *setupWizard) confirm(text string, defaultYes bool) bool {
	out := w.cmd.OutOrStdout()
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}
	if w.opts.defaults {
		fmt.Fprintf(out, "%s%s%c (accepted by --defaults)\n", text, suffix, yesNoRune(defaultYes))
		return defaultYes
	}
	for {
		fmt.Fprint(out, text, suffix)
		answer, ok := w.readLine()
		if !ok || answer == "" {
			fmt.Fprintln(out)
			return defaultYes
		}
		switch strings.ToLower(answer) {
		case "y", "yes":
			fmt.Fprintln(out)
			return true
		case "n", "no":
			fmt.Fprintln(out)
			return false
		default:
			fmt.Fprintf(out, "Please answer y or n.\n")
		}
	}
}

func yesNoRune(b bool) rune {
	if b {
		return 'y'
	}
	return 'n'
}

func optionNames(options []wizardOption) string {
	names := make([]string, 0, len(options))
	for _, opt := range options {
		names = append(names, opt.value)
	}
	return strings.Join(names, ", ")
}

// detectedProviders returns the roster rows whose executables are all on PATH
// — undetected providers are dropped outright, never offered.
func (w *setupWizard) detectedProviders() []setupcheck.ProviderProbe {
	var found []setupcheck.ProviderProbe
	for _, p := range w.report.Providers {
		if p.Found() {
			found = append(found, p)
		}
	}
	return found
}

// providerOptions renders the detected providers as annotated options, e.g.
// `claude (interactive, headless, native)`.
func (w *setupWizard) providerOptions() []wizardOption {
	var options []wizardOption
	for _, p := range w.detectedProviders() {
		var caps []string
		if p.Interactive {
			caps = append(caps, "interactive")
		}
		if p.Headless {
			caps = append(caps, "headless")
		}
		if p.Native {
			caps = append(caps, "native")
		}
		annotation := ""
		if len(caps) > 0 {
			annotation = "(" + strings.Join(caps, ", ") + ")"
		}
		options = append(options, wizardOption{value: p.Name, annotation: annotation})
	}
	return options
}

// dispatchModeOptions filters the ladder rungs by viability: pane only when
// the probe's tmux signal is present, native/headless only when some DETECTED
// provider declares the capability (offering an unrunnable rung would
// contradict "capability is detected, never asked").
func (w *setupWizard) dispatchModeOptions() []wizardOption {
	var options []wizardOption
	if w.tmux {
		options = append(options, wizardOption{value: "pane"})
	}
	native, headless := false, false
	for _, p := range w.detectedProviders() {
		native = native || p.Native
		headless = headless || p.Headless
	}
	if native {
		options = append(options, wizardOption{value: "native"})
	}
	if headless {
		options = append(options, wizardOption{value: "headless"})
	}
	return options
}

// askDefaultPath runs the 4-question default path: agent.session,
// agent.workers, dispatch.mode, and the advanced-section opt-in.
func (w *setupWizard) askDefaultPath() {
	providers := w.providerOptions()
	w.ask(wizardQuestion{
		key:     "agent.session",
		text:    "Q1: agent.session — provider for the session tier (the agents you talk to).",
		options: providers,
	})
	w.ask(wizardQuestion{
		key:     "agent.workers",
		text:    "Q2: agent.workers — provider for the workers tier (the agents stages dispatch to).",
		options: providers,
	})
	w.ask(wizardQuestion{
		key: "dispatch.mode",
		text: "Q3: dispatch.mode — stage-dispatch preference.\n" +
			"This is a preference CEILING over pane → native → headless: resolution descends from your choice and never ascends, so a setting this machine cannot reach degrades softly instead of erroring.",
		options: w.dispatchModeOptions(),
	})
}

// askAdvanced runs the opt-in advanced section (Q4). Opting in asks every
// advanced question — including keys sitting at their built-in default or
// never set at all — so first-time overrides are settable through the wizard.
// Enter keeps each key's current effective value, so an all-Enter pass writes
// nothing. The sparse agent.profiles keys have no built-in value; their empty
// current renders as the role's depth-correct inherit indication (operator is
// a session-depth role, review a workers-depth role).
func (w *setupWizard) askAdvanced() {
	if !w.confirm("Q4: Configure advanced options (agent.profiles.operator/review, dispatch.column_width, dispatch.reap_done)?", false) {
		return
	}
	providers := w.providerOptions()
	questions := []wizardQuestion{
		{key: "agent.profiles.operator.provider", text: "agent.profiles.operator.provider — provider override for the operator role.", options: providers, inheritAs: "(inherit agent.session)"},
		{key: "agent.profiles.review.provider", text: "agent.profiles.review.provider — provider override for the review role.", options: providers, inheritAs: "(inherit agent.workers)"},
		{key: "dispatch.column_width", text: "dispatch.column_width — pane-worker column width, in percent of the window.", validate: func(s string) bool {
			n, err := strconv.Atoi(s)
			return err == nil && n > 0 && n < 100
		}},
		{key: "dispatch.reap_done", text: "dispatch.reap_done — kill a done worker's pane once its result lands (true/false).", validate: func(s string) bool {
			_, err := strconv.ParseBool(s)
			return err == nil
		}},
	}
	for _, q := range questions {
		w.ask(q)
	}
}

// diffAndWrite renders the diff-before-write summary and applies confirmed
// changes through the surgical config-set path. Zero changed answers means this
// step writes nothing; the command's earlier system-scaffold warm-up may already
// have refreshed an existing stale file (Constitution III).
func (w *setupWizard) diffAndWrite() error {
	out := w.cmd.OutOrStdout()
	var changes []wizardAnswer
	for _, a := range w.answers {
		if a.answer != a.current {
			changes = append(changes, a)
		}
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "nothing to change — current configuration already matches your answers")
		return nil
	}

	fmt.Fprintf(out, "Change summary (target: %s tier — %s):\n", mutationTier(!w.opts.project), w.path)
	for _, c := range changes {
		fmt.Fprintf(out, "  %s: %s → %s\n", c.key, c.currentDisplay, c.answer)
	}
	if !w.confirm("Write these changes?", true) {
		fmt.Fprintln(out, "No changes written.")
		return nil
	}
	// A non-EOF stdin failure anywhere in the interview means the answers —
	// and the write confirmation itself, whose default is Yes — may be error
	// fallbacks rather than choices. Refuse to write on a broken stdin.
	if w.readErr != nil {
		return fmt.Errorf("stdin read failed during the interview: %w — no changes written; re-run `fab setup`", w.readErr)
	}
	for _, c := range changes {
		if err := w.writeOne(c); err != nil {
			return err
		}
	}
	return nil
}

// writeOne applies one confirmed change in-process through the same surgical,
// fence-aware write `fab config set [--system]` performs, then runs the shared
// shadow warning so a write shadowed by a higher tier says so.
func (w *setupWizard) writeOne(c wizardAnswer) error {
	out := w.cmd.OutOrStdout()
	var result configupgrade.Result
	var err error
	if w.opts.project {
		result, err = configupgrade.Set(w.path, c.key, c.answer, version)
	} else {
		result, err = configupgrade.SetSystem(w.path, c.key, c.answer, version)
	}
	if err != nil {
		return err
	}
	if result.Changed {
		fmt.Fprintf(out, "Set %s in %s\n", c.key, w.path)
	} else {
		fmt.Fprintf(out, "%s already has that value in %s\n", c.key, w.path)
	}
	for _, line := range result.Report {
		fmt.Fprintf(out, "  - %s\n", line)
	}
	warnIfShadowed(w.cmd, c.key, mutationTier(!w.opts.project))
	return nil
}
