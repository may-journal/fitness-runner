package version

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
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

// toolchainErrors names each toolchain release Gradle can neither find
// installed nor download. A release the running JDK has needs no run,
// since Gradle always detects the JDK it runs on. Otherwise Gradle plans
// a compile once per build root, which resolves the toolchain as the
// build would: through its installed JDKs, resolvers and init scripts.
func toolchainErrors(root string, have int, pins []pin) ([]string, error) {
	var errors []string
	for _, build := range toolchainBuilds(have, pins) {
		bin, ok := java.Tool(root, build)
		if !ok {
			errors = append(errors, build.Build.Install)
			continue
		}
		o := java.Run(root, bin, build, "compileJava", "--dry-run")
		if err := o.EnvErr(build.Build.Command); err != nil {
			return nil, err
		}
		errors = append(errors, unresolved(o, build, pins)...)
	}
	return errors, nil
}

// unresolved names each of build's toolchain releases a failed plan
// found no JDK for, and how to install it, or else the plan's own reason.
func unresolved(o checkkit.Outcome, build java.Project, pins []pin) []string {
	if o.Code == 0 {
		return nil
	}
	if errors := missingJDKs(o.Stdout, build, pins); len(errors) > 0 {
		return errors
	}
	return java.Fallback(o, "Gradle could not plan the build's compile; run its compileJava task by hand")
}

// missingJDKs names each of build's toolchain releases Gradle's output
// says it found no JDK for.
func missingJDKs(out string, build java.Project, pins []pin) []string {
	var errors []string
	for _, m := range missingRe.FindAllStringSubmatch(out, -1) {
		for _, p := range pins {
			if p.toolchain == build && p.version == m[1] {
				need, _ := strconv.Atoi(m[1])
				errors = append(errors, fmt.Sprintf("%s asks for a JDK %d toolchain, which is not installed; install JDK %d, such as from https://adoptium.net, or configure a toolchain resolver to download it", p.file, need, need))
			}
		}
	}
	return errors
}

// toolchainBuilds lists, once each, the build roots asking for a
// toolchain release the running JDK is not.
func toolchainBuilds(have int, pins []pin) []java.Project {
	var out []java.Project
	for _, p := range pins {
		need, _ := java.Major(p.version)
		if p.toolchain != (java.Project{}) && need != have && !slices.Contains(out, p.toolchain) {
			out = append(out, p.toolchain)
		}
	}
	return out
}
