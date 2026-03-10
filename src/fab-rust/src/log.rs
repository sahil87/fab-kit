use anyhow::Result;
use serde_json::json;
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};

use crate::resolve;

/// Logs a skill invocation.
pub fn command(fab_root: &str, cmd: &str, change_arg: &str, args: &str) -> Result<()> {
    let change_dir;

    if !change_arg.is_empty() {
        change_dir = resolve::to_abs_dir(fab_root, change_arg)?;
    } else {
        // No change arg: resolve from .fab-status.yaml symlink, graceful degradation
        let repo_root = Path::new(fab_root).parent().unwrap_or(Path::new("/"));
        let symlink_path = repo_root.join(".fab-status.yaml");
        if symlink_path.symlink_metadata().is_err() {
            return Ok(()); // silent exit
        }
        match resolve::to_abs_dir(fab_root, "") {
            Ok(dir) => {
                if !Path::new(&dir).exists() {
                    return Ok(()); // silent exit
                }
                change_dir = dir;
            }
            Err(_) => return Ok(()), // silent exit
        }
    }

    let mut entry = json!({
        "ts": now_iso(),
        "event": "command",
        "cmd": cmd,
    });
    if !args.is_empty() {
        entry["args"] = json!(args);
    }

    append_json(&change_dir, &entry)
}

/// Logs a confidence score change.
pub fn confidence_log(
    fab_root: &str,
    change_arg: &str,
    score: f64,
    delta: &str,
    trigger: &str,
) -> Result<()> {
    let change_dir = resolve::to_abs_dir(fab_root, change_arg)?;

    let entry = json!({
        "ts": now_iso(),
        "event": "confidence",
        "score": score,
        "delta": delta,
        "trigger": trigger,
    });

    append_json(&change_dir, &entry)
}

/// Logs a review outcome.
pub fn review(fab_root: &str, change_arg: &str, result: &str, rework: &str) -> Result<()> {
    let change_dir = resolve::to_abs_dir(fab_root, change_arg)?;

    let mut entry = json!({
        "ts": now_iso(),
        "event": "review",
        "result": result,
    });
    if !rework.is_empty() {
        entry["rework"] = json!(rework);
    }

    append_json(&change_dir, &entry)
}

/// Logs a stage transition.
pub fn transition(
    fab_root: &str,
    change_arg: &str,
    stage: &str,
    action: &str,
    from: &str,
    reason: &str,
    driver: &str,
) -> Result<()> {
    let change_dir = resolve::to_abs_dir(fab_root, change_arg)?;

    let mut entry = json!({
        "ts": now_iso(),
        "event": "stage-transition",
        "stage": stage,
        "action": action,
    });
    if !from.is_empty() {
        entry["from"] = json!(from);
    }
    if !reason.is_empty() {
        entry["reason"] = json!(reason);
    }
    if !driver.is_empty() {
        entry["driver"] = json!(driver);
    }

    append_json(&change_dir, &entry)
}

fn append_json(change_dir: &str, entry: &serde_json::Value) -> Result<()> {
    let history_file = PathBuf::from(change_dir).join(".history.jsonl");

    let data = serde_json::to_string(entry)?;

    // Ensure directory exists
    if let Some(parent) = history_file.parent() {
        fs::create_dir_all(parent)?;
    }

    let mut f = OpenOptions::new()
        .append(true)
        .create(true)
        .open(&history_file)?;

    writeln!(f, "{}", data)?;

    Ok(())
}

fn now_iso() -> String {
    chrono::Local::now().to_rfc3339()
}
