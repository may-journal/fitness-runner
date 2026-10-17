// Package tests runs each Java build root's tests once for the java-test
// check, then judges what the run wrote: the JUnit XML of every test, and
// the JaCoCo XML coverage totals against the shared coverage goal.
//
// Most Java test frameworks run on the JUnit Platform or through Surefire,
// so one JUnit XML reader serves them all, and JaCoCo measures whichever
// ran. The frameworks folder holds one file per framework the check spots
// from build files and imports, to explain a build whose tests ran none and
// to ask for what a framework needs before the run, such as Docker.
//
// Only reports written during the run count, so a stale report never
// passes. Gradle reruns the test and report tasks even when it holds them
// up to date. Gradle runs only the test task, so another test task counts
// when the test or report task depends on it. A failing test is reported before coverage, since a failed
// run's coverage says little.
//
// Requirements finds the tests named for requirement acceptances, which
// the requirements check reads.
package tests

// cspell:ignore jacoco

import (
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

// mavenArgs runs the build through verify, so Surefire, Failsafe and the
// JaCoCo reports bound there all run, and lets the run go on past failing
// tests so their reports and the coverage report are written.
func mavenArgs(string, java.Project) []string {
	return []string{"verify", "-D", "maven.test.failure.ignore=true"}
}

// gradleArgs reruns the test task and the coverage report task, the
// aggregate one when the build applies aggregation, and goes on past a
// failing task. Each task's clean rule deletes its outputs first, so the
// task runs again on any Gradle version, where the --rerun option needs
// Gradle 7.6.
func gradleArgs(root string, p java.Project) []string {
	report := "jacocoTestReport"
	if java.Declares(root, p, aggregationRe) {
		report = "testCodeCoverageReport"
	}
	return []string{"cleanTest", "test", "clean" + strings.ToUpper(report[:1]) + report[1:], report, "--continue"}
}

// judge names each failing test and each report that does not parse; else
// the build's own reason for a failed
// run; else a missing test run for a root with product code; else each
// coverage total short of the goal, or the missing coverage report.
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

// noTests explains a run that ran no test: none for a root without product
// code, the detected frameworks' hints for a root with test files, and a
// request for tests otherwise.
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
