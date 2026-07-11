/**
 * Globs for test/spec and type-only sources — excluded from lint type-check and coverage.
 * Twin of `config/constants.cjs` (the plain-JS configs cannot import this TS module).
 */

/** Declaration-file glob. */
export const DTS_GLOB = '**/*.d.ts';
/** Spec-file glob. */
export const SPEC_GLOB = '**/*.spec.ts';
/** Test-file glob. */
export const TEST_GLOB = '**/*.test.ts';
/** Type-only-file glob. */
export const TYPES_GLOB = '**/*.types.ts';
