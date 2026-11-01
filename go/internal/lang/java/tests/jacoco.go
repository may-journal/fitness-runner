package tests

// cspell:ignore jacoco

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/goal"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
	"github.com/may-journal/fitness-runner/go/internal/lang/java/views"
)

// report is a JaCoCo XML report, whose own totals are read as written because
// summing its classes would count a shared line twice.
type report struct {
	Counters []counter `xml:"counter"`
	Groups   []struct {
		Name string `xml:"name,attr"`
	} `xml:"group"`
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

// listing is what aggregate reports list: Maven's module groups by artifact id
// and Gradle's source files by package path.
type listing struct {
	groups, sources map[string]bool
}

// add adds what the report lists to l.
func (l listing) add(r report) {
	for _, g := range r.Groups {
		l.groups[g.Name] = true
	}
	for _, p := range r.Packages {
		for _, s := range p.Sources {
			l.sources[path.Join(p.Name, s.Name)] = true
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

// coverage judges the JaCoCo XML reports p's projects wrote since start.
// found is false when the run wrote none.
func coverage(root string, p java.Project, start time.Time) (findings []string, found bool) {
	files := reports(root, p, start)
	for _, file := range files {
		findings = append(findings, judgeReport(root, file)...)
	}
	return findings, len(files) > 0
}

// reports lists the reports to judge: the aggregates and the own report of each
// project they leave uncovered.
func reports(root string, p java.Project, start time.Time) []string {
	members := java.Members(root, p)
	aggregates := fresh(root, members, func(b *java.Build) []string { return b.Aggregate }, start)
	if len(aggregates) == 0 {
		return fresh(root, members, ownReport, start)
	}
	l, ok := listed(aggregates)
	if !ok {
		return aggregates
	}
	own := fresh(root, uncovered(root, p, members, l), ownReport, start)
	return slices.Concat(aggregates, slices.DeleteFunc(own, listsNoClass))
}

// ownReport is the pattern of a project's report of its own tests.
func ownReport(b *java.Build) []string { return b.Coverage }

// listsNoClass reports a report that parses and lists no package.
func listsNoClass(file string) bool {
	var r report
	return readXML(file, &r) && len(r.Groups)+len(r.Packages) == 0
}

// listed reads what the reports list, or false when one does not parse.
func listed(files []string) (listing, bool) {
	l := listing{groups: map[string]bool{}, sources: map[string]bool{}}
	for _, file := range files {
		var r report
		if !readXML(file, &r) {
			return l, false
		}
		l.add(r)
	}
	return l, true
}

// uncovered lists the members of root p no aggregate covers, by module group
// for Maven and listed sources for Gradle.
func uncovered(root string, p java.Project, members []java.Project, l listing) []java.Project {
	covered := map[java.Project]bool{}
	for _, q := range members {
		covered[q] = l.groups[artifactID(root, q)]
	}
	for _, f := range productFiles(root, p) {
		if q := java.Nearest(members, p.Build, f); !covered[q] {
			covered[q] = lists(l.sources, root, f)
		}
	}
	return slices.DeleteFunc(slices.Clone(members), func(q java.Project) bool { return covered[q] })
}

// artifactID is the artifact id a Maven project's pom.xml declares, or ""
// for a Gradle project.
func artifactID(root string, q java.Project) string {
	var pom struct {
		ArtifactID string `xml:"artifactId"`
	}
	if q.Build == java.Maven {
		readXML(filepath.Join(root, filepath.FromSlash(q.Dir), "pom.xml"), &pom)
	}
	return pom.ArtifactID
}

// lists reports whether sources holds the tracked Java source by the package it
// declares and its file name.
func lists(sources map[string]bool, root, file string) bool {
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	return err == nil && sources[path.Join(views.Package(string(body)), path.Base(file))]
}

// judgeReport names the totals of one report short of the goal, or the whole
// report if it is unusable.
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
