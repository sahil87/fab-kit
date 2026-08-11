# fab-help
Displays the workflow overview and command reference, grouped by category. Delegates to the `fab fab-help` Go subcommand; no context loading, no file modification.
## Flow
```
User invokes /fab-help
├─ Bash: fab log command "fab-help"
└─ Bash: fab fab-help (scans skill frontmatter, prints grouped help)
```
### Tools used
Bash (`fab log command`, `fab fab-help`).
### Sub-agents
None.
