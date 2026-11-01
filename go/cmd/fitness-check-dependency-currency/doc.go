// Command fitness-check-dependency-currency fails a declared npm dependency
// behind its latest published version, since an update means the project
// isn't at peak fitness. It runs in every repo and passes clean
// without a `package.json`.
//
// # Behavior
//
//   - Queries the npm registry over HTTPS (`dist-tags` latest), without npm, for each dependency the root `package.json` and its workspaces declare.
//   - Direct declared deps only, since you can only bump what you declare.
//   - The registry URL comes from the nearest `.npmrc` `registry=` entry (repo root, then home), else the public registry.
//   - Pass: every dependency is at its latest published version.
//   - Fail: each behind dependency reports `name: current → latest`, with identical lines collapsed to one.
//   - A declared-but-uninstalled dep reads as `name: missing → latest`.
//   - Internal `@mayjournal/*` workspace packages are skipped, since they are versioned in-repo and never published.
//   - A registry that is unreachable or answers with an error or garbage gives no finding, and info counts those packages.
//
// One proxy blocks GitHub and the registry alike. So when the runner's probe
// finds GitHub's API host unreachable, the check reads no registry and
// reports info naming the host and proxy.
//
// # Notes
//
//   - `filesChecked` counts the tracked `package.json` manifests.
//   - Reports against `latest`, not `wanted`, so a dependency inside its semver range but behind the newest release is still flagged.
//   - The point is to stay current, not merely in-range.
package check
