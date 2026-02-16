import type { Linter } from 'eslint';
import jsdoc from 'eslint-plugin-jsdoc';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';

const rules: Linter.Config['rules'] = {
  'jsdoc/require-jsdoc': 'error',
  complexity: ['error', { max: 5 }],
};

export default [
  { ignores: ['dist/**', 'coverage/**', 'node_modules/**'] },
  {
    files: ['**/*.ts'],
    languageOptions: {
      parser: tsParser,
      parserOptions: { project: './tsconfig.json' },
    },
    plugins: {
      '@typescript-eslint': tseslint,
      jsdoc,
    },
    rules,
  },
];
