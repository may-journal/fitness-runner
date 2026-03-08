import type { CheckResult } from '../types/check-result.types.js';

/** Build a CheckResult with normalized shape; defaults errors to [], filesChecked to 0. */
export function checkResult(ok: boolean, errors?: string[], filesChecked?: number): CheckResult {
  return {
    errors: errors ?? [],
    meta: { filesChecked: filesChecked ?? 0 },
    ok,
  };
}

/** Build CheckResult for exec-based checks: ok when exitCode 0 and no errors; fallback message when non-zero but no parsed errors. */
export function buildExecCheckResult(
  exitCode: number,
  errors: string[],
  filesChecked: number,
  fallbackMessage: string
): CheckResult {
  const ok = exitCode === 0 && errors.length === 0;
  const fallback = !ok && errors.length === 0 ? [fallbackMessage] : [];
  return checkResult(ok, errors.length > 0 ? errors : fallback, filesChecked);
}
