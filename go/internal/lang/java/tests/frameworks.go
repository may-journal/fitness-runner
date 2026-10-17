package tests

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// frameworkFiles holds one file per test framework or test library the
// check spots. Each says in comments how the framework reaches the build's
// JUnit XML, then gives `key: value` lines: its name, the build coordinate
// groups and import prefixes that reveal it, the endings of test files it
// runs, a hint for a build whose tests ran none, and a command that must
// succeed before its tests can run, with the finding when it fails. A
// framework of Java tests also names the annotations that mark a test
// method, and those that mark a class whose public methods all run as
// tests.
//
//go:embed frameworks
var frameworkFiles embed.FS

// framework is one entry of the frameworks folder.
type framework struct {
	name    string
	groups  []string
	imports []string
	files   []string
	hint    string
	ready   []string
	unready string
	tests   []string
	classes []string
}

// frameworks reads the frameworks folder once, in file name order.
var frameworks = sync.OnceValue(func() []framework {
	entries, err := fs.ReadDir(frameworkFiles, "frameworks")
	if err != nil {
		panic("tests: " + err.Error())
	}
	out := make([]framework, 0, len(entries))
	for _, e := range entries {
		body, err := fs.ReadFile(frameworkFiles, path.Join("frameworks", e.Name()))
		if err != nil {
			panic("tests: " + err.Error())
		}
		out = append(out, parse(string(body)))
	}
	return out
})

// keys sets each field a framework file's key names from its value.
var keys = map[string]func(f *framework, value string){
	"name":    func(f *framework, v string) { f.name = v },
	"group":   func(f *framework, v string) { f.groups = strings.Fields(v) },
	"import":  func(f *framework, v string) { f.imports = strings.Fields(v) },
	"files":   func(f *framework, v string) { f.files = strings.Fields(v) },
	"hint":    func(f *framework, v string) { f.hint = v },
	"ready":   func(f *framework, v string) { f.ready = strings.Fields(v) },
	"unready": func(f *framework, v string) { f.unready = v },
	"tests":   func(f *framework, v string) { f.tests = strings.Fields(v) },
	"classes": func(f *framework, v string) { f.classes = strings.Fields(v) },
}

// parse reads one framework file's `key: value` lines, comments left out.
// An unknown key is a mistake in the folder, so it panics.
func parse(body string) framework {
	var f framework
	for _, line := range strings.Split(body, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		set, known := keys[strings.TrimSpace(key)]
		if !known {
			panic("tests: unknown framework key " + key)
		}
		set(&f, strings.TrimSpace(value))
	}
	return f
}

// testEndings are the endings of the test files the check reads for
// imports and counts as tests: Java sources and each framework's own files.
func testEndings() []string {
	out := []string{java.Ext}
	for _, f := range frameworks() {
		out = append(out, f.files...)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// testFiles lists the tracked test files build root p owns: the files in a
// test source set's folder with a test ending.
func testFiles(root string, p java.Project) []string {
	var out []string
	for _, f := range java.Owned(root, p, testEndings()...) {
		if java.Fixture(f) {
			out = append(out, f)
		}
	}
	return out
}

// productFiles lists the tracked Java sources build root p owns outside
// its test source sets.
func productFiles(root string, p java.Project) []string {
	var out []string
	for _, f := range java.Owned(root, p, java.Ext) {
		if !java.Fixture(f) {
			out = append(out, f)
		}
	}
	return out
}

// detect lists the frameworks build root p uses: those a build script or
// version catalog names by group, or a test file imports.
func detect(root string, p java.Project) []framework {
	scripts := readAll(root, java.Owned(root, p, append(slices.Clone(p.Build.Scripts), ".versions.toml")...))
	sources := readAll(root, testFiles(root, p))
	var out []framework
	for _, f := range frameworks() {
		if slices.ContainsFunc(f.groups, func(g string) bool { return groupRe(g).MatchString(scripts) }) ||
			slices.ContainsFunc(f.imports, func(i string) bool { return importRe(i).MatchString(sources) }) {
			out = append(out, f)
		}
	}
	return out
}

// groupRe matches a build coordinate group as a whole name, such as
// org.junit in org.junit:junit-bom or <groupId>org.junit</groupId>.
func groupRe(group string) *regexp.Regexp {
	return regexp.MustCompile(`(?:^|[^\w.-])` + regexp.QuoteMeta(group) + `(?:[^\w.-]|$)`)
}

// importRe matches an import of a name that starts with prefix.
func importRe(prefix string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?` + regexp.QuoteMeta(prefix) + `\b`)
}

// readAll joins the text of the tracked files, one per line.
func readAll(root string, files []string) string {
	var b strings.Builder
	for _, f := range files {
		if body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f))); err == nil {
			b.Write(body)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// readyTimeout bounds a readiness command, so a hung service fails fast
// instead of holding the check to its own time limit.
const readyTimeout = 10 * time.Second

// ready runs each detected framework's readiness command in p's folder and
// returns the finding of each that fails or outlasts readyTimeout.
func ready(root string, p java.Project) []string {
	var out []string
	for _, f := range detect(root, p) {
		if len(f.ready) > 0 && !succeeds(filepath.Join(root, filepath.FromSlash(p.Dir)), f.ready) {
			out = append(out, f.unready)
		}
	}
	return out
}

// succeeds runs the command in dir and reports whether it exits zero
// within readyTimeout.
func succeeds(dir string, command []string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), readyTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = dir
	return cmd.Run() == nil
}
