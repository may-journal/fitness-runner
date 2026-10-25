// Command fitness-check-release-changelog binds Release Please's manifest to
// the version file a reviewer can inspect. The check applies when
// `.release-please-manifest.json` exists and contains a root release
// version.
//
// # Behavior
//
//   - Use an explicit availability gate and pass with zero files when the repository has no Release Please manifest.
//   - Pass during bootstrap when the manifest is an empty JSON object.
//   - Require the root manifest value and `version.txt` to contain the same numeric `major.minor.patch` version.
//   - Reject malformed JSON, unsupported manifests without the root `.` package, a missing `version.txt`, and mismatched versions.
//
// Release notes come from [entry files], not `CHANGELOG.md`, so this check
// asks nothing of `CHANGELOG.md`. Set `"skip-changelog": true` in
// `release-please-config.json` so Release Please stops writing it.
//
// [entry files]: https://github.com/may-journal/fitness-runner/blob/main/go/cmd/fitness-check-changelog/README.md#entry-files
package check
