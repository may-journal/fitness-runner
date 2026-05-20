import { describe, it, expect } from 'vitest';
import { quoteForShell } from '@mayjournal/fitness-shared';

describe('quoteForShell', () => {
  it('wraps string in double quotes', () => {
    expect(quoteForShell('foo')).toBe('"foo"');
  });
  it('escapes internal double quotes', () => {
    expect(quoteForShell('a"b')).toBe('"a\\"b"');
  });
  it('handles empty string', () => {
    expect(quoteForShell('')).toBe('""');
  });
});
