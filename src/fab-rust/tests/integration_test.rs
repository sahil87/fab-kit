use std::fs;
use std::os::unix::fs as unix_fs;
use std::path::PathBuf;
use std::process::Command;

/// Returns the path to the built binary.
fn binary_path() -> PathBuf {
    let mut path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    path.push("target");
    // Use the profile from the current test build
    if cfg!(debug_assertions) {
        path.push("debug");
    } else {
        path.push("release");
    }
    path.push("fab");
    path
}

/// Sets up a temporary repo with fixtures copied from the Go test fixtures.
fn setup_temp_repo() -> tempfile::TempDir {
    let tmp = tempfile::tempdir().expect("create temp dir");
    let root = tmp.path();

    // Create fab directory structure
    let fab = root.join("fab");
    fs::create_dir_all(fab.join("changes").join("260305-t3st-parity-test-change")).unwrap();
    fs::create_dir_all(fab.join("project")).unwrap();
    fs::create_dir_all(fab.join(".kit").join("templates")).unwrap();

    // Copy fixture files
    let fixture_base = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("tests")
        .join("fixtures");

    let change_dir = fab.join("changes").join("260305-t3st-parity-test-change");

    // Copy status file
    fs::copy(
        fixture_base.join("fab/changes/260305-t3st-parity-test-change/.status.yaml"),
        change_dir.join(".status.yaml"),
    ).unwrap();

    // Copy intake.md
    fs::copy(
        fixture_base.join("fab/changes/260305-t3st-parity-test-change/intake.md"),
        change_dir.join("intake.md"),
    ).unwrap();

    // Copy spec.md
    fs::copy(
        fixture_base.join("fab/changes/260305-t3st-parity-test-change/spec.md"),
        change_dir.join("spec.md"),
    ).unwrap();

    // Copy config.yaml
    fs::copy(
        fixture_base.join("fab/project/config.yaml"),
        fab.join("project").join("config.yaml"),
    ).unwrap();

    // Copy constitution.md
    fs::copy(
        fixture_base.join("fab/project/constitution.md"),
        fab.join("project").join("constitution.md"),
    ).unwrap();

    // Copy status template
    let template_src = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .parent().unwrap().parent().unwrap()
        .join("fab").join(".kit").join("templates").join("status.yaml");
    if template_src.exists() {
        fs::copy(&template_src, fab.join(".kit").join("templates").join("status.yaml")).unwrap();
    }

    // Create .fab-status.yaml symlink
    unix_fs::symlink(
        "fab/changes/260305-t3st-parity-test-change/.status.yaml",
        root.join(".fab-status.yaml"),
    ).unwrap();

    tmp
}

fn run_fab(tmp: &tempfile::TempDir, args: &[&str]) -> (String, String, i32) {
    let bin = binary_path();
    let output = Command::new(&bin)
        .args(args)
        .current_dir(tmp.path())
        .output()
        .expect(&format!("failed to run {:?}", bin));

    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).to_string();
    let code = output.status.code().unwrap_or(-1);
    (stdout, stderr, code)
}

// ===== T019: Core subcommand tests =====

mod resolve_tests {
    use super::*;

    #[test]
    fn test_resolve_folder() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["resolve", "--folder"]);
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "260305-t3st-parity-test-change");
    }

    #[test]
    fn test_resolve_id() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["resolve", "--id"]);
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "t3st");
    }

    #[test]
    fn test_resolve_dir() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["resolve", "--dir"]);
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "fab/changes/260305-t3st-parity-test-change/");
    }

    #[test]
    fn test_resolve_status() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["resolve", "--status"]);
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "fab/changes/260305-t3st-parity-test-change/.status.yaml");
    }

    #[test]
    fn test_resolve_substring_match() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["resolve", "--folder", "parity"]);
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "260305-t3st-parity-test-change");
    }

    #[test]
    fn test_resolve_ambiguous_match() {
        let tmp = setup_temp_repo();
        // Create a second change that also matches "test"
        let change2 = tmp.path().join("fab/changes/260305-abcd-another-test");
        fs::create_dir_all(&change2).unwrap();

        let (_, stderr, code) = run_fab(&tmp, &["resolve", "--folder", "test"]);
        assert_eq!(code, 1);
        assert!(stderr.contains("Multiple changes match"), "stderr: {}", stderr);
    }
}

mod status_tests {
    use super::*;

