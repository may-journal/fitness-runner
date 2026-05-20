'use strict';

const { createRequire } = require('node:module');
const requireFromFile = createRequire(__filename);

/** @type {import('prettier').Config} */
// package.json sorted via sort-package-json (conventional order); package-lock.json in .prettierignore.
const config = {
  jsonRecursiveSort: true,
  plugins: [
    requireFromFile.resolve('prettier-plugin-packagejson'),
    requireFromFile.resolve('prettier-plugin-sort-json'),
  ],
  printWidth: 100,
  semi: true,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
};

module.exports = config;
