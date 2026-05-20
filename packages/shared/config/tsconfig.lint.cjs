const { lint } = require('./tsconfig.compiler.cjs');

/** Compiler defaults for ESLint type-aware linting in consumer projects (used via temp tsconfig). */
module.exports = { compilerOptions: lint };
