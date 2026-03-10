use anyhow::Result;
use serde::Deserialize;
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

/// StageHook holds pre/post shell commands for a pipeline stage.
#[derive(Clone, Debug, Deserialize, Default)]
pub struct StageHook {
    #[serde(default)]
    pub pre: String,
    #[serde(default)]
    pub post: String,
}

/// Config holds the parsed project config relevant to the fab binary.
#[derive(Clone, Debug, Deserialize, Default)]
pub struct Config {
    #[serde(default)]
    pub stage_hooks: HashMap<String, StageHook>,
}

impl Config {
    /// Returns the hook config for a stage, or an empty hook if none configured.
    pub fn get_stage_hook(&self, stage: &str) -> StageHook {
        self.stage_hooks
            .get(stage)
            .cloned()
            .unwrap_or_default()
    }
}

/// Loads fab/project/config.yaml from fabRoot.
/// Returns an empty config if the file doesn't exist.
pub fn load(fab_root: &str) -> Result<Config> {
    let path = PathBuf::from(fab_root).join("project").join("config.yaml");

    let data = match fs::read_to_string(&path) {
        Ok(d) => d,
        Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
            return Ok(Config::default());
        }
        Err(e) => return Err(e.into()),
    };

    let mut cfg: Config = serde_yaml::from_str(&data)?;
    if cfg.stage_hooks.is_empty() {
        cfg.stage_hooks = HashMap::new();
    }

    Ok(cfg)
}
