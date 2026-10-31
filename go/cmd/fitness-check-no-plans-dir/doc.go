// Command fitness-check-no-plans-dir keeps plans and temporary design work
// out of the repo. Plans live as `Plan`-labeled GitHub Issues. Mockups,
// screenshots and validation reports live in issue or pull request comments.
// It judges every path in the repo, not only `docs/plans/`, and points each
// offending file to the comment it belongs in.
//
// It reads the tracked-file list fitness shares with every check, so a
// staged addition or a rename's target counts before its commit. It is wired
// into the file suite, so the pre-commit hook catches a file before it lands.
//
// # Behavior
//
//   - Fail: a path with a `plans`, `mockups`, `validation` or
//     `review-artifacts` folder, compared case-insensitively.
//   - Fail: a file named `*-plan.md`, `*-mockup.*` or `pr-<number>-*`.
//   - Fail: a Markdown file outside `.github/` holding a
//     `## What needs to happen` heading followed by a checklist.
//   - Fail: git or the walk cannot list the repo's files, so the check
//     never passes on a list it could not read.
//   - Pass: maintained docs, shipped assets, and tests that quote plan text
//     inline. A vaguely named image, such as `screen.png`, passes too.
//   - Each error names the path, its rule, and the comment to move it to,
//     sorted by path. The check never deletes a file.
//   - `filesChecked` is the number of paths judged.
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
