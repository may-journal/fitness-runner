// Command fitness-check-dependency-currency fails a declared npm dependency
// that is behind its latest published version.
// An installable update means the project isn't at peak fitness.
// It runs in every repo and passes clean without a `package.json`.
//
// # Behavior
//
//   - Queries the npm registry over HTTPS (`dist-tags` latest) for each dependency declared in the root `package.json` and its workspaces. No npm binary.
//   - Direct declared deps only, since you can only bump what you declare.
//   - The registry URL comes from the nearest `.npmrc` `registry=` entry (repo root, then home). It defaults to the public registry.
//   - Pass: every dependency is at its latest published version.
//   - Fail: one or more dependencies are behind. It reports `name: current → latest` per dep, and identical lines collapse to one.
//   - A declared-but-uninstalled dep reads as `name: missing → latest`.
//   - Internal `@mayjournal/*` workspace packages are skipped, since they are versioned in-repo and never published to the registry.
//   - Registry unreachable or answering an error or garbage: no finding for that package. Info counts the packages it could not query.
//
// One proxy blocks GitHub and the registry alike. So when the runner's probe
// finds GitHub's API host unreachable, the check reads no registry and
// reports info naming the host and proxy.
//
// # Notes
//
//   - `filesChecked` counts the tracked `package.json` manifests.
//   - Reports against `latest`, not `wanted`. A dependency inside its declared semver range but behind the newest release is still flagged.
//   - The point is to stay current, not merely in-range.
package check
