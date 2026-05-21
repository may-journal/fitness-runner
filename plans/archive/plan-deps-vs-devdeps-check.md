---
relatedConfigurations: ['../../package.json']
---

# Plan: Fitness check for dependencies vs devDependencies

High-level outline for a check that validates `package.json` dependency placement (dependencies vs devDependencies). Classification is inferred from source; no config or allowlist.

## Scope

- Check name and location (e.g. `deps-vs-devdeps` under `packages/checks/`).
- Input: repo root; read `package.json` and scan `src/` (or configured source dir).

## Classification (inference only)

- Functional: a package is “functional” if any non-test file under `src/` imports or requires it (directly or via a subpath, e.g. `chalk`, `cspell-lib`). Exclude test files (e.g. `*.test.ts`, `*.spec.ts`).
- Dev-only: every other listed package is dev-only and should be in `devDependencies`.
- No config or allowlist: the check only looks at what the source references.

## Implementation outline

- Parse `package.json`; collect all keys from `dependencies` and `devDependencies`.
- Scan `src/` (excluding test files) for `import … from 'pkg'` / `require('pkg')` and map package names (including scoped names and common subpaths) to “referenced by functional code.”
- For each listed package: expected in `dependencies` iff referenced by functional code; else expected in `devDependencies`.
- Compare expected vs actual; emit check result: ok vs list of misplaced packages (e.g. in deps but not referenced, or referenced but in devDependencies).

## Integration

- Add to `CheckName` and registry.
- Document in checks README and check-specific README.
- Tests (no config for classification).

## Out of scope (for this plan)

- Fixing packages automatically (report only).
- Validating version ranges or lockfile consistency.
- Config/allowlist for overriding classification.
