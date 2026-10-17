package tests

// cspell:ignore chardata

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/lang/java"
)

// suite is a JUnit XML <testsuite>, or a <testsuites> that wraps several.
type suite struct {
	Suites []suite    `xml:"testsuite"`
	Cases  []testcase `xml:"testcase"`
}

// testcase is one test a report lists, with its failure or error if any.
type testcase struct {
	Name      string   `xml:"name,attr"`
	Classname string   `xml:"classname,attr"`
	Failure   *problem `xml:"failure"`
	Error     *problem `xml:"error"`
}

// problem is a failed test's message and stack trace.
type problem struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

// results is what the run's JUnit XML reports hold for one build root.
type results struct {
	// tests counts the test cases the reports list.
	tests int
	// failed names each test that failed or erred, with its message.
	failed []string
}

// readResults reads the JUnit XML reports p's projects wrote since start.
// A report that does not parse is left out.
func readResults(root string, p java.Project, start time.Time) results {
	var r results
	for _, file := range fresh(root, java.Members(root, p), func(b *java.Build) []string { return b.TestReports }, start) {
		var s suite
		if readXML(file, &s) {
			r.add(s)
		}
	}
	return r
}

// readXML decodes the XML file into v, reporting whether it could.
func readXML(file string, v any) bool {
	body, err := os.ReadFile(file)
	return err == nil && xml.Unmarshal(body, v) == nil
}

// add counts a suite's cases, nested suites included, and names its
// failures.
func (r *results) add(s suite) {
	for _, inner := range s.Suites {
		r.add(inner)
	}
	for _, c := range s.Cases {
		r.tests++
		if p := firstProblem(c); p != nil {
			r.failed = append(r.failed, c.Classname+"."+c.Name+" failed: "+p.reason())
		}
	}
}

// firstProblem is the case's failure, else its error, else nil.
func firstProblem(c testcase) *problem {
	if c.Failure != nil {
		return c.Failure
	}
	return c.Error
}

// reason is the message, else the first line of the stack trace.
func (p problem) reason() string {
	if text := strings.TrimSpace(p.Message); text != "" {
		return firstLine(text)
	}
	return firstLine(strings.TrimSpace(p.Text))
}

// firstLine is text up to its first line break.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(line)
}

// fresh lists the files the projects' patterns match, from each project's
// folder, that were written since start, by project then pattern.
func fresh(root string, projects []java.Project, patterns func(*java.Build) []string, start time.Time) []string {
	var out []string
	for _, q := range projects {
		for _, pattern := range patterns(q.Build) {
			out = append(out, written(filepath.Join(root, filepath.FromSlash(q.Dir), filepath.FromSlash(pattern)), start)...)
		}
	}
	return out
}

// written lists the files pattern matches that were written since start.
func written(pattern string, start time.Time) []string {
	matches, _ := filepath.Glob(pattern)
	var out []string
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && !info.ModTime().Before(start) {
			out = append(out, m)
		}
	}
	return out
}
