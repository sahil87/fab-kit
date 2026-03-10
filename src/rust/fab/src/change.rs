use anyhow::{bail, Result};
use rand::Rng;
use regex::Regex;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

use crate::log;
use crate::resolve;
use crate::status;
use crate::statusfile;
use crate::types;

const ID_CHARS: &[u8] = b"abcdefghijklmnopqrstuvwxyz0123456789";

/// Creates a new change directory with initialized .status.yaml.
pub fn new(fab_root: &str, slug: &str, change_id: &str, log_args: &str) -> Result<String> {
    let slug_re = Regex::new(r"^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$").unwrap();
    let id_re = Regex::new(r"^[a-z0-9]{4}$").unwrap();

    if slug.is_empty() {
        bail!("--slug is required");
    }
    if !slug_re.is_match(slug) {
        bail!(
            "Invalid slug format '{}' (expected alphanumeric and hyphens, no leading/trailing hyphen)",
            slug
        );
    }

    let id_provided = !change_id.is_empty();
    let mut cid = change_id.to_string();

    if id_provided {
        if !id_re.is_match(change_id) {
            bail!(
                "Invalid change-id '{}' (expected 4 lowercase alphanumeric chars)",
                change_id
            );
        }
    }

    let changes_dir = PathBuf::from(fab_root).join("changes");
    let date_prefix = chrono::Utc::now().format("%y%m%d").to_string();

    if id_provided {
        if has_id_collision(&changes_dir, &cid) {
            let existing = find_collision(&changes_dir, &cid);
            bail!("Change ID '{}' already in use ({})", cid, existing);
        }
    } else {
        cid = generate_unique_id(&changes_dir, 10)?;
    }

    let folder_name = format!("{}-{}-{}", date_prefix, cid, slug);
    let change_dir = changes_dir.join(&folder_name);

    fs::create_dir(&change_dir)?;

    let created_by = detect_created_by();
    let now = chrono::Utc::now().to_rfc3339();

    // Initialize .status.yaml from template
    let template_path = PathBuf::from(fab_root)
        .join(".kit")
        .join("templates")
        .join("status.yaml");
    let tmpl_data = fs::read_to_string(&template_path)
        .map_err(|e| anyhow::anyhow!("read template: {}", e))?;

    let content = tmpl_data
        .replace("{ID}", &cid)
        .replace("{NAME}", &folder_name)
        .replace("{CREATED}", &now)
        .replace("{CREATED_BY}", &created_by);

    let status_path = change_dir.join(".status.yaml");
    fs::write(&status_path, &content)?;

    // Start intake stage
    let mut sf = statusfile::load(&status_path.to_string_lossy())?;
    status::start(
        &mut sf,
        &status_path.to_string_lossy(),
        fab_root,
        "intake",
        "fab-new",
        "",
        "",
    )?;

    if !log_args.is_empty() {
        let _ = log::command(fab_root, "fab-new", &folder_name, log_args);
    }

    Ok(folder_name)
}

