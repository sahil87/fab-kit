use anyhow::{bail, Result};
use std::fs;
use std::path::PathBuf;
use std::process::Command;

use crate::resolve;

pub fn run_send_keys(change_arg: &str, text: &str) -> Result<()> {
    // Validate preconditions
    validate_send_keys_inputs(change_arg)?;

    // Resolve the change argument to a folder name
    let fab_root = resolve::fab_root()?;
    let folder = resolve::to_folder(&fab_root, change_arg)?;

    // Discover all tmux panes and find the one matching this change
    let (pane_id, warning) = resolve_change_pane(&folder)?;

    if !warning.is_empty() {
        eprintln!("{}", warning);
    }

    // Send keys to the resolved pane
    let status = Command::new("tmux")
        .args(["send-keys", "-t", &pane_id, text, "Enter"])
        .status()?;

    if !status.success() {
        bail!("Error: failed to send keys to pane {}", pane_id);
    }

    Ok(())
}

fn validate_send_keys_inputs(change_arg: &str) -> Result<()> {
    if std::env::var("TMUX").unwrap_or_default().is_empty() {
        eprintln!("Error: not inside a tmux session");
        std::process::exit(1);
    }
    if change_arg.trim().is_empty() {
        bail!("change argument must not be empty");
    }
    Ok(())
}

fn resolve_change_pane(folder: &str) -> Result<(String, String)> {
    let panes = discover_panes()?;

    let mut matches = Vec::new();
    for p in &panes {
        if resolve_pane_change(p) == folder {
            matches.push(p.id.clone());
        }
    }

    if matches.is_empty() {
        bail!("No tmux pane found for change {:?}.", folder);
    }

    let warning = if matches.len() > 1 {
        format!(
            "Warning: multiple panes found for {}, using {}",
            resolve::extract_id(folder),
            matches[0]
        )
    } else {
        String::new()
    };

    Ok((matches[0].clone(), warning))
}

struct PaneEntry {
    id: String,
    cwd: String,
}

fn discover_panes() -> Result<Vec<PaneEntry>> {
    let output = Command::new("tmux")
        .args(["list-panes", "-a", "-F", "#{pane_id} #{pane_current_path}"])
        .output()?;

    if !output.status.success() {
        bail!("tmux list-panes: {}", String::from_utf8_lossy(&output.stderr));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut panes = Vec::new();
    for line in stdout.trim().lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.splitn(2, ' ').collect();
        if parts.len() != 2 {
            continue;
        }
        panes.push(PaneEntry {
            id: parts[0].to_string(),
            cwd: parts[1].to_string(),
        });
    }
    Ok(panes)
}

fn resolve_pane_change(p: &PaneEntry) -> String {
    let wt_root = match git_worktree_root(&p.cwd) {
        Some(r) => r,
        None => return String::new(),
    };

    let fab_dir = PathBuf::from(&wt_root).join("fab");
    if !fab_dir.exists() {
        return String::new();
    }

    let symlink_path = PathBuf::from(&wt_root).join(".fab-status.yaml");
    match fs::read_link(&symlink_path) {
        Ok(target) => {
            let target_str = target.to_string_lossy().to_string();
            resolve::extract_folder_from_symlink(&target_str)
        }
        Err(_) => String::new(),
    }
}

fn git_worktree_root(dir: &str) -> Option<String> {
    let output = Command::new("git")
        .args(["-C", dir, "rev-parse", "--show-toplevel"])
        .output()
        .ok()?;

    if !output.status.success() {
        return None;
    }

    Some(
        String::from_utf8_lossy(&output.stdout)
            .trim()
            .to_string(),
    )
}
