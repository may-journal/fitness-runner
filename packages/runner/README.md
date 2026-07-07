---
relatedConfigurations: ['package.json']
---

# @mayjournal/fitness

CLI runner for fitness checks. See [architecture-index.md](../../architecture-index.md) and [architecture/](../../architecture/) for the C4 model.

## Performance and publish footprint

Issue [#13](https://github.com/may-journal/fitness-runner/issues/13) tracks bundle size and runtime tooling. Plan: [`plans/archive/plan-issue-13-bundle-size-performance.md`](../../plans/archive/plan-issue-13-bundle-size-performance.md).

### Publish audit (monorepo root)

```bash
npm run audit:publish
npm run audit:publish -- --strict --gate-runner
```

### `loadCheck` microbenchmarks

```bash
npm run bench
# or from repo root:
npm run bench:load-check
```

Measures `loadCheck`, `resolveCheckNames`, and sequential loads for the default check bundle.

### CLI latency (hyperfine)

```bash
hyperfine --warmup 3 'node bin/fitness.js --help'
hyperfine --warmup 3 'npm run fitness -- --help'
```

Cold `npx fitness` includes an extra child process (`bin/fitness.js` spawns the runner); that fixed cost usually dominates short runs.
