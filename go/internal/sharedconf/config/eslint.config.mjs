import { createEslintConfig } from './eslint.base.mjs';

// Syntactic parse only — no enabled rule is type-aware (see eslint.base.mjs), so we skip building a
// TypeScript program (`projectService`). That program cost ~5s over the whole repo for no findings.
export default createEslintConfig();
