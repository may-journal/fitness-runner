package java

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/codemodel"
	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Technology is the name a Java product's container must give its
// language, and the language the code view's messages name.
const Technology = "Java"

// application is the kind of product a main method proves.
const application = "Java application"

// mainRe matches a main method the java launcher can start, on one line:
// static or, since JDK 25 (JEP 512), an instance method, taking nothing or
// one String array, of fixed or variable arity.
var mainRe = regexp.MustCompile(`\bvoid\s+main\s*\(\s*(?:final\s+)?(?:String\s*(?:\[\s*\]|\.\.\.)\s*\w+|String\s+\w+\s*\[\s*\])?\s*\)`)

// Runtime reads Java product sources for applications: a product .java
// file declaring a main method proves one application per Java project,
// whose container names Java. The application is built from the
// project's folder and the main file's folder, so a container may claim
// either, or take the project folder's name.
var Runtime = runtimes.Reader{Sources: map[string]func(*runtimes.Finder, string){Ext: entry}}

// entry records the Java project holding file as one application the
// first time a product file in it declares a main method. Comments are
// stripped first, so a commented-out main proves nothing.
func entry(f *runtimes.Finder, file string) {
	dir := projectDir(f.Root(), file)
	if f.Known(application, dir) {
		return
	}
	if line := mainLine(f, file); line > 0 {
		f.Products = append(f.Products, runtimes.Product{Kind: application, Name: applicationName(f, dir), Technology: Technology, Paths: []string{dir, path.Dir(file)}, At: runtimes.Location{Path: file, Line: line}})
	}
}

// mainLine is the line declaring a main method in file, or 0.
func mainLine(f *runtimes.Finder, file string) int {
	lines, _ := f.Lines(file)
	for i, line := range codemodel.Strip(lines) {
		if mainRe.MatchString(line) {
			return i + 1
		}
	}
	return 0
}

// projectDir is the folder of the deepest Java project holding file, or
// the file's own folder outside every project.
func projectDir(root, file string) string {
	best, depth := path.Dir(file), -1
	for _, p := range Projects(root) {
		if d := folderDepth(p.Dir); inProject(p, file) && d > depth {
			best, depth = p.Dir, d
		}
	}
	return best
}

// folderDepth counts a folder's names, 0 for the repository root.
func folderDepth(dir string) int {
	if dir == "." {
		return 0
	}
	return strings.Count(dir, "/") + 1
}

// applicationName is the project folder's name, or the repository
// folder's for the root project.
func applicationName(f *runtimes.Finder, dir string) string {
	if dir == "." {
		return filepath.Base(f.Root())
	}
	return path.Base(dir)
}
