/** Optional context passed when running with --staged (e.g. pre-commit) or --validate-commit-msg (commit-msg hook). */
export type RunContext = {
  stagedFiles?: string[];
  /** When set, semantic-commit validates this instead of HEAD (used by commit-msg hook). */
  proposedCommitMessage?: string;
};
