---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0002 — One binary per check retires the shared-lint-path decision

## Status

Supersedes [0001](./0001-fix-the-work-not-the-limit.md), decision 4 and consequence 4 only. Everything else in 0001 still stands.

## Context

0001 assumed two code paths ran the same lint analysis — an in-process check and a `fitness-shared lint` CLI — sharing one config so a change landed once. The Go rewrite removed that split: every check now ships as a single static binary (`fitness-check-<name>`) with no separate CLI path.

## Decision

There is one lint path, not two. The eslint check's config is owned by that one binary, so "shared across two paths" no longer applies. The timeout-budget decisions and consequences of 0001 (items 1 to 3) are unchanged.

## Consequences

The lint config is still a single point of change, but only one path consumes it, so there is no cross-path drift left to guard against.
