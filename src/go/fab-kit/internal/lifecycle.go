package internal

// LifecycleCommand pairs a fab-kit workspace command name with its cobra
// Short description.
type LifecycleCommand struct {
	Name  string
	Short string
}

// LifecycleCommands is the single source of truth for the fab-kit workspace
// command set. Everything else derives from this table:
//
//   - the `fab` router's fabKitArgs allowlist (cmd/fab/main.go)
//   - the router's "Workspace commands" help section (rendered in-process so
//     help works even when the fab-kit binary is absent)
//   - cmd/fab-kit's fabKitCommands map and its registration cross-check test
//     (each registered cobra command's Short must match the table entry)
//   - the `_cli-fab.md` router-line contract test and the fab module's
//     command-name collision test (both parse the documented allowlist)
//
// The Short strings here are the canonical help text — the cobra commands in
// cmd/fab-kit must register the same Short, enforced by test.
var LifecycleCommands = []LifecycleCommand{
	{Name: "init", Short: "Initialize fab in the current repo"},
	{Name: "upgrade-repo", Short: "Upgrade to a specific or latest version"},
	{Name: "sync", Short: "Sync workspace (skills, directories, scaffold)"},
	{Name: "update", Short: "Update fab-kit itself via Homebrew"},
	{Name: "doctor", Short: "Validate fab-kit prerequisites"},
	{Name: "migrations-status", Short: "Report which migrations apply between the local and engine versions"},
}

// LifecycleCommandSet returns the command names as a membership set — the
// shape the router's dispatch switch and cmd/fab-kit's test map consume.
func LifecycleCommandSet() map[string]bool {
	set := make(map[string]bool, len(LifecycleCommands))
	for _, c := range LifecycleCommands {
		set[c.Name] = true
	}
	return set
}
