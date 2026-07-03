/** Shape of .fitnessrc.ts / .fitnessrc.js. */
export type FitnessConfig = {
  /** Check names and/or local module paths to run, in order. Path entries (containing `/` or `\`, or ending in `.js`/`.mjs`/`.cjs`/`.ts`) load a local `Check` module instead of an npm check package. If omitted, `defaultChecks` from `@mayjournal/fitness-checks` is used. */
  checks?: string[];
  /** Check names to exclude from the resolved list (from `checks` or bundle `defaultChecks`). Path entries in `checks` are opt-in only and are never removed by this list. */
  disabledChecks?: string[];
  /** Dir names to skip when walking for files. If omitted, cspell.json ignorePaths (dir names only) are used. */
  skipTheseDirectories?: string[];
};
