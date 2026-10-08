package main

import (
	"time"

	"github.com/may-journal/fitness-runner/go/internal/conf"
)

// defaultTimeout is a check's execution budget when neither the check nor
// .fitnessrc.json sets one. A breach means wasted work, so cut the work, not
// the limit (ADR 0001).
const defaultTimeout = 5000 * time.Millisecond

// withBudgets gives every check that declares no budget of its own the
// repo's timeoutMs from .fitnessrc.json, when set. A check's own budget names
// an irreducible external cost, so it stands.
func withBudgets(cfg *conf.Config, checks []resolved) []resolved {
	if cfg == nil || cfg.TimeoutMs <= 0 {
		return checks
	}
	for i := range checks {
		if checks[i].desc.TimeoutMs == 0 {
			checks[i].desc.TimeoutMs = cfg.TimeoutMs
		}
	}
	return checks
}

// timeoutFor returns the check's budget: its own or the configured one,
// else defaultTimeout.
func timeoutFor(c resolved) time.Duration {
	if c.desc.TimeoutMs > 0 {
		return time.Duration(c.desc.TimeoutMs) * time.Millisecond
	}
	return defaultTimeout
}
