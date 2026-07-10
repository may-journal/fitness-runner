import { buildExecCheckResult, execSyncResult, getExecSync } from '@mayjournal/fitness-shared';
import type { ExecSyncFn } from '@mayjournal/fitness-shared';
import { CheckName, type Check, type RunContext } from '@mayjournal/fitness';
import { enUS } from './enUS.js';

/** Default flags: proven thresholds from consumer repos, scan-from-root so no per-repo path list is needed. Respects .gitignore by default (no flag needed). Ignores lockfiles/markdown/JSON and test/spec files — repeated mock setup and fixtures there read as false-positive duplication, not production code to refactor. */
const JSCPD_FLAGS =
  '--min-lines 5 --min-tokens 50 --threshold 1 --ignore "**/*.md,**/*.json,**/*.lock,**/*.test.*,**/*.spec.*" --reporters console .';

const ANSI_RE = /\x1b\[[0-9;]*m/g;
const ERROR_LINE_RE = /^ERROR: .+$/m;
const TOTAL_ROW_RE = /Total:\s*\S*\s*(\d+)/;

/** Strips ANSI color codes jscpd emits even when stdout is piped (not a TTY). */
function stripAnsi(output: string): string {
  return output.replace(ANSI_RE, '');
}

/** Runs npx jscpd with the shared default flags; returns exit code and combined stdout+stderr. */
function execJscpd(root: string, execSyncFn: ExecSyncFn): { exitCode: number; output: string } {
  return execSyncResult(root, `npx jscpd ${JSCPD_FLAGS} 2>&1`, execSyncFn);
}

/** Extracts jscpd's "ERROR: jscpd found too many duplicates (X%) over threshold (Y%)" line, if present. */
function parseIssues(output: string): string[] {
  const clean = stripAnsi(output);
  const match = clean.match(ERROR_LINE_RE);
  return match ? [match[0]] : [];
}

/** Extracts the "Total:" row's files-analyzed count from the console reporter's summary table. */
function parseFilesChecked(output: string): number {
  const clean = stripAnsi(output);
  const match = clean.match(TOTAL_ROW_RE);
  return match ? parseInt(match[1], 10) : 0;
}

/** jscpd check: fails when duplicate lines exceed the configured threshold. */
export const jscpdCheck: Check = {
  name: CheckName.Jscpd,
  async run(root = process.cwd(), context?: RunContext) {
    const execSyncFn = getExecSync(context);
    const { exitCode, output } = execJscpd(root, execSyncFn);
    return buildExecCheckResult(
      exitCode,
      parseIssues(output),
      parseFilesChecked(output),
      enUS.FallbackRunHint
    );
  },
};
