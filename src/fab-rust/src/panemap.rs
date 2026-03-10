use anyhow::{bail, Result};
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;

use crate::resolve;
use crate::status;
use crate::statusfile;

struct PaneEntry {
    id: String,
    cwd: String,
}

struct PaneRow {
    pane: String,
    worktree: String,
    change: String,
    stage: String,
    agent: String,
}

pub fn run_pane_map() -> Result<()> {
    // Tmux session guard
    if std::env::var("TMUX").unwrap_or_default().is_empty() {
        eprintln!("Error: not inside a tmux session");
        std::process::exit(1);
    }

    // Discover tmux panes
    let panes = discover_panes()?;

    // Determine main worktree root
    let main_root = find_main_worktree_root(&panes);

    // Resolve each pane to a row
    let mut rows = Vec::new();
    let mut runtime_cache: HashMap<String, Option<HashMap<String, serde_yaml::Value>>> = HashMap::new();

    for p in &panes {
        if let Some(row) = resolve_pane(p, &main_root, &mut runtime_cache) {
            rows.push(row);
        }
    }

    // Output
    if rows.is_empty() {
        println!("No fab worktrees found in tmux panes.");
        return Ok(());
    }

    print_pane_table(&rows);
    Ok(())
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

fn find_main_worktree_root(panes: &[PaneEntry]) -> String {
    for p in panes {
        if let Ok(output) = Command::new("git")
            .args(["-C", &p.cwd, "worktree", "list", "--porcelain"])
            .output()
        {
            if output.status.success() {
                let stdout = String::from_utf8_lossy(&output.stdout);
                for line in stdout.lines() {
                    if let Some(path) = line.strip_prefix("worktree ") {
                        return path.to_string();
                    }
                }
            }
        }
    }
    String::new()
}

fn resolve_pane(
    p: &PaneEntry,
    main_root: &str,
    runtime_cache: &mut HashMap<String, Option<HashMap<String, serde_yaml::Value>>>,
) -> Option<PaneRow> {
    // Resolve git worktree root
    let wt_root = git_worktree_root(&p.cwd)?;

    // Check for fab/ directory
    let fab_dir = PathBuf::from(&wt_root).join("fab");
    if !fab_dir.exists() {
        return None;
    }

    // Compute worktree display path
    let wt_display = worktree_display_path(&wt_root, main_root);

    // Read .fab-status.yaml symlink
    let (change_name, folder_name) = read_fab_current(&wt_root);

    // Read stage from .status.yaml
    let stage_name = if !folder_name.is_empty() {
        let status_path = fab_dir
            .join("changes")
            .join(&folder_name)
            .join(".status.yaml");
        if let Ok(sf) = statusfile::load(&status_path.to_string_lossy()) {
            let (stage, _) = status::display_stage(&sf);
            stage
        } else {
            "\u{2014}".to_string() // em dash
        }
    } else {
        "\u{2014}".to_string() // em dash
    };

    // Determine agent state
    let agent_state = resolve_agent_state(&wt_root, &folder_name, runtime_cache);

    Some(PaneRow {
        pane: p.id.clone(),
        worktree: wt_display,
        change: change_name,
        stage: stage_name,
        agent: agent_state,
    })
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

fn worktree_display_path(wt_root: &str, main_root: &str) -> String {
    if !main_root.is_empty() && wt_root == main_root {
        return "(main)".to_string();
    }
    if !main_root.is_empty() {
        let parent = Path::new(main_root).parent().unwrap_or(Path::new("/"));
        if let Ok(rel) = pathdiff(wt_root, &parent.to_string_lossy()) {
            return format!("{}/", rel);
        }
    }
    // Fallback: basename with trailing slash
    format!(
        "{}/",
        Path::new(wt_root)
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_default()
    )
}

fn pathdiff(path: &str, base: &str) -> Result<String, ()> {
    let path = Path::new(path);
    let base = Path::new(base);
    path.strip_prefix(base)
        .map(|p| p.to_string_lossy().to_string())
        .map_err(|_| ())
}

fn read_fab_current(wt_root: &str) -> (String, String) {
    let symlink_path = PathBuf::from(wt_root).join(".fab-status.yaml");
    match fs::read_link(&symlink_path) {
        Ok(target) => {
            let target_str = target.to_string_lossy().to_string();
            let folder_name = resolve::extract_folder_from_symlink(&target_str);
            if folder_name.is_empty() {
                ("(no change)".to_string(), String::new())
            } else {
                (folder_name.clone(), folder_name)
            }
        }
        Err(_) => ("(no change)".to_string(), String::new()),
    }
}

fn resolve_agent_state(
    wt_root: &str,
    folder_name: &str,
    cache: &mut HashMap<String, Option<HashMap<String, serde_yaml::Value>>>,
) -> String {
    if folder_name.is_empty() {
        return "\u{2014}".to_string(); // em dash
    }

    let rt_path = PathBuf::from(wt_root).join(".fab-runtime.yaml");

    // Check cache
    if !cache.contains_key(wt_root) {
        match load_panemap_runtime_file(&rt_path.to_string_lossy()) {
            Ok(data) => {
                cache.insert(wt_root.to_string(), Some(data));
            }
            Err(e) => {
                if e.kind() == std::io::ErrorKind::NotFound {
                    cache.insert(wt_root.to_string(), None);
                } else {
                    eprintln!("warning: failed to load {}: {}", rt_path.display(), e);
                    return "?".to_string();
                }
            }
        }
    }

    let rt_data = match cache.get(wt_root) {
        Some(Some(data)) => data,
        Some(None) => return "?".to_string(), // file missing
        None => return "?".to_string(),
    };

    // Look up change entry
    let folder_entry = match rt_data.get(folder_name) {
        Some(v) => v,
        None => return "active".to_string(),
    };

    let folder_map = match folder_entry.as_mapping() {
        Some(m) => m,
        None => return "active".to_string(),
    };

    let agent_block = match folder_map.get(&serde_yaml::Value::String("agent".to_string())) {
        Some(serde_yaml::Value::Mapping(m)) => m,
        _ => return "active".to_string(),
    };

    let idle_since = match agent_block.get(&serde_yaml::Value::String("idle_since".to_string())) {
        Some(v) => v,
        None => return "active".to_string(),
    };

    let ts = match idle_since {
        serde_yaml::Value::Number(n) => n.as_i64().unwrap_or(0),
        _ => return "active".to_string(),
    };

    let now = chrono::Utc::now().timestamp();
    let elapsed = (now - ts).max(0);

    format!("idle ({})", format_idle_duration(elapsed))
}

fn load_panemap_runtime_file(path: &str) -> Result<HashMap<String, serde_yaml::Value>, std::io::Error> {
    let data = fs::read_to_string(path)?;
    let v: serde_yaml::Value = serde_yaml::from_str(&data).map_err(|e| {
        std::io::Error::new(std::io::ErrorKind::InvalidData, format!("parsing {}: {}", path, e))
    })?;
    let mut result = HashMap::new();
    if let serde_yaml::Value::Mapping(m) = v {
        for (k, v) in m {
            if let serde_yaml::Value::String(key) = k {
                result.insert(key, v);
            }
        }
    }
    Ok(result)
}

fn format_idle_duration(seconds: i64) -> String {
    if seconds < 60 {
        format!("{}s", seconds)
    } else if seconds < 3600 {
        format!("{}m", seconds / 60)
    } else {
        format!("{}h", seconds / 3600)
    }
}

fn print_pane_table(rows: &[PaneRow]) {
    let headers = ["Pane", "Worktree", "Change", "Stage", "Agent"];
    let mut widths = [
        headers[0].len(),
        headers[1].len(),
        headers[2].len(),
        headers[3].len(),
        headers[4].len(),
    ];

    for r in rows {
        let cols = [&r.pane, &r.worktree, &r.change, &r.stage, &r.agent];
        for (i, c) in cols.iter().enumerate() {
            if c.len() > widths[i] {
                widths[i] = c.len();
            }
        }
    }

    // Print header
    println!(
        "{:<w0$}  {:<w1$}  {:<w2$}  {:<w3$}  {}",
        headers[0], headers[1], headers[2], headers[3], headers[4],
        w0 = widths[0], w1 = widths[1], w2 = widths[2], w3 = widths[3],
    );

    // Print data rows
    for r in rows {
        println!(
            "{:<w0$}  {:<w1$}  {:<w2$}  {:<w3$}  {}",
            r.pane, r.worktree, r.change, r.stage, r.agent,
            w0 = widths[0], w1 = widths[1], w2 = widths[2], w3 = widths[3],
        );
    }
}
