// Rust benchmark contender for statusman operations.
// Uses serde_yaml for YAML parsing/serialization.

use std::env;
use std::fs;
use std::io::Write;
use std::path::Path;
use std::process;

const STAGES: &[&str] = &[
    "intake",
    "spec",
    "tasks",
    "apply",
    "review",
    "hydrate",
    "ship",
    "review-pr",
];

fn read_status(path: &str) -> serde_yaml::Value {
    let content = fs::read_to_string(path).unwrap_or_else(|e| {
        eprintln!("ERROR: Cannot read {}: {}", path, e);
        process::exit(1);
    });
    serde_yaml::from_str(&content).unwrap_or_else(|e| {
        eprintln!("ERROR: Cannot parse {}: {}", path, e);
        process::exit(1);
    })
}

fn write_status_atomic(path: &str, data: &serde_yaml::Value) {
    let dir = Path::new(path).parent().unwrap_or(Path::new("."));
    let tmp_path = dir.join(format!(
        ".status.yaml.{}.{}",
        process::id(),
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_millis()
    ));

    let yaml_str = serde_yaml::to_string(data).unwrap_or_else(|e| {
        eprintln!("ERROR: Cannot serialize YAML: {}", e);
        process::exit(1);
    });

    let mut file = fs::File::create(&tmp_path).unwrap_or_else(|e| {
        eprintln!("ERROR: Cannot create temp file: {}", e);
        process::exit(1);
    });
    file.write_all(yaml_str.as_bytes()).unwrap_or_else(|e| {
        eprintln!("ERROR: Cannot write to temp file: {}", e);
        process::exit(1);
    });
    drop(file);

    fs::rename(&tmp_path, path).unwrap_or_else(|e| {
        eprintln!("ERROR: Cannot rename temp file: {}", e);
        process::exit(1);
    });
}

fn now_iso() -> String {
    // Simple ISO 8601 timestamp without chrono dependency
    use std::time::SystemTime;
    let duration = SystemTime::now()
        .duration_since(SystemTime::UNIX_EPOCH)
        .unwrap();
    let secs = duration.as_secs();
    // Format as ISO 8601 (UTC)
    let s = secs % 60;
    let m = (secs / 60) % 60;
    let h = (secs / 3600) % 24;
    let days = secs / 86400;
    // Approximate date calculation
    let mut y = 1970i64;
    let mut remaining = days as i64;
    loop {
        let days_in_year = if y % 4 == 0 && (y % 100 != 0 || y % 400 == 0) {
            366
        } else {
            365
        };
        if remaining < days_in_year {
            break;
        }
        remaining -= days_in_year;
        y += 1;
    }
    let leap = y % 4 == 0 && (y % 100 != 0 || y % 400 == 0);
    let days_in_month = [
        31,
        if leap { 29 } else { 28 },
        31,
        30,
        31,
        30,
        31,
        31,
        30,
        31,
        30,
        31,
    ];
    let mut month = 0;
    for (i, &d) in days_in_month.iter().enumerate() {
        if remaining < d as i64 {
            month = i + 1;
            break;
        }
        remaining -= d as i64;
    }
    let day = remaining + 1;
    format!(
        "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}+00:00",
        y, month, day, h, m, s
    )
}

// ─── progress-map ─────────────────────────────────────────────────────────────
fn progress_map(status_file: &str) {
    let data = read_status(status_file);
    let progress = data.get("progress").and_then(|v| v.as_mapping());

    for stage in STAGES {
        let state = progress
            .and_then(|m| {
                m.get(serde_yaml::Value::String(stage.to_string()))
                    .and_then(|v| v.as_str())
            })
            .unwrap_or("pending");
        println!("{}:{}", stage, state);
    }
}

// ─── set-change-type ──────────────────────────────────────────────────────────
fn set_change_type(status_file: &str, change_type: &str) {
    let valid = ["feat", "fix", "refactor", "docs", "test", "ci", "chore"];
    if !valid.contains(&change_type) {
        eprintln!("ERROR: Invalid change type '{}'", change_type);
        process::exit(1);
    }

    let mut data = read_status(status_file);
    if let Some(mapping) = data.as_mapping_mut() {
        mapping.insert(
            serde_yaml::Value::String("change_type".to_string()),
            serde_yaml::Value::String(change_type.to_string()),
        );
        mapping.insert(
            serde_yaml::Value::String("last_updated".to_string()),
            serde_yaml::Value::String(now_iso()),
        );
    }
    write_status_atomic(status_file, &data);
}

