// Package version checks a Java repo pins the JDK it builds with, and that
// the running JDK satisfies each pin, for the java-version check.
//
// A pin is a version file at the repo root, the ones setup-java reads
// (.java-version, .sdkmanrc and .tool-versions), or a build's own target:
// Maven's maven.compiler.release or compiler release, one ${property} deep,
// or a Gradle toolchain's languageVersion. The running JDK satisfies a pin
// when its feature release is at least the pinned one, since a newer JDK
// builds for an older release. The running JDK is JAVA_HOME's, as the
// build tools pick it, else the one on PATH.
package version

// cspell:ignore sdkmanrc

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// noPin is the failure for a Java repo that pins no JDK.
const noPin = "no JDK version is pinned; add a .java-version file naming the release, such as 21, " +
	"or set <maven.compiler.release>21</maven.compiler.release> in pom.xml, " +
	"or java { toolchain { languageVersion = JavaLanguageVersion.of(21) } } in the Gradle build"

// pin is one place the repo names the JDK release it needs.
type pin struct {
	file, version string
}

var (
	// mavenReleaseRe reads Maven's release, as a property or compiler option.
	mavenReleaseRe = regexp.MustCompile(`<(maven\.compiler\.release|release)>\s*([^<\s]+)\s*</`)
	// propertyRe reads a ${name} reference.
	propertyRe = regexp.MustCompile(`^\$\{([\w.-]+)\}$`)
	// toolchainRe reads a Gradle toolchain's language version.
	toolchainRe = regexp.MustCompile(`JavaLanguageVersion\.of\(\s*["']?(\d+)`)
)

// Run checks the pins of the repo at root against the running JDK.
func Run(root string, _ []string) (checkkit.Result, error) {
	if !java.Applies(root) {
		return checkkit.Pass(0), nil
	}
	pins := pins(root)
	if len(pins) == 0 {
		return checkkit.Fail(0, noPin), nil
	}
	running, ok := java.JDK()
	if !ok {
		return checkkit.Fail(len(pins), java.NoJDK), nil
	}
	if errors := judge(running, pins); len(errors) > 0 {
		return checkkit.Fail(len(pins), errors...), nil
	}
	return checkkit.Pass(len(pins)), nil
}

// judge names each pin the running JDK does not satisfy.
func judge(running string, pins []pin) []string {
	have, haveOk := java.Major(running)
	var errors []string
	for _, p := range pins {
		need, ok := java.Major(p.version)
		switch {
		case !ok:
			errors = append(errors, fmt.Sprintf("%s pins JDK %q, which names no release", p.file, p.version))
		case !haveOk || have < need:
			errors = append(errors, fmt.Sprintf("JDK %s does not satisfy %s (requires %d); install JDK %d or point JAVA_HOME at one", running, p.file, need, need))
		}
	}
	return errors
}

// pins lists the repo's version files, then each build's own pins.
func pins(root string) []pin {
	out := versionFiles(root)
	for _, p := range java.Projects(root) {
		for _, name := range p.Build.Files {
			out = appendNew(out, filePins(root, path.Join(p.Dir, name))...)
		}
	}
	return out
}

// filePins reads the pins one build file names, or none when it is
// missing.
func filePins(root, file string) []pin {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return nil
	}
	var out []pin
	for _, v := range buildPins(string(body)) {
		out = append(out, pin{file, v})
	}
	return out
}

// appendNew appends each pin out does not hold yet.
func appendNew(out []pin, pins ...pin) []pin {
	for _, p := range pins {
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// versionFiles reads the root version files setup-java reads.
func versionFiles(root string) []pin {
	var out []pin
	readers := []struct {
		name string
		read func(fields []string, line string) string
	}{
		{".java-version", func(_ []string, line string) string { return line }},
		{".sdkmanrc", func(_ []string, line string) string {
			if v, ok := strings.CutPrefix(line, "java="); ok {
				return strings.TrimSpace(v)
			}
			return ""
		}},
		{".tool-versions", func(fields []string, _ string) string {
			if len(fields) > 1 && fields[0] == "java" {
				return fields[1]
			}
			return ""
		}},
	}
	for _, r := range readers {
		if v := readVersionFile(filepath.Join(root, r.name), r.read); v != "" {
			out = append(out, pin{r.name, v})
		}
	}
	return out
}

// readVersionFile returns the first version read gives for a line of the
// file that is not blank or a comment, or empty for none.
func readVersionFile(file string, read func(fields []string, line string) string) string {
	body, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, line := range contentLines(string(body)) {
		if v := read(strings.Fields(line), line); v != "" {
			return v
		}
	}
	return ""
}

// contentLines lists body's trimmed lines that are not blank or comments.
func contentLines(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

// buildPins reads the releases a build file names, resolving a Maven
// ${property} from the same file.
func buildPins(body string) []string {
	var out []string
	for _, m := range mavenReleaseRe.FindAllStringSubmatch(body, -1) {
		out = append(out, resolve(body, m[2]))
	}
	for _, m := range toolchainRe.FindAllStringSubmatch(body, -1) {
		out = append(out, m[1])
	}
	return out
}

// resolve replaces a ${name} value with the file's <name> property.
func resolve(body, value string) string {
	m := propertyRe.FindStringSubmatch(value)
	if m == nil {
		return value
	}
	prop := regexp.MustCompile(`<` + regexp.QuoteMeta(m[1]) + `>\s*([^<\s]+)\s*</`).FindStringSubmatch(body)
	if prop == nil {
		return value
	}
	return prop[1]
}
