use anyhow::{bail, Result};
use std::fs;
use std::io::{BufRead, BufReader, Write};
use std::path::{Path, PathBuf};

use crate::change;
use crate::resolve;

/// Archive result.
pub struct ArchiveResult {
    pub action: String,
    pub name: String,
    pub clean: String,
    pub mov: String,
    pub index: String,
    pub pointer: String,
}

/// Restore result.
pub struct RestoreResult {
    pub action: String,
    pub name: String,
    pub mov: String,
    pub index: String,
    pub pointer: String,
}

/// Moves a change to the archive directory.
pub fn archive(fab_root: &str, change_arg: &str, description: &str) -> Result<ArchiveResult> {
    if change_arg.is_empty() {
        bail!("<change> argument is required for archive");
    }
    if description.is_empty() {
        bail!("--description is required for archive");
    }

    let folder = resolve::to_folder(fab_root, change_arg)?;

    let changes_dir = PathBuf::from(fab_root).join("changes");
    let archive_dir = changes_dir.join("archive");
    let change_dir = changes_dir.join(&folder);

    // 1. Clean: delete .pr-done if present
    let clean_status;
    let pr_done_path = change_dir.join(".pr-done");
    if pr_done_path.exists() {
        let _ = fs::remove_file(&pr_done_path);
        clean_status = "removed".to_string();
    } else {
        clean_status = "not_present".to_string();
    }

    // 2. Move to archive/yyyy/mm/
    let (bucket_year, bucket_month) = parse_date_bucket(&folder)?;
    let dest_dir = archive_dir.join(&bucket_year).join(&bucket_month);
    let _ = fs::create_dir_all(&dest_dir);
    let dest_path = dest_dir.join(&folder);
    if dest_path.exists() {
        bail!("Archive destination already exists: {}", dest_path.display());
    }
    fs::rename(&change_dir, &dest_path)?;

    // 3. Update index
    let index_file = archive_dir.join("index.md");
    let index_status = update_index(&index_file, &folder, description);

    // Backfill unindexed
    backfill_index(&archive_dir, &index_file);

    // 4. Clear pointer if active
    let mut pointer_status = "skipped".to_string();
    if let Ok(active_folder) = resolve::to_folder(fab_root, "") {
        if active_folder == folder {
            change::switch_blank(fab_root);
            pointer_status = "cleared".to_string();
        }
    }

    Ok(ArchiveResult {
        action: "archive".to_string(),
        name: folder,
        clean: clean_status,
        mov: "moved".to_string(),
        index: index_status,
        pointer: pointer_status,
    })
}

/// Restores a change from the archive.
pub fn restore(fab_root: &str, change_arg: &str, do_switch: bool) -> Result<RestoreResult> {
    if change_arg.is_empty() {
        bail!("<change> argument is required for restore");
    }

    let (folder, resolved_dir) = resolve_archive(fab_root, change_arg)?;

    let changes_dir = PathBuf::from(fab_root).join("changes");
    let archive_dir = changes_dir.join("archive");

    // 1. Move from archive
    let move_status;
    let dest_path = changes_dir.join(&folder);
    if dest_path.exists() {
        move_status = "already_in_changes".to_string();
    } else {
        fs::rename(&resolved_dir, &dest_path)?;
        move_status = "restored".to_string();
    }

    // 2. Remove from index
    let index_file = archive_dir.join("index.md");
    let index_status = remove_from_index(&index_file, &folder);

    // 3. Optionally switch
    let mut pointer_status = "skipped".to_string();
    if do_switch {
        if change::switch(fab_root, &folder).is_ok() {
            pointer_status = "switched".to_string();
        }
    }

    Ok(RestoreResult {
        action: "restore".to_string(),
        name: folder,
        mov: move_status,
        index: index_status,
        pointer: pointer_status,
    })
}

