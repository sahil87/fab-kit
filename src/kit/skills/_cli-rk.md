---
name: _cli-rk
description: "Run-kit (rk) capabilities — iframe windows, proxy, visual display recipe. Always loaded; fails silently if rk unavailable."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Run-Kit (rk) Capabilities

> Always loaded via `_preamble.md`. All rk usage MUST fail silently if rk is not installed — check `command -v rk` before any rk operation. Do not surface errors or warnings to the user when rk is absent.

---

## Detection

Before using any rk capability, check availability:

```sh
command -v rk >/dev/null 2>&1 || return  # in functions
command -v rk >/dev/null 2>&1            # in conditionals
```

If `rk` is not available, skip all rk operations silently. Never error, never warn.

---

## Iframe Windows

Create a tmux window that displays a web page instead of a terminal:

```sh
tmux new-window -n <name>
tmux set-option -w @rk_type iframe
tmux set-option -w @rk_url <url>
```

Change the URL of an existing iframe window:

```sh
tmux set-option -w @rk_url <new-url>
```

The rk server detects `@rk_type` and `@rk_url` changes automatically via SSE polling — no manual refresh needed.

---

## Proxy

Access local services through the rk server using the proxy URL pattern:

```
{server_url}/proxy/{port}/...
```

For example, a service on port 8080 is available at `{server_url}/proxy/8080/`.

---

## Server URL Discovery

Discover the server URL at **use-time** by running:

```sh
rk context 2>/dev/null | grep 'Server URL' | awk '{print $NF}'
```

Never hardcode the server URL — it can change between sessions.

---

## Visual Display Recipe

Any skill that wants to show HTML content to the user follows this centralized 4-step recipe. Fail silently at any step if rk is unavailable or a step fails.

1. **Generate HTML** to a known location (e.g., `~/.agent/diagrams/`, a temp directory, or the project tree)
2. **Serve it** — start a local HTTP server bound to loopback (not exposed on LAN):
   ```sh
   python3 -m http.server --bind 127.0.0.1 <port> -d <dir> &
   ```
3. **Open an iframe window** pointing to the proxy URL:
   ```sh
   tmux new-window -n <name>
   tmux set-option -w @rk_type iframe
   tmux set-option -w @rk_url {server_url}/proxy/<port>/<filename>
   ```
4. **Fail silently** — if any step fails (rk missing, port in use, server start fails), skip remaining steps without error

### Visual-Explainer Integration

When the `visual-explainer` plugin is available, skills MAY delegate HTML generation to it (Step 1), then follow Steps 2–4 above to display the result. If visual-explainer is not available, skip the visual display entirely — no error, no fallback.
