# Operator

The operator is owned by HexoKit. Use `rk operator` to launch it, `rk skill
operator` to read its agent-facing skill, and `rk operator autopilot` for merge
automation.

Fab retains only the `operator` agent role through `fab agent operator -o yaml`;
HexoKit uses that resolution when launching the coordinator. Fab ships no
`/fab-operator` skill at project or user level.

`fab sync` removes stale user-level copies that carry fab's generated-by marker
while preserving foreign files and symlinks. Fab's former operator state file is
left on disk for `rk operator adopt` to read.
