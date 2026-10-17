package langs

import (
	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Swift is the Swift language: a repo with tracked .swift sources.
var Swift = lang.Module{
	Name:      "swift",
	Detect:    func(root string) bool { return len(walkfs.FilesByExt(root, ".swift")) > 0 },
	Exts:      []string{".swift"},
	Manifests: []string{"Package.swift"},
	Lockfiles: []string{"Package.resolved"},
	Tools:     []string{"swiftlint"},
	Checks:    []string{"swiftlint"},
}
