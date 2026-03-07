import { describe, it, expect } from 'vitest';
import { checkResult } from './checkResult.js';

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
