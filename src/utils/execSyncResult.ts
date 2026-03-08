import type { ExecSyncFn } from './runContext.js';

/** Shared options for CLI exec calls (encoding, maxBuffer). */
export const EXEC_OPTS = { encoding: 'utf8' as const, maxBuffer: 1024 * 1024 };

/** Runs command in root with execSyncFn; returns exitCode and combined stdout+stderr. */
export function execSyncResult(
  root: string,
  command: string,
  execSyncFn: ExecSyncFn
): { exitCode: number; output: string } {
  try {
    const out = execSyncFn(command, { ...EXEC_OPTS, cwd: root });
    return { exitCode: 0, output: out };
  } catch (e: unknown) {
    const err = e as { status?: number; stderr?: string; stdout?: string };
    const output = [err.stdout, err.stderr].filter(Boolean).join('\n');
    const exitCode = typeof err.status === 'number' ? err.status : 1;
    return { exitCode, output };
  }
}
