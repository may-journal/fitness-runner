'use strict';
const jsdoc = require('eslint-plugin-jsdoc');
const tseslint = require('@typescript-eslint/eslint-plugin');
const tsParser = require('@typescript-eslint/parser');
const eslintConfigPrettier = require('eslint-config-prettier/flat');
const cspellConfig = require('./cspell.json');

const rules = {
  'jsdoc/require-jsdoc': 'error',
  complexity: ['error', { max: 5 }],
};

const ignores = cspellConfig.ignorePaths;

module.exports = [
  { ignores },
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
  eslintConfigPrettier,
];
