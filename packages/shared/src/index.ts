export { buildExecCheckResult, checkResult } from './checkResult.js';
export { loadConfig } from './config/load.js';
export { EXEC_OPTS, execSyncResult } from './execSyncResult.js';
export { findFilesByExtension } from './findFilesByExtension.js';
export { getFitnessRunnerRoot } from './getFitnessRunnerRoot.js';
export {
  extractDiagramNumbers,
  isCalloutHeader,
  pairDiagramsWithTables,
  parseDoc,
} from './mermaid.js';
export type { CalloutTableBlock, DiagramBlock, DiagramTablePair, DocBlock } from './mermaid.js';
export { getSkipDirs, getSkipDirsForWalk, RUNNER_SKIP_DIRS } from './getSkipDirs.js';
export { resolveFitnessConfigPath } from './resolveFitnessConfigPath.js';
export { resolveLintTsconfig } from './resolveLintTsconfig.js';
export { runMermaidDocCheck } from './runMermaidDocCheck.js';
export {
  ALLOWED_MARKDOWN_BASENAMES,
  runMarkdownFilenameCheck,
  validateMarkdownFilename,
} from './runMarkdownFilenameCheck.js';
export type { FilenameConvention } from './runMarkdownFilenameCheck.js';
export { getExecSync, getStagedFiles } from './runContext.js';
export type { ExecSyncFn } from './runContext.js';
export { quoteForShell } from './shellQuote.js';
export { getColumns } from './terminal.js';
export {
  configFromMod,
  getCoverageBlock,
  getCoverageExcludeFromConfig,
  getThresholdsFromConfig,
  isObject,
  loadVitestConfig,
  loadVitestConfigFromRoot,
  tryLoadConfigFile,
  tryLoadPackageJsonVitest,
  VITEST_CONFIG_NAMES,
} from './vitest-config/index.js';
export type { VitestConfigRaw } from './vitest-config/index.js';
export type { CheckResult } from './types/check-result.types.js';
export type { FitnessConfig } from './types/fitness-config.types.js';
export type { RunContext } from './types/run-context.types.js';
