---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0003 — Quality is executable fitness functions

## Context

may-journal spans many repos in several languages. Written guidelines and ad hoc CI let each repo drift. Rules lived in prose no tool enforced, and every repo wired its own checks. Quality was inconsistent, and a good rule in one repo was absent in the next.

An architectural fitness function is an executable test of a quality goal. Making the rules runnable, not advisory, is the only way they hold across repos and over time.

## Decision

Quality lives as executable fitness functions in the `fitness-runner`. Each check is a self-contained binary that reads a target and emits a JSON result (ADR 0002). The runner ships a comprehensive default; language checks self-gate, so one list fits Swift, Go, JS, and docs repos.

The checks also lint Issue and PR descriptions, not just files. Repos adopt through reusable workflows and an org rule, so the check set rolls out centrally, not repo by repo. Plans and approvals live as Issues the same checks validate.

## Consequences

The bar is strict and uniform: a rule added once applies everywhere on the next run. The cost is one binary per check and a per-repo build, paid for by a plugin model with no shared lint path.

Repos track `main`, so they stay current but ride unreleased changes; a caller can pin a tag instead. Strict, network, and format-specific checks stay opt-in, so the default never over-reaches. See [checks](../../checks.md) and [adoption](../../adoption.md); this builds on ADRs [0001](0001-fix-the-work-not-the-limit.md) and [0002](0002-one-binary-per-check-no-shared-lint-path.md).
