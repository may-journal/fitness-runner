### 2026.10.17.1910

- Feat: fitness: `--json=<file>` records each check's time, size, and errors; the job summary adds a timing table, under `--blob` too.
- Feat: fitness: `boxes <report> <page>` draws the report as an HTML page, one box per check sized by runtime, size, or error lines.
- Feat: fitness: `localChecks` in `.fitnessrc.json` names a repo directory of checks, searched first; external `disabledChecks` turns checks off, in `--blob` and `pr-check` too.
- Fix: hooks: commit-msg and pre-push skip a check that external `disabledChecks` turns off, including the Plan gate.
- Docs: ADR 0009, the check protocol, and the CI guide cover these; requirements 0009.9, 0010.12, 0077.9, 0089, and 0090 prove them.
