# workflow-timeouts

Fails a workflow `timeout-minutes` of 10 or more whose comment cites no observed maximum and date. Long limits hide slow regressions, and the cited maximum shows why the limit is that long.

- Inputs: the `.yml` and `.yaml` files directly under `.github/workflows`, or only the changed ones in a scoped run.
- Pass: a whole number under 10, bare or quoted, or one whose line or comment lines just above cite its maximum.
- A cite names hours, minutes or seconds and a real date, and an expression value is not judged.
- Fail: a long limit without that comment, naming the file, the line and the comment to add.
- Fix: measure the job's maximum from recent runs, set twice it rounded up, and cite it as `# Observed max 4m12s on 2026-11-08.`

It is one of this repo's [local checks](../../docs/local-checks.md). [Requirement 0147](../../docs/requirements/0147-checks-workflow-timeouts.md) states its behavior.
