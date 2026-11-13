---
name: fab-operator
description: "Use when coordinating multiple fab agents across tmux panes — opens the operator chat (tracked items, monitoring, routing, spawning) through HexoKit's operator skill (`rk skill operator`), with fab's legacy operator skill as the fallback when rk is absent or older."
---

# /fab-operator

Thin pointer. The operator chat skill is owned by HexoKit (`rk`); fab keeps only a fallback copy.
Do NOT load the `_preamble` always-load layer — this pointer needs no project context.

## Step 1 — Ask rk for the operator skill

Run, as one probe:

```
command -v rk >/dev/null 2>&1 && rk skill operator
```

The probe **succeeds** when it exits 0 **and** the output's first heading names the operator topic
(`# HexoKit skill: operator`). Anything else — rk absent, an older rk with no `skill operator`,
any other output — is a **failure**.

## Step 2 — On success: follow rk's skill

Follow the `rk skill operator` output as the operator chat skill for the rest of the session.
Do not read the legacy skill.

Hand-over hint (once, at startup): run `rk operator adopt --dry-run`. If it reports items to
import, tell the user that `rk operator adopt` imports fab's tracked items into rk — it is
idempotent and only reads fab's state file. Never run `rk operator adopt` yourself.

## Step 3 — On failure: follow the legacy skill

Read the legacy operator skill and follow it:

1. `.agents/skills/_operator-legacy/SKILL.md` (the project's deployed copy), else
2. `$(fab kit-path)/skills/_operator-legacy.md` (the flat kit copy — where it or `_preamble` says
   to read `.agents/skills/{x}/SKILL.md`, read `$(fab kit-path)/skills/{x}.md` instead).

That skill is today's operator, driven by fab's own `fab operator` verbs.
