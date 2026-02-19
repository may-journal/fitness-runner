---
relatedConfigurations: ['../eslint.config.cjs', '../package.json']
---

# Research: ESLint Architecture and Fitness Checks

This document explores how ESLint organizes a project with hundreds of rules, what lessons
apply here, and whether ESLint could serve as the runtime engine for fitness checks.

## How ESLint Organizes a Project With Many Rules

ESLint manages over 250 core rules plus a rich plugin ecosystem. Several architectural
decisions make that scale tractable.

### Rule-per-file layout

Every core rule lives in its own file under `lib/rules/<rule-name>.js`. A central registry
(`lib/rules/index.js`) imports them all and exposes a `Map<string, Rule>`. No rule file
knows about any other rule; coupling flows only through the registry.

```
lib/
  rules/
    no-unused-vars.js
    sort-keys.js
    complexity.js
    ...  (250+ files)
  rules/index.js   ← central registry
```

### Uniform rule shape

Each rule exports a single object with two required keys:

```js
module.exports = {
  meta: {
    type: 'suggestion', // 'problem' | 'suggestion' | 'layout'
    docs: { description: '...', recommended: true },
    fixable: 'code', // or 'whitespace', or omitted
    schema: [
      /* JSON Schema for options */
    ],
    messages: { key: 'template {{var}}' },
  },
  create(context) {
    // returns an AST visitor object
    return {
      Identifier(node) {
        /* ... */
      },
    };
  },
};
```

`meta` is the contract the rule makes with the outside world (docs site, config validators,
`--fix` support). `create` is the implementation. They never bleed into each other, which
makes rules easy to test in isolation.

### RuleTester for isolated testing

ESLint ships `RuleTester` so each rule has its own test file (`tests/lib/rules/<name>.js`).
Each test specifies valid and invalid code snippets plus expected messages, without booting
a full linter process. This one-rule-one-test-file pattern keeps test suites fast and
failures easy to locate.

### Plugin system for namespace isolation

Third-party rules are packaged as ESLint plugins. A plugin is an npm package that exports:

```js
module.exports = {
  rules: { 'my-rule': { meta: { ... }, create(ctx) { ... } } },
  configs: { recommended: { rules: { 'my-plugin/my-rule': 'error' } } },
};
```

In a flat config the consumer imports the plugin object directly — no global registry
pollution:

```js
import myPlugin from 'eslint-plugin-my-plugin';

export default [{ plugins: { my: myPlugin }, rules: { 'my/my-rule': 'error' } }];
```

The slash-namespace convention (`plugin-name/rule-name`) keeps third-party rules visually
distinct from core rules, which aids discoverability in large config files.

### Flat config as an ordered array of partial configs

ESLint v9 moved from `.eslintrc` to `eslint.config.js`, an ordered array of config objects.
Later entries override earlier ones for the same file glob. This replaces implicit
`extends` chains with explicit, predictable layering:

```js
export default [
  baseConfig, // layer 1: shared base
  { files: ['**/*.ts'], ...tsConfig }, // layer 2: TypeScript overrides
  eslintConfigPrettier, // layer 3: disable formatting rules
];
```

This is the same shape this project already uses in `eslint.config.cjs`.

### Category tagging instead of folder hierarchy

Because rules are flat in one directory, ESLint tags them by `meta.type` (problem /
suggestion / layout) and surfaces them by category on the docs site. The "recommended"
preset is nothing more than an object mapping `ruleName → severity` for every rule that
sets `meta.docs.recommended = true`. No folder structure is needed to express groupings.

### Docs-as-code

Every rule has a corresponding Markdown file in `docs/src/rules/<name>.md` generated (in
part) from `meta.docs`. The rule name is the canonical identifier across source, test, and
docs, so `grep no-unused-vars` surfaces everything at once.

---

## How We Could Apply the Same Ideas to This Project

Fitness checks already share the spirit of ESLint rules: each check has a name and a
`run` function. The following changes would bring the architecture closer to ESLint's
without discarding what works today.

### 1. Add a `meta` object to each check

Extend the `Check` type to carry metadata alongside `name` and `run`:

```ts
export type Check = {
  name: string;
  meta: {
    type: 'format' | 'content' | 'git' | 'config';
    docs: { description: string };
    fixable?: boolean; // true when the check can auto-repair (e.g. prettier --write)
  };
  run: (root?: string, context?: RunContext) => Promise<CheckResult>;
};
```

`type` would replace the informal categories implied by README prose. `fixable` would let
the runner display a "run with --fix" hint automatically rather than hard-coding it per
check.

### 2. One check, one folder, one README — already done

This project already follows the rule-per-folder pattern. Reinforcing it explicitly in
`src/checks/README.md` (e.g. "each folder is the canonical unit of a check") would make
the parallel to ESLint explicit for contributors.

### 3. A central registry mirroring ESLint's `lib/rules/index.js`

`src/checks/index.ts` already acts as the registry. It could be made more explicit by
exporting a `Map<string, Check>` alongside the ordered array:

```ts
export const checkRegistry = new Map<string, Check>(allChecks.map((c) => [c.name, c]));
```

Consumers (`.fitnessrc.ts`, the CLI) could then look checks up by name without array
scans, matching ESLint's `O(1)` rule lookup.

### 4. Plugin-style third-party checks

`.fitnessrc.ts` could accept a `plugins` array:

