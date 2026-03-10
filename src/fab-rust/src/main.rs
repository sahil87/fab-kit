mod archive;
mod change;
mod config;
mod hooks;
mod log;
mod panemap;
mod preflight;
mod resolve;
mod runtime;
mod score;
mod sendkeys;
mod status;
mod statusfile;
mod types;
mod worktree;

use clap::{Parser, Subcommand};
use std::process;

#[derive(Parser)]
#[command(name = "fab", bin_name = "fab", about = "Fab workflow engine — single binary replacement for kit shell scripts")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand)]
enum Commands {
    /// Resolve a change reference to a canonical output
    Resolve {
        /// Change reference (optional)
        change: Option<String>,
        /// Output 4-char change ID (default)
        #[arg(long)]
        id: bool,
        /// Output full folder name
        #[arg(long)]
        folder: bool,
        /// Output directory path
        #[arg(long)]
        dir: bool,
        /// Output .status.yaml path
        #[arg(long)]
        status: bool,
    },
    /// Append-only JSON logging to .history.jsonl
    Log {
        #[command(subcommand)]
        command: LogCommands,
    },
    /// Manage workflow stages, states, and .status.yaml
    Status {
        #[command(subcommand)]
        command: StatusCommands,
    },
    /// Validate project state and output structured YAML
    Preflight {
        /// Change name override
        change_name: Option<String>,
    },
    /// Change lifecycle management
    Change {
        #[command(subcommand)]
        command: ChangeCommands,
    },
    /// Compute confidence score from Assumptions table
    Score {
        /// Change reference
        change: String,
        /// Gate check mode (read-only)
        #[arg(long)]
        check_gate: bool,
        /// Stage for scoring (intake or spec)
        #[arg(long, default_value = "spec")]
        stage: String,
    },
    /// Manage runtime state (.fab-runtime.yaml)
    Runtime {
        #[command(subcommand)]
        command: RuntimeCommands,
    },
    /// Show tmux pane-to-worktree mapping with fab pipeline state
    PaneMap,
    /// Send text to a change's tmux pane
    SendKeys {
        /// Change reference
        change: String,
        /// Text to send
        text: String,
    },
}

#[derive(Subcommand)]
enum LogCommands {
    /// Log a skill invocation
    Command {
        /// Command name
        cmd: String,
        /// Change reference (optional)
        change: Option<String>,
        /// Extra args (optional)
        args: Option<String>,
    },
    /// Log a confidence score change
    Confidence {
        /// Change reference
        change: String,
        /// Score value
        score: String,
        /// Delta value
        delta: String,
        /// Trigger description
        trigger: String,
    },
    /// Log a review outcome
    Review {
        /// Change reference
        change: String,
        /// Result (passed/failed)
        result: String,
        /// Rework description (optional)
        rework: Option<String>,
    },
    /// Log a stage transition
    Transition {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Action name
        action: String,
        /// From state (optional)
        from: Option<String>,
        /// Reason (optional)
        reason: Option<String>,
        /// Driver (optional)
        driver: Option<String>,
    },
}