    #[test]
    fn test_all_stages() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["status", "all-stages"]);
        assert_eq!(code, 0);
        let stages: Vec<&str> = stdout.trim().lines().collect();
        assert_eq!(stages, vec![
            "intake", "spec", "tasks", "apply", "review", "hydrate", "ship", "review-pr"
        ]);
    }

    #[test]
    fn test_progress_map() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["status", "progress-map", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("intake:done"));
        assert!(stdout.contains("spec:done"));
        assert!(stdout.contains("tasks:active"));
        assert!(stdout.contains("apply:pending"));
    }

    #[test]
    fn test_current_stage() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["status", "current-stage", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "tasks");
    }

    #[test]
    fn test_display_stage() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["status", "display-stage", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "tasks:active");
    }

    #[test]
    fn test_checklist() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["status", "checklist", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("generated:false"));
        assert!(stdout.contains("completed:0"));
        assert!(stdout.contains("total:0"));
    }

    #[test]
    fn test_confidence() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["status", "confidence", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("certain:5"));
        assert!(stdout.contains("confident:1"));
        assert!(stdout.contains("score:4.7"));
        assert!(stdout.contains("indicative:false"));
    }

    #[test]
    fn test_validate_status_file() {
        let tmp = setup_temp_repo();
        let (_, _, code) = run_fab(
            &tmp,
            &["status", "validate-status-file", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
    }

    #[test]
    fn test_finish_auto_activate() {
        let tmp = setup_temp_repo();
        // Finish tasks stage (currently active)
        let (_, _, code) = run_fab(
            &tmp,
            &["status", "finish", "260305-t3st-parity-test-change", "tasks"],
        );
        assert_eq!(code, 0);

        // Check that apply is now active
        let (stdout, _, code) = run_fab(
            &tmp,
            &["status", "progress-map", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("tasks:done"), "stdout: {}", stdout);
        assert!(stdout.contains("apply:active"), "stdout: {}", stdout);
    }

    #[test]
    fn test_reset_cascading() {
        let tmp = setup_temp_repo();
        // First finish tasks and apply to set up
        run_fab(&tmp, &["status", "finish", "260305-t3st-parity-test-change", "tasks"]);
        run_fab(&tmp, &["status", "finish", "260305-t3st-parity-test-change", "apply"]);

        // Now reset spec — should cascade tasks, apply, review, etc. to pending
        let (_, _, code) = run_fab(
            &tmp,
            &["status", "reset", "260305-t3st-parity-test-change", "spec"],
        );
        assert_eq!(code, 0);

        let (stdout, _, _) = run_fab(
            &tmp,
            &["status", "progress-map", "260305-t3st-parity-test-change"],
        );
        assert!(stdout.contains("spec:active"), "stdout: {}", stdout);
        assert!(stdout.contains("tasks:pending"), "stdout: {}", stdout);
        assert!(stdout.contains("apply:pending"), "stdout: {}", stdout);
    }

    #[test]
    fn test_skip_cascading() {
        let tmp = setup_temp_repo();
        // Skip tasks (currently active) — downstream pending should cascade to skipped
        let (_, _, code) = run_fab(
            &tmp,
            &["status", "skip", "260305-t3st-parity-test-change", "tasks"],
        );
        assert_eq!(code, 0);

        let (stdout, _, _) = run_fab(
            &tmp,
            &["status", "progress-map", "260305-t3st-parity-test-change"],
        );
        assert!(stdout.contains("tasks:skipped"), "stdout: {}", stdout);
        assert!(stdout.contains("apply:skipped"), "stdout: {}", stdout);
        assert!(stdout.contains("review:skipped"), "stdout: {}", stdout);
    }

    #[test]
    fn test_fail_for_review() {
        let tmp = setup_temp_repo();
        // Progress to review stage
        run_fab(&tmp, &["status", "finish", "260305-t3st-parity-test-change", "tasks"]);
        run_fab(&tmp, &["status", "finish", "260305-t3st-parity-test-change", "apply"]);

        // review should now be active
        let (stdout, _, _) = run_fab(
            &tmp,
            &["status", "progress-map", "260305-t3st-parity-test-change"],
        );
        assert!(stdout.contains("review:active"), "stdout: {}", stdout);

        // Fail review
        let (_, _, code) = run_fab(
            &tmp,
            &["status", "fail", "260305-t3st-parity-test-change", "review"],
        );
        assert_eq!(code, 0);

        let (stdout, _, _) = run_fab(
            &tmp,
            &["status", "progress-map", "260305-t3st-parity-test-change"],
        );
        assert!(stdout.contains("review:failed"), "stdout: {}", stdout);
    }
}

mod change_tests {
    use super::*;

    #[test]
    fn test_change_switch_and_display() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["change", "switch", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains(".fab-status.yaml"), "stdout: {}", stdout);
        assert!(stdout.contains("260305-t3st-parity-test-change"), "stdout: {}", stdout);
        assert!(stdout.contains("Stage:"), "stdout: {}", stdout);
        assert!(stdout.contains("Confidence:"), "stdout: {}", stdout);
    }

    #[test]
    fn test_change_switch_blank() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["change", "switch", "--blank"]);
        assert_eq!(code, 0);
        assert!(stdout.contains("No active change"), "stdout: {}", stdout);
    }

    #[test]
    fn test_change_list() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["change", "list"]);
        assert_eq!(code, 0);
        assert!(stdout.contains("260305-t3st-parity-test-change"), "stdout: {}", stdout);
        assert!(stdout.contains("tasks:active"), "stdout: {}", stdout);
    }

    #[test]
    fn test_change_resolve() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["change", "resolve", "parity"]);
        assert_eq!(code, 0);
        assert_eq!(stdout.trim(), "260305-t3st-parity-test-change");
    }
}

