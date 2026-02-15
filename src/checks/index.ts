import { changelogCheck } from './changelog.js';
import { semanticCheck } from './semantic-commit.js';
import type { Check } from '../types/index.js';

export const registry: Check[] = [changelogCheck, semanticCheck];
