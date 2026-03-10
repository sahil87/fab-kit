use anyhow::{bail, Result};
use regex::Regex;
use std::collections::HashMap;
use std::fs;
use std::io::{BufRead, BufReader};
use std::path::PathBuf;

use crate::log;
use crate::resolve;
use crate::status;
use crate::statusfile;

/// Penalty weights per decision grade.
const W_CERTAIN: f64 = 0.0;
const W_CONFIDENT: f64 = 0.3;
const W_TENTATIVE: f64 = 1.0;

/// Grade counts from parsing assumptions table.
struct GradeCount {
    certain: i64,
    confident: i64,
    tentative: i64,
    unresolved: i64,
    has_fuzzy: bool,
    dim_count: i64,
    sum_s: i64,
    sum_r: i64,
    sum_a: i64,
    sum_d: i64,
}

/// Gate check result.
pub struct GateResult {
    pub gate: String,
    pub score: f64,
    pub threshold: f64,
    pub change_type: String,
    pub certain: i64,
    pub confident: i64,
    pub tentative: i64,
    pub unresolved: i64,
}

/// Normal scoring result.
pub struct ScoreResult {
    pub certain: i64,
    pub confident: i64,
    pub tentative: i64,
    pub unresolved: i64,
    pub score: f64,
    pub delta: String,
    pub has_fuzzy: bool,
    pub mean_s: f64,
    pub mean_r: f64,
    pub mean_a: f64,
    pub mean_d: f64,
}

/// Runs the gate check mode.
pub fn check_gate(fab_root: &str, change_arg: &str, stage: &str) -> Result<GateResult> {
    let change_dir = resolve::to_abs_dir(fab_root, change_arg)?;

    let status_path = PathBuf::from(&change_dir).join(".status.yaml");
    if !status_path.exists() {
        bail!(".status.yaml not found in {}", change_dir);
    }

    let mut change_type = "feat".to_string();
    if let Ok(sf) = statusfile::load(&status_path.to_string_lossy()) {
        let ct = &sf.change_type;
        if !ct.is_empty() && ct != "null" {
            change_type = ct.clone();
        }
    }

    let (score_file, threshold) = if stage == "intake" {
        (PathBuf::from(&change_dir).join("intake.md"), 3.0)
    } else {
        (
            PathBuf::from(&change_dir).join("spec.md"),
            get_gate_threshold(&change_type),
        )
    };

    if !score_file.exists() {
        bail!(
            "{} not found in {}",
            score_file.file_name().unwrap().to_string_lossy(),
            change_dir
        );
    }

    let gc = count_grades(&score_file.to_string_lossy());
    let total = gc.certain + gc.confident + gc.tentative + gc.unresolved;
    let expected_min = get_expected_min(stage, &change_type);
    let score = compute_score(gc.certain, gc.confident, gc.tentative, gc.unresolved, total, expected_min);

    let gate = if score < threshold { "fail" } else { "pass" };

    Ok(GateResult {
        gate: gate.to_string(),
        score,
        threshold,
        change_type,
        certain: gc.certain,
        confident: gc.confident,
        tentative: gc.tentative,
        unresolved: gc.unresolved,
    })
}

