'use strict';

/** @type {import('prettier').Config} */
// package.json sorted via sort-package-json (conventional order); package-lock.json in .prettierignore.
const config = {
  jsonRecursiveSort: true,
  plugins: ['prettier-plugin-packagejson', 'prettier-plugin-sort-json'],
  printWidth: 100,
  semi: true,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
};

module.exports = config;
