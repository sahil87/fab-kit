use anyhow::{bail, Result};
use serde_yaml::Value;
use std::collections::HashMap;
use std::fs;
use std::io::Write;
use std::path::Path;

use crate::types::STAGE_ORDER;

/// Checklist metadata.
#[derive(Clone, Debug)]
pub struct Checklist {
    pub generated: bool,
    pub path: String,
    pub completed: i64,
    pub total: i64,
}

/// Fuzzy SRAD dimension means.
#[derive(Clone, Debug)]
pub struct Dimensions {
    pub signal: f64,
    pub reversibility: f64,
    pub competence: f64,
    pub disambiguation: f64,
}

/// Confidence scoring block.
#[derive(Clone, Debug)]
pub struct Confidence {
    pub certain: i64,
    pub confident: i64,
    pub tentative: i64,
    pub unresolved: i64,
    pub score: f64,
    pub indicative: Option<bool>,
    pub fuzzy: Option<bool>,
    pub dimensions: Option<Dimensions>,
}

/// Stage timing/driver metadata.
#[derive(Clone, Debug)]
pub struct StageMetric {
    pub started_at: String,
    pub driver: String,
    pub iterations: i64,
    pub completed_at: String,
}

/// Stage name and its state.
#[derive(Clone, Debug)]
pub struct StageState {
    pub stage: String,
    pub state: String,
}

/// Represents the .status.yaml structure.
/// Uses serde_yaml::Value for the raw document to preserve YAML formatting on round-trip.
pub struct StatusFile {
    pub id: String,
    pub name: String,
    pub created: String,
    pub created_by: String,
    pub change_type: String,
    pub issues: Vec<String>,
    pub checklist: Checklist,
    pub confidence: Confidence,
    pub stage_metrics: HashMap<String, StageMetric>,
    pub prs: Vec<String>,
    pub last_updated: String,
    /// Raw YAML value for round-trip preservation
    raw: Value,
}

impl StatusFile {
    /// Gets the state of a stage from the progress map.
    pub fn get_progress(&self, stage: &str) -> String {
        if let Some(Value::Mapping(progress)) = self.get_raw_field("progress") {
            let key = Value::String(stage.to_string());
            if let Some(Value::String(state)) = progress.get(&key) {
                return state.clone();
            }
        }
        "pending".to_string()
    }

    /// Sets the state of a stage in the progress map.
    pub fn set_progress(&mut self, stage: &str, state: &str) {
        if let Some(root) = self.raw.as_mapping_mut() {
            let progress_key = Value::String("progress".to_string());
            if let Some(progress_val) = root.get_mut(&progress_key) {
                if let Some(progress) = progress_val.as_mapping_mut() {
                    let key = Value::String(stage.to_string());
                    progress.insert(key, Value::String(state.to_string()));
                }
            }
        }
    }

    /// Returns an ordered slice of stage:state pairs.
    pub fn get_progress_map(&self) -> Vec<StageState> {
        STAGE_ORDER
            .iter()
            .map(|s| StageState {
                stage: s.to_string(),
                state: self.get_progress(s),
            })
            .collect()
    }

    /// Save writes the StatusFile back to disk atomically (temp + rename).
    pub fn save(&mut self, path: &str) -> Result<()> {
        self.last_updated = now_iso();
        self.sync_to_raw();

        let data = serde_yaml::to_string(&self.raw)?;

        let dir = Path::new(path).parent().unwrap_or(Path::new("."));
        let tmp_path = dir.join(format!(".status.yaml.{}.tmp", std::process::id()));
        let tmp_path_str = tmp_path.to_string_lossy().to_string();

        let mut tmp = fs::File::create(&tmp_path)?;
        tmp.write_all(data.as_bytes())?;
        tmp.sync_all()?;
        drop(tmp);

        fs::rename(&tmp_path, path).map_err(|e| {
            let _ = fs::remove_file(&tmp_path_str);
            e
        })?;

        Ok(())
    }

    fn get_raw_field(&self, key: &str) -> Option<Value> {
        if let Value::Mapping(ref root) = self.raw {
            let k = Value::String(key.to_string());
            return root.get(&k).cloned();
        }
        None
    }

    fn sync_to_raw(&mut self) {
        let root = match self.raw.as_mapping_mut() {
            Some(m) => m,
            None => return,
        };

        set_scalar(root, "id", &self.id);
        set_scalar(root, "name", &self.name);
        set_scalar(root, "created", &self.created);
        set_scalar(root, "created_by", &self.created_by);
        set_scalar(root, "change_type", &self.change_type);
        set_scalar(root, "last_updated", &self.last_updated);

        // issues
        let issues_key = Value::String("issues".to_string());
        let issues_val = encode_string_sequence(&self.issues);
        root.insert(issues_key, issues_val);

        // prs
        let prs_key = Value::String("prs".to_string());
        let prs_val = encode_string_sequence(&self.prs);
        root.insert(prs_key, prs_val);

        // checklist
        let checklist_key = Value::String("checklist".to_string());
        let checklist_val = encode_checklist(&self.checklist);
        root.insert(checklist_key, checklist_val);

        // confidence
        let confidence_key = Value::String("confidence".to_string());
        let confidence_val = encode_confidence(&self.confidence);
        root.insert(confidence_key, confidence_val);

        // stage_metrics
        let metrics_key = Value::String("stage_metrics".to_string());
        let metrics_val = encode_stage_metrics(&self.stage_metrics);
        root.insert(metrics_key, metrics_val);
    }
}