/// Renames a change folder's slug.
pub fn rename(fab_root: &str, current_folder: &str, new_slug: &str) -> Result<String> {
    let slug_re = Regex::new(r"^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$").unwrap();

    if current_folder.is_empty() {
        bail!("--folder is required");
    }
    if new_slug.is_empty() {
        bail!("--slug is required");
    }
    if !slug_re.is_match(new_slug) {
        bail!(
            "Invalid slug format '{}' (expected alphanumeric and hyphens, no leading/trailing hyphen)",
            new_slug
        );
    }

    let changes_dir = PathBuf::from(fab_root).join("changes");
    let old_path = changes_dir.join(current_folder);
    if !old_path.exists() {
        bail!("Change folder '{}' not found", current_folder);
    }

    // Extract YYMMDD-XXXX prefix
    let parts: Vec<&str> = current_folder.splitn(3, '-').collect();
    if parts.len() < 2 {
        bail!("invalid folder name format");
    }
    let prefix = format!("{}-{}", parts[0], parts[1]);
    let new_name = format!("{}-{}", prefix, new_slug);

    if new_name == current_folder {
        bail!("New name is the same as current name");
    }

    let new_path = changes_dir.join(&new_name);
    if new_path.exists() {
        bail!("Folder '{}' already exists", new_name);
    }

    fs::rename(&old_path, &new_path)?;

    // Update .status.yaml name field
    let status_path = new_path.join(".status.yaml");
    if let Ok(mut sf) = statusfile::load(&status_path.to_string_lossy()) {
        sf.name = new_name.clone();
        let _ = sf.save(&status_path.to_string_lossy());
    }

    // Update .fab-status.yaml symlink if it points to old folder
    let repo_root = Path::new(fab_root).parent().unwrap_or(Path::new("/"));
    let symlink_path = repo_root.join(".fab-status.yaml");
    if let Ok(target) = fs::read_link(&symlink_path) {
        let expected_old = format!("fab/changes/{}/.status.yaml", current_folder);
        if target.to_string_lossy() == expected_old {
            let new_target = format!("fab/changes/{}/.status.yaml", new_name);
            let _ = fs::remove_file(&symlink_path);
            #[cfg(unix)]
            {
                let _ = std::os::unix::fs::symlink(&new_target, &symlink_path);
            }
        }
    }

    let _ = log::command(
        fab_root,
        "changeman-rename",
        &new_name,
        &format!("--folder {} --slug {}", current_folder, new_slug),
    );

    Ok(new_name)
}

/// Switch changes the active change pointer.
pub fn switch(fab_root: &str, name: &str) -> Result<String> {
    let folder = resolve::to_folder(fab_root, name)?;

    let repo_root = Path::new(fab_root).parent().unwrap_or(Path::new("/"));
    let symlink_path = repo_root.join(".fab-status.yaml");
    let target = format!("fab/changes/{}/.status.yaml", folder);
    let _ = fs::remove_file(&symlink_path);
    #[cfg(unix)]
    {
        std::os::unix::fs::symlink(&target, &symlink_path)?;
    }
    #[cfg(not(unix))]
    {
        bail!("symlinks not supported on this platform");
    }

    // Derive display info
    let status_path = PathBuf::from(fab_root)
        .join("changes")
        .join(&folder)
        .join(".status.yaml");

    let mut display_stage = "unknown".to_string();
    let mut display_state = "pending".to_string();
    let mut routing_stage = "unknown".to_string();
    let mut conf_display = "not yet scored".to_string();

    if let Ok(sf) = statusfile::load(&status_path.to_string_lossy()) {
        let (ds, dstate) = status::display_stage(&sf);
        display_stage = ds;
        display_state = dstate;
        routing_stage = status::current_stage(&sf);

        let c = &sf.confidence;
        let total_counts = c.certain + c.confident + c.tentative + c.unresolved;
        if c.score == 0.0 && total_counts == 0 {
            conf_display = "not yet scored".to_string();
        } else if c.indicative == Some(true) {
            conf_display = format!("{:.1} of 5.0 (indicative)", c.score);
        } else {
            conf_display = format!("{:.1} of 5.0", c.score);
        }
    }

    let dnum = types::stage_number(&display_stage);

    let mut output = String::new();
    output.push_str(&format!(".fab-status.yaml \u{2192} {}\n", folder));
    output.push('\n');
    output.push_str(&format!(
        "Stage:       {} ({}/8) \u{2014} {}\n",
        display_stage, dnum, display_state
    ));
    output.push_str(&format!("Confidence:  {}\n", conf_display));

    let cmd = default_command(&routing_stage);
    if let Some(nstage) = types::next_stage(&routing_stage) {
        output.push_str(&format!("Next:        {} (via {})", nstage, cmd));
    } else {
        output.push_str(&format!("Next:        {}", cmd));
    }

    Ok(output)
}

/// Deactivates the current change.
pub fn switch_blank(fab_root: &str) -> String {
    let repo_root = Path::new(fab_root).parent().unwrap_or(Path::new("/"));
    let symlink_path = repo_root.join(".fab-status.yaml");

    match fs::symlink_metadata(&symlink_path) {
        Ok(_) => {
            let _ = fs::remove_file(&symlink_path);
            "No active change.".to_string()
        }
        Err(_) => "No active change (already blank).".to_string(),
    }
}

