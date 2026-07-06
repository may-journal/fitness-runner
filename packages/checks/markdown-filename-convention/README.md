---
relatedConfigurations: ['../../../package.json']
---

# markdown-filename-convention

Enforces that every `.md` file's basename is either kebab-case or camelCase, so doc trees stay predictable for links, tooling, and case-sensitive CI. Opt-in — not part of `defaultChecks`.

## Enable

```ts
// .fitnessrc.ts
export default { checks: ['markdown-filename-convention'] };
```

## What passes

Kebab-case (lowercase words joined by single hyphens) or camelCase (lowercase first letter, then letters/digits):

```
docs/api-design.md      (kebab-case)
plans/plan-checks.md    (kebab-case)
docs/releaseNotes.md    (camelCase)
docs/adr001.md          (camelCase, digits allowed)
```

Standard root docs are always allowed regardless of case:

```
README.md   CHANGELOG.md   CONTRIBUTING.md   CODE_OF_CONDUCT.md   LICENSE.md   SECURITY.md
```

## What fails

`snake_case`, `PascalCase`, spaces, or any other style:

```
docs/Api_Design.md
Architecture.md
docs/My Notes.md
```

each report the path and the rule:

```
docs/Api_Design.md: filename must be kebab-case or camelCase
Architecture.md: filename must be kebab-case or camelCase
docs/My Notes.md: filename must be kebab-case or camelCase
```

## Advanced

Only the basename is validated, so directory casing is ignored — `SomeDir/api-design.md` is judged solely by `api-design.md`. The two accepted patterns are:

```
kebab:  ^[a-z0-9]+(-[a-z0-9]+)*\.md$
camel:  ^[a-z][a-zA-Z0-9]*\.md$
```

On case-insensitive filesystems (macOS default), two camelCase names that differ only in letter case resolve to the same file, so prefer kebab-case when that risk matters. To keep a non-conforming legacy filename, rename it to kebab-case or add its basename to the allowed set.

## Behavior

- Discovers files via `findFilesByExtension(root, '.md')` (standard skip dirs like `node_modules`, `dist`, `coverage`, `.git` excluded).
- Pass: every basename is a standard root doc, kebab-case, or camelCase.
- Fail: `path: filename must be kebab-case or camelCase` per non-conforming file.
- `filesChecked` counts every `.md` file scanned.
