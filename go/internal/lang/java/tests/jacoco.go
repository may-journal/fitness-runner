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

// coverage judges the JaCoCo XML reports p's projects wrote since start.
// found is false when the run wrote none.
func coverage(root string, p java.Project, start time.Time) (findings []string, found bool) {
	files := reports(root, p, start)
	for _, file := range files {
		findings = append(findings, judgeReport(root, file)...)
	}
	return findings, len(files) > 0
}

// reports lists the reports to judge: each project's own when the build
// wrote no aggregate report, else every aggregate and the own report of
// each project that no aggregate covers. An aggregate leaves out the
// projects it does not depend on, and by default the project that writes
// it. An aggregate that does not parse is named alone, since what it
// covers is unknown.
func reports(root string, p java.Project, start time.Time) []string {
	members := java.Members(root, p)
	aggregates := fresh(root, members, func(b *java.Build) []string { return b.Aggregate }, start)
	if len(aggregates) == 0 {
		return fresh(root, members, ownReport, start)
	}
	listed, ok := listedSources(aggregates)
	if !ok {
		return aggregates
	}
	return slices.Concat(aggregates, fresh(root, uncovered(root, p, members, listed), ownReport, start))
}

// ownReport is the pattern of a project's report of its own tests.
func ownReport(b *java.Build) []string { return b.Coverage }

// listedSources reads the source files the reports list, or false when
// one does not parse.
func listedSources(files []string) (map[string]bool, bool) {
	listed := map[string]bool{}
	for _, file := range files {
		var r report
		if !readXML(file, &r) {
			return nil, false
		}
		r.list(listed)
	}
	return listed, true
}

// uncovered lists the members of build root p with product sources none
// of which listed holds. An aggregate takes a project's compiled classes
// whole, so one listed source shows it covers the project, and an unlisted
// one is a class the build's own filters leave out. A project without
// product sources, such as one of integration tests only, has no code of
// its own for a report to cover.
func uncovered(root string, p java.Project, members []java.Project, listed map[string]bool) []java.Project {
	sources := map[java.Project]bool{}
	covered := map[java.Project]bool{}
	for _, f := range productFiles(root, p) {
		q := java.Nearest(members, p.Build, f)
		sources[q] = true
		covered[q] = covered[q] || listed[sourceKey(root, f)]
	}
	return slices.DeleteFunc(slices.Clone(members), func(q java.Project) bool { return !sources[q] || covered[q] })
}

// sourceKey names a tracked Java source as a JaCoCo report does: by the
// package it declares and its file name.
func sourceKey(root, file string) string {
	body, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	return path.Join(views.Package(string(body)), path.Base(file))
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
