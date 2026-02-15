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
  {
    ignores: [
      'dist/**',
      'coverage/**',
      'node_modules/**',
    ],
  },
  {
    files: [
      'src/**/*.ts',
      'tests/**/*.ts',
      '*.ts',
    ],
    languageOptions,
    plugins: {
      '@typescript-eslint': tseslint,
      '@stylistic': stylistic,
      jsdoc,
    },
    rules: {
      '@stylistic/indent': ['error',
        2],
      '@stylistic/quotes': ['error',
        'single'],
      '@stylistic/semi': ['error',
        'always'],
      '@stylistic/comma-dangle': ['error',
        'always-multiline'],
      '@stylistic/comma-style': ['error',
        'last'],
      '@stylistic/array-element-newline': ['error',
        { minItems: 2 }],
      '@stylistic/object-curly-spacing': ['error',
        'always'],
      '@stylistic/brace-style': ['error',
        '1tbs'],
      '@stylistic/no-trailing-spaces': 'error',
      '@stylistic/no-multiple-empty-lines': ['error',
        { max: 1, maxEOF: 0 }],
      'jsdoc/require-jsdoc': 'error',
      complexity: ['error',
        { max: 5 }],
    },
  },
];
