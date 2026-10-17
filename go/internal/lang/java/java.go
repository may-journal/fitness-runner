// Package java is the Java language module's home. It holds what the Java
// checks share: when a repo counts as a Java project, which Maven and
// Gradle builds it holds, how a check finds the build tool and the JDK, and
// how it runs a build tool plugin once per build.
//
// A Java project is a folder with a Maven pom.xml or a Gradle build or
// settings script. Files in a test source set's folder, such as
// src/test/resources or src/it, are test fixtures, never a project. A build
// root is a project no other project builds: Maven builds the modules a pom
// lists, and Gradle builds the folders below a settings script that hold no
// settings script of their own. One run from each root covers its modules.
// The build output git must not track is target/ for Maven, build/ and
// .gradle/ for Gradle, and compiled .class files outside fixtures. The
// packages in this folder hold each check's own logic: spotless, checkstyle
// and version.
package java

// cspell:ignore mvnw gradlew

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// Ext is the Java source ending.
const Ext = ".java"

// Build is one build tool: its build files, wrapper, command, the flags
// every run passes, and the build output it writes.
type Build struct {
	// Name labels the tool in messages.
	Name string
	// Files are the file names that mark a project the tool builds.
	Files []string
	// Scripts are the endings of every file that may apply a plugin to
	// the build, convention scripts included.
	Scripts []string
	// Wrapper is the script a repo commits to pin the tool's version.
	Wrapper string
	// Command is the tool's name on PATH.
	Command string
	// Flags run the tool without prompts, color, or a background daemon.
	Flags []string
	// Output are the folders the tool writes beside each build file.
	Output []string
	// Install is the hint for a machine with neither wrapper nor tool.
	Install string
	// Builds reports whether a run from parent builds p as one of its
	// modules.
	Builds func(root string, parent, p Project) bool
}

// Maven runs once from each root pom.xml in batch mode, without download
// progress.
var Maven = &Build{
	Name:    "Maven",
	Files:   []string{"pom.xml"},
	Scripts: []string{"pom.xml"},
	Wrapper: "mvnw",
	Command: "mvn",
	Flags:   []string{"--batch-mode", "--no-transfer-progress"},
	Output:  []string{"target"},
	Install: "Maven not installed: add the Maven wrapper (mvn wrapper:wrapper) or install Maven",
	Builds:  listsModule,
}

// Gradle runs once from each root build with the daemon off and plain
// console output.
var Gradle = &Build{
	Name:    "Gradle",
	Files:   []string{"settings.gradle", "settings.gradle.kts", "build.gradle", "build.gradle.kts"},
	Scripts: []string{".gradle", ".gradle.kts"},
	Wrapper: "gradlew",
	Command: "gradle",
	Flags:   []string{"--no-daemon", "--console=plain"},
	Output:  []string{"build", ".gradle"},
	Install: "Gradle not installed: add the Gradle wrapper (gradle wrapper) or install Gradle",
	Builds:  settlesBelow,
}

// gradleSettings are the scripts that make a folder a Gradle build of its
// own.
var gradleSettings = []string{"settings.gradle", "settings.gradle.kts"}

// moduleRe reads one module a pom lists.
var moduleRe = regexp.MustCompile(`<module>\s*([^<\s]+)\s*</module>`)

// listsModule reports whether parent's pom lists p's folder as a module.
func listsModule(root string, parent, p Project) bool {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(parent.Dir), "pom.xml"))
	if err != nil {
		return false
	}
	for _, m := range moduleRe.FindAllStringSubmatch(string(body), -1) {
		if path.Join(parent.Dir, m[1]) == p.Dir {
			return true
		}
	}
	return false
}

// settlesBelow reports whether p sits below parent's settings script and
// holds none of its own, so Gradle builds it from parent.
func settlesBelow(root string, parent, p Project) bool {
	return within(p.Dir, parent.Dir) && holds(root, parent.Dir, gradleSettings) && !holds(root, p.Dir, gradleSettings)
}

