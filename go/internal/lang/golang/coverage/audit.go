package coverage

import (
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"sync"
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

// audit measures each tested package from its prebuilt binary. Packages run
// on one bounded pool, as go test runs packages, while each package's runs
// stay in sequence. The first failure stops the remaining work.
func (r runner) audit(packages []Package) ([]testGroup, error) {
	tested := testedPackages(packages)
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()
	r.ctx = ctx
	var failure firstError
	results := par.Map(len(tested), par.Processes(), func(i int) []testGroup {
		if err := ctx.Err(); err != nil {
			failure.set(err)
			return nil
		}
		groups, err := r.auditPackage(tested[i])
		if err != nil {
			failure.set(err)
			cancel()
		}
		return groups
	})
	if err := failure.get(); err != nil {
		return nil, err
	}
	return slices.Concat(results...), nil
}

// firstError keeps the first error any pool worker reports.
type firstError struct {
	mu  sync.Mutex
	err error
}

func (f *firstError) set(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err == nil {
		f.err = err
	}
}
func (f *firstError) get() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.err
}

// auditSuite measures shared setup, each top-level test twice, then shared
// setup again, in that order. Changed setup marks every group unstable.
func (r runner) auditSuite(s suite) ([]testGroup, error) {
	baseline, err := r.measureBinary(s, "^$")
	if err != nil {
		return nil, err
	}
	groups, err := r.auditTests(s, baseline)
	if err != nil {
		return nil, err
	}
	return r.verifyBaseline(s, baseline, groups)
}

// verifyBaseline measures shared setup again and marks every group unstable
// when it changed.
func (r runner) verifyBaseline(s suite, baseline Profile, groups []testGroup) ([]testGroup, error) {
	repeated, err := r.measureBinary(s, "^$")
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

// auditTests audits the suite's top-level tests in list order.
func (r runner) auditTests(s suite, baseline Profile) ([]testGroup, error) {
	groups := make([]testGroup, len(s.names))
	for i, name := range s.names {
		group, err := r.auditTest(s, name, baseline)
		if err != nil {
			return nil, err
		}
		groups[i] = group
	}
	return groups, nil
}

// auditTest runs one top-level test twice and compares its contributions.
func (r runner) auditTest(s suite, name string, baseline Profile) (testGroup, error) {
	start := time.Now()
	pattern := "^" + regexp.QuoteMeta(name) + "$"
	first, err := r.measureBinary(s, pattern)
	if err != nil {
		return testGroup{}, err
	}
	second, err := r.measureBinary(s, pattern)
	if err != nil {
		return testGroup{}, err
	}
	a, b := contribution(first, baseline), contribution(second, baseline)
	return testGroup{Name: s.pkg.ImportPath + "/" + name, Covered: a, Setup: contribution(baseline, nil), Stable: maps.Equal(a, b), Duration: time.Since(start)}, nil
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
