package main

import (
	"cmp"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/checkkit"
	"github.com/may-journal/fitness-runner/go/internal/render"
)

// fastTotalEnv names a lower fast total in milliseconds, which requirement
// tests set to see the total stop a check without waiting a minute. It never
// raises the total.
const fastTotalEnv = "FITNESS_INTERNAL_FAST_TOTAL_MS"

// fastTotal is checkkit.FastTotal, or the lower total fastTotalEnv names.
func fastTotal() time.Duration {
	ms, err := strconv.ParseInt(os.Getenv(fastTotalEnv), 10, 64)
	if err != nil || ms <= 0 {
		return checkkit.FastTotal
	}
	return min(checkkit.FastTotal, time.Duration(ms)*time.Millisecond)
}

// unlimited is the limit of a check that no total bounds.
const unlimited = time.Duration(math.MaxInt64)

// stageOf is the stage c declares, fast when it declares none.
func stageOf(c resolved) string {
	return checkkit.StageOf(c.desc)
}

// stageTotal is the total that bounds a stage's checks together, unlimited
// for the test stage.
func stageTotal(stage string) time.Duration {
	if stage == checkkit.StageTests {
		return unlimited
	}
	return fastTotal()
}

// stageEnds holds when each stage's total runs out in one run.
type stageEnds map[string]time.Time

// startStages starts every stage's total at now.
func startStages(now time.Time) stageEnds {
	ends := stageEnds{}
	for _, stage := range []string{checkkit.StageFast, checkkit.StageTests} {
		if total := stageTotal(stage); total != unlimited {
			ends[stage] = now.Add(total)
		}
	}
	return ends
}

// limit is how long a check in stage may still run: the time left in the
// stage's total.
func (e stageEnds) limit(stage string) time.Duration {
	end, ok := e[stage]
	if !ok {
		return unlimited
	}
	return time.Until(end)
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

// breachLine names the slowest fast checks when the fast total stopped one,
// and is empty otherwise.
func breachLine(checks []resolved, outcomes []outcome, rows []render.Row) string {
	fast, breached := fastRows(checks, outcomes, rows)
	if !breached {
		return ""
	}
	return fmt.Sprintf("The fast checks ran past their %ss total. Slowest: %s",
		trimFloat(fastTotal().Seconds()), slowest(fast))
}

// fastRows keeps the fast checks' rows, and reports whether the fast total
// stopped one of them.
func fastRows(checks []resolved, outcomes []outcome, rows []render.Row) (fast []render.Row, stopped bool) {
	for i, c := range checks {
		if stageOf(c) == checkkit.StageFast {
			fast = append(fast, rows[i])
			stopped = stopped || stoppedByTotal(outcomes[i])
		}
	}
	return fast, stopped
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
