/** Optional context the runner passes to checks (e.g. staged file list, proposed commit message). */
export type RunContext = {
  stagedFiles?: string[];
  /** When set, semantic-commit validates this instead of HEAD (used by commit-msg hook). */
  proposedCommitMessage?: string;
  /** Names of all registered checks; used by rules-front-matter to allow check names in front matter. */
  registeredCheckNames?: string[];
  /** Test-only: override for child_process.execSync (cspell check). */
  _execSync?: (command: string, options: { encoding: 'utf8'; cwd: string; maxBuffer: number }) => string;
};
