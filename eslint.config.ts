import type { Linter } from 'eslint';
import stylistic from '@stylistic/eslint-plugin';
import jsdoc from 'eslint-plugin-jsdoc';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';

const languageOptions: Linter.Config['languageOptions'] = {
  parser: tsParser,
  parserOptions: { project: './tsconfig.eslint.json' },
};

export default [
  { ignores: ['dist/**',
'coverage/**',
'node_modules/**'] },
  {
    files: ['src/**/*.ts',
'tests/**/*.ts',
'*.ts'],
    languageOptions,
    plugins: {
      '@typescript-eslint': tseslint,
      '@stylistic': stylistic,
      jsdoc,
    },
    rules: {
      '@stylistic/array-element-newline': ['error',
{ minItems: 2 }],
      'jsdoc/require-jsdoc': 'error',
      complexity: ['error',
{ max: 5 }],
    },
  },
];
