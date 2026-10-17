// Package java is the Java language module's home. It holds what the Java
// checks share: when a repo counts as a Java project, which Maven and
// Gradle builds it holds, how a check finds the build tool and the JDK, and
// how it runs a build tool plugin once per build.
//
// A Java project is a folder with a Maven pom.xml or a Gradle build or
// settings script. Build files under a src folder are test fixtures, never
// a project. A build root is a project no folder above it builds with the
// same tool, so one run from the root covers a multi-module build. The
// build output git must not track is target/ for Maven, build/ and .gradle/
// for Gradle, and compiled .class files anywhere. The packages in this
// folder hold each check's own logic: spotless, checkstyle and jdk.
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
		if slices.Contains(b.Files, path.Base(f)) && !fixture(f) {
			dirs = append(dirs, path.Dir(f))
		}
	}
	slices.Sort(dirs)
	return slices.Compact(dirs)
}

// Roots lists the build roots: the projects no folder above them builds
// with the same tool.
func Roots(root string) []Project {
	all := Projects(root)
	var out []Project
	for _, p := range all {
		if !slices.ContainsFunc(all, func(q Project) bool { return q.Build == p.Build && within(p.Dir, q.Dir) }) {
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

// fixture reports a build file under a src folder, where only test inputs
// keep one.
func fixture(rel string) bool {
	return slices.Contains(strings.Split(path.Dir(rel), "/"), "src")
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

// BuildOutput lists the git path patterns of the build output the repo's
// projects write, plus compiled classes anywhere once a project exists.
func BuildOutput(root string) []string {
	projects := Projects(root)
	if len(projects) == 0 {
		return nil
	}
	var out []string
	for _, p := range projects {
		for _, o := range p.Build.Output {
			out = append(out, path.Join(p.Dir, o))
		}
	}
	return append(out, "*.class")
}

// Declares reports whether any of the repo's build scripts for b matches
// the plugin pattern.
func Declares(root string, b *Build, plugin *regexp.Regexp) bool {
	for _, f := range walkfs.FilesByExt(root, b.Scripts...) {
		if body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f))); err == nil && plugin.Match(body) {
			return true
		}
	}
	return false
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
