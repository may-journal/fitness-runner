package langs

import (
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang"
	"github.com/may-journal/fitness-runner/go/internal/lang/golang/toolchain"
	goViews "github.com/may-journal/fitness-runner/go/internal/lang/golang/views"
)

// Go is the Go language: a repo with a go.mod outside test fixtures.
var Go = lang.Module{
	Name:            "go",
	Detect:          toolchain.Applies,
	Exts:            []string{golang.Ext},
	Manifests:       []string{golang.Manifest},
	Lockfiles:       []string{golang.Sum},
	Tools:           []string{"go", "gofmt"},
	Checks:          []string{"go-complexity", "go-vet", "gofmt", "go-test"},
	Dicts:           []string{"golang.txt"},
	Technology:      golang.Technology,
	TechnologyWords: strings.Fields("go golang"),
	Runtime:         golang.Runtime,
	Views:           goViews.Reader,
	Inventory:       []string{golang.Ext},
	Tests:           golang.Tests,
}