// ===== T020: Remaining subcommand tests =====

mod preflight_tests {
    use super::*;

    #[test]
    fn test_preflight_valid_repo() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(&tmp, &["preflight"]);
        assert_eq!(code, 0);
        assert!(stdout.contains("id: t3st"), "stdout: {}", stdout);
        assert!(stdout.contains("name: 260305-t3st-parity-test-change"), "stdout: {}", stdout);
        assert!(stdout.contains("stage: tasks"), "stdout: {}", stdout);
        assert!(stdout.contains("display_stage: tasks"), "stdout: {}", stdout);
        assert!(stdout.contains("display_state: active"), "stdout: {}", stdout);
        assert!(stdout.contains("progress:"), "stdout: {}", stdout);
        assert!(stdout.contains("checklist:"), "stdout: {}", stdout);
        assert!(stdout.contains("confidence:"), "stdout: {}", stdout);
    }

    #[test]
    fn test_preflight_missing_config() {
        let tmp = setup_temp_repo();
        fs::remove_file(tmp.path().join("fab/project/config.yaml")).unwrap();

        let (_, stderr, code) = run_fab(&tmp, &["preflight"]);
        assert_eq!(code, 1);
        assert!(stderr.contains("config.yaml not found"), "stderr: {}", stderr);
    }

    #[test]
    fn test_preflight_missing_constitution() {
        let tmp = setup_temp_repo();
        fs::remove_file(tmp.path().join("fab/project/constitution.md")).unwrap();

        let (_, stderr, code) = run_fab(&tmp, &["preflight"]);
        assert_eq!(code, 1);
        assert!(stderr.contains("constitution.md not found"), "stderr: {}", stderr);
    }
}

mod score_tests {
    use super::*;

    #[test]
    fn test_score_intake() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["score", "260305-t3st-parity-test-change", "--stage", "intake"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("confidence:"), "stdout: {}", stdout);
        assert!(stdout.contains("certain: 5"), "stdout: {}", stdout);
        assert!(stdout.contains("confident: 1"), "stdout: {}", stdout);
        assert!(stdout.contains("score:"), "stdout: {}", stdout);
    }

    #[test]
    fn test_score_spec() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["score", "260305-t3st-parity-test-change", "--stage", "spec"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("confidence:"), "stdout: {}", stdout);
    }

    #[test]
    fn test_score_gate_check_pass() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &["score", "260305-t3st-parity-test-change", "--check-gate", "--stage", "intake"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("gate: pass"), "stdout: {}", stdout);
    }
}

mod log_tests {
    use super::*;

    #[test]
    fn test_log_command() {
        let tmp = setup_temp_repo();
        let (_, _, code) = run_fab(
            &tmp,
            &["log", "command", "fab-continue", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);

        // Check JSONL was written
        let history = fs::read_to_string(
            tmp.path().join("fab/changes/260305-t3st-parity-test-change/.history.jsonl"),
        ).unwrap();
        let last_line = history.lines().last().unwrap();
        let entry: serde_json::Value = serde_json::from_str(last_line).unwrap();
        assert_eq!(entry["event"], "command");
        assert_eq!(entry["cmd"], "fab-continue");
        assert!(entry["ts"].as_str().unwrap().len() > 0);
    }

    #[test]
    fn test_log_transition() {
        let tmp = setup_temp_repo();
        let (_, _, code) = run_fab(
            &tmp,
            &["log", "transition", "260305-t3st-parity-test-change", "tasks", "enter"],
        );
        assert_eq!(code, 0);

        let history = fs::read_to_string(
            tmp.path().join("fab/changes/260305-t3st-parity-test-change/.history.jsonl"),
        ).unwrap();
        let last_line = history.lines().last().unwrap();
        let entry: serde_json::Value = serde_json::from_str(last_line).unwrap();
        assert_eq!(entry["event"], "stage-transition");
        assert_eq!(entry["stage"], "tasks");
        assert_eq!(entry["action"], "enter");
    }

    #[test]
    fn test_log_review() {
        let tmp = setup_temp_repo();
        let (_, _, code) = run_fab(
            &tmp,
            &["log", "review", "260305-t3st-parity-test-change", "passed"],
        );
        assert_eq!(code, 0);

        let history = fs::read_to_string(
            tmp.path().join("fab/changes/260305-t3st-parity-test-change/.history.jsonl"),
        ).unwrap();
        let last_line = history.lines().last().unwrap();
        let entry: serde_json::Value = serde_json::from_str(last_line).unwrap();
        assert_eq!(entry["event"], "review");
        assert_eq!(entry["result"], "passed");
    }
}

mod archive_tests {
    use super::*;

