import jsdoc from 'eslint-plugin-jsdoc';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import perfectionist from 'eslint-plugin-perfectionist';
import eslintConfigPrettier from 'eslint-config-prettier/flat';
import cspellConfig from './cspell.json' with { type: 'json' };
import { DTS_GLOB } from './constants.cjs';

const readonlyGlobal = 'readonly';
/** ESLint severity used for every enabled rule. */
const ERROR = 'error';
/** Rule id shared by the three sort-keys rule blocks below. */
const SORT_KEYS = 'sort-keys';

const sortKeys = [ERROR, 'asc', { caseSensitive: true, natural: true }];
// perfectionist replaces the unmaintained eslint-plugin-typescript-sort-keys (no eslint 10
// support): natural ascending, case-sensitive, matching the `sort-keys` object rule above.
const sortMembers = [ERROR, { ignoreCase: false, order: 'asc', type: 'natural' }];

const rules = {
  [SORT_KEYS]: sortKeys,
  complexity: [ERROR, { max: 5 }],
  'jsdoc/require-jsdoc': ERROR,
  'max-lines': [ERROR, { max: 200, skipBlankLines: true, skipComments: true }],
  'perfectionist/sort-enums': sortMembers,
  'perfectionist/sort-interfaces': sortMembers,
};

const ignores = [...cspellConfig.ignorePaths, 'packages/shared/types/**', DTS_GLOB];

/** Shared ESLint flat config; pass TypeScript parserOptions per caller (CLI vs fitness check). */
export function createEslintConfig(tsParserOptions) {
  return [
    { ignores },
    {
      files: ['**/*.cjs'],
      languageOptions: {
        globals: {
          __dirname: readonlyGlobal,
          __filename: readonlyGlobal,
          module: readonlyGlobal,
          require: readonlyGlobal,
        },
        parser: tsParser,
        parserOptions: { ecmaVersion: 'latest', sourceType: 'script' },
      },
      rules: { [SORT_KEYS]: sortKeys },
    },
    {
      files: ['**/*.js', '**/*.mjs'],
      languageOptions: {
        parser: tsParser,
        parserOptions: { ecmaVersion: 'latest', sourceType: 'module' },
      },
      rules: { [SORT_KEYS]: sortKeys },
    },
    {
      files: ['**/*.ts'],
      languageOptions: {
        parser: tsParser,
        parserOptions: tsParserOptions,
      },
      plugins: {
        '@typescript-eslint': tseslint,
        jsdoc,
        perfectionist,
      },
      rules,
    },
    eslintConfigPrettier,
  ];
}
