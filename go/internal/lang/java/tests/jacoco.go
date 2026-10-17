package tests

// cspell:ignore jacoco

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/goal"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// report is a JaCoCo XML report's own totals: the counters directly under
// <report>. JaCoCo counts a source line once even when several classes or
// methods share it, so the report's totals are read as written and never
// summed from its packages, classes or methods. Its packages, listed
// directly or inside groups, name the source files it covers.
type report struct {
	Counters []counter `xml:"counter"`
	group
}

// group is a part of a report: Maven's aggregate report writes one group
// per module, named by artifact id, while Gradle's aggregate report and
// every module report list packages directly.
type group struct {
	Groups   []group       `xml:"group"`
	Packages []javaPackage `xml:"package"`
}

// javaPackage is one package a report covers, slash-separated, "" for the
// default package, with the name of each source file its classes came from.
type javaPackage struct {
	Name    string `xml:"name,attr"`
	Sources []struct {
		Name string `xml:"name,attr"`
	} `xml:"sourcefile"`
}

// list adds each source file the group covers, as its package path and
// file name, to listed.
func (g group) list(listed map[string]bool) {
	for _, inner := range g.Groups {
		inner.list(listed)
	}
	for _, p := range g.Packages {
		for _, s := range p.Sources {
			listed[path.Join(p.Name, s.Name)] = true
		}
	}
}

// counter is one JaCoCo total, such as LINE or BRANCH.
type counter struct {
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

// judged maps each counter the check holds to the coverage goal to the
// word findings name it by. A report leaves out a counter with nothing to
// count, such as BRANCH for code without branches, which meets the goal.
var judged = map[string]string{"LINE": "lines", "BRANCH": "branches"}

// coverage judges the JaCoCo XML reports p's projects wrote since start:
// every aggregate report, and each project's own report unless an
// aggregate covers that project. An aggregate covers a project when it
// lists any of the project's product sources, since an aggregate leaves
// out the projects it does not depend on, and by default the project that
// writes it. found is false when the run wrote neither.
func coverage(root string, p java.Project, start time.Time) (findings []string, found bool) {
	members := java.Members(root, p)
	aggregates := fresh(root, members, func(b *java.Build) []string { return b.Aggregate }, start)
	files := slices.Concat(aggregates, ownReports(root, members, covered(root, p, members, aggregates), start))
	for _, file := range files {
		findings = append(findings, judgeReport(root, file)...)
	}
	return findings, len(files) > 0
}

// covered lists the projects among members whose product sources build
// root p's aggregate reports list.
func covered(root string, p java.Project, members []java.Project, aggregates []string) map[java.Project]bool {
	listed := map[string]bool{}
	for _, file := range aggregates {
		var r report
		if readXML(file, &r) {
			r.list(listed)
		}
	}
	out := map[java.Project]bool{}
	for _, f := range productFiles(root, p) {
		if lists(listed, f) {
			out[java.Nearest(members, p.Build, f)] = true
		}
	}
	return out
}

// ownReports lists the reports of their own tests that the projects not
// covered wrote since start.
func ownReports(root string, projects []java.Project, covered map[java.Project]bool, start time.Time) []string {
	var out []string
	for _, q := range projects {
		if !covered[q] {
			out = append(out, fresh(root, []java.Project{q}, func(b *java.Build) []string { return b.Coverage }, start)...)
		}
	}
	return out
}

// lists reports whether listed holds the source file, as a report names
// it by its package path and file name, which ends the file's path below
// its source folder.
func lists(listed map[string]bool, file string) bool {
	segments := strings.Split(file, "/")
	for i := range segments {
		if listed[strings.Join(segments[i:], "/")] {
			return true
		}
	}
	return false
}

// judgeReport reads one report's totals and names those short of the goal,
// or the report itself when it does not parse or counts no lines, as for
// classes compiled without debug information.
func judgeReport(root, file string) []string {
	rel := java.Relative(root, ".", file)
	var r report
	if !readXML(file, &r) {
		return []string{rel + " is not a JaCoCo XML report; rerun the build"}
	}
	if !slices.ContainsFunc(r.Counters, func(c counter) bool { return c.Type == "LINE" }) {
		return []string{rel + " counts no lines; compile with debug information, the default, and rerun the build"}
	}
	return shortfalls(rel, r.Counters)
}

// shortfalls names each judged total of the report at rel short of the
// goal.
func shortfalls(rel string, counters []counter) []string {
	var out []string
	for _, c := range counters {
		if word, ok := judged[c.Type]; ok && !goal.Met(c.Covered, c.Covered+c.Missed) {
			out = append(out, fmt.Sprintf("%s: %s %d/%d covered; the goal is %s", rel, word, c.Covered, c.Covered+c.Missed, goal.Text()))
		}
	}
	return out
}
