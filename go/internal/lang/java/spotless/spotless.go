// Package spotless runs the Spotless plugin's format check from each Java
// build root for the spotless check. The repo's build declares the plugin
// and its formatter, so the check never picks a style. Each file Spotless
// would rewrite becomes one finding; a failure that names no file reports
// the build's own error lines.
//
//fitness:container cli
//fitness:component kit Its language's checks share it.
package spotless

// cspell:ignore diffplug

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// violationsHeader opens Spotless's list of files it would rewrite.
const violationsHeader = "The following files had format violations"

// plugin is Spotless as each build tool runs it.
var plugin = java.Plugin{
	Name: "Spotless",
	Uses: map[*java.Build]java.Use{
		java.Maven: {
			Declared: regexp.MustCompile(`spotless-maven-plugin`),
			Missing:  "No plugin found for prefix 'spotless'",
			Snippet:  `<plugin><groupId>com.diffplug.spotless</groupId><artifactId>spotless-maven-plugin</artifactId><version>3.10.4</version><configuration><java><googleJavaFormat/></java></configuration></plugin>`,
			Args:     func(string, java.Project) []string { return []string{"spotless:check"} },
		},
		java.Gradle: {
			Declared: regexp.MustCompile(`com\.diffplug\.spotless`),
			Missing:  "task 'spotlessCheck' not found",
			Snippet:  `plugins { id("com.diffplug.spotless") version "8.10.4" } spotless { java { googleJavaFormat() } }`,
			Args:     func(string, java.Project) []string { return []string{"spotlessCheck"} },
		},
	},
	Judge: judge,
}

// apply names the task that rewrites the files, by build tool.
var apply = map[*java.Build]string{java.Maven: "spotless:apply", java.Gradle: "spotlessApply"}

// Run checks the format of the repo at root.
func Run(root string, _ []string) (checkkit.Result, error) {
	return plugin.Check(root)
}

// judge names each file the run listed, or the build's reason for a failure
// that listed none.
func judge(root string, p java.Project, o checkkit.Outcome) []string {
	var out []string
	for _, file := range listed(root, p, o.Stdout) {
		out = append(out, file+" is not formatted; run "+apply[p.Build])
	}
	if len(out) > 0 {
		return out
	}
	return java.Fallback(o, "Spotless reported an error; run "+apply[p.Build]+" or the check by hand")
}

// listed reads the files under Spotless's violations header: lines that
// name a file in the repo, past the diff lines Spotless prints beneath
// each.
func listed(root string, p java.Project, output string) []string {
	var out []string
	_, list, found := strings.Cut(output, violationsHeader)
	if !found {
		return nil
	}
	for _, line := range strings.Split(list, "\n") {
		if rel, ok := repoFile(root, p, line); ok {
			out = append(out, rel)
		}
	}
	return out
}

// repoFile reads a line that names nothing but a file in the repo, with
// Maven's [ERROR] prefix stripped, as a path relative to root.
func repoFile(root string, p java.Project, line string) (string, bool) {
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "[ERROR]"))
	if text == "" || strings.ContainsAny(text, " \t") {
		return "", false
	}
	rel := java.Relative(root, p.Dir, text)
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return rel, err == nil && !info.IsDir()
}
