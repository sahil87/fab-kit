use anyhow::{bail, Result};

use crate::config;
use crate::hooks;
use crate::log;
use crate::statusfile::{Confidence, Dimensions, StageMetric, StageState, StatusFile};
use crate::types::{self, STAGE_ORDER, VALID_CHANGE_TYPES};

struct Transition {
    from: Vec<&'static str>,
    to: &'static str,
}

fn default_transitions(event: &str) -> Option<Transition> {
    match event {
        "start" => Some(Transition { from: vec!["pending"], to: "active" }),
        "advance" => Some(Transition { from: vec!["active"], to: "ready" }),
        "finish" => Some(Transition { from: vec!["active", "ready"], to: "done" }),
        "reset" => Some(Transition { from: vec!["done", "ready", "skipped"], to: "active" }),
        "skip" => Some(Transition { from: vec!["pending", "active"], to: "skipped" }),
        _ => None,
    }
}

fn stage_transitions(stage: &str, event: &str) -> Option<Transition> {
    match (stage, event) {
        ("review", "start") | ("review-pr", "start") => {
            Some(Transition { from: vec!["pending", "failed"], to: "active" })
        }
        ("review", "advance") | ("review-pr", "advance") => {
            Some(Transition { from: vec!["active"], to: "ready" })
        }
        ("review", "finish") | ("review-pr", "finish") => {
            Some(Transition { from: vec!["active", "ready"], to: "done" })
        }
        ("review", "reset") | ("review-pr", "reset") => {
            Some(Transition { from: vec!["done", "ready", "skipped"], to: "active" })
        }
        ("review", "fail") | ("review-pr", "fail") => {
            Some(Transition { from: vec!["active"], to: "failed" })
        }
        _ => None,
    }
}

fn lookup_transition(event: &str, stage: &str, current_state: &str) -> Result<String> {
    // Check stage-specific overrides first
    if let Some(t) = stage_transitions(stage, event) {
        if t.from.contains(&current_state) {
            return Ok(t.to.to_string());
        }
        bail!(
            "Cannot {} stage '{}' — current state is '{}', no valid transition",
            event, stage, current_state
        );
    }

    // Check default transitions
    if let Some(t) = default_transitions(event) {
        if t.from.contains(&current_state) {
            return Ok(t.to.to_string());
        }
    }

    bail!(
        "Cannot {} stage '{}' — current state is '{}', no valid transition",
        event, stage, current_state
    );
}

/// Start transitions a stage from {pending,failed} to active.
pub fn start(
    sf: &mut StatusFile,
    status_path: &str,
    fab_root: &str,
    stage: &str,
    driver: &str,
    from: &str,
    reason: &str,
) -> Result<()> {
    if !types::is_valid_stage(stage) {
        bail!("Invalid stage '{}'", stage);
    }

    let current_state = sf.get_progress(stage);
    let target_state = lookup_transition("start", stage, &current_state)?;

    // Run pre hook before transitioning
    run_stage_hook(fab_root, stage, "pre")?;

    sf.set_progress(stage, &target_state);
    apply_metrics_side_effect(sf, fab_root, stage, &target_state, driver, from, reason);

    sf.save(status_path)
}

/// Advance transitions a stage from active to ready.
pub fn advance(
    sf: &mut StatusFile,
    status_path: &str,
    stage: &str,
    _driver: &str,
) -> Result<()> {
    if !types::is_valid_stage(stage) {
        bail!("Invalid stage '{}'", stage);
    }

    let current_state = sf.get_progress(stage);
    let target_state = lookup_transition("advance", stage, &current_state)?;

    sf.set_progress(stage, &target_state);

    sf.save(status_path)
}