#[derive(Subcommand)]
enum StatusCommands {
    /// Show fab pipeline status for worktrees
    Show {
        /// Worktree name (optional)
        name: Option<String>,
        /// Show status for all worktrees
        #[arg(long)]
        all: bool,
        /// Output as JSON
        #[arg(long)]
        json: bool,
    },
    /// List all stage IDs in order
    AllStages,
    /// Extract stage:state pairs
    ProgressMap {
        /// Change reference
        change: String,
    },
    /// Single-line visual progress
    ProgressLine {
        /// Change reference
        change: String,
    },
    /// Detect active stage
    CurrentStage {
        /// Change reference
        change: String,
    },
    /// Display stage as stage:state
    DisplayStage {
        /// Change reference
        change: String,
    },
    /// Extract checklist fields
    Checklist {
        /// Change reference
        change: String,
    },
    /// Extract confidence fields
    Confidence {
        /// Change reference
        change: String,
    },
    /// Validate .status.yaml against schema
    ValidateStatusFile {
        /// Change reference
        change: String,
    },
    /// {pending,failed} -> active
    Start {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Driver (optional)
        driver: Option<String>,
        /// From state (optional)
        from: Option<String>,
        /// Reason (optional)
        reason: Option<String>,
    },
    /// active -> ready
    Advance {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Driver (optional)
        driver: Option<String>,
    },
    /// {active,ready} -> done (+auto-activate next)
    Finish {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Driver (optional)
        driver: Option<String>,
    },
    /// {done,ready,skipped} -> active (+cascade)
    Reset {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Driver (optional)
        driver: Option<String>,
        /// From state (optional)
        from: Option<String>,
        /// Reason (optional)
        reason: Option<String>,
    },
    /// {pending,active} -> skipped (+cascade)
    Skip {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Driver (optional)
        driver: Option<String>,
    },
    /// active -> failed (review/review-pr only)
    Fail {
        /// Change reference
        change: String,
        /// Stage name
        stage: String,
        /// Driver (optional)
        driver: Option<String>,
        /// Rework description (optional)
        rework: Option<String>,
    },
    /// Set change_type
    SetChangeType {
        /// Change reference
        change: String,
        /// Change type
        r#type: String,
    },
    /// Update checklist field
    SetChecklist {
        /// Change reference
        change: String,
        /// Field name
        field: String,
        /// Field value
        value: String,
    },
    /// Replace confidence block
    SetConfidence {
        /// Change reference
        change: String,
        /// Certain count
        certain: String,
        /// Confident count
        confident: String,
        /// Tentative count
        tentative: String,
        /// Unresolved count
        unresolved: String,
        /// Score value
        score: String,
        /// Mark score as indicative (from intake)
        #[arg(long)]
        indicative: bool,
    },
    /// Replace confidence block with dimensions
    SetConfidenceFuzzy {
        /// Change reference
        change: String,
        /// Certain count
        certain: String,
        /// Confident count
        confident: String,
        /// Tentative count
        tentative: String,
        /// Unresolved count
        unresolved: String,
        /// Score value
        score: String,
        /// Mean signal
        mean_s: String,
        /// Mean reversibility
        mean_r: String,
        /// Mean competence
        mean_a: String,
        /// Mean disambiguation
        mean_d: String,
        /// Mark score as indicative (from intake)
        #[arg(long)]
        indicative: bool,
    },
    /// Append issue ID (idempotent)
    AddIssue {
        /// Change reference
        change: String,
        /// Issue ID
        id: String,
    },
    /// List issue IDs
    GetIssues {
        /// Change reference
        change: String,
    },
    /// Append PR URL (idempotent)
    AddPr {
        /// Change reference
        change: String,
        /// PR URL
        url: String,
    },
    /// List PR URLs
    GetPrs {
        /// Change reference
        change: String,
    },
}

#[derive(Subcommand)]
enum ChangeCommands {
    /// Create a new change directory
    New {
        /// Folder name suffix (required)
        #[arg(long)]
        slug: Option<String>,
        /// Explicit 4-char ID (optional)
        #[arg(long)]
        change_id: Option<String>,
        /// Description for logman (optional)
        #[arg(long)]
        log_args: Option<String>,
    },
    /// Rename a change folder's slug
    Rename {
        /// Current folder name (required)
        #[arg(long)]
        folder: Option<String>,
        /// New slug (required)
        #[arg(long)]
        slug: Option<String>,
    },
    /// Switch the active change
    Switch {
        /// Change name (optional)
        name: Option<String>,
        /// Deactivate the current change
        #[arg(long)]
        blank: bool,
    },
    /// List changes with stage info
    List {
        /// List archived changes
        #[arg(long)]
        archive: bool,
    },
    /// Resolve a change name
    Resolve {
        /// Override (optional)
        r#override: Option<String>,
    },
    /// Archive a change
    Archive {
        /// Change reference (optional positional)
        change: Option<String>,
        /// Description for archive index (required)
        #[arg(long)]
        description: Option<String>,
    },
    /// Restore an archived change
    Restore {
        /// Change reference
        change: String,
        /// Activate the restored change
        #[arg(long)]
        switch: bool,
    },
    /// List archived changes
    ArchiveList,
}

