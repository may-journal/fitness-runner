---
relatedConfigurations: ['../../../.fitnessrc.json']
---

# markdown-filename-convention

Enforces that every `.md` file's basename follows a single naming convention, so doc trees stay predictable for links, tooling, and case-sensitive CI. Standard root docs (`README.md`, `CHANGELOG.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `LICENSE.md`, `SECURITY.md`) are always exempt.

This one package exports two check names — two flavors of the same fitness function (`runMarkdownFilenameCheck` in `@mayjournal/fitness-shared`), each pinned to one convention:

- `markdown-filename-kebab-case` — basename must be kebab-case: `^[a-z0-9]+(-[a-z0-9]+)*\.md$`
- `markdown-filename-camel-case` — basename must be camelCase: `^[a-z][a-zA-Z0-9]*\.md$`

Both are opt-in — not part of `defaultChecks`. Enable exactly one (a repo is kebab-case or camelCase, not both).

## Enable

```ts
// .fitnessrc.ts — pick the convention your repo uses
export default { checks: ['markdown-filename-kebab-case'] };
// or
export default { checks: ['markdown-filename-camel-case'] };
```

## What passes

`markdown-filename-kebab-case` — lowercase alphanumeric words joined by single hyphens:

```
docs/api-design.md
architecture/02-containers.md
adr001.md
```

`markdown-filename-camel-case` — lowercase first letter, then letters or digits, no separators:

```
docs/releaseNotes.md
docs/apiDesign.md
adr001.md
```

## What fails

Each flavor rejects anything outside its convention (and always rejects `snake_case`, `PascalCase`, and spaces), reporting the path and the required convention:

```
docs/releaseNotes.md: filename must be kebab-case
docs/api-design.md: filename must be camelCase
docs/Api_Design.md: filename must be kebab-case
```

## Advanced

Only the basename is validated, so directory casing is ignored — `SomeDir/api-design.md` is judged solely by `api-design.md`. On case-insensitive filesystems (macOS default), two camelCase names that differ only in letter case resolve to the same file, so prefer `markdown-filename-kebab-case` when that collision risk matters. To keep a non-conforming legacy filename, rename it or add its basename to the allowed set in `runMarkdownFilenameCheck`.

## Behavior

- Discovers files via `findFilesByExtension(root, '.md')` (standard skip dirs like `node_modules`, `dist`, `coverage`, `.git` excluded).
- Pass: every basename is a standard root doc or matches the enabled flavor's convention.
- Fail: `path: filename must be <kebab-case|camelCase>` per non-conforming file.
- `filesChecked` counts every `.md` file scanned.

## Contributing

This README is the canonical description for both check names. The shared logic lives in `runMarkdownFilenameCheck` (`@mayjournal/fitness-shared`); this package's `src/kebab-case.ts` and `src/camel-case.ts` each supply one `FilenameConvention` and are exposed as separate check names by the bundler. To change the rule, edit the shared function (and its tests) and keep this README in sync.
