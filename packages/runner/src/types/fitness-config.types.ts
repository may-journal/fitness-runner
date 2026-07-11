import type { CheckName } from './check-name.js';

/** Shape of .fitnessrc.ts / .fitnessrc.js. */
export type FitnessConfig = {
  /** Check names to run, in order. If omitted, `defaultChecks` from `@mayjournal/fitness-checks` is used. */
  checks?: CheckName[];
  /** Check names to exclude from the resolved list (from `checks` or bundle `defaultChecks`). */
  disabledChecks?: CheckName[];
  /** Options for the `repeated-string-literals` check. */
  repeatedStringLiterals?: {
    /** Exact string values never flagged — a project baseline complementing the built-in idiomatic set. */
    allow?: string[];
  };
  /** Dir names to skip when walking for files. If omitted, cspell.json ignorePaths (dir names only) are used. */
  skipTheseDirectories?: string[];
};
