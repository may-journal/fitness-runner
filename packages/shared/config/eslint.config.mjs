import { createEslintConfig } from './eslint.base.mjs';

const tsconfigRootDir = process.env.FITNESS_TSCONFIG_ROOT;

export default createEslintConfig({
  projectService: true,
  ...(tsconfigRootDir && { tsconfigRootDir }),
});
