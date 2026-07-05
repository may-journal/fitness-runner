import jsdoc from 'eslint-plugin-jsdoc';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import perfectionist from 'eslint-plugin-perfectionist';
import eslintConfigPrettier from 'eslint-config-prettier/flat';
import cspellConfig from './cspell.json' with { type: 'json' };

const sortKeys = ['error', 'asc', { caseSensitive: true, natural: true }];
// perfectionist replaces the unmaintained eslint-plugin-typescript-sort-keys (no eslint 10
// support): natural ascending, case-sensitive, matching the `sort-keys` object rule above.
const sortMembers = ['error', { ignoreCase: false, order: 'asc', type: 'natural' }];

const rules = {
  complexity: ['error', { max: 5 }],
  'jsdoc/require-jsdoc': 'error',
  'max-lines': ['error', { max: 200, skipBlankLines: true, skipComments: true }],
  'perfectionist/sort-enums': sortMembers,
  'perfectionist/sort-interfaces': sortMembers,
  'sort-keys': sortKeys,
};

const ignores = [...cspellConfig.ignorePaths, 'packages/shared/types/**', '**/*.d.ts'];

/** Shared ESLint flat config; pass TypeScript parserOptions per caller (CLI vs fitness check). */
export function createEslintConfig(tsParserOptions) {
  return [
    { ignores },
    {
      files: ['**/*.cjs'],
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
      rules: { 'sort-keys': sortKeys },
    },
    {
      files: ['**/*.js', '**/*.mjs'],
      languageOptions: {
        parser: tsParser,
        parserOptions: { ecmaVersion: 'latest', sourceType: 'module' },
      },
      rules: { 'sort-keys': sortKeys },
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
