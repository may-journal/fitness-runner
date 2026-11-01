'use strict';

const { createRequire } = require('node:module');
const path = require('node:path');
const requireFromFile = createRequire(__filename);

/** Escapes glob syntax, so a folder named like `app (copy)` matches itself. */
const literal = (text) => text.replace(/[()[\]{}*?!+@\\]/g, '\\$&');

/** Returns glob, a path from the repo root, as a path from this file's folder. */
const fromRepoRoot = (glob) =>
  path.posix.join(literal(path.relative(__dirname, process.cwd()).split(path.sep).join('/')), glob);

/** @type {import('prettier').Config} */
// package.json sorted via sort-package-json (conventional order); JSON keys sorted recursively.
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
  // Prettier matches override globs relative to this file's cache folder, so
  // the globs start at the repo root.
  overrides: [
    {
      files: ['.github/ISSUE_TEMPLATE/**/*.md', '.github/fitness-setup.md'].map(fromRepoRoot),
      options: { jsonRecursiveSort: false, printWidth: 80, singleQuote: false, trailingComma: 'all' },
    },
  ],
};

module.exports = config;
