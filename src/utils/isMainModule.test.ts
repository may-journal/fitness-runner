import { afterEach, beforeEach, describe, it, expect } from 'vitest';
import { isMainModule } from './isMainModule.js';

describe('isMainModule', () => {
  let argv: string[];

  beforeEach(() => {
    argv = process.argv.slice();
  });

  afterEach(() => {
    process.argv = argv;
  });

  it('returns true when argv[1] equals module path exactly', () => {
    const moduleUrl = 'file:///tmp/entry.js';
    process.argv = ['node', '/tmp/entry.js'];
    expect(isMainModule(moduleUrl)).toBe(true);
  });

  it('returns true when argv[1] omits extension (node foo → foo.js)', () => {
    const moduleUrl = 'file:///tmp/entry.js';
    process.argv = ['node', '/tmp/entry'];
    expect(isMainModule(moduleUrl)).toBe(true);
  });

  it('returns true when argv[1] omits .mjs extension', () => {
    const moduleUrl = 'file:///app/cli.mjs';
    process.argv = ['node', '/app/cli'];
    expect(isMainModule(moduleUrl)).toBe(true);
  });

  it('returns false when argv[1] is a different path (e.g. test runner)', () => {
    const moduleUrl = 'file:///project/src/index.js';
    process.argv = ['node', '/project/node_modules/vitest/run.js'];
    expect(isMainModule(moduleUrl)).toBe(false);
  });

  it('returns false when argv[1] is missing', () => {
    const moduleUrl = 'file:///tmp/entry.js';
    process.argv = ['node'];
    expect(isMainModule(moduleUrl)).toBe(false);
  });

  it('returns false when argv path is only a prefix (suffix not a file extension)', () => {
    const moduleUrl = 'file:///path/to/index.js';
    process.argv = ['node', '/path'];
    expect(isMainModule(moduleUrl)).toBe(false);
  });
});
