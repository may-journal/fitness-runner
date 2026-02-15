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
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(pkg): init\n\n');
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
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => '');
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
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs single check when check name is positional (e.g. npx fitness semantic-commit)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(api): add endpoint\n\nBody');
    await run([
      'node',
      'fitness',
      'semantic-commit',
    ]);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs check from path when --check=./path/to/check.js', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-path-check-'));
    const checkPath = join(dir, 'check.js');
    writeFileSync(
      checkPath,
      'export default { name: "path-check", run: async () => ({ ok: true, errors: [], meta: { filesChecked: 0 } }) };',
    );
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- init\n');
    writeFileSync(join(dir, '.nvmrc'), '18');
    const origCwd = process.cwd();
    process.chdir(dir);
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementation(() => '');
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    await run([
      'node',
      'fitness',
      '--check=./check.js',
    ]);
    process.chdir(origCwd);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/path-check: .+/));
    expect(process.exit).toHaveBeenCalledWith(0);
    logSpy.mockRestore();
  });

  it('runs check from path when module exports named Check only (no default)', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-path-named-'));
    writeFileSync(
      join(dir, 'check.js'),
      'export const myCheck = { name: "named-check", run: async () => ({ ok: true, errors: [], meta: { filesChecked: 0 } }) };',
    );
    writeFileSync(join(dir, 'CHANGELOG.md'), '# Changelog\n\n### 2026.02.15.1100\n\n- init\n');
    writeFileSync(join(dir, '.nvmrc'), '18');
    const origCwd = process.cwd();
    process.chdir(dir);
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementation(() => '');
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    await run([
      'node',
      'fitness',
      '--check=./check.js',
    ]);
    process.chdir(origCwd);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/named-check: .+/));
    expect(process.exit).toHaveBeenCalledWith(0);
    logSpy.mockRestore();
  });

  it('exits 1 when path points to module with no Check export', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    writeFileSync(join(dir, 'check.js'), 'export const x = 1;');
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run([
      'node',
      'fitness',
      '--check=./check.js',
    ])).rejects.toThrow('exit');
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: ./check.js');
    errSpy.mockRestore();
  });

  it('exits 1 when path points to module that throws on import', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    writeFileSync(join(dir, 'check.js'), 'throw new Error("load error");');
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run([
      'node',
      'fitness',
      '--check=./check.js',
    ])).rejects.toThrow('exit');
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: ./check.js');
    errSpy.mockRestore();
  });

  it('exits 1 for unknown path (file not found or invalid module)', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { mkdtempSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    const origCwd = process.cwd();
    process.chdir(dir);
    await expect(run([
      'node',
      'fitness',
      '--check=./nonexistent.js',
    ])).rejects.toThrow('exit');
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: ./nonexistent.js');
    errSpy.mockRestore();
  });

  it('runs with staged context (git diff mocked)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'chore(deps): bump\n\n');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('handles git diff failure (catch)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => {
        throw new Error('not a git repo');
      })
      .mockImplementationOnce(() => 'feat(x): y\n\n');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('handles non-empty staged output', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => 'a.md\nb.md')
      .mockImplementationOnce(() => 'feat(runner): add tests\n\n');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('outputs check timing with filesChecked meta', async () => {
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(pkg): init\n\n');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/semantic-commit: checked 1 files in \d+ms/));
    logSpy.mockRestore();
  });

  it('logs errors and exits 1 when check fails', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'Bad commit');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('✖ [semantic-commit]'));
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('exits 1 when check fails', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync)
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'Initial commit\n\n');
    await run(['node',
      'fitness',
      '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(1);
  });

  it('--check=semantic-commit with positional path: exits 0 for semantic message', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    const msgPath = join(dir, 'msg.txt');
    writeFileSync(msgPath, 'feat(checks): add commit-msg hook\n\nBody');
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    await run(['node',
      'fitness',
      '--check=semantic-commit',
      msgPath]);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('semantic-commit with two positionals (check then msg path): uses second as message file', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    const msgPath = join(dir, 'msg.txt');
    writeFileSync(msgPath, 'feat(scope): two positionals\n\nBody');
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    await run([
      'node',
      'fitness',
      'semantic-commit',
      msgPath,
    ]);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('--check=semantic-commit with positional path: exits 0 when file unreadable or empty (treats as empty)', async () => {
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => '');
    await run(['node',
      'fitness',
      '--check=semantic-commit',
      '/nonexistent/msg.txt']);
    expect(process.exit).toHaveBeenCalledWith(0);
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-'));
    writeFileSync(join(dir, 'empty.txt'), '');
    vi.mocked(execSync).mockImplementationOnce(() => '');
    await run(['node',
      'fitness',
      '--check=semantic-commit',
      join(dir, 'empty.txt')]);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('--check=semantic-commit with positional path: exits 1 for non-semantic message', async () => {
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    const msgPath = join(dir, 'msg.txt');
    writeFileSync(msgPath, 'oops I forgot');
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => '');
    await run(['node',
      'fitness',
      '--check=semantic-commit',
      msgPath]);
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

  it('exits 1 with (semantic-commit) when --check=semantic-commit but registry has no semantic-commit', async () => {
    vi.resetModules();
    vi.doMock('../checks/index.js', () => ({ registry: [{ name: 'other', run: async () => ({ ok: true, errors: [], meta: {} }) }] }));
    const { run: runWithNoSemantic } = await import('../index.js');
    const { mkdtempSync, writeFileSync } = await import('node:fs');
    const { join } = await import('node:path');
    const { tmpdir } = await import('node:os');
    const dir = mkdtempSync(join(tmpdir(), 'fitness-commit-msg-'));
    writeFileSync(join(dir, 'msg.txt'), 'feat(x): y');
    const { execSync } = await import('node:child_process');
    vi.mocked(execSync).mockImplementationOnce(() => '');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    vi.stubGlobal('process', Object.assign(process, { exit: vi.fn() }));
    await expect(runWithNoSemantic(['node',
      'fitness',
      '--check=semantic-commit',
      join(dir, 'msg.txt')])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith('Unknown check: semantic-commit');
    errSpy.mockRestore();
    vi.unstubAllGlobals();
  });
});
