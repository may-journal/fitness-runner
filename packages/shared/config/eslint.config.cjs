'use strict';
const { createEslintConfig } = require('./eslint.base.cjs');

const tsconfigRootDir = process.env.FITNESS_TSCONFIG_ROOT;

module.exports = createEslintConfig({
  projectService: true,
  ...(tsconfigRootDir && { tsconfigRootDir }),
});
