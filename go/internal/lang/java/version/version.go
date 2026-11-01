// Package version checks that a Java repo pins its JDK and that the machine's
// JDKs satisfy each pin.
package version

// cspell:ignore sdkmanrc

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/comments"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// noPin is the failure for a Java repo that pins no JDK.
const noPin = "no JDK version is pinned; add a .java-version file naming the release, such as 21, " +
	"or set <maven.compiler.release>21</maven.compiler.release> in pom.xml, " +
	"or java { toolchain { languageVersion = JavaLanguageVersion.of(21) } } in the Gradle build"

// pin is one place the repo names the JDK release it needs, with the Gradle
// toolchain root that must supply exactly it.
type pin struct {
	file, version string
	// vendor is the lower-case text a toolchain looks for in the JDK's vendor,
	// and j9 whether it requires the OpenJ9 implementation.
	vendor    string
	j9        bool
	toolchain java.Project
}

// readers read the pins a build file names, by the build's tool, and what
// stopped them reading one the build inherits.
var readers = map[*java.Build]func(root, file string, roots []java.Project) ([]pin, string){
	java.Maven:  mavenPins,
	java.Gradle: gradlePins,
}

var (
	// mavenReleaseRe reads Maven's release, as a property or compiler option.
	mavenReleaseRe = regexp.MustCompile(`<(maven\.compiler\.release|release)>\s*([^<\s]+)\s*</`)
	// propertyRe reads a ${name} reference.
	propertyRe = regexp.MustCompile(`^\$\{([\w.-]+)\}$`)
)

// Run checks the pins of the repo at root against the machine's JDKs.
func Run(root string, _ []string) (checkkit.Result, error) {
	if !java.Applies(root) {
		return checkkit.Pass(0), nil
	}
	pins, unread := pins(root)
	if len(pins) == 0 {
		return checkkit.Fail(0, append(unread, noPin)...), nil
	}
	errors := pinErrors(root, pins)
	if len(errors) > 0 {
		return checkkit.Fail(len(pins), errors...), nil
	}
	return checkkit.Pass(len(pins)), nil
}

// pinErrors names each pin the machine's JDKs do not satisfy: a toolchain
// that must match an installed JDK, else the running JDK. With no running
// JDK, it asks for one.
func pinErrors(root string, pins []pin) []string {
	running, ok := java.JDK()
	if !ok {
		return []string{java.NoJDK}
	}
	var newer []pin
	for _, p := range pins {
		if p.toolchain == (java.Project{}) {
			newer = append(newer, p)
		}
	}
	have, _ := java.Major(running)
	errors := toolchainErrors(root, have, pins)
	return append(judge(running, newer), errors...)
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

// pins lists the repo's version files, then each build's own pins, and
// what kept a build's inherited release unread.
func pins(root string) ([]pin, []string) {
	out := versionFiles(root)
	roots := java.Roots(root)
	var unread []string
	for _, p := range java.Projects(root) {
		for _, name := range p.Build.Files {
			found, why := readers[p.Build](root, path.Join(p.Dir, name), roots)
			out = appendNew(out, found...)
			if why != "" {
				unread = append(unread, why)
			}
		}
	}
	return out, unread
}

// read returns a tracked file's code view, or false when it is missing, so
// a commented-out pin never counts.
func read(root, file string) (string, bool) {
	body, err := comments.CodeFile(root, file)
	return string(body), err == nil
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
		if v := readVersionFile(root, r.name, r.read); v != "" {
			out = append(out, pin{file: r.name, version: v})
		}
	}
	return out
}

// readVersionFile returns the first version read gives for a line of the
// file's code view, or empty for none.
func readVersionFile(root, file string, read func(fields []string, line string) string) string {
	body, err := comments.CodeFile(root, file)
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

// contentLines lists body's trimmed lines that are not blank.
func contentLines(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// releases reads the releases body names, resolving each ${property}
// from the first of poms that sets it.
func releases(body string, poms []string) []string {
	var out []string
	for _, m := range mavenReleaseRe.FindAllStringSubmatch(body, -1) {
		out = append(out, resolve(m[2], poms))
	}
	return out
}

// resolve replaces a ${name} value with the <name> property of the first
// of poms that sets it.
func resolve(value string, poms []string) string {
	m := propertyRe.FindStringSubmatch(value)
	if m == nil {
		return value
	}
	prop := regexp.MustCompile(`<` + regexp.QuoteMeta(m[1]) + `>\s*([^<\s]+)\s*</`)
	for _, body := range poms {
		if found := prop.FindStringSubmatch(body); found != nil {
			return found[1]
		}
	}
	return value
}
