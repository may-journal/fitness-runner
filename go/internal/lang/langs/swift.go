package langs

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/swift"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Swift is the Swift language: a repo with tracked .swift sources.
var Swift = lang.Module{
	Name:            "swift",
	Detect:          func(root string) bool { return len(walkfs.FilesByExt(root, swift.Ext)) > 0 },
	Exts:            []string{swift.Ext},
	Manifests:       []string{"Package.swift"},
	Lockfiles:       []string{"Package.resolved"},
	Tools:           []string{"swiftlint"},
	Checks:          []string{"swiftlint"},
	Technology:      swift.Technology,
	TechnologyWords: strings.Fields("swift swiftui objective-c ios macos watchos tvos visionos xcode"),
	Runtime:         swift.Runtime,
	Tests:           swift.Tests,
}
