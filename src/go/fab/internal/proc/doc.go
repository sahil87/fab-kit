// Package proc provides platform-specific helpers for walking the process
// tree. The primary consumer is the Claude Code hook handler, which needs to
// identify Claude's own PID. Hooks are invoked as
// `claude → sh -c '<command>' → hook`, so the hook's immediate parent is the
// `sh` process and its grandparent is Claude itself. This package exposes a
// single-function API — ClaudePID — that walks up one level past the shell
// wrapper to return Claude's PID.
//
// The platform split mirrors the convention used elsewhere in the codebase
// (see cmd/fab/pane_process_{linux,darwin}.go): Linux reads /proc/$PPID/status;
// macOS shells out to `ps -o ppid= -p $PPID`.
package proc
