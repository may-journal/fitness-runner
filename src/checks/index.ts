import { changelogCheck } from './changelog/index.js';
import { changelogUpdatedCheck } from './changelog-updated/index.js';
import { cspellCheck } from './cspell/index.js';
import { eslintCheck } from './eslint/index.js';
import { markdownNoBoldItalicCheck } from './markdown-no-bold-italic/index.js';
import { nodeVersionCheck } from './node-version/index.js';
import { readRepoFirstCheck } from './read-repo-first/index.js';
import { rulesFrontMatterCheck } from './rules-front-matter/index.js';
import { semanticCheck } from './semantic-commit/index.js';
import { vitestCoverageExcludeCheck } from './vitest-coverage-exclude/index.js';
import type { Check } from '../types/index.types.js';

export const registry: Check[] = [
  readRepoFirstCheck,
  changelogCheck,
  changelogUpdatedCheck,
  cspellCheck,
  eslintCheck,
  markdownNoBoldItalicCheck,
  nodeVersionCheck,
  rulesFrontMatterCheck,
  semanticCheck,
  vitestCoverageExcludeCheck,
];
