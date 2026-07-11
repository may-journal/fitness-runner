/**
 * markdown filename convention check — two flavors of one shared fitness function
 * (`runMarkdownFilenameCheck` in `@mayjournal/fitness-shared`), exposed from this one
 * package under two check names. The bundler maps each named export's entry module to a
 * `@mayjournal/fitness-checks/checks/<name>` subpath, so `.fitnessrc` can enable either.
 */
export { default as markdownFilenameKebabCaseCheck, KEBAB_RE } from './kebab-case.js';
export { default as markdownFilenameCamelCaseCheck, CAMEL_RE } from './camel-case.js';
