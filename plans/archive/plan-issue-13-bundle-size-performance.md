---
relatedConfigurations: ['../../.fitnessrc.json']
issue: https://github.com/may-journal/fitness-runner/issues/13
---

# Plan: Bundle size and runtime performance (issue #13)

Track publish footprint and CLI runtime for this monorepo (`@mayjournal/fitness` + 15 publishable workspace packages, dynamic check loading). GitHub issue: [#13](https://github.com/may-journal/fitness-runner/issues/13).

## Context (May 2026)

- Node ≥24, ESM, `files: ["dist"]` (and `bin` on runner).
- Checks load via `createRequire` + `import(pathToFileURL(...))`.
- Most checks run in worker threads that call `loadCheck` again by name.

Consumers care about install size (tarball + transitive `node_modules`) and startup/import cost, not browser bundle graphs. Heavy cost lives in check packages (e.g. `eslint`, `@typescript-eslint/*`), not in the slim runner.

## Architecture notes (no new tools)

- Double Node process (`bin/fitness.js` → spawn): largest fixed cost for `npx fitness`.
- Double module load (main + worker `loadCheck`): profile before optimizing.
- Check choice dominates — eslint/prettier/cspell dwarf runner import time; à la carte installs are the real consumer perf win.

## Implementation phases

| Phase                     | Status | Work                                                                                                                                               |
| ------------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. `audit:publish`        | Done   | [`scripts/audit-publish/`](../../scripts/audit-publish/)                                                                                           |
| 1b. Publint `exports` fix | Done   | `types` first in all 15 publishable `package.json` files                                                                                           |
| 2. Knip + CI advisory     | Done   | Per-workspace `knip --no-exit-code`; root `npm run knip -ws --if-present` (reports issues, exit 0 until config is tuned)                           |
| 3. Runtime benches        | Done   | [`load-check.bench.ts`](../../packages/runner/src/checks/load-check.bench.ts), [runner README](../../packages/runner/README.md)                    |
| 4. PR size comment        | Done   | [`.github/workflows/publish-audit.yml`](../../.github/workflows/publish-audit.yml), [`pr-comment.mjs`](../../scripts/audit-publish/pr-comment.mjs) |
| 5. attw + runner gate     | Done   | `--attw` (advisory), `--gate-runner` (24 KiB tarball, enforced on `main` push)                                                                     |

## Commands

```bash
npm run audit:publish
npm run audit:publish -- --no-build --strict --gate-runner
npm run audit:publish:comment
npm run knip
npm run bench:load-check
```

## CI

[`.github/workflows/publish-audit.yml`](../../.github/workflows/publish-audit.yml):

- PR: build → `audit:publish --json --attw` → sticky PR comment (advisory).
- push `main`: `audit:publish --strict --gate-runner` (fails on publint or runner size).
- knip: advisory (`continue-on-error`).
- bench: weekly schedule + `workflow_dispatch`.

## Baseline (local, May 2026)

| Package                      | Tarball           |
| ---------------------------- | ----------------- |
| `@mayjournal/fitness`        | ~20 KiB           |
| `@mayjournal/fitness-shared` | ~15 KiB           |
| Check packages (12)          | ~1.7–4.1 KiB each |
| `@mayjournal/fitness-checks` | ~1 KiB            |
| Total (15 packages)          | ~70 KiB           |

Runner tarball gate: 24 KiB (`--gate-runner`, overridable with `--runner-max-tarball N`).

## References

- [publint docs](https://publint.dev/docs)
- [Knip monorepos](https://knip.dev/features/monorepos-and-workspaces)
- [@arethetypeswrong/cli](https://www.npmjs.com/package/@arethetypeswrong/cli)
- [Vitest bench](https://vitest.dev/guide/features.html#benchmarking)
- [hyperfine](https://github.com/sharkdp/hyperfine)
