package langs

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
	"github.com/may-journal/fitness-runner/go/internal/toolchain"
)

// Go is the Go language: a repo with a go.mod outside test fixtures.
var Go = lang.Module{
	Name:            "go",
	Detect:          func(root string) bool { return len(toolchain.Modules(root)) > 0 },
	Exts:            []string{golang.Ext},
	Manifests:       []string{golang.Manifest},
	Lockfiles:       []string{"go.sum"},
	Tools:           []string{"go", "gofmt"},
	Checks:          []string{"go-complexity", "go-vet", "gofmt", "go-test"},
	Dicts:           []string{"golang.txt"},
	Technology:      golang.Technology,
	TechnologyWords: strings.Fields("go golang"),
	Runtime:         golang.Runtime,
	Tests:           golang.Tests,
}
