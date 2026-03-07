/** Optional context the runner passes to checks (e.g. staged file list, proposed commit message). */
export type RunContext = {
  /** Test-only: override check timeout ms so timeout tests don't wait 5s. */
  _checkTimeoutMsForTesting?: number;
  /** Test-only: override for child_process.execSync (cspell check). */
  _execSync?: (
    command: string,
    options: { cwd: string; encoding: 'utf8'; maxBuffer: number }
  ) => string;
  /** Test-only: override for fitness-runner package root (vitest-coverage-full check). */
  _fitnessRunnerRootForTesting?: string;
  /** Test-only: override for current time (changelog-updated check). */
  _now?: () => Date;
  /** Names of checks enabled for this run (from config or full registry). */
  enabledCheckNames?: string[];
  /** Args after the check name when running a single check (e.g. npx fitness prettier --write). */
  passthroughArgs?: string[];
  /** When set, checks (e.g. semantic-commit) can validate this instead of HEAD; runner may set via contextInline. */
  proposedCommitMessage?: string;
  /** Names of all registered checks; used by rules-front-matter to allow check names in front matter. */
  registeredCheckNames?: string[];
  stagedFiles?: string[];
};
