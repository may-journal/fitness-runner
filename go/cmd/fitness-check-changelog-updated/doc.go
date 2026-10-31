// Command fitness-check-changelog-updated checks that each staged commit
// writes a changelog entry about its change. It runs when the runner
// provides staged file context. The added lines in the branch's [entry file]
// must share at least three words with the rest of the staged diff.
//
// # One entry per branch
//
// A branch keeps one entry file, so later commits add bullets to the file
// its first commit wrote. You write its `### yyyy.mm.dd.HHMM` heading; the [changelog] check
// judges its format.
//
// `CHANGELOG.md` is not judged: a commit may add, edit, or delete it, and it
// counts as part of the rest of the diff.
//
// A branch's own commits are those no other `origin` branch holds, so a
// stacked branch starts where its base ends. On the default branch, or
// without `origin` refs, each commit is judged alone. `fitness pr-check`
// holds the whole pull request to one entry.
//
// # Behavior
//
//   - Pass (replay in progress): mid-merge, mid-cherry-pick, or mid-revert (`MERGE_HEAD`, `CHERRY_PICK_HEAD`, or `REVERT_HEAD` present) skips.
//   - Pass: no staged context skips.
//   - Pass: entry file additions share at least 3 words with the rest of the staged diff.
//   - Fail: no staged entry file → "Stage a changelog/<yyyy-mm-dd-HHMM>-<change>.md entry file that mentions your staged changes".
//   - Fail: the branch already has its entry file and the commit adds another → error naming the branch's entry file and each new one.
//   - Fail: a staged new entry file whose name lacks its heading's date → error naming the file it should be.
//   - Fail: a staged edit or deletion of an entry file the newest release tag holds → error asking for a new entry file.
//   - Fail: entry file additions share fewer than three words with the rest of the diff → error with the count.
//
// A replay skips, since it re-stages historical entries. A too-small overlap
// adds a second line, "use words like: …", listing up to 10 words from the
// staged diff. Only the added parts of the entry
// file in the diff count, with words lowercased and at least 3 long.
//
// [entry file]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-changelog/README.md#entry-files
// [changelog]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-changelog/README.md
//
//fitness:container cli
//fitness:component check Each check is its own package, built into the fitness binary.
package check
