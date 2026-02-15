import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

vi.mock('../src/checks/index.js', () => ({ registry: [] }));

const { run } = await import('../src/index.js');

describe('exitUnknown', () => {
  const exit = process.exit;

  beforeEach(() => {
    vi.stubGlobal('process', Object.assign(process, { exit: vi.fn() }));
  });

  afterEach(() => {
    process.exit = exit;
    vi.unstubAllGlobals();
  });

  it('exits 1 with (none) when no checks and no --check', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node',
'fitness']);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: (none)');
    errSpy.mockRestore();
  });
});
