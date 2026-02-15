import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

const { run } = await import('../src/index.js');

describe('fitness run', () => {
  const exit = process.exit;

  beforeEach(() => {
    vi.stubGlobal(
      'process',
      Object.assign(process, {
        exit: vi.fn(),
      }),
    );
  });

  afterEach(() => {
    process.exit = exit;
    vi.unstubAllGlobals();
  });

  it('exits 0 when all checks pass', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026-02-15\n\n- init\n');
    writeFileSync(join(dir, '.nvmrc'), '18');
    const origCwd = process.cwd();
    process.chdir(dir);
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(pkg): init');
    await run(['node', 'fitness']);
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 for unknown check', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node',
'fitness',
'--check=unknown']);
    expect(process.exit).toHaveBeenCalledWith(1);
  });

  it('runs single check when --check=semantic-commit', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    await run(['node',
'fitness',
'--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs with --staged', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'chore(deps): bump');
    await run(['node',
'fitness',
'--staged',
'--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('handles git diff failure when --staged (catch)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => {
        throw new Error('not a git repo');
      })
      .mockImplementationOnce(() => 'feat(x): y');
    await run(['node',
'fitness',
'--staged',
'--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('handles non-empty staged output when --staged', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => 'a.md\nb.md')
      .mockImplementationOnce(() => 'feat(runner): add tests');
    await run(['node',
'fitness',
'--staged',
'--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('outputs check timing with filesChecked meta', async () => {
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(pkg): init');
    await run(['node',
'fitness',
'--check=semantic-commit']);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/semantic-commit: checked 1 files in \d+ms/));
    logSpy.mockRestore();
  });

  it('logs errors and exits 1 when check fails', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'Bad commit');
    await run(['node',
'fitness',
'--check=semantic-commit']);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('✖ [semantic-commit]'));
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('exits 1 when check fails', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'Initial commit\n\n');
    await run(['node',
'fitness',
'--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(1);
  });
});
