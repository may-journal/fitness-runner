package java

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/runtimes"
)

// Technology is the name a Java product's container must give its
// language, and the language the code view's messages name.
const Technology = "Java"

// application is the kind of product a main method proves.
const application = "Java application"

// mainRe matches a main method the java launcher can start, its first group
// holding the modifiers so a private main is ruled out.
var mainRe = regexp.MustCompile(`((?:\b(?:public|protected|private|static|final|synchronized|strictfp)\s+)*)\bvoid\s+main\s*\(\s*(?:final\s+)?(?:String\s*(?:\[\s*\]|\.\.\.)\s*\w+|String\s+\w+\s*\[\s*\])?\s*\)`)

// Runtime reads Java product sources, where a file declaring a main method
// proves one application per Java project.
var Runtime = runtimes.Reader{Sources: map[string]func(*runtimes.Finder, string){Ext: entry}}

// entry records the Java project holding file as one application the
// first time a product file's code view declares a main method. A
// commented-out main proves nothing.
func entry(f *runtimes.Finder, file string) {
	dir := projectDir(f, file)
	if f.Known(application, dir) {
		return
	}
	if line := mainLine(f, file); line > 0 {
		f.Products = append(f.Products, runtimes.Product{Kind: application, Name: applicationName(f, dir), Technology: Technology, Paths: []string{dir, path.Dir(file)}, At: runtimes.Location{Path: file, Line: line}})
	}
}

// mainLine is the line a main method's declaration starts on in file, or
// 0.
func mainLine(f *runtimes.Finder, file string) int {
	lines, _ := f.Lines(file)
	code := strings.Join(lines, "\n")
	for _, m := range mainRe.FindAllStringSubmatchIndex(code, -1) {
		if !strings.Contains(code[m[2]:m[3]], "private") {
			return strings.Count(code[:m[0]], "\n") + 1
		}
	}
	return 0
}

// projectDir is the nearest folder above file with a tracked build file, else
// the file's own folder.
func projectDir(f *runtimes.Finder, file string) string {
	for dir := path.Dir(file); ; dir = path.Dir(dir) {
		if builds(f, dir) {
			return dir
		}
		if dir == "." {
			return path.Dir(file)
		}
	}
}

// builds reports a tracked build file in dir, fixtures left out, as
// Projects finds them.
func builds(f *runtimes.Finder, dir string) bool {
	for _, b := range Builds {
		for _, name := range b.Files {
			if file := path.Join(dir, name); f.Exists(file) && !Fixture(file) {
				return true
			}
		}
	}
	return false
}

// applicationName is the project folder's name, or the repository
// folder's for the root project.
func applicationName(f *runtimes.Finder, dir string) string {
	if dir == "." {
		return filepath.Base(f.Root())
	}
	return path.Base(dir)
}
