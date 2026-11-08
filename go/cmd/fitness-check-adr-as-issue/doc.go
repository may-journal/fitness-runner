// Command fitness-check-adr-as-issue keeps architecture decisions in `ADR`
// issues, never in files. An issue can be discussed, linked and superseded
// in one place, while a file drifts from the talk that made it. The `ADR`
// issue template and issue-check judge each issue's shape.
//
// It runs in every repo, the runner's own included.
//
// # Behavior
//
//   - Fail: a Markdown file in a folder named `adr`, such as
//     `docs/architecture/adr/0003-sync.md`, compared case-insensitively.
//     The folder's README or index lists ADRs and passes.
//   - Fail: a Markdown file whose first top-level heading starts with the
//     word ADR, such as `# ADR 12: Use SQLite`.
//   - Fail: a Markdown file with `## Context`, `## Decision` and
//     `## Consequences` sections, such as an unmoved check README.
//   - Pass: a file whose whole text is one full GitHub issue URL.
//   - Pass: a check README that links its ADR issue in place of the decision.
//   - Pass: files under `.github/`, where the issue template lives.
//   - Each error names the file and its rule, and says to open an `ADR`
//     issue with `fitness adr-migrate`.
//   - Fail: git or the walk cannot list the repo's files.
//
// # Moving existing ADRs
//
// `fitness adr-migrate` opens one `ADR` issue per file and rewrites each
// link to the file into the issue's full URL. It removes the file, or
// trims a check README to its title, summary, issue URL and `## Check`.
// Its `--dry-run` flag prints the plan and changes nothing, and a rerun
// reuses the issues it opened.
//
// When the migration cannot run yet, `adrAsIssue.allow` lists the paths
// still waiting, each glob with the reason it waits.
//
//   - Fail: an allow entry without a reason, or whose glob is malformed.
//   - Fail: an allow entry that matches no ADR file, so the list shrinks
//     as files move and ends empty.
//   - `filesChecked` is the number of paths judged.
package check
