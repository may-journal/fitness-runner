package main

import (
	"os"
	"strconv"
	"time"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

// defaultTimeout is a check's execution budget when neither the check nor
// .fitnessrc.json sets one; a breach means cutting the work (ADR 0001).
var defaultTimeout = time.Duration(conf.Defaults.TimeoutMs) * time.Millisecond

// withBudgets gives every check that declares no budget of its own the
// repo's timeoutMs from .fitnessrc.json, when set. A check's own budget names
// an irreducible external cost, so it stands.
func withBudgets(cfg *conf.Config, checks []resolved) []resolved {
	if cfg == nil || cfg.TimeoutMs <= 0 {
		return checks
	}
	for i := range checks {
		checks[i].configMs = cfg.TimeoutMs
	}
	return checks
}

// timeoutFor returns the check's budget: its own or the configured one,
// else defaultTimeout.
func timeoutFor(c resolved) time.Duration {
	if c.desc.TimeoutMs == 0 && c.configMs > 0 {
		return time.Duration(c.configMs) * time.Millisecond
	}
	return declaredBudget(c)
}

// declaredBudget is the budget the check declares, else defaultTimeout.
func declaredBudget(c resolved) time.Duration {
	if c.desc.TimeoutMs > 0 {
		return time.Duration(c.desc.TimeoutMs) * time.Millisecond
	}
	return defaultTimeout
}

// deadlineEnv holds the Unix time in milliseconds by which a run must end.
// A Claude hook sets it so the run reports inside the hook's timeout.
const deadlineEnv = "FITNESS_INTERNAL_DEADLINE"

// cutNote is the info line of a check the run's deadline stopped or never
// let start.
const cutNote = "skipped: the run's deadline passed"

// deadlineBudget caps budget at the time left before the run's deadline, and
// reports whether the deadline is the tighter limit.
func deadlineBudget(budget time.Duration) (time.Duration, bool) {
	ms, err := strconv.ParseInt(os.Getenv(deadlineEnv), 10, 64)
	if err != nil {
		return budget, false
	}
	left := time.Until(time.UnixMilli(ms))
	return min(budget, left), left < budget
}
