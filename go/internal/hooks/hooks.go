// Package hooks embeds the git hooks `fitness init` installs, as
// go/docs/hooks.md describes.
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
