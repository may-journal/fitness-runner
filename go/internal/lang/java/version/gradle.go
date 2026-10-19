package version

// cspell:ignore jdks

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

var (
	// toolchainRe reads a Gradle toolchain's language version.
	toolchainRe = regexp.MustCompile(`JavaLanguageVersion\.of\(\s*["']?(\d+)`)
	// blockRe reads the body of a toolchain { } block.
	blockRe = regexp.MustCompile(`toolchain\s*\{([^{}]*)\}`)
	// vendorRe reads a toolchain's vendor: a JvmVendorSpec constant, or
	// matching("text").
	vendorRe = regexp.MustCompile(`JvmVendorSpec\.(?:matching\(\s*["']([^"']+)["']|(\w+))`)
	// implementationRe reads a toolchain's JVM implementation.
	implementationRe = regexp.MustCompile(`JvmImplementation\.(\w+)`)
)

// vendorNames maps a JvmVendorSpec constant to the text Gradle looks for
// in a JDK's vendor, where it is not the constant in lower case without
// underscores.
var vendorNames = map[string]string{"HEWLETT_PACKARD": "hewlett"}

// gradlePins reads the toolchain releases a Gradle script names, each to
// be matched against the JDKs installed for the build root.
func gradlePins(root, file string, roots []java.Project) ([]pin, string) {
	body, ok := read(root, file)
	if !ok {
		return nil, ""
	}
	build := java.Nearest(roots, java.Gradle, file)
	var out []pin
	for _, m := range toolchainRe.FindAllStringSubmatchIndex(body, -1) {
		p := pin{file: file, version: body[m[2]:m[3]], toolchain: build}
		p.vendor, p.j9 = constraints(enclosing(body, m[0]))
		out = append(out, p)
	}
	return out, ""
}

// enclosing returns the toolchain { } block holding body[at], or the text
// of the line when it sits in none.
func enclosing(body string, at int) string {
	for _, b := range blockRe.FindAllStringSubmatchIndex(body, -1) {
		if b[0] <= at && at < b[1] {
			return body[b[2]:b[3]]
		}
	}
	start := strings.LastIndex(body[:at], "\n") + 1
	end := strings.Index(body[at:], "\n")
	if end < 0 {
		return body[start:]
	}
	return body[start : at+end]
}

// constraints reads the vendor text and J9 requirement a toolchain block
// sets, or none for a toolchain that takes any vendor.
func constraints(block string) (vendor string, j9 bool) {
	if m := vendorRe.FindStringSubmatch(block); m != nil {
		vendor = strings.ToLower(m[1])
		if m[2] != "" {
			vendor = vendorText(m[2])
		}
	}
	m := implementationRe.FindStringSubmatch(block)
	return vendor, m != nil && m[1] == "J9"
}

// vendorText is the text Gradle looks for in a JDK's vendor for a
// JvmVendorSpec constant.
func vendorText(constant string) string {
	if text, ok := vendorNames[constant]; ok {
		return text
	}
	return strings.ToLower(strings.ReplaceAll(constant, "_", ""))
}

// toolchainErrors names each toolchain no installed JDK matches, when the
// build configures no way to download one. A release the running JDK has
// needs no lookup unless the toolchain names a vendor or implementation,
// since Gradle always detects the JDK it runs on. The check reads folders
// and files and never starts Gradle.
func toolchainErrors(root string, have int, pins []pin) []string {
	var errors []string
	look := lookups{root: root, machine: sync.OnceValue(machineJDKs), builds: map[java.Project]buildJDKs{}}
	for _, p := range pins {
		need, _ := java.Major(p.version)
		if wantsInstalled(p, need, have) && !look.provide(p, need) {
			errors = append(errors, fmt.Sprintf("%s asks for a JDK %d toolchain, which is not installed; install JDK %d, such as from https://adoptium.net, or configure a toolchain resolver to download it", p.file, need, need))
		}
	}
	return errors
}

// wantsInstalled reports whether p is a toolchain that needs a JDK found
// on the machine: another release than the running JDK's, or the running
// release from a named vendor or implementation.
func wantsInstalled(p pin, need, have int) bool {
	if p.toolchain == (java.Project{}) {
		return false
	}
	return need != have || p.vendor != "" || p.j9
}

// buildJDKs is what one build root may use: the JDKs it can find, and
// whether Gradle may download others.
type buildJDKs struct {
	jdks     []jdk
	download bool
}

// lookups holds the JDKs found so far in one run. The machine-wide list
// is read once, on first use, and each build root's own once.
type lookups struct {
	root    string
	machine func() []jdk
	builds  map[java.Project]buildJDKs
}

// provide reports whether the pin's build can get a JDK matching it.
func (l lookups) provide(p pin, need int) bool {
	b, ok := l.builds[p.toolchain]
	if !ok {
		props := java.GradleProperties(l.root, p.toolchain)
		b = buildJDKs{
			jdks:     append(slices.Clone(l.machine()), propertyJDKs(props)...),
			download: canDownload(l.root, p.toolchain, props),
		}
		l.builds[p.toolchain] = b
	}
	return b.download || slices.ContainsFunc(b.jdks, func(j jdk) bool { return j.matches(p, need) })
}

// matches reports whether j is the release, vendor and implementation the
// pin's toolchain asks for.
func (j jdk) matches(p pin, need int) bool {
	return j.major == need && strings.Contains(j.vendor, p.vendor) && (j.j9 || !p.j9)
}