// holds reports whether dir holds any of the files.
func holds(root, dir string, files []string) bool {
	return slices.ContainsFunc(files, func(name string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir), name))
		return err == nil
	})
}

// Builds are the build tools the module reads, in the order checks run
// them.
var Builds = []*Build{Maven, Gradle}

// Manifests are every build tool's build file names.
func Manifests() []string {
	var out []string
	for _, b := range Builds {
		out = append(out, b.Files...)
	}
	return out
}

// Project is one folder a build tool builds, slash-separated and relative
// to the repo root, "." for the root itself.
type Project struct {
	Build *Build
	Dir   string
}

// Projects lists every Java project in the repo at root, by build tool
// order, then folder.
func Projects(root string) []Project {
	var out []Project
	for _, b := range Builds {
		for _, d := range projectDirs(root, b) {
			out = append(out, Project{Build: b, Dir: d})
		}
	}
	return out
}

// projectDirs lists the folders holding b's build files, fixtures left
// out, sorted.
func projectDirs(root string, b *Build) []string {
	var dirs []string
	for _, f := range walkfs.FilesByExt(root, b.Files...) {
		if slices.Contains(b.Files, path.Base(f)) && !Fixture(f) {
			dirs = append(dirs, path.Dir(f))
		}
	}
	slices.Sort(dirs)
	return slices.Compact(dirs)
}

// Roots lists the build roots: the projects no other project of the same
// tool builds as a module.
func Roots(root string) []Project {
	all := Projects(root)
	var out []Project
	for _, p := range all {
		if !slices.ContainsFunc(all, func(q Project) bool { return q.Build == p.Build && p.Build.Builds(root, q, p) }) {
			out = append(out, p)
		}
	}
	return out
}

// within reports whether dir sits strictly inside parent.
func within(dir, parent string) bool {
	if dir == parent {
		return false
	}
	return parent == "." || strings.HasPrefix(dir, parent+"/")
}

// Fixture reports a file in a test source set's folder, a src folder's
// test, it, or *Test child, where only test inputs live.
func Fixture(rel string) bool {
	segments := strings.Split(path.Dir(rel), "/")
	for i := 0; i+1 < len(segments); i++ {
		if segments[i] == "src" && testSet(segments[i+1]) {
			return true
		}
	}
	return false
}

// testSet reports a source set folder name that holds tests.
func testSet(name string) bool {
	return name == "it" || strings.HasPrefix(name, "test") || strings.HasSuffix(name, "Test")
}

// Applies reports whether the repo at root holds a Java project. It is the
// one place that rule lives.
func Applies(root string) bool {
	return len(Projects(root)) > 0
}

// Sources lists the repo's tracked Java sources.
func Sources(root string) []string {
	return walkfs.FilesByExt(root, Ext)
}

// BuildOutput lists the tracked files the repo's projects wrote as build
// output, and compiled classes outside fixtures once a project exists.
func BuildOutput(root string) []string {
	projects := Projects(root)
	if len(projects) == 0 {
		return nil
	}
	var out []string
	for _, f := range walkfs.FilesByExt(root, "") {
		if output(projects, f) {
			out = append(out, f)
		}
	}
	return out
}

// output reports whether a tracked file is build output: under a project's
// output folder, or a compiled class outside fixtures.
func output(projects []Project, file string) bool {
	if strings.HasSuffix(file, ".class") && !Fixture(file) {
		return true
	}
	return slices.ContainsFunc(projects, func(p Project) bool {
		return slices.ContainsFunc(p.Build.Output, func(o string) bool {
			return strings.HasPrefix(file, path.Join(p.Dir, o)+"/")
		})
	})
}

