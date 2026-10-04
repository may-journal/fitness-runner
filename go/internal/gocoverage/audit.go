package gocoverage

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"
)

type testGroup struct {
	Name     string
	Covered  map[Block]bool
	Setup    map[Block]bool
	Duration time.Duration
	Stable   bool
}

func (r runner) audit(packages []Package) ([]testGroup, error) {
	var groups []testGroup
	for _, pkg := range packages {
		if len(pkg.TestGoFiles)+len(pkg.XTestGoFiles) == 0 {
			continue
		}
		tests, err := r.auditPackage(pkg.ImportPath)
		if err != nil {
			return nil, err
		}
		groups = append(groups, tests...)
	}
	return groups, nil
}
func (r runner) auditPackage(pkg string) ([]testGroup, error) {
	out, err := r.command("test", "-list=^Test", pkg)
	if err != nil {
		return nil, err
	}
	baseline, err := r.measure(pkg, "^$")
	if err != nil {
		return nil, err
	}
	groups, err := r.auditNames(pkg, string(out), baseline)
	if err != nil {
		return nil, err
	}
	return r.verifyBaseline(pkg, baseline, groups)
}
func (r runner) auditNames(pkg, out string, baseline Profile) ([]testGroup, error) {
	var groups []testGroup
	for _, name := range strings.Split(out, "\n") {
		if !testName.MatchString(name) {
			continue
		}
		group, err := r.auditTest(pkg, name, baseline)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, nil
}

var testName = regexp.MustCompile(`^Test[\pL\pN_]*$`)

func (r runner) auditTest(pkg, name string, baseline Profile) (testGroup, error) {
	start := time.Now()
	first, err := r.measure(pkg, "^"+regexp.QuoteMeta(name)+"$")
	if err != nil {
		return testGroup{}, err
	}
	second, err := r.measure(pkg, "^"+regexp.QuoteMeta(name)+"$")
	if err != nil {
		return testGroup{}, err
	}
	a, b := contribution(first, baseline), contribution(second, baseline)
	return testGroup{Name: pkg + "/" + name, Covered: a, Setup: contribution(baseline, nil), Stable: maps.Equal(a, b), Duration: time.Since(start)}, nil
}
func contribution(profile, baseline Profile) map[Block]bool {
	result := map[Block]bool{}
	for b, hit := range profile {
		if hit && !baseline[b] && b.Statements > 0 {
			result[b] = true
		}
	}
	return result
}
func overlap(groups []testGroup) []string {
	owners := map[Block]int{}
	for _, group := range groups {
		if !group.Stable {
			continue
		}
		for b := range group.Covered {
			owners[b]++
		}
	}
	var findings []string
	for i, group := range groups {
		findings = append(findings, groupFindings(group, groups[:i], owners)...)
	}
	sort.Strings(findings)
	return findings
}
func groupFindings(group testGroup, previous []testGroup, owners map[Block]int) []string {
	timing := fmt.Sprintf("%s (%s, two isolated runs including setup)", group.Name, group.Duration.Round(time.Millisecond))
	if !group.Stable {
		return []string{timing + ": unstable coverage; excluded from overlap advice"}
	}
	findings := []string{fmt.Sprintf("%s: %d covered blocks, %d unique; subtests grouped with parent", timing, len(group.Covered), uniqueBlocks(group, owners))}
	if len(group.Covered) == 0 {
		return append(findings, group.Name+": no measured production coverage; check instrumentation and assertions")
	}
	if uniqueBlocks(group, owners) == 0 {
		findings = append(findings, group.Name+": no unique coverage; review assertions before removing tests")
	}
	return append(findings, identicalGroups(group, previous)...)
}
func uniqueBlocks(group testGroup, owners map[Block]int) int {
	count := 0
	for b := range group.Covered {
		if owners[b] == 1 {
			count++
		}
	}
	return count
}
func identicalGroups(group testGroup, previous []testGroup) []string {
	var findings []string
	for _, other := range previous {
		if other.Stable && maps.Equal(group.Covered, other.Covered) {
			findings = append(findings, group.Name+": identical coverage to "+other.Name+"; assertions may differ")
		}
	}
	return findings
}

func (r runner) verifyBaseline(pkg string, baseline Profile, groups []testGroup) ([]testGroup, error) {
	repeated, err := r.measure(pkg, "^$")
	if err != nil {
		return nil, err
	}
	if !maps.Equal(baseline, repeated) {
		for i := range groups {
			groups[i].Stable = false
		}
	}
	return groups, nil
}
