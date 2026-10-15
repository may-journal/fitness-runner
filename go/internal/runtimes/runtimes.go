// Package runtimes finds what a repository ships: the runtime products its
// build targets, entry points, and deploy files prove, and the platform
// services its code uses. A dependency manifest alone proves nothing.
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

// Product is one runtime the repository builds: a Go command, an Xcode app
// or extension target, or a Node service or library. Paths are the folders
// it is built from; Technology is the language its container must name.
// Entries are the paths a package.json publishes, built or not.
type Product struct {
	Kind, Name, Technology string
	Paths, Entries         []string
	At                     Location
}

// Service is a platform service product code uses, such as CloudKit.
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

// Find reads the product sources and manifests among files under root.
// product reports whether a tracked file is product code rather than a
// test, tool, or generated file.
func Find(root string, files []string, product func(string) bool) Report {
	f := &finder{root: root, exists: map[string]bool{}}
	for _, file := range files {
		f.remember(file)
	}
	for _, file := range files {
		if product(file) {
			f.read(file)
		}
	}
	sort.Slice(f.Products, func(i, j int) bool { return f.Products[i].At.String() < f.Products[j].At.String() })
	return f.Report
}

type finder struct {
	Report
	root   string
	exists map[string]bool
}

func (f *finder) remember(file string) {
	f.exists[file] = true
	for dir := path.Dir(file); dir != "."; dir = path.Dir(dir) {
		f.exists[dir] = true
	}
}

// manifests are the build and package files read by name.
var manifests = map[string]func(*finder, string){
	"project.yml":     (*finder).xcodegen,
	"project.pbxproj": (*finder).pbxproj,
	"package.json":    (*finder).node,
}

// Manifest reports a build or package file read by name, such as
// project.pbxproj. A manifest proves runtimes even when .gitattributes marks
// it linguist-generated, since Xcode writes the project file itself.
func Manifest(file string) bool {
	return manifests[path.Base(file)] != nil
}

func (f *finder) read(file string) {
	if manifest := manifests[path.Base(file)]; manifest != nil {
		manifest(f, file)
		return
	}
	if ext := path.Ext(file); ext == ".go" {
		f.goCommand(file)
	} else if ext == ".swift" {
		f.platformServices(file)
	}
}

func (f *finder) fail(at Location, format string, args ...any) {
	f.Errors = append(f.Errors, at.String()+": "+fmt.Sprintf(format, args...))
}

// lines reads a tracked file as lines; an unreadable file is an error.
func (f *finder) lines(file string) ([]string, bool) {
	body, err := os.ReadFile(filepath.Join(f.root, filepath.FromSlash(file)))
	if err != nil {
		f.fail(Location{file, 1}, "cannot read: %v", err)
		return nil, false
	}
	return strings.Split(string(body), "\n"), true
}

// beside joins a manifest's folder with a path it names, keeping "." for
// the repository root.
func beside(manifest, rel string) string {
	return path.Clean(path.Join(path.Dir(manifest), rel))
}

// lineOf finds the first line containing text, or 1.
func lineOf(lines []string, text string) int {
	for i, line := range lines {
		if strings.Contains(line, text) {
			return i + 1
		}
	}
	return 1
}
