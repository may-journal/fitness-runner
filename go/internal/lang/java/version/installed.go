package version

// cspell:ignore sdkman jabba jdks libexec foojay openj9

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

var (
	// listedRe reads the version and vendor `java_home -V` lists for each
	// JDK.
	listedRe = regexp.MustCompile(`(?m)^\s+(\d[^\s]*)\s+\([^)]*\)\s+"([^"]*)"`)
	// resolverRe reads a script that registers a toolchain download
	// repository: the Foojay plugin, or a Java repository in
	// toolchainManagement.
	resolverRe = regexp.MustCompile(`foojay|javaRepositories\s*\{[^}]*\brepository\b`)
	// blockCommentRe and lineCommentRe read the comments of a Gradle script.
	blockCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineCommentRe  = regexp.MustCompile(`(?m)(^|\s)//.*$`)
)

// jdkParents are the folders, relative to the user's home, whose children
// are JDKs that Gradle's toolchain detection finds: its own downloads,
// SDKMAN, asdf, jabba, IntelliJ and the macOS user folder.
var jdkParents = []string{
	".gradle/jdks",
	".sdkman/candidates/java",
	".asdf/installs/java",
	".jabba/jdk",
	".jdks",
	"Library/Java/JavaVirtualMachines",
}

// systemParents are the folders, from the filesystem root, whose children
// are JDKs the operating system or Homebrew installs.
var systemParents = []string{
	"/usr/lib/jvm",
	"/usr/java",
	"/Library/Java/JavaVirtualMachines",
}

// brewHomes are the patterns for Homebrew's versioned OpenJDK formulae.
var brewHomes = []string{
	"/opt/homebrew/opt/openjdk*/libexec/openjdk.jdk/Contents/Home",
	"/usr/local/opt/openjdk*/libexec/openjdk.jdk/Contents/Home",
}

// javaHomeTool lists the JDKs macOS registers.
const javaHomeTool = "/usr/libexec/java_home"

// jdk is an installed JDK as its release file describes it.
type jdk struct {
	major  int
	vendor string
	j9     bool
}

// machineJDKs lists the JDKs every build on this machine may use, without
// starting Gradle: JAVA_HOME's, the children of the folders above and
// macOS's list. It reads folders and files only.
func machineJDKs() []jdk {
	out := listedJDKs()
	homes := []string{os.Getenv("JAVA_HOME")}
	if user, err := os.UserHomeDir(); err == nil {
		homes = append(homes, children(user, jdkParents)...)
	}
	homes = append(homes, children("", systemParents)...)
	for _, pattern := range brewHomes {
		matches, _ := filepath.Glob(pattern)
		homes = append(homes, matches...)
	}
	return append(out, jdksIn(homes)...)
}

// children lists the folders directly inside each parent, which is
// relative to base when base is set.
func children(base string, parents []string) []string {
	var out []string
	for _, parent := range parents {
		entries, _ := os.ReadDir(filepath.Join(base, filepath.FromSlash(parent)))
		for _, e := range entries {
			out = append(out, filepath.Join(base, filepath.FromSlash(parent), e.Name()))
		}
	}
	return out
}

// jdksIn lists the JDK each folder holds, skipping empty names and folders
// holding none.
func jdksIn(homes []string) []jdk {
	var out []jdk
	for _, h := range homes {
		if j, ok := homeJDK(h); ok && h != "" {
			out = append(out, j)
		}
	}
	return out
}

// homeJDK reads the JDK a folder holds. A macOS bundle keeps its release
// file under Contents/Home, and Gradle's downloads nest the JDK one folder
// below the one it names.
func homeJDK(dir string) (jdk, bool) {
	if dir == "" {
		return jdk{}, false
	}
	for _, file := range releaseFiles(dir) {
		if j, ok := readJDK(file); ok {
			return j, true
		}
	}
	return jdk{}, false
}

// releaseFiles lists where dir's release file may be.
func releaseFiles(dir string) []string {
	files := []string{filepath.Join(dir, "release"), filepath.Join(dir, "Contents", "Home", "release")}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		files = append(files, filepath.Join(dir, e.Name(), "release"), filepath.Join(dir, e.Name(), "Contents", "Home", "release"))
	}
	return files
}

// readJDK reads a JDK's release and vendor from its release file.
func readJDK(file string) (jdk, bool) {
	props := java.Properties(file)
	major, ok := java.Major(unquote(props["JAVA_VERSION"]))
	if !ok {
		return jdk{}, false
	}
	vendor := unquote(props["IMPLEMENTOR"]) + " " + unquote(props["IMPLEMENTOR_VERSION"])
	return jdk{major: major, vendor: strings.ToLower(vendor), j9: strings.Contains(strings.ToLower(props["JVM_VARIANT"]), "openj9")}, true
}

// unquote drops the quotes a release file puts around a value.
func unquote(value string) string {
	return strings.Trim(value, `"`)
}

// listedJDKs reads the JDKs macOS's java_home lists, or none where the
// tool is missing.
func listedJDKs() []jdk {
	if _, err := os.Stat(javaHomeTool); err != nil {
		return nil
	}
	out, _ := exec.Command(javaHomeTool, "-V").CombinedOutput()
	var jdks []jdk
	for _, m := range listedRe.FindAllStringSubmatch(string(out), -1) {
		if n, ok := java.Major(m[1]); ok {
			jdks = append(jdks, jdk{major: n, vendor: strings.ToLower(m[2])})
		}
	}
	return jdks
}

// propertyJDKs lists the JDKs in the folders org.gradle.java.installations
// .paths names, and the environment variables its fromEnv names. Gradle
// splits both lists on commas.
func propertyJDKs(props map[string]string) []jdk {
	homes := splitList(props["org.gradle.java.installations.paths"])
	for _, name := range splitList(props["org.gradle.java.installations.fromEnv"]) {
		homes = append(homes, os.Getenv(name))
	}
	return jdksIn(homes)
}

// splitList splits a comma list, dropping blanks.
func splitList(list string) []string {
	var out []string
	for _, item := range strings.Split(list, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// canDownload reports whether Gradle may download a missing JDK for the
// build: auto-download is not turned off, and a settings or init script
// registers a toolchain download repository.
func canDownload(root string, build java.Project, props map[string]string) bool {
	if props["org.gradle.java.installations.auto-download"] == "false" {
		return false
	}
	for _, body := range scripts(root, build) {
		body = lineCommentRe.ReplaceAllString(blockCommentRe.ReplaceAllString(body, ""), "$1")
		if resolverRe.MatchString(body) {
			return true
		}
	}
	return false
}

// scripts reads the build's settings scripts and the init scripts of the
// Gradle user home.
func scripts(root string, build java.Project) []string {
	var files []string
	for _, name := range []string{"settings.gradle", "settings.gradle.kts"} {
		files = append(files, filepath.Join(root, filepath.FromSlash(path.Join(build.Dir, name))))
	}
	for _, name := range []string{"init.gradle", "init.gradle.kts"} {
		files = append(files, java.GradleUserFile(name))
	}
	initD, _ := filepath.Glob(java.GradleUserFile("init.d", "*"))
	var out []string
	for _, f := range append(files, initD...) {
		if body, err := os.ReadFile(f); err == nil {
			out = append(out, string(body))
		}
	}
	return out
}
