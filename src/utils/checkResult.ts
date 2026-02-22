import type { CheckResult } from '../types/check-result.types.js';

/** Build a CheckResult with normalized shape; defaults errors to [], filesChecked to 0. */
export function checkResult(ok: boolean, errors?: string[], filesChecked?: number): CheckResult {
  return {
    errors: errors ?? [],
    meta: { filesChecked: filesChecked ?? 0 },
    ok,
  };
}
