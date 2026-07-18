'use strict';
/**
 * Shared literals for the config files in this directory — CJS so both the .cjs and .mjs
 * configs can load it (require and import both work). The TS sources have their own twin
 * (`src/testGlobs.ts`); the two runtimes cannot share one module.
 */
/** Declaration-file glob, excluded from lint and coverage. */
exports.DTS_GLOB = '**/*.d.ts';
/** Compilation target shared by the esbuild/oxc/tsc configs. */
exports.ES_TARGET = 'ES2022';
/** Spec-file glob, excluded from lint type-check and coverage. */
exports.SPEC_GLOB = '**/*.spec.ts';
/** Test-file glob, excluded from lint type-check and coverage. */
exports.TEST_GLOB = '**/*.test.ts';
/** Type-only-file glob, excluded from coverage. */
exports.TYPES_GLOB = '**/*.types.ts';
