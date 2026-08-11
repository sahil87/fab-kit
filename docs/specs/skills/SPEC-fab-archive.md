# fab-archive
Archives a completed change (post-hydrate) or restores an archived one. All mechanics delegate to `fab change archive`/`fab change restore`; the skill only formats the YAML output.
## Flow
```
User invokes /fab-archive [change-name | restore <name> [--switch]]
├─ Read: _preamble.md
├─ Archive: Bash: fab preflight → [hydrate not done] STOP → Bash: fab change archive <change>
├─ Restore: Bash: fab change restore <name> [--switch] (preflight + hydrate guard waived)
└─ Format report (index/pointer results; failed = op done, exit non-zero)
```
### Tools used
Read (preamble), Bash (`fab preflight`, `fab change archive`, `fab change restore`, `fab change archive-list`).
### Sub-agents
None.
