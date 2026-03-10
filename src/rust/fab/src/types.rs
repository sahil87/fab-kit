/// Ordered stage list — pipeline order.
pub const STAGE_ORDER: &[&str] = &[
    "intake", "spec", "tasks", "apply", "review", "hydrate", "ship", "review-pr",
];

/// Returns the 1-indexed position of a stage.
pub fn stage_number(stage: &str) -> usize {
    for (i, s) in STAGE_ORDER.iter().enumerate() {
        if *s == stage {
            return i + 1;
        }
    }
    0
}

/// Returns the next stage in the pipeline, or None if at the end.
pub fn next_stage(stage: &str) -> Option<&'static str> {
    for (i, s) in STAGE_ORDER.iter().enumerate() {
        if *s == stage && i + 1 < STAGE_ORDER.len() {
            return Some(STAGE_ORDER[i + 1]);
        }
    }
    None
}

/// Valid change types.
pub const VALID_CHANGE_TYPES: &[&str] = &["feat", "fix", "refactor", "docs", "test", "ci", "chore"];

/// Valid states.
pub const VALID_STATES: &[&str] = &["pending", "active", "ready", "done", "failed", "skipped"];

/// Allowed states per stage.
pub fn allowed_states(stage: &str) -> Option<&'static [&'static str]> {
    match stage {
        "intake" => Some(&["active", "ready", "done"]),
        "spec" => Some(&["pending", "active", "ready", "done", "skipped"]),
        "tasks" => Some(&["pending", "active", "ready", "done", "skipped"]),
        "apply" => Some(&["pending", "active", "ready", "done", "skipped"]),
        "review" => Some(&["pending", "active", "ready", "done", "failed", "skipped"]),
        "hydrate" => Some(&["pending", "active", "ready", "done", "skipped"]),
        "ship" => Some(&["pending", "active", "done", "skipped"]),
        "review-pr" => Some(&["pending", "active", "done", "failed", "skipped"]),
        _ => None,
    }
}

pub fn is_valid_stage(stage: &str) -> bool {
    STAGE_ORDER.contains(&stage)
}

pub fn is_valid_state(state: &str) -> bool {
    VALID_STATES.contains(&state)
}