fn set_scalar(root: &mut serde_yaml::Mapping, key: &str, value: &str) {
    let k = Value::String(key.to_string());
    root.insert(k, Value::String(value.to_string()));
}

fn encode_string_sequence(items: &[String]) -> Value {
    if items.is_empty() {
        Value::Sequence(vec![])
    } else {
        Value::Sequence(items.iter().map(|s| Value::String(s.clone())).collect())
    }
}

fn encode_checklist(c: &Checklist) -> Value {
    let mut m = serde_yaml::Mapping::new();
    m.insert(
        Value::String("generated".to_string()),
        Value::Bool(c.generated),
    );
    m.insert(
        Value::String("path".to_string()),
        Value::String(c.path.clone()),
    );
    m.insert(
        Value::String("completed".to_string()),
        Value::Number(c.completed.into()),
    );
    m.insert(
        Value::String("total".to_string()),
        Value::Number(c.total.into()),
    );
    Value::Mapping(m)
}

fn encode_confidence(c: &Confidence) -> Value {
    let mut m = serde_yaml::Mapping::new();
    m.insert(
        Value::String("certain".to_string()),
        Value::Number(c.certain.into()),
    );
    m.insert(
        Value::String("confident".to_string()),
        Value::Number(c.confident.into()),
    );
    m.insert(
        Value::String("tentative".to_string()),
        Value::Number(c.tentative.into()),
    );
    m.insert(
        Value::String("unresolved".to_string()),
        Value::Number(c.unresolved.into()),
    );
    m.insert(
        Value::String("score".to_string()),
        encode_float(c.score),
    );

    if let Some(true) = c.indicative {
        m.insert(
            Value::String("indicative".to_string()),
            Value::Bool(true),
        );
    }

    if let Some(true) = c.fuzzy {
        m.insert(
            Value::String("fuzzy".to_string()),
            Value::Bool(true),
        );
        if let Some(ref dims) = c.dimensions {
            let mut dm = serde_yaml::Mapping::new();
            dm.insert(
                Value::String("signal".to_string()),
                encode_float(dims.signal),
            );
            dm.insert(
                Value::String("reversibility".to_string()),
                encode_float(dims.reversibility),
            );
            dm.insert(
                Value::String("competence".to_string()),
                encode_float(dims.competence),
            );
            dm.insert(
                Value::String("disambiguation".to_string()),
                encode_float(dims.disambiguation),
            );
            m.insert(
                Value::String("dimensions".to_string()),
                Value::Mapping(dm),
            );
        }
    }

    Value::Mapping(m)
}

fn encode_float(f: f64) -> Value {
    // serde_yaml represents floats; we use the Number type
    Value::Number(serde_yaml::Number::from(f))
}

fn encode_stage_metrics(metrics: &HashMap<String, StageMetric>) -> Value {
    if metrics.is_empty() {
        return Value::Mapping(serde_yaml::Mapping::new());
    }

    let mut m = serde_yaml::Mapping::new();
    // Preserve stage order
    for stage in STAGE_ORDER {
        if let Some(sm) = metrics.get(*stage) {
            let mut inner = serde_yaml::Mapping::new();
            if !sm.started_at.is_empty() {
                inner.insert(
                    Value::String("started_at".to_string()),
                    Value::String(sm.started_at.clone()),
                );
            }
            if !sm.driver.is_empty() {
                inner.insert(
                    Value::String("driver".to_string()),
                    Value::String(sm.driver.clone()),
                );
            }
            if sm.iterations > 0 {
                inner.insert(
                    Value::String("iterations".to_string()),
                    Value::Number(sm.iterations.into()),
                );
            }
            if !sm.completed_at.is_empty() {
                inner.insert(
                    Value::String("completed_at".to_string()),
                    Value::String(sm.completed_at.clone()),
                );
            }
            m.insert(
                Value::String(stage.to_string()),
                Value::Mapping(inner),
            );
        }
    }
    Value::Mapping(m)
}

