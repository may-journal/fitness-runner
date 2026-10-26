// Command fitness-check-adr-checks pairs each numbered architecture
// decision record with the check that keeps it. An ADR is prose, so a
// decision holds only while a check in `fitness/adr-NNNN/` proves it. A
// requirement doc holds the same way, while its tests pass.
//
// The check applies to a repo with a `fitness/` folder of its own checks
// and numbered ADRs. An ADR is a `NNNN-kebab-title.md` file in a folder
// named `adr`. A repo without either passes with `filesChecked: 0`. The
// [local checks guide] shows how to write a check.
//
// # Pairs
//
//   - Check: each ADR needs a `fitness/adr-NNNN/` folder, unless its status
//     line says it is superseded.
//   - Links: the ADR links its check's README, and the README links the ADR,
//     so either leads a reader to the other.
//   - Orphans: a `fitness/adr-NNNN/` folder needs an ADR of that number.
//   - Changes: a changed ADR fails unless a file in its check changed too.
//   - Check changes: a changed check file fails unless its ADR changed.
//   - Scope: a full scan judges no changes; a commit judges its staged
//     files, and a pull request its diff.
//
// A paired check that judges no files fails as a broken scan. So an ADR
// check whose file list goes empty never passes by reading nothing.
//
// [local checks guide]: https://github.com/may-journal/fitness-runner/blob/main/docs/local-checks.md
package check