#[derive(Subcommand)]
enum RuntimeCommands {
    /// Record agent idle timestamp for a change
    SetIdle {
        /// Change reference
        change: String,
    },
    /// Clear agent idle state for a change
    ClearIdle {
        /// Change reference
        change: String,
    },
}

fn main() {
    let cli = Cli::parse();

    let result = match cli.command {
        Commands::Resolve { change, id, folder, dir, status } => {
            run_resolve(change, id, folder, dir, status)
        }
        Commands::Log { command } => run_log(command),
        Commands::Status { command } => run_status(command),
        Commands::Preflight { change_name } => run_preflight(change_name),
        Commands::Change { command } => run_change(command),
        Commands::Score { change, check_gate, stage } => run_score(change, check_gate, stage),
        Commands::Runtime { command } => run_runtime(command),
        Commands::PaneMap => run_pane_map(),
        Commands::SendKeys { change, text } => run_send_keys(change, text),
    };

    if let Err(e) = result {
        eprintln!("ERROR: {}", e);
        process::exit(1);
    }
}

fn run_resolve(
    change_arg: Option<String>,
    _id: bool, // accepted but unused — --id is the default behavior
    folder: bool,
    dir: bool,
    status: bool,
) -> anyhow::Result<()> {
    let fab_root = resolve::fab_root()?;
    let change_str = change_arg.unwrap_or_default();
    let resolved = resolve::to_folder(&fab_root, &change_str)?;

    if folder {
        println!("{}", resolved);
    } else if dir {
        println!("fab/changes/{}/", resolved);
    } else if status {
        println!("fab/changes/{}/.status.yaml", resolved);
    } else {
        // default is --id
        println!("{}", resolve::extract_id(&resolved));
    }
    Ok(())
}

fn run_log(command: LogCommands) -> anyhow::Result<()> {
    let fab_root = resolve::fab_root()?;

    match command {
        LogCommands::Command { cmd, change, args } => {
            log::command(
                &fab_root,
                &cmd,
                &change.unwrap_or_default(),
                &args.unwrap_or_default(),
            )
        }
        LogCommands::Confidence { change, score, delta, trigger } => {
            let score_val: f64 = score.parse().map_err(|_| anyhow::anyhow!("invalid score: {}", score))?;
            log::confidence_log(&fab_root, &change, score_val, &delta, &trigger)
        }
        LogCommands::Review { change, result, rework } => {
            log::review(&fab_root, &change, &result, &rework.unwrap_or_default())
        }
        LogCommands::Transition { change, stage, action, from, reason, driver } => {
            log::transition(
                &fab_root,
                &change,
                &stage,
                &action,
                &from.unwrap_or_default(),
                &reason.unwrap_or_default(),
                &driver.unwrap_or_default(),
            )
        }
    }
}

