---
relatedConfigurations: ['../../../package.json']
---

# dependency-currency

Flags declared npm dependencies that are behind their latest published version — an installable update means the project isn't at peak fitness. Opt-in — not part of `defaultChecks`. Add `'dependency-currency'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Runs: `npm outdated --json` from the project root (direct declared deps only — no `--all` — since you can only bump what you declare).
- Pass: every dependency is at its latest published version.
- Fail: one or more dependencies are behind — reports `name: current → latest` per dep. A declared-but-uninstalled dep reads as `name: missing → latest`.
- Internal `@mayjournal/*` workspace packages are skipped (versioned in-repo, not published to the registry).
- Registry unreachable (offline / CI without network): degrades to a pass rather than hard-failing the run.

## Notes

- `filesChecked` counts the `package.json` manifests under the project (excluding `node_modules` and other skip dirs).
- Reports against `latest`, not `wanted`: a dependency inside its declared semver range but behind the newest release is still flagged, since the point is to stay current, not merely in-range.
