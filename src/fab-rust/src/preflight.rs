use anyhow::{bail, Result};
use std::fs;
use std::path::PathBuf;

use crate::resolve;
use crate::status;
use crate::statusfile::{self, Checklist, Confidence, StageState};

/// Result holds the structured preflight output.
pub struct PreflightResult {
    pub id: String,
    pub name: String,
    pub change_dir: String,
    pub stage: String,
    pub display_stage: String,
    pub display_state: String,
    pub progress: Vec<StageState>,
    pub checklist: Checklist,
    pub confidence: Confidence,
}

/// Performs preflight validation and returns structured result.
pub fn run(fab_root: &str, change_override: &str) -> Result<PreflightResult> {
    // 1. Check project initialization
    let config_path = PathBuf::from(fab_root).join("project").join("config.yaml");
    if !config_path.exists() {
        bail!("Project not initialized — fab/project/config.yaml not found. Run /fab-setup.");
    }
    let const_path = PathBuf::from(fab_root)
        .join("project")
        .join("constitution.md");
    if !const_path.exists() {
        bail!("Project not initialized — fab/project/constitution.md not found. Run /fab-setup.");
    }

    // 2. Sync staleness warning (non-blocking)
    check_sync_staleness(fab_root);

    // 3. Resolve change
    let folder = resolve::to_folder(fab_root, change_override)?;

    // 4. Check change directory
    let change_dir = PathBuf::from(fab_root).join("changes").join(&folder);
    if !change_dir.exists() {
        bail!("Change directory not found: fab/changes/{}", folder);
    }

    // 5. Check .status.yaml
    let status_path = change_dir.join(".status.yaml");
    if !status_path.exists() {
        bail!(".status.yaml not found in fab/changes/{}", folder);
    }

    // 6. Load and validate
    let sf = statusfile::load(&status_path.to_string_lossy())
        .map_err(|e| anyhow::anyhow!("Failed to load .status.yaml: {}", e))?;

    status::validate(&sf).map_err(|e| anyhow::anyhow!("Invalid .status.yaml: {}", e))?;

    // Build result
    let id = resolve::extract_id(&folder);
    let current_stage = status::current_stage(&sf);
    let (display_stage, display_state) = status::display_stage(&sf);
    let rel_change_dir = format!("fab/changes/{}", folder);

    Ok(PreflightResult {
        id,
        name: folder,
        change_dir: rel_change_dir,
        stage: current_stage,
        display_stage,
        display_state,
        progress: sf.get_progress_map(),
        checklist: sf.checklist,
        confidence: sf.confidence,
    })
}

/// Produces the YAML output matching preflight.sh format.
pub fn format_yaml(r: &PreflightResult) -> String {
    let mut b = String::new();

    b.push_str(&format!("id: {}\n", r.id));
    b.push_str(&format!("name: {}\n", r.name));
    b.push_str(&format!("change_dir: {}\n", r.change_dir));
    b.push_str(&format!("stage: {}\n", r.stage));
    b.push_str(&format!("display_stage: {}\n", r.display_stage));
    b.push_str(&format!("display_state: {}\n", r.display_state));
    b.push_str("progress:\n");
    for ss in &r.progress {
        b.push_str(&format!("  {}: {}\n", ss.stage, ss.state));
    }
    b.push_str("checklist:\n");
    b.push_str(&format!("  generated: {}\n", r.checklist.generated));
    b.push_str(&format!("  completed: {}\n", r.checklist.completed));
    b.push_str(&format!("  total: {}\n", r.checklist.total));
    b.push_str("confidence:\n");
    b.push_str(&format!("  certain: {}\n", r.confidence.certain));
    b.push_str(&format!("  confident: {}\n", r.confidence.confident));
    b.push_str(&format!("  tentative: {}\n", r.confidence.tentative));
    b.push_str(&format!("  unresolved: {}\n", r.confidence.unresolved));
    b.push_str(&format!("  score: {:.1}\n", r.confidence.score));

    if let Some(true) = r.confidence.indicative {
        b.push_str("  indicative: true\n");
    }

    b
}

fn check_sync_staleness(fab_root: &str) {
    let version_file = PathBuf::from(fab_root).join(".kit").join("VERSION");
    let kit_version = fs::read_to_string(&version_file)
        .ok()
        .map(|s| s.trim().to_string())
        .unwrap_or_default();

    let sync_file = PathBuf::from(fab_root).join(".kit-sync-version");
    match fs::read_to_string(&sync_file) {
        Ok(data) => {
            let sync_version = data.trim().to_string();
            if !kit_version.is_empty() && !sync_version.is_empty() && kit_version != sync_version {
                eprintln!(
                    "\u{26a0} Skills out of sync — run fab-sync.sh to refresh (engine {}, last synced {})",
                    kit_version, sync_version
                );
            }
        }
        Err(_) => {
            if !kit_version.is_empty() {
                eprintln!(
                    "\u{26a0} Skills may be out of sync — run fab-sync.sh to refresh"
                );
            }
        }
    }
}
