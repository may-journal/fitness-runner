// Package tests runs each Java build root's tests once, then judges the JUnit
// XML and the JaCoCo coverage against the goal.
package tests

// cspell:ignore jacoco

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// aggregationRe marks a Gradle build that applies JaCoCo report
// aggregation, whose report task joins every project's coverage.
var aggregationRe = regexp.MustCompile(`["'\x60]jacoco-report-aggregation["'\x60]`)

// cachingProperty is the Gradle property that turns the build cache on
// or off.
const cachingProperty = "org.gradle.caching"

// Run runs and judges the tests of the repo at root.
func Run(root string, _ []string) (checkkit.Result, error) {
	return plugin(time.Now().Truncate(time.Second)).Check(root)
}

// plugin is JaCoCo as each build tool runs it with the tests, judging only
// reports written since start.
func plugin(start time.Time) java.Plugin {
	return java.Plugin{
		Name: "JaCoCo",
		Uses: map[*java.Build]java.Use{
			java.Maven: {
				Declared: regexp.MustCompile(`jacoco-maven-plugin`),
				Snippet:  `<plugin><groupId>org.jacoco</groupId><artifactId>jacoco-maven-plugin</artifactId><version>0.8.15</version><executions><execution><goals><goal>prepare-agent</goal><goal>report</goal></goals></execution></executions></plugin>`,
				Args:     mavenArgs,
			},
			java.Gradle: {
				Declared: regexp.MustCompile(`(?m)id\s*\(?\s*["']jacoco["']|plugin\s*[:=]\s*["']jacoco["']|^\s*jacoco\s*(?:\{|$)|` + aggregationRe.String()),
				Missing:  "task 'jacocoTestReport' not found",
				Snippet:  `plugins { id("jacoco") } tasks.jacocoTestReport { dependsOn(tasks.test); reports { xml.required = true } }`,
				Args:     gradleArgs,
			},
		},
		Judge: func(root string, p java.Project, o checkkit.Outcome) []string { return judge(root, p, o, start) },
		Ready: ready,
	}
}

// mavenArgs runs the build through verify and past failing tests, so
// every report is written.
func mavenArgs(string, java.Project) []string {
	return []string{"verify", "-D", "maven.test.failure.ignore=true"}
}

// gradleArgs reruns the test and coverage report tasks past failures, cleaning
// them first because --rerun needs Gradle 7.6.
func gradleArgs(root string, p java.Project) []string {
	report := "jacocoTestReport"
	if java.Declares(root, p, aggregationRe) {
		report = "testCodeCoverageReport"
	}
	args := []string{"cleanTest", "test", "clean" + strings.ToUpper(report[:1]) + report[1:], report, "--continue"}
	if cachingAllowed(root, p) {
		args = append(args, "--build-cache")
	}
	return args
}

// cachingAllowed reports whether no source sets org.gradle.caching, or one
// sets it to true.
func cachingAllowed(root string, p java.Project) bool {
	value, set := java.GradleProperties(root, p)[cachingProperty]
	for _, opts := range []string{os.Getenv("JAVA_OPTS"), os.Getenv("GRADLE_OPTS")} {
		for _, field := range strings.Fields(opts) {
			if v, ok := strings.CutPrefix(field, "-D"+cachingProperty+"="); ok {
				value, set = v, true
			}
		}
	}
	return !set || strings.EqualFold(strings.TrimSpace(value), "true")
}

// judge names the failing tests, else the failed build's reason, else the
// missing tests, else the coverage short of the goal.
func judge(root string, p java.Project, o checkkit.Outcome, start time.Time) []string {
	r := readResults(root, p, start)
	if problems := slices.Concat(r.failed, r.unreadable); len(problems) > 0 {
		return problems
	}
	if o.Code != 0 {
		return java.Fallback(o, "the "+p.Build.Name+" build at "+p.Dir+" failed; run its tests by hand")
	}
	if r.tests == 0 {
		return noTests(root, p)
	}
	findings, found := coverage(root, p, start)
	if !found {
		return []string{"no JaCoCo XML report after the " + p.Build.Name + " build at " + p.Dir + " ran its tests; add: " + plugin(start).Uses[p.Build].Snippet}
	}
	return findings
}

// noTests explains a run that ran no test, with the detected frameworks' hints
// for a root that has test files.
func noTests(root string, p java.Project) []string {
	product := productFiles(root, p)
	if len(product) == 0 {
		return nil
	}
	at := "the " + p.Build.Name + " build at " + p.Dir
	if len(testFiles(root, p)) == 0 {
		return []string{at + " has product code such as " + product[0] + " and no tests; add tests under src/test/java"}
	}
	out := []string{at + " has test files but ran no test"}
	for _, f := range detect(root, p) {
		if f.hint != "" {
			out = append(out, f.name+": "+f.hint)
		}
	}
	return out
}