/// Finish transitions a stage to done and auto-activates the next pending stage.
pub fn finish(
    sf: &mut StatusFile,
    status_path: &str,
    fab_root: &str,
    stage: &str,
    driver: &str,
) -> Result<()> {
    if !types::is_valid_stage(stage) {
        bail!("Invalid stage '{}'", stage);
    }

    let current_state = sf.get_progress(stage);
    let target_state = lookup_transition("finish", stage, &current_state)?;

    sf.set_progress(stage, &target_state);
    apply_metrics_side_effect(sf, fab_root, stage, &target_state, "", "", "");

    // Auto-activate next pending stage
    if let Some(next_stage) = types::next_stage(stage) {
        let next_state = sf.get_progress(next_stage);
        if next_state == "pending" {
            sf.set_progress(next_stage, "active");
            apply_metrics_side_effect(sf, fab_root, next_stage, "active", driver, "", "");
        }
    }

    sf.save(status_path)?;

    // Run post hook after transition is saved
    run_stage_hook(fab_root, stage, "post")?;

    // Auto-log review/review-pr pass
    if stage == "review" || stage == "review-pr" {
        let _ = log::review(fab_root, &sf.name, "passed", "");
    }

    Ok(())
}

/// Reset transitions a stage to active and cascades downstream to pending.
pub fn reset(
    sf: &mut StatusFile,
    status_path: &str,
    fab_root: &str,
    stage: &str,
    driver: &str,
    from: &str,
    reason: &str,
) -> Result<()> {
    if !types::is_valid_stage(stage) {
        bail!("Invalid stage '{}'", stage);
    }

    let current_state = sf.get_progress(stage);
    let target_state = lookup_transition("reset", stage, &current_state)?;

    sf.set_progress(stage, &target_state);
    apply_metrics_side_effect(sf, fab_root, stage, &target_state, driver, from, reason);

    // Cascade downstream to pending
    let mut found_target = false;
    for s in STAGE_ORDER {
        if found_target {
            sf.set_progress(s, "pending");
            apply_metrics_side_effect(sf, fab_root, s, "pending", "", "", "");
        }
        if *s == stage {
            found_target = true;
        }
    }

    sf.save(status_path)
}

/// Skip transitions a stage to skipped and cascades downstream pending to skipped.
pub fn skip(
    sf: &mut StatusFile,
    status_path: &str,
    fab_root: &str,
    stage: &str,
    _driver: &str,
) -> Result<()> {
    if !types::is_valid_stage(stage) {
        bail!("Invalid stage '{}'", stage);
    }

    let current_state = sf.get_progress(stage);
    let target_state = lookup_transition("skip", stage, &current_state)?;

    sf.set_progress(stage, &target_state);
    apply_metrics_side_effect(sf, fab_root, stage, &target_state, "", "", "");

    // Forward cascade: downstream pending → skipped
    let mut found_target = false;
    for s in STAGE_ORDER {
        if found_target {
            if sf.get_progress(s) == "pending" {
                sf.set_progress(s, "skipped");
                apply_metrics_side_effect(sf, fab_root, s, "skipped", "", "", "");
            }
        }
        if *s == stage {
            found_target = true;
        }
    }

    sf.save(status_path)
}

/// Fail transitions a stage to failed (review/review-pr only).
pub fn fail(
    sf: &mut StatusFile,
    status_path: &str,
    fab_root: &str,
    stage: &str,
    _driver: &str,
    rework: &str,
) -> Result<()> {
    if !types::is_valid_stage(stage) {
        bail!("Invalid stage '{}'", stage);
    }

    let current_state = sf.get_progress(stage);
    let target_state = lookup_transition("fail", stage, &current_state)?;

    sf.set_progress(stage, &target_state);

    sf.save(status_path)?;

    // Auto-log review/review-pr failure
    if stage == "review" || stage == "review-pr" {
        let _ = log::review(fab_root, &sf.name, "failed", rework);
    }

    Ok(())
}

/// Sets the change_type field.
pub fn set_change_type(sf: &mut StatusFile, status_path: &str, change_type: &str) -> Result<()> {
    if !VALID_CHANGE_TYPES.contains(&change_type) {
        bail!(
            "Invalid change type '{}' (valid: {})",
            change_type,
            VALID_CHANGE_TYPES.join(", ")
        );
    }
    sf.change_type = change_type.to_string();
    sf.save(status_path)
}

