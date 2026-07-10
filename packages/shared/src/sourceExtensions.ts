/** Source-file extension sets shared by the file-scanning checks — each extension literal lives once. */

/** TypeScript-only source extensions (build-output-untracked's import scan). */
export const TS_FILE_EXTENSIONS = ['.cts', '.mts', '.ts'];

/** Every JS/TS source extension the lexer-based checks scan (no-eslint-disable, repeated-string-literals). */
export const SOURCE_FILE_EXTENSIONS = [...TS_FILE_EXTENSIONS, '.cjs', '.js', '.mjs', '.tsx'];
