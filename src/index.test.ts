import { describe, it, expect, vi } from 'vitest';
import { run, enUS } from './index.js';

describe('index', () => {
  it('exports run and enUS', () => {
    expect(typeof run).toBe('function');
    expect(enUS).toBeDefined();
    expect(typeof enUS).toBe('object');
  });

  it('calls run when isMainModule returns true', async () => {
    const runMock = vi.fn();
    vi.resetModules();
    vi.doMock('./utils/isMainModule.js', () => ({ isMainModule: () => true }));
    vi.doMock('./runner/index.js', () => ({ run: runMock, enUS: {} }));
    await import('./index.js');
    expect(runMock).toHaveBeenCalled();
  });
});