/// Runs the normal scoring mode.
pub fn compute(fab_root: &str, change_arg: &str, stage: &str) -> Result<ScoreResult> {
    let change_dir = resolve::to_abs_dir(fab_root, change_arg)?;
    let status_path = PathBuf::from(&change_dir).join(".status.yaml");

    let score_file = if stage == "intake" {
        let f = PathBuf::from(&change_dir).join("intake.md");
        if !f.exists() {
            bail!("intake.md required for scoring at intake stage");
        }
        f
    } else {
        let f = PathBuf::from(&change_dir).join("spec.md");
        if !f.exists() {
            bail!("spec.md required for scoring");
        }
        f
    };

    // Load status file for change type, previous score, and writing back
    let load_result = statusfile::load(&status_path.to_string_lossy());

    let mut change_type = "feat".to_string();
    let mut prev_score = 0.0;
    if let Ok(ref sf) = load_result {
        let ct = &sf.change_type;
        if !ct.is_empty() && ct != "null" {
            change_type = ct.clone();
        }
        prev_score = sf.confidence.score;
    }

    let expected_min = get_expected_min(stage, &change_type);
    let gc = count_grades(&score_file.to_string_lossy());
    let total = gc.certain + gc.confident + gc.tentative + gc.unresolved;
    let score = compute_score(gc.certain, gc.confident, gc.tentative, gc.unresolved, total, expected_min);

    // Compute dimension means
    let (mean_s, mean_r, mean_a, mean_d) = if gc.dim_count > 0 {
        (
            round_to_1(gc.sum_s as f64 / gc.dim_count as f64),
            round_to_1(gc.sum_r as f64 / gc.dim_count as f64),
            round_to_1(gc.sum_a as f64 / gc.dim_count as f64),
            round_to_1(gc.sum_d as f64 / gc.dim_count as f64),
        )
    } else {
        (0.0, 0.0, 0.0, 0.0)
    };

    let delta = score - prev_score;
    let delta_str = format!("{:+.1}", delta);

    // Write to .status.yaml
    if let Ok(mut sf) = load_result {
        let indicative = stage == "intake";
        if gc.has_fuzzy {
            let _ = status::set_confidence_fuzzy(
                &mut sf,
                &status_path.to_string_lossy(),
                gc.certain, gc.confident, gc.tentative, gc.unresolved,
                score, mean_s, mean_r, mean_a, mean_d, indicative,
            );
        } else {
            let _ = status::set_confidence(
                &mut sf,
                &status_path.to_string_lossy(),
                gc.certain, gc.confident, gc.tentative, gc.unresolved,
                score, indicative,
            );
        }

        let folder = PathBuf::from(&change_dir)
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_default();
        let _ = log::confidence_log(fab_root, &folder, score, &delta_str, "calc-score");
    }

    Ok(ScoreResult {
        certain: gc.certain,
        confident: gc.confident,
        tentative: gc.tentative,
        unresolved: gc.unresolved,
        score,
        delta: delta_str,
        has_fuzzy: gc.has_fuzzy,
        mean_s,
        mean_r,
        mean_a,
        mean_d,
    })
}

/// Formats a GateResult as YAML.
pub fn format_gate_yaml(r: &GateResult) -> String {
    format!(
        "gate: {}\nscore: {:.1}\nthreshold: {:.1}\nchange_type: {}\ncertain: {}\nconfident: {}\ntentative: {}\nunresolved: {}",
        r.gate, r.score, r.threshold, r.change_type, r.certain, r.confident, r.tentative, r.unresolved
    )
}

/// Formats a ScoreResult as YAML.
pub fn format_score_yaml(r: &ScoreResult) -> String {
    let mut b = String::new();
    b.push_str("confidence:\n");
    b.push_str(&format!("  certain: {}\n", r.certain));
    b.push_str(&format!("  confident: {}\n", r.confident));
    b.push_str(&format!("  tentative: {}\n", r.tentative));
    b.push_str(&format!("  unresolved: {}\n", r.unresolved));
    b.push_str(&format!("  score: {:.1}\n", r.score));
    b.push_str(&format!("  delta: {}\n", r.delta));

    if r.has_fuzzy {
        b.push_str("  fuzzy: true\n");
        b.push_str("  dimensions:\n");
        b.push_str(&format!("    signal: {:.1}\n", r.mean_s));
        b.push_str(&format!("    reversibility: {:.1}\n", r.mean_r));
        b.push_str(&format!("    competence: {:.1}\n", r.mean_a));
        b.push_str(&format!("    disambiguation: {:.1}\n", r.mean_d));
    }

    b
}