/// Lists archived change folder names from both flat and nested entries.
pub fn list(fab_root: &str) -> Result<Vec<String>> {
    let archive_dir = PathBuf::from(fab_root).join("changes").join("archive");
    if !archive_dir.exists() {
        return Ok(Vec::new());
    }

    let top_level = fs::read_dir(&archive_dir)?;
    let mut results = Vec::new();

    let mut top_entries: Vec<_> = top_level.flatten().collect();
    top_entries.sort_by_key(|e| e.file_name());

    // Flat entries: archive/{name}/ (skip year directories)
    for e in &top_entries {
        if e.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            let name = e.file_name().to_string_lossy().to_string();
            if !is_year_dir(&name) {
                results.push(name);
            }
        }
    }

    // Nested entries: archive/yyyy/mm/{name}/
    for year_entry in &top_entries {
        if !year_entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
            continue;
        }
        let year_name = year_entry.file_name().to_string_lossy().to_string();
        if !is_year_dir(&year_name) {
            continue;
        }
        let year_dir = archive_dir.join(&year_name);
        if let Ok(month_entries) = fs::read_dir(&year_dir) {
            let mut months: Vec<_> = month_entries.flatten().collect();
            months.sort_by_key(|e| e.file_name());
            for month_entry in &months {
                if !month_entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                    continue;
                }
                let month_dir = year_dir.join(month_entry.file_name());
                if let Ok(change_entries) = fs::read_dir(&month_dir) {
                    let mut changes: Vec<_> = change_entries.flatten().collect();
                    changes.sort_by_key(|e| e.file_name());
                    for ce in &changes {
                        if ce.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                            results.push(ce.file_name().to_string_lossy().to_string());
                        }
                    }
                }
            }
        }
    }

    Ok(results)
}

/// Formats an ArchiveResult.
pub fn format_archive_yaml(r: &ArchiveResult) -> String {
    format!(
        "action: {}\nname: {}\nclean: {}\nmove: {}\nindex: {}\npointer: {}",
        r.action, r.name, r.clean, r.mov, r.index, r.pointer
    )
}

/// Formats a RestoreResult.
pub fn format_restore_yaml(r: &RestoreResult) -> String {
    format!(
        "action: {}\nname: {}\nmove: {}\nindex: {}\npointer: {}",
        r.action, r.name, r.mov, r.index, r.pointer
    )
}

fn parse_date_bucket(name: &str) -> Result<(String, String)> {
    if name.len() < 6 {
        bail!(
            "invalid folder name '{}': expected YYMMDD prefix",
            name
        );
    }
    let prefix = &name[..6];
    for c in prefix.chars() {
        if !c.is_ascii_digit() {
            bail!(
                "invalid folder name '{}': expected YYMMDD prefix",
                name
            );
        }
    }
    let yy = &prefix[0..2];
    let mm = &prefix[2..4];
    Ok((format!("20{}", yy), mm.to_string()))
}

fn resolve_archive(fab_root: &str, override_arg: &str) -> Result<(String, PathBuf)> {
    if override_arg.is_empty() {
        bail!("<change> argument is required for restore");
    }

    let archive_dir = PathBuf::from(fab_root).join("changes").join("archive");
    if !archive_dir.exists() {
        bail!("No archive folder found.");
    }

    struct Entry {
        name: String,
        dir: PathBuf,
    }
    let mut entries = Vec::new();

    // Flat entries
    if let Ok(top_level) = fs::read_dir(&archive_dir) {
        for e in top_level.flatten() {
            if !e.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                continue;
            }
            let name = e.file_name().to_string_lossy().to_string();
            if is_year_dir(&name) {
                continue;
            }
            entries.push(Entry {
                name,
                dir: e.path(),
            });
        }
    }

    // Nested entries
    if let Ok(top_level) = fs::read_dir(&archive_dir) {
        for year_entry in top_level.flatten() {
            if !year_entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                continue;
            }
            let year_name = year_entry.file_name().to_string_lossy().to_string();
            if !is_year_dir(&year_name) {
                continue;
            }
            let year_dir = archive_dir.join(&year_name);
            if let Ok(month_entries) = fs::read_dir(&year_dir) {
                for month_entry in month_entries.flatten() {
                    if !month_entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                        continue;
                    }
                    let month_dir = year_dir.join(month_entry.file_name());
                    if let Ok(change_entries) = fs::read_dir(&month_dir) {
                        for ce in change_entries.flatten() {
                            if ce.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                                entries.push(Entry {
                                    name: ce.file_name().to_string_lossy().to_string(),
                                    dir: ce.path(),
                                });
                            }
                        }
                    }
                }
            }
        }
    }

    if entries.is_empty() {
        bail!("No archived changes found.");
    }

    let override_lower = override_arg.to_lowercase();

    // Exact match
    for e in &entries {
        if e.name.to_lowercase() == override_lower {
            return Ok((e.name.clone(), e.dir.clone()));
        }
    }

    // Substring match
    let partials: Vec<&Entry> = entries
        .iter()
        .filter(|e| e.name.to_lowercase().contains(&override_lower))
        .collect();

    if partials.len() == 1 {
        return Ok((partials[0].name.clone(), partials[0].dir.clone()));
    }
    if partials.len() > 1 {
        let names: Vec<&str> = partials.iter().map(|e| e.name.as_str()).collect();
        bail!(
            "Multiple archives match \"{}\": {}.",
            override_arg,
            names.join(", ")
        );
    }

    bail!("No archive matches \"{}\".", override_arg);
}