/// Lists changes with stage info.
pub fn list(fab_root: &str, archive: bool) -> Result<Vec<String>> {
    let scan_dir = if archive {
        PathBuf::from(fab_root).join("changes").join("archive")
    } else {
        PathBuf::from(fab_root).join("changes")
    };

    if !scan_dir.exists() {
        if archive {
            return Ok(Vec::new());
        }
        bail!("fab/changes/ not found.");
    }

    let entries = fs::read_dir(&scan_dir)?;
    let mut results = Vec::new();

    for entry in entries.flatten() {
        if !entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            continue;
        }
        let name = entry.file_name().to_string_lossy().to_string();
        if !archive && name == "archive" {
            continue;
        }

        let status_path = scan_dir.join(&name).join(".status.yaml");
        match statusfile::load(&status_path.to_string_lossy()) {
            Ok(sf) => {
                let (ds, dstate) = status::display_stage(&sf);
                let c = &sf.confidence;
                let indicative = if c.indicative == Some(true) {
                    "true"
                } else {
                    "false"
                };
                results.push(format!(
                    "{}:{}:{}:{:.1}:{}",
                    name, ds, dstate, c.score, indicative
                ));
            }
            Err(_) => {
                results.push(format!("{}:unknown:unknown", name));
                eprintln!("Warning: .status.yaml not found for {}", name);
            }
        }
    }

    Ok(results)
}

/// Passthrough to resolve.to_folder.
pub fn resolve_change(fab_root: &str, override_arg: &str) -> Result<String> {
    resolve::to_folder(fab_root, override_arg)
}

fn detect_created_by() -> String {
    // Try gh api user
    if let Ok(output) = Command::new("gh")
        .args(["api", "user", "--jq", ".login"])
        .output()
    {
        let user = String::from_utf8_lossy(&output.stdout).trim().to_string();
        if !user.is_empty() {
            return user;
        }
    }

    // Try git config
    if let Ok(output) = Command::new("git").args(["config", "user.name"]).output() {
        let user = String::from_utf8_lossy(&output.stdout).trim().to_string();
        if !user.is_empty() {
            return user;
        }
    }

    "unknown".to_string()
}

fn generate_random_id() -> String {
    let mut rng = rand::thread_rng();
    (0..4)
        .map(|_| ID_CHARS[rng.gen_range(0..ID_CHARS.len())] as char)
        .collect()
}

fn generate_unique_id(changes_dir: &Path, max_retries: usize) -> Result<String> {
    for _ in 0..max_retries {
        let id = generate_random_id();
        if !has_id_collision(changes_dir, &id) {
            return Ok(id);
        }
    }
    bail!(
        "Failed to generate unique change ID after {} attempts",
        max_retries
    );
}

fn has_id_collision(changes_dir: &Path, change_id: &str) -> bool {
    let entries = match fs::read_dir(changes_dir) {
        Ok(e) => e,
        Err(_) => return false,
    };
    let pattern = format!("-{}-", change_id);
    for entry in entries.flatten() {
        if entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            let name = entry.file_name().to_string_lossy().to_string();
            // Check YYMMDD-{id}-slug pattern
            if name.len() > 6 && name[6..].starts_with(&pattern) {
                return true;
            }
        }
    }
    false
}

fn find_collision(changes_dir: &Path, change_id: &str) -> String {
    let entries = match fs::read_dir(changes_dir) {
        Ok(e) => e,
        Err(_) => return String::new(),
    };
    let pattern = format!("-{}-", change_id);
    for entry in entries.flatten() {
        if entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            let name = entry.file_name().to_string_lossy().to_string();
            if name.len() > 6 && name[6..].starts_with(&pattern) {
                return name;
            }
        }
    }
    String::new()
}

fn default_command(stage: &str) -> &'static str {
    match stage {
        "intake" | "spec" | "tasks" | "apply" | "review" => "/fab-continue",
        "hydrate" => "/git-pr",
        "ship" => "/git-pr-review",
        "review-pr" => "/fab-archive",
        _ => "/fab-status",
    }
}
