const { readdirSync } = require('node:fs');
const { join } = require('node:path');
const { lint } = require('./tsconfig.compiler.cjs');
const { ignorePaths } = require('./cspell.json');

const checksDir = join(__dirname, '../../../packages/checks');
const checkNames = readdirSync(checksDir, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name);

const include = checkNames.map((name) => `../../../packages/checks/${name}/src/**/*.ts`);

/** Type-aware ESLint project covering all check packages (generated as tsconfig.checks.json). */
module.exports = {
  compilerOptions: lint,
  exclude: [...ignorePaths, '**/*.test.ts'],
  include,
};
