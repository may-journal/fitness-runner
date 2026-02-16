'use strict';

// Exported config for consumers: "prettier": "@fitness/runner/prettier.config"
module.exports = {
  jsonRecursiveSort: true,
  plugins: ['prettier-plugin-sort-json'],
  printWidth: 100,
  semi: true,
  singleQuote: true,
  tabWidth: 2,
  trailingComma: 'es5',
};
