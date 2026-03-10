use anyhow::Result;
use serde_yaml::Value;
use std::fs;
use std::io::Write;
use std::path::Path;

use crate::resolve;

/// Records agent idle timestamp for a change.
pub fn set_idle(fab_root: &str, change_arg: &str) -> Result<()> {
    let folder = resolve::to_folder(fab_root, change_arg)?;

    let rt_path = runtime_file_path(fab_root);
    let mut m = load_runtime_file(&rt_path)?;

    let root = m.as_mapping_mut().unwrap();

    // Ensure folder entry exists as a mapping
    let folder_key = Value::String(folder.clone());
    let folder_entry = root
        .entry(folder_key)
        .or_insert_with(|| Value::Mapping(serde_yaml::Mapping::new()));

    let folder_map = match folder_entry.as_mapping_mut() {
        Some(m) => m,
        None => {
            *folder_entry = Value::Mapping(serde_yaml::Mapping::new());
            folder_entry.as_mapping_mut().unwrap()
        }
    };

    // Set or update agent.idle_since
    let agent_key = Value::String("agent".to_string());
    let agent_entry = folder_map
        .entry(agent_key)
        .or_insert_with(|| Value::Mapping(serde_yaml::Mapping::new()));

    let agent_map = match agent_entry.as_mapping_mut() {
        Some(m) => m,
        None => {
            *agent_entry = Value::Mapping(serde_yaml::Mapping::new());
            agent_entry.as_mapping_mut().unwrap()
        }
    };

    let now = chrono::Utc::now().timestamp();
    agent_map.insert(
        Value::String("idle_since".to_string()),
        Value::Number(serde_yaml::Number::from(now)),
    );

    save_runtime_file(&rt_path, &m)
}

/// Clears agent idle state for a change.
pub fn clear_idle(fab_root: &str, change_arg: &str) -> Result<()> {
    let folder = resolve::to_folder(fab_root, change_arg)?;

    let rt_path = runtime_file_path(fab_root);
    if !Path::new(&rt_path).exists() {
        return Ok(()); // no-op if file doesn't exist
    }

    let mut m = load_runtime_file(&rt_path)?;
    let root = m.as_mapping_mut().unwrap();

    let folder_key = Value::String(folder.clone());

    // Delete the agent block for this folder
    let should_remove_folder;
    if let Some(folder_entry) = root.get_mut(&folder_key) {
        if let Some(folder_map) = folder_entry.as_mapping_mut() {
            folder_map.remove(&Value::String("agent".to_string()));
            should_remove_folder = folder_map.is_empty();
        } else {
            should_remove_folder = false;
        }
    } else {
        should_remove_folder = false;
    }

    if should_remove_folder {
        root.remove(&folder_key);
    }

    save_runtime_file(&rt_path, &m)
}

fn runtime_file_path(fab_root: &str) -> String {
    let repo_root = Path::new(fab_root).parent().unwrap_or(Path::new("/"));
    repo_root
        .join(".fab-runtime.yaml")
        .to_string_lossy()
        .to_string()
}

fn load_runtime_file(path: &str) -> Result<Value> {
    match fs::read_to_string(path) {
        Ok(data) => {
            let v: Value = serde_yaml::from_str(&data)?;
            match v {
                Value::Mapping(_) => Ok(v),
                Value::Null => Ok(Value::Mapping(serde_yaml::Mapping::new())),
                _ => Ok(Value::Mapping(serde_yaml::Mapping::new())),
            }
        }
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            Ok(Value::Mapping(serde_yaml::Mapping::new()))
        }
        Err(e) => Err(e.into()),
    }
}

fn save_runtime_file(path: &str, m: &Value) -> Result<()> {
    let data = serde_yaml::to_string(m)?;

    let dir = Path::new(path).parent().unwrap_or(Path::new("."));
    let tmp_path = dir.join(format!(
        ".fab-runtime-{}.tmp",
        std::process::id()
    ));

    let mut tmp = fs::File::create(&tmp_path)?;
    tmp.write_all(data.as_bytes())?;
    tmp.sync_all()?;
    drop(tmp);

    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        fs::set_permissions(&tmp_path, fs::Permissions::from_mode(0o644))?;
    }

    fs::rename(&tmp_path, path).map_err(|e| {
        let _ = fs::remove_file(&tmp_path);
        e
    })?;

    Ok(())
}