fn count_grades(file: &str) -> GradeCount {
    let f = match fs::File::open(file) {
        Ok(f) => f,
        Err(_) => {
            return GradeCount {
                certain: 0,
                confident: 0,
                tentative: 0,
                unresolved: 0,
                has_fuzzy: false,
                dim_count: 0,
                sum_s: 0,
                sum_r: 0,
                sum_a: 0,
                sum_d: 0,
            };
        }
    };

    let scores_re = Regex::new(r"S:(\d+)\s+R:(\d+)\s+A:(\d+)\s+D:(\d+)").unwrap();

    let reader = BufReader::new(f);
    let mut gc = GradeCount {
        certain: 0,
        confident: 0,
        tentative: 0,
        unresolved: 0,
        has_fuzzy: false,
        dim_count: 0,
        sum_s: 0,
        sum_r: 0,
        sum_a: 0,
        sum_d: 0,
    };
    let mut in_section = false;
    let mut header_seen = false;

    for line in reader.lines().flatten() {
        if line.starts_with("## Assumptions") {
            in_section = true;
            header_seen = false;
            continue;
        }
        if in_section && line.starts_with("## ") {
            break;
        }

        if !in_section {
            continue;
        }

        if line.starts_with("| #") || line.starts_with("| # ") {
            header_seen = true;
            continue;
        }

        // Skip separator lines
        let trimmed = line.trim();
        if header_seen && is_table_separator(trimmed) {
            continue;
        }

        if header_seen && line.starts_with('|') {
            let cols: Vec<&str> = line.split('|').collect();
            if cols.len() < 6 {
                continue;
            }

            let grade = cols[2].trim().to_lowercase();

            match grade.as_str() {
                "certain" => gc.certain += 1,
                "confident" => gc.confident += 1,
                "tentative" => gc.tentative += 1,
                "unresolved" => gc.unresolved += 1,
                _ => {}
            }

            let scores_col = if cols.len() >= 6 { cols[5].trim() } else { "" };

            if let Some(m) = scores_re.captures(scores_col) {
                gc.has_fuzzy = true;
                gc.dim_count += 1;
                gc.sum_s += m[1].parse::<i64>().unwrap_or(0);
                gc.sum_r += m[2].parse::<i64>().unwrap_or(0);
                gc.sum_a += m[3].parse::<i64>().unwrap_or(0);
                gc.sum_d += m[4].parse::<i64>().unwrap_or(0);
            }
        }
    }

    gc
}

fn is_table_separator(line: &str) -> bool {
    if !line.starts_with('|') {
        return false;
    }
    for c in line.chars() {
        if c != '|' && c != '-' && c != ' ' {
            return false;
        }
    }
    true
}

fn compute_score(certain: i64, confident: i64, tentative: i64, unresolved: i64, total: i64, expected_min: i64) -> f64 {
    if unresolved > 0 {
        return 0.0;
    }

    let base = 5.0 - W_CERTAIN * certain as f64 - W_CONFIDENT * confident as f64 - W_TENTATIVE * tentative as f64;
    let base = if base < 0.0 { 0.0 } else { base };

    let cover = if expected_min > 0 {
        let c = total as f64 / expected_min as f64;
        if c > 1.0 { 1.0 } else { c }
    } else {
        1.0
    };

    round_to_1(base * cover)
}

fn get_expected_min(stage: &str, change_type: &str) -> i64 {
    let intake_mins: HashMap<&str, i64> = [("feat", 5), ("refactor", 4), ("fix", 3)].iter().cloned().collect();
    let spec_mins: HashMap<&str, i64> = [("feat", 7), ("refactor", 6), ("fix", 5)].iter().cloned().collect();

    if stage == "intake" {
        *intake_mins.get(change_type).unwrap_or(&2)
    } else {
        *spec_mins.get(change_type).unwrap_or(&3)
    }
}

fn get_gate_threshold(change_type: &str) -> f64 {
    let thresholds: HashMap<&str, f64> = [
        ("fix", 2.0), ("feat", 3.0), ("refactor", 3.0),
        ("docs", 2.0), ("test", 2.0), ("ci", 2.0), ("chore", 2.0),
    ].iter().cloned().collect();

    *thresholds.get(change_type).unwrap_or(&3.0)
}

fn round_to_1(f: f64) -> f64 {
    (f * 10.0).round() / 10.0
}
