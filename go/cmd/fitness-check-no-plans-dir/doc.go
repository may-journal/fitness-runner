// Command fitness-check-no-plans-dir guards the plans-to-Issues migration:
// it fails when any file exists under `docs/plans/`, because plans now live
// as `Plan`-labeled GitHub Issues, not checked-in files. It points the author
// to open a Plan Issue instead.
//
// It is wired into the file suite, so the pre-commit hook catches a
// `docs/plans/` file before it lands.
//
// # Behavior
//
//   - Pass: no file exists under `docs/plans/`.
//   - Fail: one error per file found there, naming the file and the fix.
//   - `filesChecked` is the number of offending files (zero when clean).
package check
