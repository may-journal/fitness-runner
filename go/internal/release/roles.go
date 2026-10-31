package release

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/mermaid"
)

// containerDirective names, in a package doc comment, the container of the
// containers view the package is built into: `//fitness:container <id>`.
const containerDirective = "//fitness:container "

// componentDirective names, in a package doc comment, the component of the
// components view the package belongs to, and why:
// `//fitness:component <id> <reason>`.
const componentDirective = "//fitness:component "

// containersView and componentsView are the views whose element IDs a
// package's directives name, relative to the repository root.
const (
	containersView = "docs/architecture/02-containers.md"
	componentsView = "docs/architecture/03-components.md"
)

// role is the place in the architecture views a package's doc comment
// names for it.
type role struct {
	// Package is the package path under go/.
	Package string
	// Container is the ID of the container the package is built into.
	Container string
	// Component is the ID of the component the package belongs to.
	Component string
	// Reason says why the package belongs to that component.
	Reason string
}

// drawn holds the IDs of the containers and components the views draw.
type drawn struct {
	containers, components map[string]bool
}

// roles reads the container and component every Go package under go/
// names, outside the folders the go tool ignores and test-only folders, in
// package order. A package that names either of them never, twice, or by
// an ID its view does not draw, or names a component without a reason, is
// an error, each named in the one error returned.
func (c Config) roles() ([]role, error) {
	ids, err := c.drawnIDs()
	if err != nil {
		return nil, err
	}
	goDir := filepath.Join(c.Root, "go")
	dirs, err := packageDirs(goDir)
	if err != nil {
		return nil, err
	}
	return readRoles(goDir, dirs, ids)
}

// drawnIDs reads the IDs the containers and components views draw.
func (c Config) drawnIDs() (d drawn, err error) {
	if d.containers, err = c.viewIDs(containersView, "Container"); err != nil {
		return d, err
	}
	d.components, err = c.viewIDs(componentsView, "Component")
	return d, err
}

// packageDirs lists every folder under goDir, outside the folders the go
// tool ignores, relative to goDir with slashes, in walk order.
func packageDirs(goDir string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(goDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if ignoredDir(entry.Name()) {
			return fs.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})
	return dirs, err
}

// readRoles reads the role of the package in each folder under goDir that
// holds one,
// joining what is wrong with each into the error returned.
func readRoles(goDir string, dirs []string, ids drawn) ([]role, error) {
	var roles []role
	var errs []error
	for _, dir := range dirs {
		r, ok, err := readRole(goDir, dir)
		if err != nil {
			return nil, err
		}
		if ok {
			roles, errs = append(roles, r), append(errs, r.problem(ids))
		}
	}
	return roles, errors.Join(errs...)
}

// problem is what is wrong with a role read from a package's directives,
// or nil.
func (r role) problem(ids drawn) error {
	if r.Container == "" || r.Component == "" {
		return fmt.Errorf("go/%s: its package doc comment needs one %q line and one %q line", r.Package, containerDirective+"<id>", componentDirective+"<id> <reason>")
	}
	if err := ids.unknown(r); err != nil {
		return err
	}
	if r.Reason == "" {
		return fmt.Errorf("go/%s: its %q line needs a reason after the ID", r.Package, componentDirective+r.Component)
	}
	return nil
}

// unknown fails a role whose container or component the views do not draw.
func (d drawn) unknown(r role) error {
	if !d.containers[r.Container] {
		return fmt.Errorf("go/%s: container %q is not a container %s draws", r.Package, r.Container, containersView)
	}
	if !d.components[r.Component] {
		return fmt.Errorf("go/%s: component %q is not a component %s draws", r.Package, r.Component, componentsView)
	}
	return nil
}

// readRole reads the directives of the package in dir, a folder under the
// repository's go folder goDir, reporting whether dir holds a package: a Go
// source that is not a test.
func readRole(goDir, dir string) (role, bool, error) {
	files, err := parseSources(token.NewFileSet(), dir, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil || len(files) == 0 {
		return role{}, false, err
	}
	rel, err := filepath.Rel(goDir, dir)
	r := role{Package: filepath.ToSlash(rel)}
	containers, components := directiveLines(files, containerDirective), directiveLines(files, componentDirective)
	if len(containers) == 1 && len(components) == 1 {
		r.Container = containers[0]
		r.Component, r.Reason, _ = strings.Cut(components[0], " ")
		r.Reason = strings.TrimSpace(r.Reason)
	}
	return r, true, err
}

// directiveLines are the values of every line in the files' package doc
// comments that opens with prefix.
func directiveLines(files []*ast.File, prefix string) []string {
	var values []string
	for _, file := range files {
		if file.Doc == nil {
			continue
		}
		for _, line := range file.Doc.List {
			if value, ok := strings.CutPrefix(line.Text, prefix); ok {
				values = append(values, strings.TrimSpace(value))
			}
		}
	}
	return values
}

// viewIDs are the IDs of every element drawn with macro, such as
// Container, in the diagrams of the view at rel, legends left out.
func (c Config) viewIDs(rel, macro string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(c.Root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	element := regexp.MustCompile(`(?m)^\s*` + macro + `\(\s*([A-Za-z0-9_]+)\s*,`)
	ids := map[string]bool{}
	for _, body := range diagrams(string(data)) {
		for _, match := range element.FindAllStringSubmatch(body, -1) {
			ids[match[1]] = true
		}
	}
	return ids, nil
}

// diagrams are the bodies of a Markdown document's Mermaid diagrams,
// legends left out.
func diagrams(md string) []string {
	var bodies []string
	for _, block := range mermaid.ParseDoc(md) {
		if d, ok := block.(*mermaid.DiagramBlock); ok && !strings.Contains(d.Body, "title Legend") {
			bodies = append(bodies, d.Body)
		}
	}
	return bodies
}
