import { describe, it, expect } from 'vitest';
import { interpolate } from './interpolate.js';

describe('interpolate', () => {
  it('replaces all placeholders with values', () => {
    expect(interpolate('a {{x}} b {{y}}', { x: 1, y: 'two' })).toBe('a 1 b two');
  });

  it('replaces missing key with empty string', () => {
    expect(interpolate('{{a}}{{b}}', { a: 'x' })).toBe('x');
  });
});
