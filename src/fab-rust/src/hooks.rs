use anyhow::{bail, Result};
use std::path::Path;
use std::process::Command;

/// Executes a shell command string in the repo root directory.
/// The command is run via "sh -c" so it supports pipes, redirects, etc.
/// If command is empty, returns Ok (no-op).
pub fn run(fab_root: &str, command: &str) -> Result<()> {
    if command.is_empty() {
        return Ok(());
    }

    let repo_root = Path::new(fab_root)
        .parent()
        .unwrap_or(Path::new("/"));

    let status = Command::new("sh")
        .arg("-c")
        .arg(command)
        .current_dir(repo_root)
        .status()?;

    if !status.success() {
        bail!("stage hook failed: {}", command);
    }
    Ok(())
}
