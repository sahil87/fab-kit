# idea - Idea Backlog Manager

Full CRUD for a per-repo ideas backlog. Stores ideas as markdown checkboxes in `fab/backlog.md` at your git root. When run from a worktree, ideas are stored in the main repository.

Each idea gets a random 4-char ID and a date stamp:

```
- [ ] [a3f2] 2025-06-01: Some cool idea
- [x] [b7k9] 2025-05-28: DES-123 Already shipped this one
```

To link an idea to a Linear ticket, add the ticket ID at the start of the description.

## Commands

### idea \<text\>

Add a new idea to the backlog.

```bash
idea "some cool idea"
idea --id abcd "specific slug"
idea --date 2025-01-15 "backdated idea"
```

| Option | Description |
|---|---|
| `--id <slug>` | Use a specific 4-char ID instead of random |
| `--date <YYYY-MM-DD>` | Use a specific date instead of today |

Errors if `--id` conflicts with an existing entry.

### idea list

List ideas.

```bash
idea list              # Open ideas only
idea list -a           # All ideas
idea list --done       # Completed ideas only
idea list --json       # JSON output
idea list --sort id    # Sort by ID instead of date
idea list --reverse    # Reverse sort order
```

| Option | Description |
|---|---|
| `-a` | List all ideas, including completed ones |
| `--done` | List only completed ideas |
| `--json` | Output as JSON array |
| `--sort <field>` | Sort by `date` (default) or `id` |
| `--reverse` | Reverse sort order |

### idea show \<query\>

Show a single idea by ID or text match.

```bash
idea show a3f2
idea show "cool idea"
idea show a3f2 --json
```

| Option | Description |
|---|---|
| `--json` | Output as JSON object |

### idea done \<query\>

Mark an idea as completed.

```bash
idea done a3f2
idea done "cool idea"
```

### idea reopen \<query\>

Reopen a completed idea.

```bash
idea reopen b7k9
```

### idea edit \<query\> \<new-text\>

Modify an idea's text, preserving ID, date, and status.

```bash
idea edit a3f2 "revised cool idea"
idea edit a3f2 "revised" --date 2025-02-01
idea edit a3f2 "revised" --id zz99
```

| Option | Description |
|---|---|
| `--date <YYYY-MM-DD>` | Also update the date |
| `--id <slug>` | Also change the ID (errors on conflict) |

### idea rm \<query\>

Delete an idea entirely. Prompts for confirmation unless `--force` is passed.

```bash
idea rm a3f2
idea rm a3f2 --force
```

| Option | Description |
|---|---|
| `--force` | Skip confirmation prompt |

## Global Options

Available on all commands:

| Option | Description |
|---|---|
| `-h, --help` | Show usage |
| `--file <path>` | Override backlog file (relative to git root) |

Precedence: `--file` flag > `IDEAS_FILE` env var > default `fab/backlog.md`.

| Environment Variable | Default | Description |
|---|---|---|
| `IDEAS_FILE` | `fab/backlog.md` | Backlog file path, relative to git root |

## Claude Code Integration

The companion Claude Code skill (`cc/plugins/idea`) auto-detects idea management intent from conversation context — no slash command needed.