fn run_status(command: StatusCommands) -> anyhow::Result<()> {
    match command {
        StatusCommands::Show { name, all, json } => {
            if let Some(name) = name {
                let info = worktree::find_by_name(&name)?;
                if json {
                    println!("{}", worktree::format_json(&info)?);
                } else {
                    println!("{}", worktree::format_human(&info));
                }
                return Ok(());
            }

            if all {
                let infos = worktree::list()?;
                if json {
                    println!("{}", worktree::format_all_json(&infos)?);
                } else {
                    println!("{}", worktree::format_all_human(&infos));
                }
                return Ok(());
            }

            // Default: current worktree
            let info = worktree::current()?;
            if json {
                println!("{}", worktree::format_json(&info)?);
            } else {
                println!("{}", worktree::format_human(&info));
            }
            Ok(())
        }
        StatusCommands::AllStages => {
            for s in status::all_stages() {
                println!("{}", s);
            }
            Ok(())
        }
        StatusCommands::ProgressMap { change } => {
            let (sf, _, _) = load_status(&change)?;
            for ss in status::progress_map(&sf) {
                println!("{}:{}", ss.stage, ss.state);
            }
            Ok(())
        }
        StatusCommands::ProgressLine { change } => {
            let (sf, _, _) = load_status(&change)?;
            let line = status::progress_line(&sf);
            if !line.is_empty() {
                println!("{}", line);
            }
            Ok(())
        }
        StatusCommands::CurrentStage { change } => {
            let (sf, _, _) = load_status(&change)?;
            println!("{}", status::current_stage(&sf));
            Ok(())
        }
        StatusCommands::DisplayStage { change } => {
            let (sf, _, _) = load_status(&change)?;
            let (stage, state) = status::display_stage(&sf);
            println!("{}:{}", stage, state);
            Ok(())
        }
        StatusCommands::Checklist { change } => {
            let (sf, _, _) = load_status(&change)?;
            println!("generated:{}", sf.checklist.generated);
            println!("completed:{}", sf.checklist.completed);
            println!("total:{}", sf.checklist.total);
            Ok(())
        }
        StatusCommands::Confidence { change } => {
            let (sf, _, _) = load_status(&change)?;
            println!("certain:{}", sf.confidence.certain);
            println!("confident:{}", sf.confidence.confident);
            println!("tentative:{}", sf.confidence.tentative);
            println!("unresolved:{}", sf.confidence.unresolved);
            println!("score:{:.1}", sf.confidence.score);
            let indicative = sf.confidence.indicative.unwrap_or(false);
            println!("indicative:{}", indicative);
            Ok(())
        }
        StatusCommands::ValidateStatusFile { change } => {
            let (sf, _, _) = load_status(&change)?;
            status::validate(&sf)
        }
        StatusCommands::Start { change, stage, driver, from, reason } => {
            let (mut sf, status_path, fab_root) = load_status(&change)?;
            status::start(
                &mut sf, &status_path, &fab_root, &stage,
                &driver.unwrap_or_default(),
                &from.unwrap_or_default(),
                &reason.unwrap_or_default(),
            )
        }
        StatusCommands::Advance { change, stage, driver } => {
            let (mut sf, status_path, _fab_root) = load_status(&change)?;
            status::advance(&mut sf, &status_path, &stage, &driver.unwrap_or_default())
        }
        StatusCommands::Finish { change, stage, driver } => {
            let (mut sf, status_path, fab_root) = load_status(&change)?;
            status::finish(
                &mut sf, &status_path, &fab_root, &stage,
                &driver.unwrap_or_default(),
            )
        }
        StatusCommands::Reset { change, stage, driver, from, reason } => {
            let (mut sf, status_path, fab_root) = load_status(&change)?;
            status::reset(
                &mut sf, &status_path, &fab_root, &stage,
                &driver.unwrap_or_default(),
                &from.unwrap_or_default(),
                &reason.unwrap_or_default(),
            )
        }
        StatusCommands::Skip { change, stage, driver } => {
            let (mut sf, status_path, fab_root) = load_status(&change)?;
            status::skip(
                &mut sf, &status_path, &fab_root, &stage,
                &driver.unwrap_or_default(),
            )
        }
        StatusCommands::Fail { change, stage, driver, rework } => {
            let (mut sf, status_path, fab_root) = load_status(&change)?;
            status::fail(
                &mut sf, &status_path, &fab_root, &stage,
                &driver.unwrap_or_default(),
                &rework.unwrap_or_default(),
            )
        }
        StatusCommands::SetChangeType { change, r#type } => {
            let (mut sf, status_path, _) = load_status(&change)?;
            status::set_change_type(&mut sf, &status_path, &r#type)
        }
        StatusCommands::SetChecklist { change, field, value } => {
            let (mut sf, status_path, _) = load_status(&change)?;
            status::set_checklist(&mut sf, &status_path, &field, &value)
        }
        StatusCommands::SetConfidence {
            change, certain, confident, tentative, unresolved, score, indicative,
        } => {
            let (mut sf, status_path, _) = load_status(&change)?;
            let certain: i64 = certain.parse().map_err(|_| anyhow::anyhow!("invalid value for 'certain' ({:?})", certain))?;
            let confident: i64 = confident.parse().map_err(|_| anyhow::anyhow!("invalid value for 'confident' ({:?})", confident))?;
            let tentative: i64 = tentative.parse().map_err(|_| anyhow::anyhow!("invalid value for 'tentative' ({:?})", tentative))?;
            let unresolved: i64 = unresolved.parse().map_err(|_| anyhow::anyhow!("invalid value for 'unresolved' ({:?})", unresolved))?;
            let score: f64 = score.parse().map_err(|_| anyhow::anyhow!("invalid value for 'score' ({:?})", score))?;
            status::set_confidence(
                &mut sf, &status_path,
                certain, confident, tentative, unresolved, score, indicative,
            )
        }
        StatusCommands::SetConfidenceFuzzy {
            change, certain, confident, tentative, unresolved, score,
            mean_s, mean_r, mean_a, mean_d, indicative,
        } => {
            let (mut sf, status_path, _) = load_status(&change)?;
            let certain: i64 = certain.parse().map_err(|_| anyhow::anyhow!("invalid value for 'certain' ({:?})", certain))?;
            let confident: i64 = confident.parse().map_err(|_| anyhow::anyhow!("invalid value for 'confident' ({:?})", confident))?;
            let tentative: i64 = tentative.parse().map_err(|_| anyhow::anyhow!("invalid value for 'tentative' ({:?})", tentative))?;
            let unresolved: i64 = unresolved.parse().map_err(|_| anyhow::anyhow!("invalid value for 'unresolved' ({:?})", unresolved))?;
            let score: f64 = score.parse().map_err(|_| anyhow::anyhow!("invalid value for 'score' ({:?})", score))?;
            let mean_s: f64 = mean_s.parse().map_err(|_| anyhow::anyhow!("invalid value for 'mean_s' ({:?})", mean_s))?;
            let mean_r: f64 = mean_r.parse().map_err(|_| anyhow::anyhow!("invalid value for 'mean_r' ({:?})", mean_r))?;
            let mean_a: f64 = mean_a.parse().map_err(|_| anyhow::anyhow!("invalid value for 'mean_a' ({:?})", mean_a))?;
            let mean_d: f64 = mean_d.parse().map_err(|_| anyhow::anyhow!("invalid value for 'mean_d' ({:?})", mean_d))?;
            status::set_confidence_fuzzy(
                &mut sf, &status_path,
                certain, confident, tentative, unresolved, score,
                mean_s, mean_r, mean_a, mean_d, indicative,
            )
        }
        StatusCommands::AddIssue { change, id } => {
            let (mut sf, status_path, _) = load_status(&change)?;
            status::add_issue(&mut sf, &status_path, &id)
        }
        StatusCommands::GetIssues { change } => {
            let (sf, _, _) = load_status(&change)?;
            for id in &sf.issues {
                println!("{}", id);
            }
            Ok(())
        }
        StatusCommands::AddPr { change, url } => {
            let (mut sf, status_path, _) = load_status(&change)?;
            status::add_pr(&mut sf, &status_path, &url)
        }
        StatusCommands::GetPrs { change } => {
            let (sf, _, _) = load_status(&change)?;
            for url in &sf.prs {
                println!("{}", url);
            }
            Ok(())
        }
    }
}

fn run_preflight(change_name: Option<String>) -> anyhow::Result<()> {
    let fab_root = resolve::fab_root()?;
    let change_override = change_name.unwrap_or_default();
    let result = preflight::run(&fab_root, &change_override)?;
    print!("{}", preflight::format_yaml(&result));
    Ok(())
}

fn run_change(command: ChangeCommands) -> anyhow::Result<()> {
    let fab_root = resolve::fab_root()?;

    match command {
        ChangeCommands::New { slug, change_id, log_args } => {
            let folder = change::new(
                &fab_root,
                &slug.unwrap_or_default(),
                &change_id.unwrap_or_default(),
                &log_args.unwrap_or_default(),
            )?;
            println!("{}", folder);
            Ok(())
        }
        ChangeCommands::Rename { folder, slug } => {
            let new_name = change::rename(
                &fab_root,
                &folder.unwrap_or_default(),
                &slug.unwrap_or_default(),
            )?;
            println!("{}", new_name);
            Ok(())
        }
        ChangeCommands::Switch { name, blank } => {
            if blank {
                println!("{}", change::switch_blank(&fab_root));
                return Ok(());
            }
            let name = name.ok_or_else(|| anyhow::anyhow!("switch requires <name> or --blank"))?;
            let output = change::switch(&fab_root, &name)?;
            println!("{}", output);
            Ok(())
        }
        ChangeCommands::List { archive: archive_flag } => {
            let results = change::list(&fab_root, archive_flag)?;
            for r in &results {
                println!("{}", r);
            }
            Ok(())
        }
        ChangeCommands::Resolve { r#override } => {
            let folder = change::resolve_change(&fab_root, &r#override.unwrap_or_default())?;
            println!("{}", folder);
            Ok(())
        }
        ChangeCommands::Archive { change, description } => {
            let change_arg = match &change {
                Some(c) if !c.is_empty() => c.as_str(),
                _ => return Ok(()),  // mimic Go: no args => show help (no-op for now)
            };
            let result = archive::archive(
                &fab_root,
                change_arg,
                &description.unwrap_or_default(),
            )?;
            println!("{}", archive::format_archive_yaml(&result));
            Ok(())
        }
        ChangeCommands::Restore { change, switch: do_switch } => {
            let result = archive::restore(&fab_root, &change, do_switch)?;
            println!("{}", archive::format_restore_yaml(&result));
            Ok(())
        }
        ChangeCommands::ArchiveList => {
            let results = archive::list(&fab_root)?;
            for r in &results {
                println!("{}", r);
            }
            Ok(())
        }
    }
}

fn run_score(change: String, check_gate: bool, stage: String) -> anyhow::Result<()> {
    let fab_root = resolve::fab_root()?;

    if check_gate {
        let result = score::check_gate(&fab_root, &change, &stage)?;
        println!("{}", score::format_gate_yaml(&result));
        return Ok(());
    }

    let result = score::compute(&fab_root, &change, &stage)?;
    print!("{}", score::format_score_yaml(&result));
    Ok(())
}

fn run_runtime(command: RuntimeCommands) -> anyhow::Result<()> {
    let fab_root = resolve::fab_root()?;

    match command {
        RuntimeCommands::SetIdle { change } => runtime::set_idle(&fab_root, &change),
        RuntimeCommands::ClearIdle { change } => runtime::clear_idle(&fab_root, &change),
    }
}

fn run_pane_map() -> anyhow::Result<()> {
    panemap::run_pane_map()
}

fn run_send_keys(change: String, text: String) -> anyhow::Result<()> {
    sendkeys::run_send_keys(&change, &text)
}

fn load_status(change_arg: &str) -> anyhow::Result<(statusfile::StatusFile, String, String)> {
    let fab_root = resolve::fab_root()?;
    let status_path = resolve::to_abs_status(&fab_root, change_arg)?;
    let sf = statusfile::load(&status_path)?;
    Ok((sf, status_path, fab_root))
}
