package version

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

var (
	// toolchainRe reads a Gradle toolchain's language version.
	toolchainRe = regexp.MustCompile(`JavaLanguageVersion\.of\(\s*["']?(\d+)`)
	// missingRe reads the release Gradle found no JDK for, nor could
	// download.
	missingRe = regexp.MustCompile(`Cannot find a Java installation .*?languageVersion=(\d+)`)
)

// gradlePins reads the toolchain releases a Gradle script names, each to
// be judged by the build root's own toolchain resolution.
func gradlePins(root, file string, roots []java.Project) ([]pin, string) {
	body, ok := read(root, file)
	if !ok {
		return nil, ""
	}
	build := java.Nearest(roots, java.Gradle, file)
	var out []pin
	for _, m := range toolchainRe.FindAllStringSubmatch(body, -1) {
		out = append(out, pin{file: file, version: m[1], toolchain: build})
	}
	return out, ""
}

// toolchainErrors names each toolchain release no installed JDK matches,
// when the build configures no resolver to download it. A release the
// running JDK has needs no lookup, since Gradle always detects the JDK it
// runs on. The check reads folders and files and never starts Gradle.
func toolchainErrors(root string, have int, pins []pin) []string {
	var errors []string
	installed := map[java.Project][]int{}
	for _, p := range pins {
		need, _ := java.Major(p.version)
		if !wantsInstalled(p, need, have) {
			continue
		}
		if _, ok := installed[p.toolchain]; !ok {
			installed[p.toolchain] = installedJDKs(root, p.toolchain)
		}
		if missing(root, p, need, installed[p.toolchain]) {
			errors = append(errors, fmt.Sprintf("%s asks for a JDK %d toolchain, which is not installed; install JDK %d, such as from https://adoptium.net, or configure a toolchain resolver to download it", p.file, need, need))
		}
	}
	return errors
}

// wantsInstalled reports whether p is a toolchain for a release other than
// the running JDK's, which must be installed.
func wantsInstalled(p pin, need, have int) bool {
	return p.toolchain != (java.Project{}) && need != have
}

// missing reports whether no installed release is need and the pin's
// build has no resolver to download it.
func missing(root string, p pin, need int, installed []int) bool {
	return !slices.Contains(installed, need) && !hasResolver(root, p.toolchain)
}
