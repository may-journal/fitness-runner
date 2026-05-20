import { describe, it, expect } from 'vitest';
import { buildExecCheckResult, checkResult } from '@mayjournal/fitness-shared';

describe('checkResult', () => {
  it('defaults errors to [] and filesChecked to 0 when omitted', () => {
    const r = checkResult(true);
    expect(r.ok).toBe(true);
    expect(r.errors).toEqual([]);
    expect(r.meta?.filesChecked).toBe(0);
  });

  it('uses provided errors and filesChecked when passed', () => {
    const r = checkResult(false, ['err1'], 2);
    expect(r.ok).toBe(false);
    expect(r.errors).toEqual(['err1']);
    expect(r.meta?.filesChecked).toBe(2);
  });
});

describe('buildExecCheckResult', () => {
  const fallback = 'Run the linter to fix.';
  it('returns ok when exitCode 0 and no errors', () => {
    const r = buildExecCheckResult(0, [], 3, fallback);
    expect(r.ok).toBe(true);
    expect(r.errors).toEqual([]);
    expect(r.meta?.filesChecked).toBe(3);
  });
  it('returns errors when exitCode 0 but parsed errors exist', () => {
    const r = buildExecCheckResult(0, ['file:1:2 - msg'], 1, fallback);
    expect(r.ok).toBe(false);
    expect(r.errors).toEqual(['file:1:2 - msg']);
  });
  it('returns fallback when non-zero exit and no parsed errors', () => {
    const r = buildExecCheckResult(1, [], 0, fallback);
    expect(r.ok).toBe(false);
    expect(r.errors).toEqual([fallback]);
    expect(r.meta?.filesChecked).toBe(0);
  });
  it('returns parsed errors when non-zero exit and errors exist', () => {
    const r = buildExecCheckResult(1, ['a:1 - err'], 1, fallback);
    expect(r.ok).toBe(false);
    expect(r.errors).toEqual(['a:1 - err']);
  });
});
