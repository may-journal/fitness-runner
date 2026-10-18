package version

// cspell:ignore sdkman jabba jdks libexec foojay

import (
	"bufio"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

var (
	// releaseRe reads a JDK's feature release from its release file.
	releaseRe = regexp.MustCompile(`(?m)^JAVA_VERSION="?([^"\s]+)`)
	// listedRe reads the version `java_home -V` lists for each JDK.
	listedRe = regexp.MustCompile(`(?m)^\s+(\d[^\s]*)\s+\(`)
	// resolverRe reads a settings script that configures a toolchain
	// resolver, such as the Foojay plugin.
	resolverRe = regexp.MustCompile(`toolchainManagement|foojay`)
)

// jdkParents are the folders, relative to the user's home, whose children
// are JDKs that Gradle's toolchain detection finds: its own downloads,
// SDKMAN, asdf, jabba, IntelliJ, Homebrew's versioned formulae and the
// macOS user folder.
var jdkParents = []string{
	".gradle/jdks/*",
	".sdkman/candidates/java/*",
	".asdf/installs/java/*",
	".jabba/jdk/*",
	".jdks/*",
	"Library/Java/JavaVirtualMachines/*/Contents/Home",
}

// systemParents are the folders, from the filesystem root, whose children
// are JDKs the operating system or Homebrew installs.
var systemParents = []string{
	"/usr/lib/jvm/*",
	"/usr/java/*",
	"/Library/Java/JavaVirtualMachines/*/Contents/Home",
	"/opt/homebrew/opt/openjdk*/libexec/openjdk.jdk/Contents/Home",
	"/usr/local/opt/openjdk*/libexec/openjdk.jdk/Contents/Home",
}

// javaHomeTool lists the JDKs macOS registers.
const javaHomeTool = "/usr/libexec/java_home"

// installedJDKs lists the feature release of every JDK Gradle's toolchain
// detection would find on this machine, without starting Gradle:
// JAVA_HOME, the folders above, macOS's list, and the installations
// gradle.properties names. It reads folders and files only.
func installedJDKs(root string, build java.Project) []int {
	majors := listedMajors()
	for _, h := range jdkHomes(root, build) {
		if m, ok := homeMajor(h); ok {
			majors = append(majors, m)
		}
	}
	return majors
}

// jdkHomes lists the JDK folders worth reading for a build.
func jdkHomes(root string, build java.Project) []string {
	home, _ := os.UserHomeDir()
	homes := []string{os.Getenv("JAVA_HOME")}
	for _, pattern := range jdkParents {
		homes = append(homes, glob(filepath.Join(home, filepath.FromSlash(pattern)))...)
	}
	for _, pattern := range systemParents {
		homes = append(homes, glob(pattern)...)
	}
	return append(homes, propertyHomes(root, build, home)...)
}

// glob lists the paths matching pattern, or none for a bad pattern.
func glob(pattern string) []string {
	out, _ := filepath.Glob(pattern)
	return out
}

// homeMajor reads the feature release a JDK home's release file names.
func homeMajor(dir string) (int, bool) {
	body, err := os.ReadFile(filepath.Join(dir, "release"))
	if err != nil {
		return 0, false
	}
	m := releaseRe.FindStringSubmatch(string(body))
	if m == nil {
		return 0, false
	}
	return java.Major(m[1])
}

// listedMajors reads the releases macOS's java_home lists, or none where
// the tool is missing.
func listedMajors() []int {
	if _, err := os.Stat(javaHomeTool); err != nil {
		return nil
	}
	out, _ := exec.Command(javaHomeTool, "-V").CombinedOutput()
	var majors []int
	for _, m := range listedRe.FindAllStringSubmatch(string(out), -1) {
		if n, ok := java.Major(m[1]); ok {
			majors = append(majors, n)
		}
	}
	return majors
}

// propertyHomes lists the JDK folders gradle.properties names through
// org.gradle.java.installations.paths, and the environment variables it
// names through org.gradle.java.installations.fromEnv. It reads the build
// folder's file, the repo root's and the Gradle user home's.
func propertyHomes(root string, build java.Project, home string) []string {
	userHome := os.Getenv("GRADLE_USER_HOME")
	if userHome == "" {
		userHome = filepath.Join(home, ".gradle")
	}
	files := []string{
		filepath.Join(userHome, "gradle.properties"),
		filepath.Join(root, "gradle.properties"),
		filepath.Join(root, filepath.FromSlash(build.Dir), "gradle.properties"),
	}
	var out []string
	for _, f := range files {
		props := properties(f)
		out = append(out, splitList(props["org.gradle.java.installations.paths"])...)
		for _, name := range splitList(props["org.gradle.java.installations.fromEnv"]) {
			out = append(out, os.Getenv(name))
		}
	}
	return out
}

// properties reads a Java properties file's key=value lines, or none for
// a missing file.
func properties(file string) map[string]string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	out := map[string]string{}
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if key, value, ok := strings.Cut(line, "="); ok && !strings.HasPrefix(line, "#") {
			out[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return out
}

// splitList splits a comma or path-separator list, dropping blanks.
func splitList(list string) []string {
	var out []string
	for _, item := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == os.PathListSeparator }) {
		out = append(out, strings.TrimSpace(item))
	}
	return out
}

// hasResolver reports whether the build's settings script configures a
// toolchain resolver, which lets Gradle download any JDK it lacks.
func hasResolver(root string, build java.Project) bool {
	for _, name := range []string{"settings.gradle", "settings.gradle.kts"} {
		if body, ok := read(root, path.Join(build.Dir, name)); ok && resolverRe.MatchString(body) {
			return true
		}
	}
	return false
}
