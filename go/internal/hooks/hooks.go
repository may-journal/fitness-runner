// Package hooks embeds the canonical may-journal git hooks that `fitness init`
// installs into a consumer repo. The hooks call the installed `fitness` binary
// on PATH, so a repo pulls them by version rather than hand-copying them.
package hooks

import "embed"

// FS holds the hook scripts under files/.
//
//go:embed files/commit-msg files/pre-commit files/pre-push
var FS embed.FS

// Names lists the installed hook filenames, in no particular order.
var Names = []string{"commit-msg", "pre-commit", "pre-push"}

// Read returns the embedded script for one hook name.
func Read(name string) ([]byte, error) {
	return FS.ReadFile("files/" + name)
}
