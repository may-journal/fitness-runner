// Package checkstyle runs the Checkstyle plugin from each Java build root
// for the checkstyle check, main and test sources alike. The repo's build
// declares the plugin and its Checkstyle config owns the rules, but that
// config must set the cyclomatic complexity ceiling every language here has.
// Every finding fails, warnings included: Maven runs with warnings as
// violations, and Gradle's warning lines are read as findings too. Each
// becomes `file:line[:col] - message (Rule)`.
//
//fitness:container cli
//fitness:component kit Its language's checks share it.
package checkstyle

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
	"github.com/may-journal/fitness-runner/go/internal/walkfs"
)

// ceiling is the house complexity ceiling the hints suggest, the one the
// other languages' complexity checks default to.
const ceiling = `<module name="CyclomaticComplexity"><property name="max" value="5"/></module>`

// defaultConfig is where Gradle looks for the config, and where the hint
// asks a repo without one to put it.
const defaultConfig = "config/checkstyle/checkstyle.xml"

// doctype opens the config the hint asks for. Maven's plugin reads no
// config without it.
const doctype = `<!DOCTYPE module PUBLIC "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN" "https://checkstyle.org/dtds/configuration_1_3.dtd">`

var (
	// checkerRe marks an XML file as a Checkstyle config.
	checkerRe = regexp.MustCompile(`<module\s+name\s*=\s*"(?:[\w.]+\.)?Checker"`)
	// complexityRe marks a config that sets the complexity ceiling.
	complexityRe = regexp.MustCompile(`<module\s+name\s*=\s*"(?:[\w.]+\.)?CyclomaticComplexity(?:Check)?"`)
	// mavenRe reads Maven's violation lines:
	// [WARN] path/X.java:[12,5] (category) Rule: message.
	mavenRe = regexp.MustCompile(`^\[(?:WARN|WARNING|ERROR)\]\s+(.+?\.java):\[(\d+)(?:,(\d+))?\]\s+\([\w-]+\)\s+(\w+):\s+(.*)$`)
	// plainRe reads Checkstyle's plain lines, which Gradle prints:
	// [ant:checkstyle] [WARN] path/X.java:12:5: message [Rule].
	plainRe = regexp.MustCompile(`^(?:\[ant:checkstyle\]\s+)?\[(?:WARN|WARNING|ERROR)\]\s+(.+?\.java):(\d+)(?::(\d+))?:\s+(.*?)\s+\[(\w+)\]$`)
)

// plugin is Checkstyle as each build tool runs it.
var plugin = java.Plugin{
	Name: "Checkstyle",
	Uses: map[*java.Build]java.Use{
		java.Maven: {
			Declared: regexp.MustCompile(`maven-checkstyle-plugin`),
			Missing:  "No plugin found for prefix 'checkstyle'",
			Snippet:  `<plugin><groupId>org.apache.maven.plugins</groupId><artifactId>maven-checkstyle-plugin</artifactId><version>3.6.0</version><configuration><configLocation>` + defaultConfig + `</configLocation></configuration></plugin>`,
			Args:     mavenArgs,
		},
		java.Gradle: {
			Declared: regexp.MustCompile(`(?m)id\s*\(?\s*["']checkstyle["']|plugin\s*[:=]\s*["']checkstyle["']|^\s*checkstyle\s*(?:\{|$)`),
			Missing:  "task 'checkstyleMain' not found",
			Snippet:  `plugins { id("checkstyle") }`,
			Args:     gradleArgs,
		},
	},
	Judge: judge,
}

// Run lints the repo at root and checks its configs set the ceiling.
func Run(root string, _ []string) (checkkit.Result, error) {
	res, err := plugin.Check(root)
	if err != nil || !java.Applies(root) {
		return res, err
	}
	if missing := ceilingErrors(root, configs(root)); len(missing) > 0 {
		return checkkit.Fail(res.FilesChecked, append(missing, res.Errors...)...), nil
	}
	return res, nil
}

// mavenArgs runs the check goal over test sources too, failing on warnings,
// with the repo's config when the build names none of its own.
func mavenArgs(root string, _ java.Project) []string {
	args := []string{"checkstyle:check", "-D", "checkstyle.violationSeverity=warning", "-D", "checkstyle.includeTestSourceDirectory=true"}
	if config, ok := mainConfig(configs(root)); ok {
		if abs, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(config))); err == nil {
			args = append(args, "-D", "checkstyle.config.location="+abs)
		}
	}
	return args
}

// mainConfig picks the config a Maven build without its own uses: the
// repo's only one, else the one at the place Gradle looks.
func mainConfig(found []string) (string, bool) {
	if len(found) == 1 {
		return found[0], true
	}
	return defaultConfig, slices.Contains(found, defaultConfig)
}

// gradleArgs runs both Checkstyle tasks even when Gradle holds them up to
// date, since an up-to-date task prints none of its warnings.
func gradleArgs(string, java.Project) []string {
	return []string{"checkstyleMain", "--rerun", "checkstyleTest", "--rerun"}
}

// configs lists the repo's tracked Checkstyle configs, test fixtures left
// out.
func configs(root string) []string {
	var out []string
	for _, f := range walkfs.FilesByExt(root, ".xml") {
		if body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f))); err == nil && !java.Fixture(f) && checkerRe.Match(body) {
			out = append(out, f)
		}
	}
	return out
}

// ceilingErrors names each config without the complexity ceiling, or asks
// for a config when the repo has none.
func ceilingErrors(root string, found []string) []string {
	if len(found) == 0 {
		return []string{"no Checkstyle config sets a complexity ceiling; add " + defaultConfig + " holding " + doctype + "<module name=\"Checker\"><module name=\"TreeWalker\">" + ceiling + "</module></module>"}
	}
	var out []string
	for _, f := range found {
		if body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f))); err == nil && complexityRe.Match(body) {
			continue
		}
		out = append(out, f+" sets no complexity ceiling; add inside TreeWalker: "+ceiling)
	}
	return out
}

// judge names each violation the run printed, or the build's reason for a
// failure that printed none.
func judge(root string, p java.Project, o checkkit.Outcome) []string {
	var out []string
	for _, line := range strings.Split(o.Stdout, "\n") {
		if finding, ok := violation(root, p, strings.TrimSpace(line)); ok {
			out = append(out, finding)
		}
	}
	if len(out) > 0 {
		return out
	}
	return java.Fallback(o, "Checkstyle reported an error; run the build's Checkstyle task by hand")
}

// violation renders one violation line as file:line[:col] - message (Rule).
func violation(root string, p java.Project, line string) (string, bool) {
	var file, row, col, rule, message string
	if m := mavenRe.FindStringSubmatch(line); m != nil {
		file, row, col, rule, message = m[1], m[2], m[3], m[4], m[5]
	} else if m := plainRe.FindStringSubmatch(line); m != nil {
		file, row, col, message, rule = m[1], m[2], m[3], m[4], m[5]
	} else {
		return "", false
	}
	at := java.Relative(root, p.Dir, file) + ":" + row
	if col != "" {
		at += ":" + col
	}
	return at + " - " + strings.TrimSuffix(message, ".") + " (" + rule + ")", true
}