fn is_year_dir(name: &str) -> bool {
    if name.len() != 4 {
        return false;
    }
    name.chars().all(|c| c.is_ascii_digit())
}

fn update_index(index_file: &Path, folder: &str, description: &str) -> String {
    let mut index_status = "updated".to_string();
    if !index_file.exists() {
        let _ = fs::write(index_file, "# Archive Index\n\n");
        index_status = "created".to_string();
    }

    // Normalize description
    let description: String = description
        .chars()
        .map(|c| if c == '\n' || c == '\r' || c == '\t' { ' ' } else { c })
        .collect();
    let description = description.trim();

    let new_entry = format!("- **{}** \u{2014} {}", folder, description);

    let data = fs::read_to_string(index_file).unwrap_or_default();
    let lines: Vec<&str> = data.split('\n').collect();

    let mut result = Vec::new();
    if lines.len() >= 2 {
        result.push(lines[0].to_string());
        result.push(lines[1].to_string());
    } else if lines.len() == 1 {
        result.push(lines[0].to_string());
        result.push(String::new());
    } else {
        result.push("# Archive Index".to_string());
        result.push(String::new());
    }
    result.push(new_entry);
    if lines.len() > 2 {
        for line in &lines[2..] {
            result.push(line.to_string());
        }
    }

    let _ = fs::write(index_file, result.join("\n"));
    index_status
}

fn backfill_index(archive_dir: &Path, index_file: &Path) {
    let index_content = fs::read_to_string(index_file).unwrap_or_default();

    let mut f = match fs::OpenOptions::new().append(true).open(index_file) {
        Ok(f) => f,
        Err(_) => return,
    };

    let backfill_entry = |f: &mut fs::File, name: &str| {
        let marker = format!("**{}**", name);
        if !index_content.contains(&marker) {
            let _ = writeln!(
                f,
                "- **{}** \u{2014} (no description \u{2014} pre-index archive)",
                name
            );
        }
    };

    // Flat entries
    if let Ok(top_level) = fs::read_dir(archive_dir) {
        for e in top_level.flatten() {
            if e.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                let name = e.file_name().to_string_lossy().to_string();
                if !is_year_dir(&name) {
                    backfill_entry(&mut f, &name);
                }
            }
        }
    }

    // Nested entries
    if let Ok(top_level) = fs::read_dir(archive_dir) {
        for year_entry in top_level.flatten() {
            if !year_entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                continue;
            }
            let year_name = year_entry.file_name().to_string_lossy().to_string();
            if !is_year_dir(&year_name) {
                continue;
            }
            let year_dir = archive_dir.join(&year_name);
            if let Ok(month_entries) = fs::read_dir(&year_dir) {
                for month_entry in month_entries.flatten() {
                    if !month_entry.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                        continue;
                    }
                    let month_dir = year_dir.join(month_entry.file_name());
                    if let Ok(change_entries) = fs::read_dir(&month_dir) {
                        for ce in change_entries.flatten() {
                            if ce.file_type().map(|t| t.is_dir()).unwrap_or(false) {
                                backfill_entry(
                                    &mut f,
                                    &ce.file_name().to_string_lossy(),
                                );
                            }
                        }
                    }
                }
            }
        }
    }
}

fn remove_from_index(index_file: &Path, folder: &str) -> String {
    if !index_file.exists() {
        return "not_found".to_string();
    }

    let marker = format!("**{}**", folder);

    let f = match fs::File::open(index_file) {
        Ok(f) => f,
        Err(_) => return "not_found".to_string(),
    };

    let mut found = false;
    let mut lines = Vec::new();
    let reader = BufReader::new(f);
    for line in reader.lines().flatten() {
        if line.contains(&marker) {
            found = true;
            continue;
        }
        lines.push(line);
    }

    if !found {
        return "not_found".to_string();
    }

    let _ = fs::write(index_file, lines.join("\n"));
    "removed".to_string()
}
