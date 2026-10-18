package coverage

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// audit measures each tested package from its prebuilt binary. Up to
// par.Processes() packages run side by side, and all their runs share as
// many process slots. A failure stops only later packages, so the earliest
// failing package is reported.
func (r runner) audit(packages []Package, parallel map[string]map[string]bool) ([]testGroup, error) {
	tested := testedPackages(packages)
	r.slots = make(chan struct{}, par.Processes())
	groups, err := par.MapErr(r.ctx, len(tested), par.Processes(), func(ctx context.Context, i int) ([]testGroup, error) {
		return r.with(ctx).auditPackage(tested[i], parallel[tested[i].ImportPath])
	})
	return slices.Concat(groups...), err
}

func (r runner) with(ctx context.Context) runner {
	r.ctx = ctx
	return r
}

// auditPackage builds pkg's binary into a private directory, audits it, and
// removes the binary on every path.
func (r runner) auditPackage(pkg Package, parallel map[string]bool) ([]testGroup, error) {
	dir, err := os.MkdirTemp(r.temp, "bin-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	s, err := r.suite(pkg, filepath.Join(dir, "pkg.test"), parallel)
	if err != nil {
		return nil, err
	}
	return r.auditSuite(s)
}

// packageAudit is one package's suite, its first setup profile, and a group
// slot per top-level test in list order.
type packageAudit struct {
	suite    suite
	baseline Profile
	groups   []testGroup
}

// auditSuite follows go test's order for one package: shared setup, the
// serial tests in sequence, the t.Parallel() tests side by side, then shared
// setup again.
func (r runner) auditSuite(s suite) ([]testGroup, error) {
	baseline, err := r.measureBinary(s, "^$")
	if err != nil {
		return nil, err
	}
	a := &packageAudit{suite: s, baseline: baseline, groups: make([]testGroup, len(s.names))}
	if err := r.runSerial(a); err != nil {
		return nil, err
	}
	if err := r.runParallel(a); err != nil {
		return nil, err
	}
	return r.finishPackage(a)
}

// runSerial audits the package's serial tests one after another.
func (r runner) runSerial(a *packageAudit) error {
	for i, name := range a.suite.names {
		if a.suite.parallel[name] {
			continue
		}
		if err := r.runTest(a, i); err != nil {
			return err
		}
	}
	return nil
}

// runParallel audits the package's t.Parallel() tests side by side.
func (r runner) runParallel(a *packageAudit) error {
	var tests []int
	for i, name := range a.suite.names {
		if a.suite.parallel[name] {
			tests = append(tests, i)
		}
	}
	_, err := par.MapErr(r.ctx, len(tests), par.Processes(), func(ctx context.Context, k int) (struct{}, error) {
		return struct{}{}, r.with(ctx).runTest(a, tests[k])
	})
	return err
}

// runTest audits test i into its group slot.
func (r runner) runTest(a *packageAudit, i int) error {
	group, err := r.auditTest(a.suite, a.suite.names[i], a.baseline)
	a.groups[i] = group
	return err
}

// finishPackage measures shared setup again and marks every group unstable
// when it changed.
func (r runner) finishPackage(a *packageAudit) ([]testGroup, error) {
	repeated, err := r.measureBinary(a.suite, "^$")
	if err != nil {
		return nil, err
	}
	if !maps.Equal(a.baseline, repeated) {
		for i := range a.groups {
			a.groups[i].Stable = false
		}
	}
	return a.groups, nil
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
