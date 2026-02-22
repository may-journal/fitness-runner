'use strict';

/** @type {import('prettier').Config} */
const config = {
  jsonRecursiveSort: true,
  plugins: ['prettier-plugin-sort-json'],
  printWidth: 100,
  semi: true,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
};

module.exports = config;
