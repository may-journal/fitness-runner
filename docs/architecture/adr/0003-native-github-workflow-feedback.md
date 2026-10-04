---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# 0003 — Every workflow provides native GitHub feedback

## Context

Fitness links file failures to PR diff lines through GitHub error annotations. Every workflow needs this reporting contract so people can act without searching logs.

## Decision

1. All repo workflows require native failure annotations and job summaries, including reusable workflows, org callers, checks, builds, installs, and releases.
2. File findings use `::error file=path,line=N::message` with accurate checkout-relative locations. Omit unknown lines; use `::error::message` when no source file applies.
3. Messages name the failed check or operation and explain the cause. Summaries link issue and PR targets without inventing source locations.
4. Each reporting job writes outcomes to `GITHUB_STEP_SUMMARY`, including success and known counts. Disclose overflow and link retained full reports.
5. Preserve exit codes, gate policy, verdict comments, and labels. Successful remediation stays successful. Reporting failures must never hide failed checks.
6. Shared Go code owns reporting. Tiny shell fallbacks cover bootstrap failures before Go runs. Accept native tool annotations without duplicates.
7. Enable workflow commands only in Actions. Escape properties, messages, and summary content for their formats; exclude secrets and retain useful local logs.
8. Review every new or changed workflow against this contract. Test locations, global errors, success, overflow, escaping, and exit codes across reusable callers.

## Consequences

1. GitHub places located findings on matching diff lines. Other findings remain in checks and summaries; inline cards depend on the diff.
2. Commands share `go/internal/report`. Every executable job has a fallback reporter; reusable callers inherit reporting from the called job.
3. Runner outages, forced cancellation, and failures before reporting starts may leave only GitHub status and logs. Report setup failures when execution permits.
4. Annotations are capped at ten; summaries at 1 MiB. Workflows upload full overflow reports. Tests cover nested checkouts and fallback execution.
5. Triggers, permissions, merge policy, and automation retain their current behavior. Changes to those controls require explicit scope.

[GitHub workflow commands](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands) define the output formats. [Workflow docs](../../../.github/workflows/README.md) describe existing command paths.
