import type { CheckName } from './check-name.js';

/** Shape of .fitnessrc.ts / .fitnessrc.js. */
export type FitnessConfig = {
  /** Check names to run, in order. If omitted, `defaultChecks` from `@mayjournal/fitness-checks` is used. */
  checks?: CheckName[];
  /** Check names to exclude from the resolved list (from `checks` or bundle `defaultChecks`). */
  disabledChecks?: CheckName[];
  /** Dir names to skip when walking for files. If omitted, cspell.json ignorePaths (dir names only) are used. */
  skipTheseDirectories?: string[];
};
