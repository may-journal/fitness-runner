package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/render"
)

// fastTotalEnv and testsTotalEnv name a lower total in milliseconds, which
// requirement tests set to see a total stop a check without waiting minutes.
// Neither raises its total.
const (
	fastTotalEnv  = "FITNESS_INTERNAL_FAST_TOTAL_MS"
	testsTotalEnv = "FITNESS_INTERNAL_TESTS_TOTAL_MS"
)

// fastTotal is checkkit.FastTotal, or the lower total fastTotalEnv names.
func fastTotal() time.Duration {
	return lowered(checkkit.FastTotal, fastTotalEnv)
}

// lowered is total, or the lower total in milliseconds the variable env
// names.
func lowered(total time.Duration, env string) time.Duration {
	ms, err := strconv.ParseInt(os.Getenv(env), 10, 64)
	if err != nil || ms <= 0 {
		return total
	}
	return min(total, time.Duration(ms)*time.Millisecond)
}

// stageOf is the stage c declares, fast when it declares none.
func stageOf(c resolved) string {
	return checkkit.StageOf(c.desc)
}

// stageTotal is the total that bounds a stage's checks together.
func stageTotal(stage string) time.Duration {
	if stage == checkkit.StageTests {
		return lowered(checkkit.TestsTotal, testsTotalEnv)
	}
	return fastTotal()
}

// stageEnds holds when each stage's total runs out in one run.
type stageEnds map[string]time.Time

// startStages starts every stage's total at now, since both stages start
// together.
func startStages(now time.Time) stageEnds {
	ends := stageEnds{}
	for _, stage := range stages {
		ends[stage] = now.Add(stageTotal(stage))
	}
	return ends
}

// runTotals holds what earlier pools spent of each stage's total, counting
// only while that stage's own checks ran.
type runTotals struct {
	spent map[string]time.Duration
}

// start returns when each stage's total runs out for a pool starting at
// now, after what earlier pools spent.
func (t *runTotals) start(now time.Time) stageEnds {
	ends := stageEnds{}
	for _, stage := range stages {
		ends[stage] = now.Add(stageTotal(stage) - t.spent[stage])
	}
	return ends
}

// spend adds, for each stage, the time from start until that stage's last
// check in the pool ended.
func (t *runTotals) spend(start time.Time, checks []resolved, outcomes []outcome) {
	if t.spent == nil {
		t.spent = map[string]time.Duration{}
	}
	last := map[string]time.Time{}
	for i, c := range checks {
		if stage := stageOf(c); outcomes[i].ended.After(last[stage]) {
			last[stage] = outcomes[i].ended
		}
	}
	for stage, end := range last {
		t.spent[stage] += max(end.Sub(start), 0)
	}
}

// stages lists each stage, the test runners first, as a run reports them.
var stages = []string{checkkit.StageTests, checkkit.StageFast}

// stageNames is how a breach line names each stage's checks.
var stageNames = map[string]string{checkkit.StageFast: "fast checks", checkkit.StageTests: "test runners"}

// limit is how long a check in stage may still run: the time left in the
// stage's total.
func (e stageEnds) limit(stage string) time.Duration {
	return time.Until(e[stage])
}

// deadlineEnv holds the Unix time in milliseconds by which a run must end.
// A Claude hook sets it so the run reports inside the hook's timeout.
const deadlineEnv = "FITNESS_INTERNAL_DEADLINE"

// cutNote is the info line of a check the run's deadline stopped or never
// let start.
const cutNote = "skipped: the run's deadline passed"

// deadlineBudget caps limit at the time left before the run's deadline, and
// reports whether the deadline is the tighter limit.
func deadlineBudget(limit time.Duration) (time.Duration, bool) {
	ms, err := strconv.ParseInt(os.Getenv(deadlineEnv), 10, 64)
	if err != nil {
		return limit, false
	}
	left := time.Until(time.UnixMilli(ms))
	return min(limit, left), left < limit
}

// neverStartedNote is the info line of a check its stage's total never let
// start, which is no finding of that check.
func neverStartedNote(c resolved) string {
	stage := stageOf(c)
	return fmt.Sprintf("not run: the %ss %s total ran out before it started", trimFloat(stageTotal(stage).Seconds()), stage)
}

// skippedCode is 1 when a stage's total kept a check from starting, since a
// run that skipped a check never passes, else 0.
func skippedCode(outcomes []outcome) int {
	if slices.ContainsFunc(outcomes, func(o outcome) bool { return o.neverStarted }) {
		return 1
	}
	return 0
}

// stoppedLine is a stopped check's first error line: the total it ran past.
func stoppedLine(c resolved) string {
	stage := stageOf(c)
	return fmt.Sprintf("Stopped at the %ss %s total", trimFloat(stageTotal(stage).Seconds()), stage)
}

// slowestShown is how many checks a slowest line names.
const slowestShown = 5

// slowest names the slowest rows with their times, slowest first, as
// "name 1.2s". It names at most slowestShown.
func slowest(rows []render.Row) string {
	sorted := slices.Clone(rows)
	slices.SortStableFunc(sorted, func(a, b render.Row) int { return cmp.Compare(b.Ms, a.Ms) })
	var names []string
	for _, r := range sorted[:min(len(sorted), slowestShown)] {
		names = append(names, r.Name+" "+took(r.Ms))
	}
	return strings.Join(names, ", ")
}

// slowestLine is the summary line naming the run's slowest checks.
func slowestLine(rows []render.Row) string {
	if len(rows) == 0 {
		return ""
	}
	return "Slowest: " + slowest(rows)
}

// breachLine names, for each stage whose total stopped a check, that stage's
// slowest checks with their times. It is empty when no total stopped one.
func breachLine(checks []resolved, outcomes []outcome, rows []render.Row) string {
	var lines []string
	for _, stage := range stages {
		in, breached := stageRows(stage, checks, outcomes, rows)
		if breached {
			lines = append(lines, fmt.Sprintf("The %s ran past their %ss total. Slowest: %s",
				stageNames[stage], trimFloat(stageTotal(stage).Seconds()), slowest(in)))
		}
	}
	return strings.Join(lines, "\n")
}

// stageRows keeps the rows of stage's checks, and reports whether the
// stage's total stopped one of them.
func stageRows(stage string, checks []resolved, outcomes []outcome, rows []render.Row) (in []render.Row, stopped bool) {
	for i, c := range checks {
		if stageOf(c) == stage {
			in = append(in, rows[i])
			stopped = stopped || stoppedByTotal(outcomes[i]) || outcomes[i].neverStarted
		}
	}
	return in, stopped
}

// stoppedByTotal reports whether a stage's total, not the run's deadline,
// stopped the check.
func stoppedByTotal(o outcome) bool {
	return o.timedOut && !o.cut
}

// took writes a check's time: milliseconds below a second, then as span does.
func took(ms int64) string {
	if ms < 1000 {
		return strconv.FormatInt(ms, 10) + "ms"
	}
	return span(time.Duration(ms) * time.Millisecond)
}

// ranMs is how long a check ran in milliseconds, or the whole
// total when the total stopped a check that started late.
func ranMs(c resolved, start time.Time, timedOut, cut bool) int64 {
	ms := time.Since(start).Milliseconds()
	if timedOut && !cut {
		return max(ms, stageTotal(stageOf(c)).Milliseconds())
	}
	return ms
}