/// Updates a checklist field.
pub fn set_checklist(
    sf: &mut StatusFile,
    status_path: &str,
    field: &str,
    value: &str,
) -> Result<()> {
    match field {
        "generated" => {
            if value != "true" && value != "false" {
                bail!(
                    "Invalid value '{}' for field 'generated' (expected true/false)",
                    value
                );
            }
            sf.checklist.generated = value == "true";
        }
        "completed" => {
            let n: i64 = parse_non_negative_int(value).map_err(|_| {
                anyhow::anyhow!(
                    "Invalid value '{}' for field 'completed' (expected non-negative integer)",
                    value
                )
            })?;
            sf.checklist.completed = n;
        }
        "total" => {
            let n: i64 = parse_non_negative_int(value).map_err(|_| {
                anyhow::anyhow!(
                    "Invalid value '{}' for field 'total' (expected non-negative integer)",
                    value
                )
            })?;
            sf.checklist.total = n;
        }
        _ => {
            bail!(
                "Invalid checklist field '{}' (expected: generated, completed, total)",
                field
            );
        }
    }
    sf.save(status_path)
}

/// Replaces the confidence block.
pub fn set_confidence(
    sf: &mut StatusFile,
    status_path: &str,
    certain: i64,
    confident: i64,
    tentative: i64,
    unresolved: i64,
    score: f64,
    indicative: bool,
) -> Result<()> {
    sf.confidence = Confidence {
        certain,
        confident,
        tentative,
        unresolved,
        score,
        indicative: if indicative { Some(true) } else { None },
        fuzzy: None,
        dimensions: None,
    };
    sf.save(status_path)
}

/// Replaces the confidence block with dimension data.
pub fn set_confidence_fuzzy(
    sf: &mut StatusFile,
    status_path: &str,
    certain: i64,
    confident: i64,
    tentative: i64,
    unresolved: i64,
    score: f64,
    mean_s: f64,
    mean_r: f64,
    mean_a: f64,
    mean_d: f64,
    indicative: bool,
) -> Result<()> {
    sf.confidence = Confidence {
        certain,
        confident,
        tentative,
        unresolved,
        score,
        indicative: if indicative { Some(true) } else { None },
        fuzzy: Some(true),
        dimensions: Some(Dimensions {
            signal: mean_s,
            reversibility: mean_r,
            competence: mean_a,
            disambiguation: mean_d,
        }),
    };
    sf.save(status_path)
}

/// Appends an issue ID (idempotent).
pub fn add_issue(sf: &mut StatusFile, status_path: &str, id: &str) -> Result<()> {
    if !sf.issues.contains(&id.to_string()) {
        sf.issues.push(id.to_string());
    }
    sf.save(status_path) // always refresh last_updated
}

/// Appends a PR URL (idempotent).
pub fn add_pr(sf: &mut StatusFile, status_path: &str, url: &str) -> Result<()> {
    if !sf.prs.contains(&url.to_string()) {
        sf.prs.push(url.to_string());
    }
    sf.save(status_path) // always refresh last_updated
}

/// Returns stage:state pairs in pipeline order.
pub fn progress_map(sf: &StatusFile) -> Vec<StageState> {
    sf.get_progress_map()
}

/// Returns a single-line visual progress string.
pub fn progress_line(sf: &StatusFile) -> String {
    let mut parts = Vec::new();
    let mut has_active = false;
    let mut has_pending = false;

    for ss in sf.get_progress_map() {
        match ss.state.as_str() {
            "done" => parts.push(ss.stage.clone()),
            "active" => {
                parts.push(format!("{} \u{23f3}", ss.stage));
                has_active = true;
            }
            "ready" => parts.push(format!("{} \u{25f7}", ss.stage)),
            "failed" => parts.push(format!("{} \u{2717}", ss.stage)),
            "skipped" => parts.push(format!("{} \u{23ed}", ss.stage)),
            "pending" => {
                has_pending = true;
            }
            _ => {}
        }
    }

    if parts.is_empty() {
        return String::new();
    }

    let mut line = parts.join(" \u{2192} ");
    if !has_active && !has_pending {
        line.push_str(" \u{2713}");
    }
    line
}

/// Determines the active/next stage.
pub fn current_stage(sf: &StatusFile) -> String {
    let pm = sf.get_progress_map();

    // First active or ready
    for ss in &pm {
        if ss.state == "active" || ss.state == "ready" {
            return ss.stage.clone();
        }
    }

    // Fallback: first pending after last done/skipped
    let mut last_done = String::new();
    for ss in &pm {
        if ss.state == "done" || ss.state == "skipped" {
            last_done = ss.stage.clone();
        }
    }

    if !last_done.is_empty() {
        let mut found_last = false;
        for ss in &pm {
            if found_last && ss.state == "pending" {
                return ss.stage.clone();
            }
            if ss.stage == last_done {
                found_last = true;
            }
        }
    }

    "review-pr".to_string() // all done
}