    #[test]
    fn test_archive_and_index() {
        let tmp = setup_temp_repo();
        let (stdout, _, code) = run_fab(
            &tmp,
            &[
                "change", "archive", "260305-t3st-parity-test-change",
                "--description", "Test change",
            ],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("action: archive"), "stdout: {}", stdout);
        assert!(stdout.contains("name: 260305-t3st-parity-test-change"), "stdout: {}", stdout);
        assert!(stdout.contains("move: moved"), "stdout: {}", stdout);

        // Check index was created
        let index = fs::read_to_string(
            tmp.path().join("fab/changes/archive/index.md"),
        ).unwrap();
        assert!(index.contains("260305-t3st-parity-test-change"), "index: {}", index);
        assert!(index.contains("Test change"), "index: {}", index);
    }

    #[test]
    fn test_restore() {
        let tmp = setup_temp_repo();
        // First archive
        run_fab(
            &tmp,
            &[
                "change", "archive", "260305-t3st-parity-test-change",
                "--description", "Test",
            ],
        );

        // Then restore
        let (stdout, _, code) = run_fab(
            &tmp,
            &["change", "restore", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);
        assert!(stdout.contains("action: restore"), "stdout: {}", stdout);
        assert!(stdout.contains("move: restored"), "stdout: {}", stdout);

        // Verify it's back in changes
        assert!(tmp.path().join("fab/changes/260305-t3st-parity-test-change").exists());
    }
}

mod runtime_tests {
    use super::*;

    #[test]
    fn test_set_idle() {
        let tmp = setup_temp_repo();
        let (_, _, code) = run_fab(
            &tmp,
            &["runtime", "set-idle", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);

        // Verify runtime file exists and has correct structure
        let rt_path = tmp.path().join(".fab-runtime.yaml");
        assert!(rt_path.exists());
        let content = fs::read_to_string(&rt_path).unwrap();
        assert!(content.contains("260305-t3st-parity-test-change"), "content: {}", content);
        assert!(content.contains("idle_since"), "content: {}", content);
    }

    #[test]
    fn test_clear_idle() {
        let tmp = setup_temp_repo();
        // First set idle
        run_fab(&tmp, &["runtime", "set-idle", "260305-t3st-parity-test-change"]);

        // Then clear
        let (_, _, code) = run_fab(
            &tmp,
            &["runtime", "clear-idle", "260305-t3st-parity-test-change"],
        );
        assert_eq!(code, 0);

        // Runtime file should be cleaned up
        let rt_path = tmp.path().join(".fab-runtime.yaml");
        if rt_path.exists() {
            let content = fs::read_to_string(&rt_path).unwrap();
            assert!(!content.contains("idle_since"), "content: {}", content);
        }
    }
}

mod panemap_tests {
    use super::*;

    #[test]
    fn test_panemap_outside_tmux() {
        let tmp = setup_temp_repo();
        let bin = binary_path();
        let output = Command::new(&bin)
            .args(["pane-map"])
            .current_dir(tmp.path())
            .env_remove("TMUX")
            .output()
            .expect("run binary");

        let stderr = String::from_utf8_lossy(&output.stderr);
        assert!(stderr.contains("not inside a tmux session"), "stderr: {}", stderr);
        assert!(!output.status.success());
    }
}

mod sendkeys_tests {
    use super::*;

    #[test]
    fn test_sendkeys_missing_tmux() {
        let tmp = setup_temp_repo();
        let bin = binary_path();
        let output = Command::new(&bin)
            .args(["send-keys", "260305-t3st-parity-test-change", "echo hello"])
            .current_dir(tmp.path())
            .env_remove("TMUX")
            .output()
            .expect("run binary");

        let stderr = String::from_utf8_lossy(&output.stderr);
        assert!(stderr.contains("not inside a tmux session"), "stderr: {}", stderr);
        assert!(!output.status.success());
    }
}
