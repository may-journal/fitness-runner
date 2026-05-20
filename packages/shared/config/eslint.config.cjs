'use strict';
const { existsSync, readdirSync } = require('node:fs');
const { join } = require('node:path');
const jsdoc = require('eslint-plugin-jsdoc');
const tseslint = require('@typescript-eslint/eslint-plugin');
const tsParser = require('@typescript-eslint/parser');
const tsSortKeys = require('eslint-plugin-typescript-sort-keys');
const eslintConfigPrettier = require('eslint-config-prettier/flat');
const cspellConfig = require('./cspell.json');

const repoRoot = join(__dirname, '../../..');

const checkPackageTsconfigs = readdirSync(join(repoRoot, 'packages/checks'))
  .map((name) => join(repoRoot, 'packages/checks', name, 'tsconfig.json'))
  .filter(existsSync);

const lintProjects = [
  join(repoRoot, 'packages/shared/config/tsconfig.json'),
  join(repoRoot, 'packages/shared/tsconfig.json'),
  join(repoRoot, 'packages/checks-bundle/tsconfig.json'),
  ...checkPackageTsconfigs,
];

const rules = {
  complexity: ['error', { max: 5 }],
  'jsdoc/require-jsdoc': 'error',
  'max-lines': ['error', { max: 200, skipBlankLines: true, skipComments: true }],
  'sort-keys': ['error', 'asc', { caseSensitive: true, natural: true }],
  'typescript-sort-keys/interface': ['error', 'asc', { caseSensitive: true, natural: true }],
  'typescript-sort-keys/string-enum': ['error', 'asc', { caseSensitive: true, natural: true }],
};

const ignores = [...cspellConfig.ignorePaths, 'packages/shared/types/**', '**/*.d.ts'];

module.exports = [
  { ignores },
  {
    files: ['**/*.cjs', '**/*.js', '**/*.mjs'],
    languageOptions: {
      globals: {
        __dirname: 'readonly',
        __filename: 'readonly',
        module: 'readonly',
        require: 'readonly',
      },
      parser: tsParser,
      parserOptions: { ecmaVersion: 'latest', sourceType: 'script' },
    },
    rules: { 'sort-keys': ['error', 'asc', { caseSensitive: true, natural: true }] },
  },
  {
    files: ['**/*.ts'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: lintProjects,
        tsconfigRootDir: repoRoot,
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
      jsdoc,
      'typescript-sort-keys': tsSortKeys,
    },
    rules,
  },
  eslintConfigPrettier,
];