// ─── finish ───────────────────────────────────────────────────────────────────
fn finish(status_file: &str, stage: &str) {
    let mut data = read_status(status_file);
    let ts = now_iso();

    // Get current state
    let current_state = data
        .get("progress")
        .and_then(|v| v.get(stage))
        .and_then(|v| v.as_str())
        .unwrap_or("pending")
        .to_string();

    if current_state != "active" && current_state != "ready" {
        eprintln!(
            "ERROR: Cannot finish stage '{}' — current state is '{}'",
            stage, current_state
        );
        process::exit(1);
    }

    // Find next stage
    let stage_idx = STAGES.iter().position(|&s| s == stage);
    let next_stage = stage_idx.and_then(|i| STAGES.get(i + 1).copied());

    let mapping = data.as_mapping_mut().unwrap_or_else(|| {
        eprintln!("ERROR: Status file root must be a mapping");
        process::exit(1);
    });

    // Update progress
    if let Some(progress) = mapping
        .get_mut(&serde_yaml::Value::String("progress".to_string()))
        .and_then(|v| v.as_mapping_mut())
    {
        progress.insert(
            serde_yaml::Value::String(stage.to_string()),
            serde_yaml::Value::String("done".to_string()),
        );

        // Auto-activate next pending stage
        if let Some(next) = next_stage {
            let next_state = progress
                .get(&serde_yaml::Value::String(next.to_string()))
                .and_then(|v| v.as_str())
                .unwrap_or("pending");
            if next_state == "pending" {
                progress.insert(
                    serde_yaml::Value::String(next.to_string()),
                    serde_yaml::Value::String("active".to_string()),
                );
            }
        }
    }

    // Update stage_metrics — completed_at for current stage
    let metrics_key = serde_yaml::Value::String("stage_metrics".to_string());
    if mapping.get(&metrics_key).is_none() {
        mapping.insert(
            metrics_key.clone(),
            serde_yaml::Value::Mapping(serde_yaml::Mapping::new()),
        );
    }
    if let Some(metrics) = mapping
        .get_mut(&metrics_key)
        .and_then(|v| v.as_mapping_mut())
    {
        let stage_key = serde_yaml::Value::String(stage.to_string());
        if metrics.get(&stage_key).is_none() {
            metrics.insert(
                stage_key.clone(),
                serde_yaml::Value::Mapping(serde_yaml::Mapping::new()),
            );
        }
        if let Some(stage_metrics) = metrics.get_mut(&stage_key).and_then(|v| v.as_mapping_mut()) {
            stage_metrics.insert(
                serde_yaml::Value::String("completed_at".to_string()),
                serde_yaml::Value::String(ts.clone()),
            );
        }

        // Metrics for next stage activation
        if let Some(next) = next_stage {
            let next_key = serde_yaml::Value::String(next.to_string());
            let mut next_metrics = serde_yaml::Mapping::new();
            next_metrics.insert(
                serde_yaml::Value::String("started_at".to_string()),
                serde_yaml::Value::String(ts.clone()),
            );
            next_metrics.insert(
                serde_yaml::Value::String("driver".to_string()),
                serde_yaml::Value::String("benchmark".to_string()),
            );
            next_metrics.insert(
                serde_yaml::Value::String("iterations".to_string()),
                serde_yaml::Value::Number(1.into()),
            );
            metrics.insert(next_key, serde_yaml::Value::Mapping(next_metrics));
        }
    }

    // Update last_updated
    mapping.insert(
        serde_yaml::Value::String("last_updated".to_string()),
        serde_yaml::Value::String(ts),
    );

    write_status_atomic(status_file, &data);
}

// ─── CLI ──────────────────────────────────────────────────────────────────────
fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        eprintln!("Usage: statusman {{progress-map|set-change-type|finish}} <status_file> [args...]");
        process::exit(1);
    }

    match args[1].as_str() {
        "--help" | "-h" => {
            println!("statusman-rust: Rust benchmark contender");
            println!(
                "Usage: statusman {{progress-map|set-change-type|finish}} <status_file> [args...]"
            );
        }
        "progress-map" => {
            if args.len() < 3 {
                eprintln!("Usage: statusman progress-map <status_file>");
                process::exit(1);
            }
            progress_map(&args[2]);
        }
        "set-change-type" => {
            if args.len() < 4 {
                eprintln!("Usage: statusman set-change-type <status_file> <type>");
                process::exit(1);
            }
            set_change_type(&args[2], &args[3]);
        }
        "finish" => {
            if args.len() < 4 {
                eprintln!("Usage: statusman finish <status_file> <stage>");
                process::exit(1);
            }
            finish(&args[2], &args[3]);
        }
        other => {
            eprintln!("Unknown command: {}", other);
            process::exit(1);
        }
    }
}
