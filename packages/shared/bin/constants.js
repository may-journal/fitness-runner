/**
 * Shared literals for the fitness-shared bin scripts and config files. These run before any
 * build output exists, so they cannot import the TS constants modules in src/ — their shared
 * literals live once, here.
 */

/** The runner package name (`@mayjournal/fitness`). */
export const FITNESS_PKG = '@mayjournal/fitness';
/** The package manifest filename. */
export const PACKAGE_JSON = 'package.json';
/** Repo-relative runner workspace directory. */
export const RUNNER_DIR = 'packages/runner';
/** Repo-relative shared-config directory. */
export const SHARED_CONFIG_DIR = 'packages/shared/config';
/** The vitest CLI run subcommand. */
export const VITEST_RUN = 'run';
