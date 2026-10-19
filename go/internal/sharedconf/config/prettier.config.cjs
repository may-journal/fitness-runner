'use strict';

const { createRequire } = require('node:module');
const path = require('node:path');
const requireFromFile = createRequire(__filename);

/** Returns glob, a path from the repo root, as a path from this file's folder. */
const fromRepoRoot = (glob) =>
  path.posix.join(path.relative(__dirname, process.cwd()).split(path.sep).join('/'), glob);

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
  // may-journal/.github syncs these files into every repo, formatted with
  // Prettier's defaults; keep them at defaults so the same bytes pass here.
  // Prettier matches override globs relative to this file's folder, which is
  // a cache folder outside the repo, so the globs start at the repo root that
  // Prettier runs in.
  overrides: [
    {
      files: ['.github/ISSUE_TEMPLATE/**/*.md', '.github/fitness-setup.md'].map(fromRepoRoot),
      options: { printWidth: 80, singleQuote: false },
    },
  ],
};

module.exports = config;
