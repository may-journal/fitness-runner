---
relatedConfigurations: ['../../../package.json']
---

# markdown-filename-convention

Enforces that every `.md` filename is either kebab-case or camelCase so doc trees stay predictable for links, tooling, and case-sensitive CI. Opt-in — not part of `defaultChecks`. Add `'markdown-filename-convention'` to `.fitnessrc` `checks` to enable it.

## Behavior

- Discovers `.md` files with `findFilesByExtension(root, '.md')` (excludes `node_modules`, dist, coverage, .git, githooks).
- Validates each file's basename only, so a path like `plans/foo-bar.md` is judged by `foo-bar.md`.
- Pass: basename matches kebab-case `^[a-z0-9]+(-[a-z0-9]+)*\.md$` or camelCase `^[a-z][a-zA-Z0-9]*\.md$`.
- Always allowed regardless of case: `README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `LICENSE.md`.
- Fail: any other basename (`snake_case`, PascalCase, spaces, etc.) — one error per file, e.g. `path/to/Bad_Name.md: filename must be kebab-case or camelCase`.
- `filesChecked` counts the `.md` files scanned.

## Notes

- On case-insensitive filesystems (macOS default) camelCase names that differ only in case can collide; prefer kebab-case where that risk matters.
