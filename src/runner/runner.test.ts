import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';

vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: vi.fn(mod.execSync) };
});

const { run } = await import('../index.js');

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
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- init\n');
    writeFileSync(join(dir, '.nvmrc'), '18');
    const origCwd = process.cwd();
    process.chdir(dir);
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(pkg): init');
    await run(['node',
      'fitness']);
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('logs check timing without filesChecked when meta omits it', async () => {
    vi.resetModules();
    vi.doMock('../checks/index.js', () => ({
      registry: [{ name: 'meta-less', run: async () => ({ ok: true, errors: [], meta: {} }) }],
    }));
    const { run: runWithMetaLess } = await import('../index.js');
    const { mkdtempSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    const origCwd = process.cwd();
    process.chdir(dir);
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    await runWithMetaLess(['node',
      'fitness',
      '--check=meta-less']);
    process.chdir(origCwd);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/meta-less: \d+ms/));
    logSpy.mockRestore();
  });

  it('exits 1 for unknown check', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node',
      'fitness',
      '--check=unknown'])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('runs single check when --check=semantic-commit', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs single check when check name is positional (e.g. npx fitness semantic-commit)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    await run([
      'node',
      'fitness',
      'semantic-commit',
    ]);
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

  it('--validate-commit-msg: exits 0 for semantic message', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    const msgPath = join(dir, 'msg.txt');
    writeFileSync(msgPath, 'feat(checks): add commit-msg hook\n\nBody');
    await run(['node',
      'fitness',
      `--validate-commit-msg=${msgPath}`]);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('--validate-commit-msg: exits 0 when message file unreadable or empty (treats as empty)', async () => {
    await run(['node',
      'fitness',
      '--validate-commit-msg=/nonexistent/msg.txt']);
    expect(process.exit).toHaveBeenCalledWith(0);
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    writeFileSync(join(dir, 'empty.txt'), '');
    await run(['node',
      'fitness',
      `--validate-commit-msg=${join(dir, 'empty.txt')}`]);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('--validate-commit-msg: exits 1 for non-semantic message', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    const msgPath = join(dir, 'msg.txt');
    writeFileSync(msgPath, 'oops I forgot');
    await run(['node',
      'fitness',
      `--validate-commit-msg=${msgPath}`]);
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });
});

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
    vi.resetModules();
    vi.doMock('../checks/index.js', () => ({ registry: [] }));
    const { run: runWithEmptyRegistry } = await import('../index.js');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(runWithEmptyRegistry(['node',
      'fitness'])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: (none)');
    errSpy.mockRestore();
  });

  it('exits 1 with (semantic-commit) when --validate-commit-msg but registry has no semantic-commit', async () => {
    vi.resetModules();
    vi.doMock('../checks/index.js', () => ({ registry: [{ name: 'other', run: async () => ({ ok: true, errors: [], meta: {} }) }] }));
    const { run: runWithNoSemantic } = await import('../index.js');
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    writeFileSync(join(dir, 'msg.txt'), 'feat(x): y');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.stubGlobal('process', Object.assign(process, { exit: vi.fn() }));
    await expect(runWithNoSemantic(['node',
      'fitness',
      `--validate-commit-msg=${join(dir, 'msg.txt')}`])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: semantic-commit');
    errSpy.mockRestore();
    vi.unstubAllGlobals();
  });
});
