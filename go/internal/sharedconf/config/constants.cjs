'use strict';
/**
 * Shared literals for the config files in this directory — CJS so both the .cjs and .mjs
 * configs can load it (require and import both work).
 */
/** Compilation target shared by the esbuild/oxc/tsc configs. */
exports.ES_TARGET = 'ES2022';
