use anyhow::{bail, Result};
use serde::Serialize;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

use crate::status;
use crate::statusfile;

/// Info holds the fab pipeline state for a single worktree.
#[derive(Clone, Debug, Serialize)]
pub struct Info {
    pub name: String,
    pub path: String,
    pub branch: String,
    pub is_main: bool,
    pub is_current: bool,
    pub change: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub stage: String,
    #[serde(skip_serializing_if = "String::is_empty")]
    pub state: String,
}

struct WorktreeEntry {
    path: String,
    branch: String,
}

/// Discovers all git worktrees and resolves their fab pipeline state.
pub fn list() -> Result<Vec<Info>> {
    let current_dir = std::env::current_dir()?;
    let current_dir = fs::canonicalize(&current_dir).unwrap_or(current_dir);
    let current_str = current_dir.to_string_lossy().to_string();

    let output = Command::new("git")
        .args(["worktree", "list", "--porcelain"])
        .output()?;
    if !output.status.success() {
        bail!("git worktree list: {}", String::from_utf8_lossy(&output.stderr));
    }

    let raw = String::from_utf8_lossy(&output.stdout).to_string();
    let entries = parse_worktree_list(&raw);

    // First entry is always the main worktree
    let main_path = entries.first().map(|e| e.path.clone()).unwrap_or_default();

    let mut infos = Vec::new();
    for e in &entries {
        let is_current = current_str == e.path
            || current_str.starts_with(&format!("{}/", e.path));

        let mut info = Info {
            path: e.path.clone(),
            branch: e.branch.clone(),
            is_current,
            is_main: e.path == main_path,
            name: if e.path == main_path {
                "(main)".to_string()
            } else {
                Path::new(&e.path)
                    .file_name()
                    .map(|n| n.to_string_lossy().to_string())
                    .unwrap_or_default()
            },
            change: String::new(),
            stage: String::new(),
            state: String::new(),
        };

        resolve_fab_state(&mut info);
        infos.push(info);
    }

    Ok(infos)
}

/// Returns the worktree matching the given name.
pub fn find_by_name(name: &str) -> Result<Info> {
    let all = list()?;
    let name_lower = name.to_lowercase();
    for info in all {
        if info.name.to_lowercase() == name_lower {
            return Ok(info);
        }
    }
    bail!("Worktree '{}' not found", name);
}

/// Returns the worktree for the current working directory.
pub fn current() -> Result<Info> {
    let all = list()?;
    for info in all {
        if info.is_current {
            return Ok(info);
        }
    }
    bail!("current directory is not a git worktree");
}

/// Formats a single worktree as a human-readable line.
pub fn format_human(info: &Info) -> String {
    if !info.stage.is_empty() {
        format!("{}  {}  {}  {}", info.name, info.change, info.stage, info.state)
    } else {
        format!("{}  {}", info.name, info.change)
    }
}

/// Formats all worktrees as a human-readable table.
pub fn format_all_human(infos: &[Info]) -> String {
    let mut repo_name = String::new();
    let mut wt_dir = String::new();

    for info in infos {
        if info.is_main {
            repo_name = Path::new(&info.path)
                .file_name()
                .map(|n| n.to_string_lossy().to_string())
                .unwrap_or_default();
            wt_dir = Path::new(&info.path)
                .parent()
                .map(|p| p.to_string_lossy().to_string())
                .unwrap_or_default();
            break;
        }
    }
    if repo_name.is_empty() && !infos.is_empty() {
        repo_name = Path::new(&infos[0].path)
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_default();
        wt_dir = Path::new(&infos[0].path)
            .parent()
            .map(|p| p.to_string_lossy().to_string())
            .unwrap_or_default();
    }

    let mut sb = String::new();
    sb.push_str(&format!("Worktrees for: {}\n", repo_name));
    sb.push_str(&format!("Location: {}\n\n", wt_dir));

    for info in infos {
        let marker = if info.is_current { "* " } else { "  " };
        if !info.stage.is_empty() {
            sb.push_str(&format!(
                "{}{:<14} {:<42} {:<10} {}\n",
                marker, info.name, info.change, info.stage, info.state
            ));
        } else {
            sb.push_str(&format!("{}{:<14} {}\n", marker, info.name, info.change));
        }
    }

    sb.push_str(&format!("\nTotal: {} worktree(s)", infos.len()));
    sb
}

/// Formats a single worktree as JSON.
pub fn format_json(info: &Info) -> Result<String> {
    Ok(serde_json::to_string_pretty(info)?)
}

/// Formats all worktrees as a JSON array.
pub fn format_all_json(infos: &[Info]) -> Result<String> {
    Ok(serde_json::to_string_pretty(infos)?)
}

fn parse_worktree_list(raw: &str) -> Vec<WorktreeEntry> {
    let mut entries = Vec::new();
    let mut current = WorktreeEntry {
        path: String::new(),
        branch: String::new(),
    };

    for line in raw.lines() {
        if let Some(path) = line.strip_prefix("worktree ") {
            if !current.path.is_empty() {
                entries.push(current);
            }
            current = WorktreeEntry {
                path: path.to_string(),
                branch: String::new(),
            };
        } else if let Some(branch) = line.strip_prefix("branch refs/heads/") {
            current.branch = branch.to_string();
        }
    }
    if !current.path.is_empty() {
        entries.push(current);
    }
    entries
}

fn resolve_fab_state(info: &mut Info) {
    let fab_dir = PathBuf::from(&info.path).join("fab");
    if !fab_dir.exists() {
        info.change = "(no fab)".to_string();
        return;
    }

    let current_file = fab_dir.join("current");
    let data = match fs::read_to_string(&current_file) {
        Ok(d) if !d.trim().is_empty() => d,
        _ => {
            info.change = "(no change)".to_string();
            return;
        }
    };

    let lines: Vec<&str> = data.trim().lines().collect();
    let folder_name = if lines.len() >= 2 {
        lines[1].trim().to_string()
    } else {
        String::new()
    };

    if folder_name.is_empty() {
        info.change = "(no change)".to_string();
        return;
    }

    let status_path = fab_dir
        .join("changes")
        .join(&folder_name)
        .join(".status.yaml");
    if !status_path.exists() {
        info.change = "(stale)".to_string();
        return;
    }

    info.change = folder_name;

    if let Ok(sf) = statusfile::load(&status_path.to_string_lossy()) {
        let (stage, state) = status::display_stage(&sf);
        info.stage = stage;
        info.state = state;
    }
}
