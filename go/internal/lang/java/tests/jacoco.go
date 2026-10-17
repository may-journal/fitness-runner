package tests

// cspell:ignore jacoco

import (
	"fmt"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/goal"
	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// report is a JaCoCo XML report's own totals: the counters directly under
// <report>. JaCoCo counts a source line once even when several classes or
// methods share it, so the report's totals are read as written and never
// summed from its packages, classes or methods.
type report struct {
	Counters []counter `xml:"counter"`
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
// the aggregate reports when the build writes any, else each project's own.
// found is false when the run wrote neither.
func coverage(root string, p java.Project, start time.Time) (findings []string, found bool) {
	members := java.Members(root, p)
	files := fresh(root, members, func(b *java.Build) []string { return b.Aggregate }, start)
	if len(files) == 0 {
		files = fresh(root, members, func(b *java.Build) []string { return b.Coverage }, start)
	}
	for _, file := range files {
		findings = append(findings, judgeReport(root, file)...)
	}
	return findings, len(files) > 0
}

// judgeReport reads one report's totals and names those short of the goal,
// or the report itself when it does not parse.
func judgeReport(root, file string) []string {
	rel := java.Relative(root, ".", file)
	var r report
	if !readXML(file, &r) {
		return []string{rel + " is not a JaCoCo XML report; rerun the build"}
	}
	var out []string
	for _, c := range r.Counters {
		if word, ok := judged[c.Type]; ok && !goal.Met(c.Covered, c.Covered+c.Missed) {
			out = append(out, fmt.Sprintf("%s: %s %d/%d covered; the goal is %s", rel, word, c.Covered, c.Covered+c.Missed, goal.Text()))
		}
	}
	return out
}
