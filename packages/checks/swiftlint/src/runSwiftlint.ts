import {
  buildExecCheckResult,
  checkResult,
  execSyncResult,
  getExecSync,
} from '@mayjournal/fitness-shared';
import type { ExecSyncFn } from '@mayjournal/fitness-shared';
import type { Check, CheckName, RunContext } from '@mayjournal/fitness';
import { enUS } from './enUS.js';

type SwiftlintViolation = {
  character?: number;
  file: string;
  line: number;
  reason: string;
  rule_id: string;
};

const COMMAND_NOT_FOUND_RE = /command not found/;
const NO_LINTABLE_FILES_RE = /No lintable files found/;

/** Runs swiftlint's JSON reporter; returns exit code and combined stdout+stderr. */
function execSwiftlint(root: string, execSyncFn: ExecSyncFn): { exitCode: number; output: string } {
  return execSyncResult(root, 'swiftlint lint --strict --reporter json --quiet . 2>&1', execSyncFn);
}

/** Formats one violation as file:line[:col] - reason (rule_id). */
function formatViolation(v: SwiftlintViolation): string {
  const col = v.character != null ? `:${v.character}` : '';
  return `${v.file}:${v.line}${col} - ${v.reason} (${v.rule_id})`;
}

/** Parses the JSON reporter's violation array; malformed output yields no parsed errors. */
function parseViolations(output: string): string[] {
  try {
    const parsed = JSON.parse(output) as SwiftlintViolation[];
    return parsed.map(formatViolation);
  } catch {
    return [];
  }
}

/** SwiftLint check: shells out to a brew-installed binary — no npm dependency, no library fallback. */
export const swiftlintCheck: Check = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'swiftlint' as CheckName,
  async run(root = process.cwd(), context?: RunContext) {
    const execSyncFn = getExecSync(context);
    const { exitCode, output } = execSwiftlint(root, execSyncFn);
    if (COMMAND_NOT_FOUND_RE.test(output)) return checkResult(false, [enUS.NotInstalled]);
    if (NO_LINTABLE_FILES_RE.test(output)) return checkResult(true, [], 0);
    return buildExecCheckResult(exitCode, parseViolations(output), 0, enUS.FallbackRunHint);
  },
};
