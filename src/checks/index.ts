import { changelogCheck } from './changelog/index.js';
import { changelogUpdatedCheck } from './changelog-updated/index.js';
import { cspellCheck } from './cspell/index.js';
import { nodeVersionCheck } from './node-version/index.js';
import { semanticCheck } from './semantic-commit/index.js';
import type { Check } from '../types/index.js';

export const registry: Check[] = [
  changelogCheck,
  changelogUpdatedCheck,
  cspellCheck,
  nodeVersionCheck,
  semanticCheck,
];
