---
relatedConfigurations: ['../../package.json']
---

# audit-publish

Audits publish footprint for every npm-publishable workspace: `publint` plus `npm pack --dry-run --json` tarball sizes. Implements [issue #13](https://github.com/may-journal/fitness-runner/issues/13) phase 1.

## Usage

```bash
npm run audit:publish                      # build, then audit all 15 packages
npm run audit:publish -- --no-build        # skip build (use when dist is already fresh)
npm run audit:publish -- --strict          # exit 1 if any publint errors
npm run audit:publish -- --gate-runner     # exit 1 if @mayjournal/fitness tarball > 24 KiB
npm run audit:publish -- --attw            # include advisory attw on runner + shared
npm run audit:publish -- --json            # machine-readable report
npm run audit:publish -- --markdown        # markdown table (stdout)
npm run audit:publish:comment              # markdown for PR comments (no build)
```

## Output

Prints a table sorted by tarball size (largest first) and a total across all publishable packages. Publint details print to stderr when there are failures (advisory by default; use `--strict` to fail CI).

## Runtime benchmarks

`loadCheck` microbenchmarks (issue #13 phase 3):

```bash
npm run bench:load-check
```

CLI latency (manual, requires [hyperfine](https://github.com/sharkdp/hyperfine)):

```bash
hyperfine --warmup 3 'node packages/runner/bin/fitness.js --help'
hyperfine --warmup 3 'npx fitness --help'
```

## API

```js
import { runPublishAudit, auditOnePackage, parsePackDryRun } from './index.mjs';
```
