/** Shape of .fitnessrc.ts / .fitnessrc.js. */
export type FitnessConfig = {
  /** Check names to run, in order. If omitted, `defaultChecks` from `@mayjournal/fitness-checks` is used. */
  checks?: string[];
  /** Dir names to skip when walking for files. If omitted, cspell.json ignorePaths (dir names only) are used. */
  skipTheseDirectories?: string[];
};
