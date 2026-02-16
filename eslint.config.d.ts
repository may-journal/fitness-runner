/** @see https://eslint.org/docs/latest/use/configure/configuration-files-eslintrc */
declare module '@fitness/runner/eslint.config' {
  import type { Linter } from 'eslint';
  const config: Linter.Config[];
  export default config;
}
