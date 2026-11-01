// Command fitness-check-workflow-ref-exists fails a workflow that calls a
// may-journal workflow or action at a tag or sha with no such path yet
// ([#485]). Such a caller fails every event in its repo with a
// missing-workflow error, as fitness-runner#249 did.
//
// # Behavior
//
//   - It reads each `uses:` line in the workflow files directly under `.github/workflows`.
//   - A `may-journal/<repo>/<path>@<ref>` line names a reusable workflow file or an action folder. A bare `may-journal/<repo>@<ref>` names the repo's root action.
//   - A ref is a tag or a full 40-character sha, and the path must exist at it.
//   - An action folder holds an `action.yml` or `action.yaml` file at the ref.
//   - Fail: the path is missing at the ref, naming the file, line, path, and ref.
//   - Fail: the ref is not a tag, a branch, or a full sha, or a sha names no commit.
//   - Ignored: local `./` actions, other owners, branch refs, and refs built from `${{ }}` expressions.
//   - Ignored: text inside a `|` or `>` block, such as a `run:` script.
//
// In a pull request, or a staged commit, it reads only the changed workflow
// files. Composite actions outside `.github/workflows` are out of scope.
//
// # GitHub access
//
// The check asks GitHub through `gh api`, so it uses `gh`'s login or
// `GH_TOKEN`. The org's `fitness-suite` and CI workflows pass `GH_TOKEN` to
// the suite for it.
//
// When GitHub cannot answer, it never passes or fails a ref it could not
// see. No `gh`, no network, a bad token, a rate limit, or 20 seconds of reads
// stops it. It keeps the findings so far and notes the calls left
// unverified. When the runner's probe found GitHub's API host unreachable,
// it asks nothing and reports info naming the host and proxy.
//
// GitHub refusing one request, such as a 403, skips that call with a note. A
// repo that answers 404 is one the token cannot read, so its calls are
// skipped with a note too. A workflow token reads only its own repo and
// public ones, so private repos are skipped in CI.
//
// [#485]: https://github.com/may-journal/fitness-runner/issues/485
package check
