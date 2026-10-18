package coverage

import (
	"fmt"
	"maps"
	"regexp"
	"sort"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/par"
)

type testGroup struct {
	Name     string
	Covered  map[Block]bool
	Setup    map[Block]bool
	Duration time.Duration
	Stable   bool
}

// audit measures shared setup and each top-level test twice from prebuilt
// binaries. Every run is independent, so all of them share one bounded pool;
// results are read back in package and test order.
func (r runner) audit(packages []Package) ([]testGroup, error) {
	suites, err := r.suites(packages)
	if err != nil {
		return nil, err
	}
	var jobs []measurement
	for i, s := range suites {
		jobs = append(jobs, s.measurements(i)...)
	}
	runs := par.Map(len(jobs), par.Processes(), func(i int) measured {
		return r.measureBinary(suites[jobs[i].suite], jobs[i].pattern)
	})
	for _, run := range runs {
		if run.err != nil {
			return nil, run.err
		}
	}
	return assemble(suites, runs), nil
}

// assemble splits runs, in measurements order, back into each suite's groups.
func assemble(suites []suite, runs []measured) []testGroup {
	var groups []testGroup
	for _, s := range suites {
		n := 2 + 2*len(s.names)
		groups = append(groups, s.groups(runs[:n])...)
		runs = runs[n:]
	}
	return groups
}

// measurement is one run of a suite's binary for one -test.run pattern.
type measurement struct {
	suite   int
	pattern string
}

// measurements lists a suite's runs: shared setup, each test twice, then
// shared setup again.
func (s suite) measurements(index int) []measurement {
	jobs := []measurement{{index, "^$"}}
	for _, name := range s.names {
		pattern := "^" + regexp.QuoteMeta(name) + "$"
		jobs = append(jobs, measurement{index, pattern}, measurement{index, pattern})
	}
	return append(jobs, measurement{index, "^$"})
}

// groups turns a suite's runs, in measurements order, into test groups. A
// test is stable when both its runs and both setup runs agree.
func (s suite) groups(runs []measured) []testGroup {
	baseline := runs[0].profile
	setupStable := maps.Equal(baseline, runs[len(runs)-1].profile)
	groups := make([]testGroup, len(s.names))
	for i, name := range s.names {
		first, second := runs[1+2*i], runs[2+2*i]
		a, b := contribution(first.profile, baseline), contribution(second.profile, baseline)
		groups[i] = testGroup{Name: s.pkg.ImportPath + "/" + name, Covered: a, Setup: contribution(baseline, nil), Stable: setupStable && maps.Equal(a, b), Duration: first.duration + second.duration}
	}
	return groups
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
