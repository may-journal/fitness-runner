package version

// cspell:ignore foojay

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

var (
	// toolchainRe reads a Gradle toolchain's language version.
	toolchainRe = regexp.MustCompile(`JavaLanguageVersion\.of\(\s*["']?(\d+)`)
	// resolverRe marks a Gradle build that configures a toolchain download
	// repository, as the foojay resolver plugin does, so Gradle fetches a
	// JDK it lacks.
	resolverRe = regexp.MustCompile(`foojay-resolver|javaRepositories`)
	// detectedRe reads the language version and the JDK flag of each
	// toolchain Gradle's javaToolchains task lists.
	detectedRe = regexp.MustCompile(`(?m)^\s*\|\s*(Language Version|Is JDK):\s*(\S+)`)
)

// gradlePins reads the toolchain releases a Gradle script names. Without a
// resolver in its build, each must match an installed JDK.
func gradlePins(root, file string, roots []java.Project) ([]pin, string) {
	body, ok := read(root, file)
	if !ok {
		return nil, ""
	}
	build := java.Nearest(roots, java.Gradle, file)
	if java.Declares(root, build, resolverRe) {
		build = java.Project{}
	}
	var out []pin
	for _, m := range toolchainRe.FindAllStringSubmatch(body, -1) {
		out = append(out, pin{file: file, version: m[1], toolchain: build})
	}
	return out, ""
}

// toolchainErrors names each toolchain release no JDK that Gradle detects
// from the build root provides, running Gradle once per build root.
func toolchainErrors(root string, pins []pin) ([]string, error) {
	var errors []string
	for _, build := range toolchainBuilds(pins) {
		installed, failed, err := detected(root, build)
		if err != nil {
			return nil, err
		}
		if failed != nil {
			errors = append(errors, failed...)
			continue
		}
		errors = append(errors, uninstalled(pins, build, installed)...)
	}
	return errors, nil
}

// uninstalled names each of build's toolchain releases no installed JDK
// provides, and how to install it.
func uninstalled(pins []pin, build java.Project, installed []int) []string {
	var errors []string
	for _, p := range pins {
		if need, _ := java.Major(p.version); p.toolchain == build && !slices.Contains(installed, need) {
			errors = append(errors, fmt.Sprintf("%s asks for a JDK %d toolchain, which is not installed; install JDK %d, such as from https://adoptium.net, or configure a toolchain resolver to download it", p.file, need, need))
		}
	}
	return errors
}

// toolchainBuilds lists the build roots whose toolchains must match an
// installed JDK, once each.
func toolchainBuilds(pins []pin) []java.Project {
	var out []java.Project
	for _, p := range pins {
		if p.toolchain != (java.Project{}) && !slices.Contains(out, p.toolchain) {
			out = append(out, p.toolchain)
		}
	}
	return out
}

// detected lists the feature releases of the JDKs Gradle detects from
// build, or what stopped it listing them.
func detected(root string, build java.Project) ([]int, []string, error) {
	bin, ok := java.Tool(root, build)
	if !ok {
		return nil, []string{build.Build.Install}, nil
	}
	o := java.Run(root, bin, build, "javaToolchains")
	if err := o.EnvErr(build.Build.Command); err != nil {
		return nil, nil, err
	}
	if failed := java.Fallback(o, "Gradle could not list the installed JDKs; run its javaToolchains task by hand"); failed != nil {
		return nil, failed, nil
	}
	return releasesListed(o.Stdout), nil, nil
}

// releasesListed reads the releases of the JDKs, not runtimes alone, in
// Gradle's toolchain list.
func releasesListed(out string) []int {
	var releases []int
	release := 0
	for _, m := range detectedRe.FindAllStringSubmatch(out, -1) {
		switch {
		case m[1] == "Language Version":
			release, _ = java.Major(m[2])
		case strings.EqualFold(m[2], "true"):
			releases = append(releases, release)
		}
	}
	return releases
}
