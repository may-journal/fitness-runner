// Package runtimes finds what a repository ships: the runtime products its
// build targets, entry points, and deploy files prove, and the platform
// services its code uses. A dependency manifest alone proves nothing. Each
// language module brings a Reader; this package knows no language itself.
package runtimes

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Location is a one-based line in tracked repository content.
type Location struct {
	Path string
	Line int
}

func (l Location) String() string { return fmt.Sprintf("%s:%d", l.Path, l.Line) }

// Product is one runtime the repository builds, such as a command, an app
// target, or a published package. Paths are the folders it is built from;
// Technology is the language its container must name. Entries are the
// paths a package publishes, built or not. Package marks a product a
// container may name by its package name or an entry path.
type Product struct {
	Kind, Name, Technology string
	Paths, Entries         []string
	Package                bool
	At                     Location
}

// Service is a platform service product code uses.
type Service struct {
	Name string
	At   Location
}

// Report lists every product and service found, with the evidence the
// adapters could not read.
type Report struct {
	Products []Product
	Services []Service
	Errors   []string
}

// Reader is what one language module brings to runtime discovery.
type Reader struct {
	// Manifests read build and package files, by file name.
	Manifests map[string]func(*Finder, string)
	// Sources read product source files, by extension.
	Sources map[string]func(*Finder, string)
	// Scripts read what a manifest's scripts run, by manifest file name.
	Scripts map[string]func(root, file string) Runs
}

// Runs is what one manifest's scripts run and publish, as repository
// paths: Start for the start script, Dev for the others.
type Runs struct {
	Dev, Start, Published []string
}

// Find reads the product sources and manifests among files under root
// with the readers given. product reports whether a tracked file is
// product code rather than a test, tool, or generated file.
func Find(root string, files []string, product func(string) bool, readers []Reader) Report {
	f := &Finder{root: root, exists: map[string]bool{}}
	for _, file := range files {
		f.remember(file)
	}
	for _, file := range files {
		if product(file) {
			f.read(file, readers)
		}
	}
	sort.Slice(f.Products, func(i, j int) bool { return f.Products[i].At.String() < f.Products[j].At.String() })
	return f.Report
}

// Finder collects the Report while readers add to it.
type Finder struct {
	Report
	root   string
	exists map[string]bool
}

func (f *Finder) remember(file string) {
	f.exists[file] = true
	for dir := path.Dir(file); dir != "."; dir = path.Dir(dir) {
		f.exists[dir] = true
	}
}

// Known reports whether a product of kind is already built from dir.
func (f *Finder) Known(kind, dir string) bool {
	for _, p := range f.Products {
		if p.Kind == kind && p.Paths[0] == dir {
			return true
		}
	}
	return false
}

// Root is the repository root being read.
func (f *Finder) Root() string { return f.root }

// Exists reports a tracked file, or a folder holding one.
func (f *Finder) Exists(p string) bool { return f.exists[p] }

// Manifest reports a build or package file some reader reads by name. A
// manifest proves runtimes even when .gitattributes marks it
// linguist-generated, since some build tools write their project file
// themselves.
func Manifest(file string, readers []Reader) bool {
	return manifest(file, readers) != nil
}

// manifest is the reader for file's name, or nil.
func manifest(file string, readers []Reader) func(*Finder, string) {
	for _, r := range readers {
		if read := r.Manifests[path.Base(file)]; read != nil {
			return read
		}
	}
	return nil
}

func (f *Finder) read(file string, readers []Reader) {
	if read := manifest(file, readers); read != nil {
		read(f, file)
		return
	}
	for _, r := range readers {
		if read := r.Sources[path.Ext(file)]; read != nil {
			read(f, file)
			return
		}
	}
}

// Fail records evidence that could not be read.
func (f *Finder) Fail(at Location, format string, args ...any) {
	f.Errors = append(f.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

// Lines reads a tracked file as lines; an unreadable file is an error.
func (f *Finder) Lines(file string) ([]string, bool) {
	body, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(file)))
	if err != nil {
		f.Fail(Location{file, 1}, "cannot read: %v", err)
		return nil, false
	}
	return strings.Split(string(body), "\n"), true
}

// Beside joins a manifest's folder with a path it names, keeping "." for
// the repository root.
func Beside(manifest, rel string) string {
	return path.Clean(path.Join(path.Dir(manifest), rel))
}