/// Loads and parses a .status.yaml file.
pub fn load(path: &str) -> Result<StatusFile> {
    let data = fs::read_to_string(path)
        .map_err(|_| anyhow::anyhow!("status file not found: {}", path))?;

    let raw: Value =
        serde_yaml::from_str(&data).map_err(|e| anyhow::anyhow!("invalid YAML in {}: {}", path, e))?;

    let root = match &raw {
        Value::Mapping(m) => m,
        _ => bail!("expected mapping at root of {}", path),
    };

    let get_str = |key: &str| -> String {
        root.get(&Value::String(key.to_string()))
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string()
    };

    let get_str_seq = |key: &str| -> Vec<String> {
        root.get(&Value::String(key.to_string()))
            .and_then(|v| v.as_sequence())
            .map(|seq| {
                seq.iter()
                    .filter_map(|v| v.as_str().map(String::from))
                    .collect()
            })
            .unwrap_or_default()
    };

    let checklist = parse_checklist(root);
    let confidence = parse_confidence(root);
    let stage_metrics = parse_stage_metrics(root);

    Ok(StatusFile {
        id: get_str("id"),
        name: get_str("name"),
        created: get_str("created"),
        created_by: get_str("created_by"),
        change_type: get_str("change_type"),
        issues: get_str_seq("issues"),
        checklist,
        confidence,
        stage_metrics,
        prs: get_str_seq("prs"),
        last_updated: get_str("last_updated"),
        raw,
    })
}

fn parse_checklist(root: &serde_yaml::Mapping) -> Checklist {
    let key = Value::String("checklist".to_string());
    let default = Checklist {
        generated: false,
        path: String::new(),
        completed: 0,
        total: 0,
    };
    let val = match root.get(&key) {
        Some(Value::Mapping(m)) => m,
        _ => return default,
    };

    Checklist {
        generated: val
            .get(&Value::String("generated".to_string()))
            .and_then(|v| v.as_bool())
            .unwrap_or(false),
        path: val
            .get(&Value::String("path".to_string()))
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string(),
        completed: val
            .get(&Value::String("completed".to_string()))
            .and_then(|v| v.as_i64())
            .unwrap_or(0),
        total: val
            .get(&Value::String("total".to_string()))
            .and_then(|v| v.as_i64())
            .unwrap_or(0),
    }
}

fn parse_confidence(root: &serde_yaml::Mapping) -> Confidence {
    let key = Value::String("confidence".to_string());
    let default = Confidence {
        certain: 0,
        confident: 0,
        tentative: 0,
        unresolved: 0,
        score: 0.0,
        indicative: None,
        fuzzy: None,
        dimensions: None,
    };
    let val = match root.get(&key) {
        Some(Value::Mapping(m)) => m,
        _ => return default,
    };

    let get_i64 = |k: &str| -> i64 {
        val.get(&Value::String(k.to_string()))
            .and_then(|v| v.as_i64())
            .unwrap_or(0)
    };
    let get_f64 = |k: &str| -> f64 {
        val.get(&Value::String(k.to_string()))
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0)
    };

    let indicative = val
        .get(&Value::String("indicative".to_string()))
        .and_then(|v| v.as_bool());
    let fuzzy = val
        .get(&Value::String("fuzzy".to_string()))
        .and_then(|v| v.as_bool());

    let dimensions = if fuzzy == Some(true) {
        val.get(&Value::String("dimensions".to_string()))
            .and_then(|v| v.as_mapping())
            .map(|dm| {
                let get_dim = |k: &str| -> f64 {
                    dm.get(&Value::String(k.to_string()))
                        .and_then(|v| v.as_f64())
                        .unwrap_or(0.0)
                };
                Dimensions {
                    signal: get_dim("signal"),
                    reversibility: get_dim("reversibility"),
                    competence: get_dim("competence"),
                    disambiguation: get_dim("disambiguation"),
                }
            })
    } else {
        None
    };

    Confidence {
        certain: get_i64("certain"),
        confident: get_i64("confident"),
        tentative: get_i64("tentative"),
        unresolved: get_i64("unresolved"),
        score: get_f64("score"),
        indicative,
        fuzzy,
        dimensions,
    }
}

fn parse_stage_metrics(root: &serde_yaml::Mapping) -> HashMap<String, StageMetric> {
    let key = Value::String("stage_metrics".to_string());
    let mut result = HashMap::new();

    let val = match root.get(&key) {
        Some(Value::Mapping(m)) => m,
        _ => return result,
    };

    for (k, v) in val.iter() {
        let stage = match k.as_str() {
            Some(s) => s.to_string(),
            None => continue,
        };
        let inner = match v.as_mapping() {
            Some(m) => m,
            None => continue,
        };

        let get_str = |k: &str| -> String {
            inner
                .get(&Value::String(k.to_string()))
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string()
        };
        let get_i64 = |k: &str| -> i64 {
            inner
                .get(&Value::String(k.to_string()))
                .and_then(|v| v.as_i64())
                .unwrap_or(0)
        };

        result.insert(
            stage,
            StageMetric {
                started_at: get_str("started_at"),
                driver: get_str("driver"),
                iterations: get_i64("iterations"),
                completed_at: get_str("completed_at"),
            },
        );
    }

    result
}

fn now_iso() -> String {
    chrono::Local::now().to_rfc3339()
}
