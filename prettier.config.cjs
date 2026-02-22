'use strict';

/** @type {import('prettier').Config} */
// Prettier uses a different parser for package.json by default, so the sort-json plugin never runs on it.
const config = {
  jsonRecursiveSort: true,
  overrides: [
    {
      files: 'package.json',
      options: {
        jsonRecursiveSort: true, // Sorts all keys recursively, including exports paths and condition keys.
        parser: 'json',
        plugins: ['prettier-plugin-sort-json'],
      },
    },
  ],
  plugins: ['prettier-plugin-sort-json'],
  printWidth: 100,
  semi: true,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
};

module.exports = config;
