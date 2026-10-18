package coverage

import (
	"context"
	"fmt"
	"maps"
	"os"
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

// audit measures each tested package from its prebuilt binary in three
// stages on one bounded pool, as go test orders a package: shared setup and
// the serial tests in sequence, then the tests that call t.Parallel()
// alongside each other, then shared setup again. Packages run side by side.
func (r runner) audit(packages []Package) ([]testGroup, error) {
	tested := testedPackages(packages)
	audits, err := par.MapErr(r.ctx, len(tested), par.Processes(), func(ctx context.Context, i int) (*packageAudit, error) {
		return r.with(ctx).startPackage(tested[i])
	})
	if err != nil {
		return nil, err
	}
	jobs := parallelJobs(audits)
	if _, err := par.MapErr(r.ctx, len(jobs), par.Processes(), func(ctx context.Context, i int) (struct{}, error) {
		return struct{}{}, r.with(ctx).runJob(jobs[i])
	}); err != nil {
		return nil, err
	}
	finished, err := par.MapErr(r.ctx, len(audits), par.Processes(), func(ctx context.Context, i int) ([]testGroup, error) {
		return r.with(ctx).finishPackage(audits[i])
	})
	return slices.Concat(finished...), err
}

func (r runner) with(ctx context.Context) runner {
	r.ctx = ctx
	return r
}

// packageAudit is one package's suite, its first setup profile, and a group
// slot per top-level test in list order.
type packageAudit struct {
	suite    suite
	baseline Profile
	groups   []testGroup
}

// startPackage builds the suite, measures shared setup, then audits the
// serial tests in list order.
func (r runner) startPackage(pkg Package) (*packageAudit, error) {
	s, err := r.suite(pkg)
	if err != nil {
		return nil, err
	}
	baseline, err := r.measureBinary(s, "^$")
	if err != nil {
		return nil, err
	}
	a := &packageAudit{suite: s, baseline: baseline, groups: make([]testGroup, len(s.names))}
	return a, r.runSerial(a)
}

// runSerial audits the package's tests that do not call t.Parallel(), one
// after another in list order.
func (r runner) runSerial(a *packageAudit) error {
	for i, name := range a.suite.names {
		if a.suite.parallel[name] {
			continue
		}
		if err := r.runJob(job{a, i}); err != nil {
			return err
		}
	}
	return nil
}

// job is one top-level test of a package audit, by list index.
type job struct {
	audit *packageAudit
	test  int
}

// parallelJobs lists every package's t.Parallel() tests, in package and list
// order.
func parallelJobs(audits []*packageAudit) []job {
	var jobs []job
	for _, a := range audits {
		for i, name := range a.suite.names {
			if a.suite.parallel[name] {
				jobs = append(jobs, job{a, i})
			}
		}
	}
	return jobs
}

// runJob audits one test into its group slot.
func (r runner) runJob(j job) error {
	group, err := r.auditTest(j.audit.suite, j.audit.suite.names[j.test], j.audit.baseline)
	j.audit.groups[j.test] = group
	return err
}

// finishPackage measures shared setup again, marks every group unstable when
// it changed, and removes the package's binary.
func (r runner) finishPackage(a *packageAudit) ([]testGroup, error) {
	defer os.RemoveAll(a.suite.dir)
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