```ts
export default {
  plugins: [myOrgChecks], // each plugin is { checks: Check[] }
  checks: ['changelog', 'my-org/custom-check'],
};
```

The slash convention mirrors ESLint plugins: checks shipped with the runner have plain
names (`changelog`), third-party checks are namespaced (`my-org/custom-check`). This
avoids name collisions and signals provenance at a glance.

### 5. Layered config (flat-config-style)

Rather than a single `checks` array in `.fitnessrc.ts`, the config could be an ordered
array of partial configs just like ESLint's flat config:

```ts
export default [
  { checks: ['changelog', 'node-version', 'semantic-commit'] }, // base
  { files: ['src/**/*.ts'], checks: ['eslint', 'prettier'] }, // code only
];
```

This would allow teams to enable different checks for different file types or contexts
(CI vs. local dev) without maintaining separate configs.

### 6. RuleTester-equivalent for checks

A `CheckTester` utility (mirroring `RuleTester`) would make writing tests for new checks
easier:

```ts
const tester = new CheckTester(myCheck);

await tester.valid({ root: '/tmp/ok-repo' });
await tester.invalid({ root: '/tmp/bad-repo', errors: ['Expected ...'] });
```

This reduces boilerplate in test files and enforces a consistent testing contract across
all checks.

---

## Could We Adopt ESLint as the Engine?

This section evaluates replacing (or substantially delegating to) ESLint as the runtime
for fitness checks.

### What would it mean to use ESLint as the engine?

ESLint's engine reads source files, parses them into ASTs, and runs visitor functions
exported by rules. "Using ESLint as the engine" would mean expressing each fitness check
as an ESLint rule (or set of rules) and letting `eslint .` be the sole entry point.

### Where the mapping works well

| Fitness check             | ESLint mapping                                                                                         |
| ------------------------- | ------------------------------------------------------------------------------------------------------ |
| `eslint`                  | Already an ESLint pass-through; could be removed as a separate check                                   |
| `prettier`                | `eslint-config-prettier` + `@stylistic/eslint-plugin` covers formatting parity; `--fix` rewrites files |
| `sort-keys`               | Already an ESLint core rule, already in `eslint.config.cjs`                                            |
| `vitest-coverage-exclude` | A custom ESLint rule reading `vitest.config.cjs` AST                                                   |
| `rules-front-matter`      | A custom ESLint rule on `.md` files (requires a Markdown parser plugin)                                |

For purely code-quality concerns, ESLint is already the right tool, and some checks
(`eslint`, `prettier` formatting, `sort-keys`) are already expressed as ESLint rules in
this project's `eslint.config.cjs`.

### Where the mapping breaks down

ESLint rules operate on file ASTs. Several fitness checks have no meaningful AST target:

| Fitness check       | Why ESLint cannot express it                                                                                            |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| `changelog`         | Validates the existence and structure of `CHANGELOG.md` relative to dates; no single-file AST captures cross-file state |
| `changelog-updated` | Compares staged git diff content against `CHANGELOG.md` text; requires git subprocess                                   |
| `semantic-commit`   | Reads `git log` or the commit-msg hook file; no source file is being parsed                                             |
| `node-version`      | Compares `process.version` against `.nvmrc`; not a linting concern                                                      |
| `cspell`            | Runs a separate spell-checker process across the repo; ESLint cannot delegate to an external CLI per-run                |
| `read-repo-first`   | Interactive TTY prompt; ESLint has no concept of interactive I/O during a lint run                                      |

Attempting to map these into ESLint rules would require abusing ESLint's file-parsing
phase (e.g. writing a fake "rule" that runs a git subprocess and always reports zero
errors but has side effects). That conflicts with ESLint's design principle that rules are
stateless, deterministic visitors on AST nodes.

### Verdict: Hybrid approach

A full adoption of ESLint as the engine is not viable because roughly half of the current
checks are workflow-level (git state, dates, interactive prompts) rather than code-level
(AST analysis). ESLint is not designed for workflow concerns.

A hybrid approach is the right fit:

1. Keep the fitness runner as the top-level engine for workflow checks.
2. Delegate all code-style and structure checks to ESLint rules expressed in
   `eslint.config.cjs` and surfaced through the existing `eslint` fitness check.
3. Move checks like `prettier` formatting parity and `sort-keys` fully into
   `eslint.config.cjs` rather than maintaining parallel implementations.
4. Evaluate `@typescript-eslint` and `eslint-plugin-jsdoc` rules as replacements for any
   structural TypeScript checks that might be added in the future.

This mirrors how ESLint itself positions in a broader toolchain: it handles code analysis
while `git hooks`, `CI scripts`, and custom runners handle workflow-level gates.

### Summary table

| Concern                                                | Recommended engine                        |
| ------------------------------------------------------ | ----------------------------------------- |
| Code formatting (indentation, quotes, trailing commas) | ESLint / Prettier via `eslint.config.cjs` |
| Code structure (sort-keys, complexity, JSDoc)          | ESLint rules in `eslint.config.cjs`       |
| TypeScript type-level analysis                         | `@typescript-eslint` rules                |
| Changelog presence and freshness                       | Fitness runner check                      |
| Semantic commit messages                               | Fitness runner check                      |
| Node version alignment                                 | Fitness runner check                      |
| Spell checking                                         | Fitness runner check (cspell subprocess)  |
| Interactive onboarding prompt                          | Fitness runner check                      |