/// Returns the display stage and state.
pub fn display_stage(sf: &StatusFile) -> (String, String) {
    let pm = sf.get_progress_map();

    // Tier 1: first active
    for ss in &pm {
        if ss.state == "active" {
            return (ss.stage.clone(), "active".to_string());
        }
    }

    // Tier 2: first ready
    for ss in &pm {
        if ss.state == "ready" {
            return (ss.stage.clone(), "ready".to_string());
        }
    }

    // Tier 3: last done/skipped
    let mut last_done = String::new();
    let mut last_done_state = String::new();
    for ss in &pm {
        if ss.state == "done" || ss.state == "skipped" {
            last_done = ss.stage.clone();
            last_done_state = ss.state.clone();
        }
    }
    if !last_done.is_empty() {
        return (last_done, last_done_state);
    }

    // Tier 4: first pending
    if !STAGE_ORDER.is_empty() {
        return (STAGE_ORDER[0].to_string(), "pending".to_string());
    }
    ("intake".to_string(), "pending".to_string())
}

/// Validates a .status.yaml against the schema.
pub fn validate(sf: &StatusFile) -> Result<()> {
    let mut active_count = 0;
    let mut errors = Vec::new();

    for stage in STAGE_ORDER {
        let state = sf.get_progress(stage);
        let state = if state.is_empty() { "pending".to_string() } else { state };

        if !types::is_valid_state(&state) {
            errors.push(format!("Invalid state '{}' for stage {}", state, stage));
            continue;
        }

        if let Some(allowed) = types::allowed_states(stage) {
            if !allowed.contains(&state.as_str()) {
                errors.push(format!("State '{}' not allowed for stage {}", state, stage));
            }
        }

        if state == "active" {
            active_count += 1;
        }
    }

    if active_count > 1 {
        errors.push("Multiple stages are active (expected 0 or 1)".to_string());
    }

    if !errors.is_empty() {
        bail!("{}", errors.join("; "));
    }
    Ok(())
}

/// Returns all stage IDs in pipeline order.
pub fn all_stages() -> &'static [&'static str] {
    STAGE_ORDER
}

fn apply_metrics_side_effect(
    sf: &mut StatusFile,
    fab_root: &str,
    stage: &str,
    state: &str,
    driver: &str,
    from: &str,
    reason: &str,
) {
    let now = chrono::Utc::now().to_rfc3339();

    match state {
        "active" => {
            let sm = sf
                .stage_metrics
                .entry(stage.to_string())
                .or_insert_with(|| StageMetric {
                    started_at: String::new(),
                    driver: String::new(),
                    iterations: 0,
                    completed_at: String::new(),
                });
            sm.iterations += 1;
            sm.started_at = now;
            sm.driver = driver.to_string();
            sm.completed_at = String::new();

            // Log transition (best-effort)
            let folder = &sf.name;
            let action = if sm.iterations > 1 { "re-entry" } else { "enter" };
            let _ = log::transition(fab_root, folder, stage, action, from, reason, driver);
        }
        "done" => {
            if let Some(sm) = sf.stage_metrics.get_mut(stage) {
                sm.completed_at = now;
            }
        }
        "pending" | "skipped" => {
            sf.stage_metrics.remove(stage);
        }
        _ => {}
    }
}

fn run_stage_hook(fab_root: &str, stage: &str, phase: &str) -> Result<()> {
    let cfg = config::load(fab_root)?;
    let hook = cfg.get_stage_hook(stage);
    let command = match phase {
        "pre" => &hook.pre,
        "post" => &hook.post,
        _ => return Ok(()),
    };
    hooks::run(fab_root, command)
}

fn parse_non_negative_int(s: &str) -> Result<i64, ()> {
    if s.is_empty() {
        return Err(());
    }
    for c in s.chars() {
        if !c.is_ascii_digit() {
            return Err(());
        }
    }
    s.parse::<i64>().map_err(|_| ())
}
