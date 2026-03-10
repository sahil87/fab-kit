use anyhow::{bail, Result};
use std::fs;
use std::path::{Path, PathBuf};

/// Searches upward from cwd to find the fab/ directory.
pub fn fab_root() -> Result<String> {
    let mut dir = std::env::current_dir()?;
    loop {
        let candidate = dir.join("fab");
        if candidate.is_dir() {
            return Ok(candidate.to_string_lossy().to_string());
        }
        if !dir.pop() {
            bail!("fab/ directory not found");
        }
    }
}

/// Resolves a change reference to a full folder name.
/// If override_arg is empty, reads .fab-status.yaml symlink at repo root.
pub fn to_folder(fab_root: &str, override_arg: &str) -> Result<String> {
    let changes_dir = PathBuf::from(fab_root).join("changes");

    if !override_arg.is_empty() {
        return resolve_override(&changes_dir, override_arg);
    }
    resolve_from_current(fab_root, &changes_dir)
}

/// Extracts the 4-char change ID from a YYMMDD-XXXX-slug folder name.
pub fn extract_id(folder: &str) -> String {
    let parts: Vec<&str> = folder.splitn(3, '-').collect();
    if parts.len() >= 2 {
        parts[1].to_string()
    } else {
        String::new()
    }
}

/// Returns the directory path relative to repo root.
#[allow(dead_code)]
pub fn to_dir(fab_root: &str, override_arg: &str) -> Result<String> {
    let folder = to_folder(fab_root, override_arg)?;
    Ok(format!("fab/changes/{}/", folder))
}

/// Returns the .status.yaml path relative to repo root.
#[allow(dead_code)]
pub fn to_status(fab_root: &str, override_arg: &str) -> Result<String> {
    let folder = to_folder(fab_root, override_arg)?;
    Ok(format!("fab/changes/{}/.status.yaml", folder))
}

/// Returns the absolute directory path.
pub fn to_abs_dir(fab_root: &str, override_arg: &str) -> Result<String> {
    let folder = to_folder(fab_root, override_arg)?;
    let path = PathBuf::from(fab_root).join("changes").join(&folder);
    Ok(path.to_string_lossy().to_string())
}

/// Returns the absolute .status.yaml path.
pub fn to_abs_status(fab_root: &str, override_arg: &str) -> Result<String> {
    let folder = to_folder(fab_root, override_arg)?;
    let path = PathBuf::from(fab_root)
        .join("changes")
        .join(&folder)
        .join(".status.yaml");
    Ok(path.to_string_lossy().to_string())
}

/// Extracts the change folder name from a symlink target path.
/// Expected format: "fab/changes/{name}/.status.yaml"
pub fn extract_folder_from_symlink(target: &str) -> String {
    let target = target.replace('\\', "/");
    let prefix = "fab/changes/";
    let suffix = "/.status.yaml";
    if target.starts_with(prefix) && target.ends_with(suffix) {
        let name = &target[prefix.len()..target.len() - suffix.len()];
        if !name.is_empty() && !name.contains('/') {
            return name.to_string();
        }
    }
    String::new()
}

fn resolve_override(changes_dir: &Path, override_arg: &str) -> Result<String> {
    if !changes_dir.exists() {
        bail!("fab/changes/ not found.");
    }

    let folders = list_change_folders(changes_dir)?;
    if folders.is_empty() {
        bail!("No active changes found.");
    }

    let override_lower = override_arg.to_lowercase();

    // Exact match
    for f in &folders {
        if f.to_lowercase() == override_lower {
            return Ok(f.clone());
        }
    }

    // Substring match
    let partials: Vec<&String> = folders
        .iter()
        .filter(|f| f.to_lowercase().contains(&override_lower))
        .collect();

    if partials.len() == 1 {
        return Ok(partials[0].clone());
    }
    if partials.len() > 1 {
        let names: Vec<&str> = partials.iter().map(|s| s.as_str()).collect();
        bail!("Multiple changes match \"{}\": {}.", override_arg, names.join(", "));
    }

    bail!("No change matches \"{}\".", override_arg);
}

fn resolve_from_current(fab_root: &str, changes_dir: &Path) -> Result<String> {
    // Read .fab-status.yaml symlink at repo root
    let repo_root = Path::new(fab_root).parent().unwrap_or(Path::new("/"));
    let symlink_path = repo_root.join(".fab-status.yaml");
    if let Ok(target) = fs::read_link(&symlink_path) {
        let target_str = target.to_string_lossy().to_string();
        let name = extract_folder_from_symlink(&target_str);
        if !name.is_empty() {
            return Ok(name);
        }
    }

    // Fallback: single-change guess
    if !changes_dir.exists() {
        bail!("No active change.");
    }

    let mut candidates = Vec::new();
    if let Ok(entries) = fs::read_dir(changes_dir) {
        for entry in entries.flatten() {
            let name = entry.file_name().to_string_lossy().to_string();
            if !entry.file_type().map(|t| t.is_dir()).unwrap_or(false) || name == "archive" {
                continue;
            }
            let status_path = changes_dir.join(&name).join(".status.yaml");
            if status_path.exists() {
                candidates.push(name);
            }
        }
    }

    if candidates.len() == 1 {
        eprintln!("(resolved from single active change)");
        return Ok(candidates.into_iter().next().unwrap());
    }
    if candidates.is_empty() {
        bail!("No active change.");
    }
    bail!("No active change (multiple changes exist — use /fab-switch).");
}

fn list_change_folders(changes_dir: &Path) -> Result<Vec<String>> {
    let mut folders = Vec::new();
    let entries = fs::read_dir(changes_dir)?;
    for entry in entries.flatten() {
        let name = entry.file_name().to_string_lossy().to_string();
        if entry.file_type().map(|t| t.is_dir()).unwrap_or(false) && name != "archive" {
            folders.push(name);
        }
    }
    Ok(folders)
}