// Declares reports whether a build script of build root p matches the
// plugin pattern: one in p's folder or below it, outside fixtures and the
// other build roots nested there.
func Declares(root string, p Project, plugin *regexp.Regexp) bool {
	roots := Roots(root)
	return slices.ContainsFunc(walkfs.FilesByExt(root, p.Build.Scripts...), func(f string) bool {
		return owner(roots, p.Build, f) == p && !Fixture(f) && matches(root, f, plugin)
	})
}

// owner is the build root of tool b nearest above file.
func owner(roots []Project, b *Build, file string) Project {
	var best Project
	for _, r := range roots {
		if r.Build == b && inProject(r, file) && len(r.Dir) >= len(best.Dir) {
			best = r
		}
	}
	return best
}

// inProject reports whether file sits in p's folder or below it.
func inProject(p Project, file string) bool {
	return p.Dir == "." || strings.HasPrefix(file, p.Dir+"/")
}

// matches reports whether the tracked file's text matches pattern.
func matches(root, file string, pattern *regexp.Regexp) bool {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	return err == nil && pattern.Match(body)
}

// Tool resolves the command that runs p's build: the wrapper in its folder
// or any folder above it up to root, then the tool on PATH. ok is false
// when neither exists.
func Tool(root string, p Project) (bin string, ok bool) {
	if w, ok := wrapper(root, p); ok {
		return w, true
	}
	if bin, err := exec.LookPath(p.Build.Command); err == nil {
		return bin, true
	}
	return "", false
}

// wrapper finds the build's wrapper in p's folder or the nearest folder
// above it, up to root.
func wrapper(root string, p Project) (string, bool) {
	for dir := p.Dir; ; dir = path.Dir(dir) {
		if w := filepath.Join(root, filepath.FromSlash(dir), p.Build.Wrapper); executable(w) {
			return w, true
		}
		if dir == "." || dir == "/" {
			return "", false
		}
	}
}

// executable reports whether p is a file with an execute bit set.
func executable(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// Run runs p's build tool once in p's folder with the tool's flags and the
// given arguments, stdout and stderr combined.
func Run(root, bin string, p Project, args ...string) checkkit.Outcome {
	cmd := exec.Command(bin, append(slices.Clone(p.Build.Flags), args...)...)
	cmd.Dir = filepath.Join(root, filepath.FromSlash(p.Dir))
	return checkkit.ExecCombined(cmd)
}

// ToolErrors lists the lines a failed build gave as its reason: Maven's
// [ERROR] lines, or what Gradle printed under "What went wrong".
func ToolErrors(output string) []string {
	var out []string
	lines := strings.Split(output, "\n")
	for i := range lines {
		if text := reason(lines, i); text != "" {
			out = append(out, text)
		}
	}
	return out
}

// reason reads the failure reason line i gives, or empty for none.
func reason(lines []string, i int) string {
	trimmed := strings.TrimSpace(lines[i])
	if text, ok := strings.CutPrefix(trimmed, "[ERROR]"); ok {
		return strings.TrimSpace(text)
	}
	if trimmed == "* What went wrong:" && i+1 < len(lines) {
		return strings.TrimSpace(lines[i+1])
	}
	return ""
}

// Relative returns file, as a build tool printed it, relative to root when
// it sits inside root, else as given.
func Relative(root, dir, file string) string {
	abs := file
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, filepath.FromSlash(dir), file)
	}
	for _, base := range forms(root) {
		if rel, ok := inside(base, forms(abs)); ok {
			return rel
		}
	}
	return file
}

// inside returns the first target's slash path relative to base, for a
// target within base.
func inside(base string, targets []string) (string, bool) {
	for _, target := range targets {
		if rel, err := filepath.Rel(base, target); err == nil && rel != ".." && !strings.HasPrefix(rel, "../") {
			return filepath.ToSlash(rel), true
		}
	}
	return "", false
}

// forms lists p made absolute and, when it differs, with symlinks
// resolved, since a tool may print either.
func forms(p string) []string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return []string{p}
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil && real != abs {
		return []string{abs, real}
	}
	return []string{abs}
}
