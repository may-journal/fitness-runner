---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# dependency-currency

Flags declared npm dependencies that are behind their latest published version — an installable update means the project isn't at peak fitness. Opt-in — not in the runner's default list. Add `"dependency-currency"` to the `checks` array in `.fitnessrc.json` to enable it.

## Behavior

- Queries the npm registry over HTTPS (`dist-tags` latest) for each dependency declared in the root `package.json` and its workspaces — no npm binary.
- Direct declared deps only, since you can only bump what you declare.
- The registry URL comes from the nearest `.npmrc` `registry=` entry (repo root, then home), defaulting to the public registry.
- Pass: every dependency is at its latest published version.
- Fail: one or more dependencies are behind — reports `name: current → latest` per dep. A declared-but-uninstalled dep reads as `name: missing → latest`.
- Internal `@mayjournal/*` workspace packages are skipped (versioned in-repo, not published to the registry).
- Registry unreachable (offline / CI without network): degrades to a pass rather than hard-failing the run.

## Notes

- `filesChecked` counts the `package.json` manifests under the project (excluding `node_modules` and other skip dirs).
- Reports against `latest`, not `wanted`: a dependency inside its declared semver range but behind the newest release is still flagged.
- The point is to stay current, not merely in-range.
