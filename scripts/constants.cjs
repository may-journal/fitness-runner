'use strict';
const { join } = require('node:path');

/**
 * Shared constants for the repo automation scripts (scripts/*, and the checks-bundle build
 * script). These are plain-JS islands that cannot import the TS `fitness-shared` constants
 * module — they run before anything is built — so their shared literals live once, here.
 * Individual `exports.` assignments keep the named exports statically visible to ESM importers.
 */
exports.CHANGELOG_MD = 'CHANGELOG.md';
exports.FITNESS_PKG = '@mayjournal/fitness';
exports.FITNESS_SHARED_PKG = '@mayjournal/fitness-shared';
/** The `--json` output flag shared by npm and this repo's own script CLIs. */
exports.JSON_FLAG = '--json';
exports.NODE_MODULES = 'node_modules';
exports.NPM = 'npm';
exports.NPMRC = '.npmrc';
exports.NPM_SCRIPT_BUILD = 'build';
exports.PACKAGE_JSON = 'package.json';
/** The `run` subcommand shared by npm and publint invocations. */
exports.RUN_SUBCOMMAND = 'run';
/** Absolute repo root — scripts use this instead of each re-deriving it from `__dirname`. */
exports.REPO_ROOT = join(__dirname, '..');
exports.RUNNER_DIR = 'packages/runner';
