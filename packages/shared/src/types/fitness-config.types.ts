/** Shape of .fitnessrc.ts / .fitnessrc.js. */
export type FitnessConfig = {
  /** Check names to run, in order. If omitted, all checks run (minus any in disabledChecks). */
  checks?: string[];
  /** Check names to skip when not using a custom checks list. Ignored if checks is set. */
  disabledChecks?: string[];
  /** Dir names to skip when walking for files. If omitted, cspell.json ignorePaths (dir names only) are used. */
  skipTheseDirectories?: string[];
};
