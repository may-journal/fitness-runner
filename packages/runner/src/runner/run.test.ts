import { existsSync, mkdtempSync, symlinkSync, writeFileSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { tmpdir } from 'node:os';
import { beforeEach, afterEach, describe, it, expect, vi } from 'vitest';
import { run, enUS } from './index.js';

const { execSyncMock } = vi.hoisted(() => ({ execSyncMock: vi.fn() }));
const workerMockRef = vi.hoisted(() => ({
  behavior: 'success' as 'success' | 'successNoMeta' | 'errorMessage' | 'errorEvent' | 'timeout',
}));
const realExistsSync = vi.hoisted(() => ({
  fn: (() => true) as unknown as (p: string) => boolean,
}));
vi.mock('node:child_process', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:child_process')>();
  return { ...mod, execSync: execSyncMock };
});
vi.mock('node:fs', async (importOriginal) => {
  const fs = await importOriginal<typeof import('node:fs')>();
  realExistsSync.fn = fs.existsSync;
  return { ...fs, existsSync: vi.fn((path: string) => realExistsSync.fn(path)) };
});

vi.mock('node:worker_threads', async (importOriginal) => {
  const mod = await importOriginal<typeof import('node:worker_threads')>();
  function MockWorker(_path: string, _opts: { workerData?: { checkName: string; root: string } }) {
    const handlers: Record<string, (arg: unknown) => void> = {};
    const instance = {
      on: vi.fn((ev: string, cb: (arg: unknown) => void) => {
        handlers[ev] = cb;
      }),
      terminate: vi.fn(),
    };
    const successPayload = {
      result: { ok: true, errors: [] as string[], meta: { filesChecked: 0 } },
      ms: 0,
    };
    const successNoMetaPayload = {
      result: { ok: true, errors: [] as string[] },
      ms: 0,
    };
    setImmediate(() => {
      if (workerMockRef.behavior === 'timeout') return;
      if (workerMockRef.behavior === 'errorEvent' && handlers['error'])
        handlers['error'](new Error('worker error'));
      else if (handlers['message']) {
        const payload =
          workerMockRef.behavior === 'errorMessage'
            ? { error: 'worker failed', ms: 0 }
            : workerMockRef.behavior === 'successNoMeta'
              ? successNoMetaPayload
              : successPayload;
        handlers['message'](payload);
        if (workerMockRef.behavior === 'success') {
          setImmediate(() => handlers['message']?.(payload));
          setImmediate(() => handlers['error']?.(new Error('late')));
        }
      }
    });
    return instance;
  }
  return { ...mod, Worker: MockWorker };
});

const REPO_ROOT = resolve(fileURLToPath(new URL('../../../..', import.meta.url)));

function tempDir(): string {
  return mkdtempSync(join(tmpdir(), 'fitness-run-'));
}

function linkNodeModules(dir: string): void {
  const target = join(REPO_ROOT, 'node_modules');
  const link = join(dir, 'node_modules');
  if (existsSync(link)) return;
  symlinkSync(target, link, 'junction');
}

function runDir(dir: string, overrides: Record<string, string> = {}): void {
  linkNodeModules(dir);
  const base: Record<string, string> = {
    'package.json': '{"version":"0.1.0-2026.02.15.1100"}',
    'CHANGELOG.md':
      '---\nfitnessFunctions: ["./package.json"]\n---\n# Changelog\n\n### 2026.02.15.1100\n\n- init\n',
    '.nvmrc': '18',
    'vitest.config.js':
      'module.exports = { test: { coverage: { thresholds: { branches: 100, functions: 100, lines: 100, statements: 100 } } } };',
    ...overrides,
  };
  for (const [file, content] of Object.entries(base)) writeFileSync(join(dir, file), content);
}

describe('run', () => {
  const exit = process.exit;

  beforeEach(() => {
    execSyncMock.mockReset();
    workerMockRef.behavior = 'success';
    vi.stubGlobal('process', Object.assign(process, { exit: vi.fn() }));
  });

  afterEach(() => {
    process.chdir(REPO_ROOT);
    process.exit = exit;
    vi.unstubAllGlobals();
  });

  it('returns without running when re-entry guard is set', async () => {
    const g = globalThis as unknown as { __fitness_run_active?: boolean };
    g.__fitness_run_active = true;
    try {
      await run(['node', 'fitness']);
      expect(process.exit).not.toHaveBeenCalled();
    } finally {
      delete g.__fitness_run_active;
    }
  });

  it('exits 0 when all checks pass', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation((cmd: string) =>
      typeof cmd === 'string' && cmd.includes('--pretty') ? 'feat(pkg): init\n\n' : ''
    );
    const eslintMock = async () => ({ errors: [], exitCode: 0, filesChecked: 0 });
    await run(['node', 'fitness'], { _eslintRunForTesting: eslintMock });
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('excludes checks listed in disabledChecks from .fitnessrc checks list', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts':
        'export default { checks: ["changelog", "semantic-commit"], disabledChecks: ["semantic-commit"] };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(pkg): init\n\n');
    const stderrChunks: string[] = [];
    const stderrSpy = vi.spyOn(process.stderr, 'write').mockImplementation((chunk: unknown) => {
      stderrChunks.push(String(chunk));
      return true;
    });
    try {
      await run(['node', 'fitness']);
      process.chdir(origCwd);
      const stderr = stderrChunks.join('');
      expect(stderr).toMatch(/→ changelog/);
      expect(stderr).not.toMatch(/→ semantic-commit/);
      expect(process.exit).toHaveBeenCalledWith(0);
    } finally {
      stderrSpy.mockRestore();
    }
  });

  it('runs only checks from .fitnessrc.ts when present', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts': 'export default { checks: ["changelog", "semantic-commit"] };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(pkg): init\n\n');
    await run(['node', 'fitness']);
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs a local check module referenced by path in .fitnessrc checks, alongside a name', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts': 'export default { checks: ["changelog", "./my-check.js"] };',
      'my-check.js':
        'export default { name: "my-check", run: async () => ({ ok: true, errors: [] }) };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementationOnce(() => '');
    const stderrChunks: string[] = [];
    const stderrSpy = vi.spyOn(process.stderr, 'write').mockImplementation((chunk: unknown) => {
      stderrChunks.push(String(chunk));
      return true;
    });
    try {
      await run(['node', 'fitness']);
      process.chdir(origCwd);
      const stderr = stderrChunks.join('');
      expect(stderr).toMatch(/→ changelog/);
      expect(stderr).toMatch(/→ my-check/);
      expect(process.exit).toHaveBeenCalledWith(0);
    } finally {
      stderrSpy.mockRestore();
    }
  });

  it('exits 1 with resolution error when a config path spec is missing or invalid', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts': 'export default { checks: ["changelog", "./nonexistent.js"] };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node', 'fitness'])).rejects.toThrow('exit');
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('./nonexistent.js'));
    errSpy.mockRestore();
  });

  it('runs each check once when config.checks has duplicate names', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts':
        'export default { checks: ["changelog", "changelog", "semantic-commit", "changelog"] };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(pkg): init\n\n');
    const stderrChunks: string[] = [];
    const stderrSpy = vi.spyOn(process.stderr, 'write').mockImplementation((chunk: unknown) => {
      stderrChunks.push(String(chunk));
      return true;
    });
    try {
      await run(['node', 'fitness']);
      process.chdir(origCwd);
      const stderr = stderrChunks.join('');
      const changelogRuns = (stderr.match(/→ changelog/g) ?? []).length;
      const semanticRuns = (stderr.match(/→ semantic-commit/g) ?? []).length;
      expect(changelogRuns).toBe(1);
      expect(semanticRuns).toBe(1);
      expect(process.exit).toHaveBeenCalledWith(0);
    } finally {
      stderrSpy.mockRestore();
    }
  });

  it('runs only known checks when config.checks includes unknown names', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts': 'export default { checks: ["changelog", "nonexistent-check"] };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock
      .mockImplementationOnce(() => '')
      .mockImplementationOnce(() => 'feat(pkg): init\n\n');
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    await run(['node', 'fitness']);
    process.chdir(origCwd);
    const out = logSpy.mock.calls.map((c) => c[0]).join('\n');
    expect(out).toMatch(/changelog/);
    expect(out).not.toMatch(/nonexistent-check/);
    logSpy.mockRestore();
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 with (none) when configured checks all fail to load', async () => {
    const dir = tempDir();
    runDir(dir, { '.fitnessrc.ts': 'export default { checks: ["nonexistent-check"] };' });
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node', 'fitness'])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(enUS.UnknownCheckPrefix + enUS.UnknownCheckSpecNone)
    );
    process.chdir(origCwd);
    errSpy.mockRestore();
  });

  it('exits 1 for unknown check name', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node', 'fitness', '--check=unknown'])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('exits 1 with resolution error when bundle and config checks are unavailable', async () => {
    vi.resetModules();
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return {
        ...mod,
        resolveCheckNames: async () => {
          throw 'no checks';
        },
      };
    });
    const { run: runNoConfig } = await import('./index.js');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(runNoConfig(['node', 'fitness'])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('no checks'));
    errSpy.mockRestore();
  });

  it('maps checkFolderByName when loaded check folder differs from name', async () => {
    vi.resetModules();
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      const check = {
        name: 'markdown-front-matter',
        folder: 'rules-front-matter',
        run: async () => ({ ok: true, errors: [], meta: {} }),
      };
      // Mock both loaders: bundle-default resolution uses loadCheck, while an explicit
      // `.fitnessrc` checks list (fromConfig) resolves through tryLoadCheck.
      return {
        ...mod,
        resolveCheckNames: async () => ['markdown-front-matter'],
        loadCheck: async () => check,
        tryLoadCheck: async () => check,
      };
    });
    const { run: runFolderMap } = await import('./index.js');
    await runFolderMap(['node', 'fitness']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 with spec when --check=name but package is not installed', async () => {
    vi.resetModules();
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return { ...mod, tryLoadCheck: async () => null };
    });
    const { run: runNoSemantic } = await import('./index.js');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(runNoSemantic(['node', 'fitness', '--check=semantic-commit'])).rejects.toThrow(
      'exit'
    );
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(enUS.UnknownCheckPrefix + 'semantic-commit')
    );
    errSpy.mockRestore();
  });

  it('runs single check by name and by positional', async () => {
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'feat(api): add endpoint\n\nBody' : ''
    );
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
    await run(['node', 'fitness', 'semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('passes through args to single check', async () => {
    const dir = tempDir();
    linkNodeModules(dir);
    writeFileSync(join(dir, 'package.json'), '{"prettier": {}}');
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockReturnValue('');
    await run(['node', 'fitness', 'prettier', '--write', '.']);
    process.chdir(origCwd);
    const call = execSyncMock.mock.calls.find((c) => String(c[0]).includes('prettier'));
    expect(call?.[0]).toContain('--write');
    expect(call?.[0]).not.toContain('--check');
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs check from path (default and named export)', async () => {
    const dir = tempDir();
    runDir(dir);
    writeFileSync(
      join(dir, 'check-default.js'),
      'export default { name: "path-check", run: async () => ({ ok: true, errors: [], meta: { filesChecked: 0 } }) };'
    );
    writeFileSync(
      join(dir, 'check-named.js'),
      'export const myCheck = { name: "named-check", run: async () => ({ ok: true, errors: [], meta: { filesChecked: 0 } }) };'
    );
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    await run(['node', 'fitness', '--check=./check-default.js']);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/path-check/));
    await run(['node', 'fitness', '--check=./check-named.js']);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/named-check/));
    process.chdir(origCwd);
    logSpy.mockRestore();
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 when path has no Check export or throws on import', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'no-check.js'), 'export const x = 1;');
    writeFileSync(join(dir, 'throws.js'), 'throw new Error("load error");');
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node', 'fitness', '--check=./no-check.js'])).rejects.toThrow('exit');
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(enUS.UnknownCheckPrefix + './no-check.js')
    );
    await expect(run(['node', 'fitness', '--check=./throws.js'])).rejects.toThrow('exit');
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(enUS.UnknownCheckPrefix + './throws.js')
    );
    process.chdir(origCwd);
    errSpy.mockRestore();
  });

  it('exits 1 when path check run() throws or times out', async () => {
    const dir = tempDir();
    writeFileSync(
      join(dir, 'throw-error.js'),
      'export default { name: "throwing", run: () => { throw new Error("check error"); } };'
    );
    writeFileSync(
      join(dir, 'throw-string.js'),
      'export default { name: "throw-string", run: async () => { throw "oops"; } };'
    );
    writeFileSync(
      join(dir, 'hang.js'),
      'export default { name: "hang", run: () => new Promise(() => {}) };'
    );
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node', 'fitness', '--check=./throw-error.js']);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('check error'));
    await run(['node', 'fitness', '--check=./throw-string.js']);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('oops'));
    await run(['node', 'fitness', '--check=./hang.js'], { _checkTimeoutMsForTesting: 10 });
    expect(errSpy).toHaveBeenCalledWith(expect.stringMatching(/Check timed out after .+s/));
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('exits 1 for nonexistent path', async () => {
    const dir = tempDir();
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node', 'fitness', '--check=./nonexistent.js'])).rejects.toThrow('exit');
    process.chdir(origCwd);
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(enUS.UnknownCheckPrefix + './nonexistent.js')
    );
    errSpy.mockRestore();
  });

  it('uses staged context when git diff succeeds; continues when git fails', async () => {
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'chore(deps): bump\n\n' : ''
    );
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
    execSyncMock.mockImplementation(() => {
      throw new Error('not a git repo');
    });
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(1);
  });

  it('builds context with empty staged list when git diff returns empty', async () => {
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'feat(pkg): init\n\n' : ''
    );
    await run(['node', 'fitness', '--check=semantic-commit', '--message=feat(pkg): init']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 and logs errors when check fails', async () => {
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'Bad commit' : ''
    );
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(`[semantic-commit] ${enUS.PleaseFix}`)
    );
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining(enUS.ErrorBullet));
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('parses --message=value (tryEqualsArg) and injects into context', async () => {
    const dir = tempDir();
    linkNodeModules(dir);
    writeFileSync(join(dir, 'package.json'), '{}');
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'feat(scope): add\n\n' : ''
    );
    await run(['node', 'fitness', '--check=semantic-commit', '--message=feat(scope): add']);
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('injects --message= and --message (space) into context for semantic-commit', async () => {
    execSyncMock.mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    await run([
      'node',
      'fitness',
      '--check=semantic-commit',
      '--message=feat(checks): add commit-msg hook',
    ]);
    expect(process.exit).toHaveBeenCalledWith(0);
    execSyncMock.mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    await run(['node', 'fitness', 'semantic-commit', '--message=feat(scope): two positionals']);
    expect(process.exit).toHaveBeenCalledWith(0);
    execSyncMock.mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    await run(['node', 'fitness', '--check=semantic-commit', '--message', 'feat(x): space form']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('fails when --message= empty or missing for semantic-commit', async () => {
    execSyncMock.mockImplementation(() => '');
    await run(['node', 'fitness', '--check=semantic-commit', '--message=']);
    expect(process.exit).toHaveBeenCalledWith(1);
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(1);
    execSyncMock.mockImplementationOnce(() => {
      throw new Error('not a git repo');
    });
    await run(['node', 'fitness', '--check=semantic-commit', '--message', '--write']);
    expect(process.exit).toHaveBeenCalledWith(1);
  });

  it('strips both --message and missing value when positional check and --message has no value', async () => {
    execSyncMock.mockImplementation(() => {
      throw new Error('not a git repo');
    });
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node', 'fitness', 'semantic-commit', '--message']);
    expect(process.exit).toHaveBeenCalledWith(1);
    errSpy.mockRestore();
  });

  it('logs timing with filesChecked and when meta omits it', async () => {
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'feat(pkg): init\n\n' : ''
    );
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(logSpy).toHaveBeenCalledWith(expect.stringMatching(/semantic-commit.*\d+ms/));
    logSpy.mockRestore();
    vi.resetModules();
    vi.doMock('../checks/load-check.js', async (importOriginal) => {
      const mod = await importOriginal<typeof import('../checks/load-check.js')>();
      return {
        ...mod,
        tryLoadCheck: async (name: string) =>
          name === 'meta-less'
            ? { name: 'meta-less', run: async () => ({ ok: true, errors: [], meta: {} }) }
            : mod.tryLoadCheck(name, process.cwd()),
      };
    });
    const { run: runMetaLess } = await import('./index.js');
    const dir = tempDir();
    linkNodeModules(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    const logSpy2 = vi.spyOn(console, 'log').mockImplementation(() => {});
    await runMetaLess(['node', 'fitness', '--check=meta-less']);
    process.chdir(origCwd);
    expect(logSpy2).toHaveBeenCalledWith(expect.stringMatching(/meta-less/));
    logSpy2.mockRestore();
  });

  it('runs read-repo-first in-process without timeout race when VITEST unset', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    const origVitest = process.env.VITEST;
    delete process.env.VITEST;
    await run(['node', 'fitness', '--check=read-repo-first']);
    process.env.VITEST = origVitest;
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('runs package check via worker when VITEST unset', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    const fs = await import('node:fs');
    vi.mocked(fs.existsSync).mockImplementation(
      (path: string) =>
        (typeof path === 'string' && path.includes('run-one-check-worker')) ||
        realExistsSync.fn(path)
    );
    try {
      const origVitest = process.env.VITEST;
      delete process.env.VITEST;
      await run(['node', 'fitness', '--check=changelog']);
      process.env.VITEST = origVitest;
      expect(process.exit).toHaveBeenCalledWith(0);
    } finally {
      vi.mocked(fs.existsSync).mockRestore();
    }
    process.chdir(origCwd);
  });

  it('uses dist worker path when run-one-check-worker.js not next to run (existsSync false)', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    const fs = await import('node:fs');
    vi.mocked(fs.existsSync).mockImplementation((path: string) =>
      path.includes('run-one-check-worker') ? false : realExistsSync.fn(path)
    );
    try {
      const origVitest = process.env.VITEST;
      delete process.env.VITEST;
      await run(['node', 'fitness', '--check=changelog']);
      process.env.VITEST = origVitest;
      expect(process.exit).toHaveBeenCalledWith(0);
    } finally {
      vi.mocked(fs.existsSync).mockRestore();
    }
    process.chdir(origCwd);
  });

  it('strips node_modules from staged paths in context', async () => {
    execSyncMock.mockImplementation((cmd: string) =>
      cmd.includes('--pretty') ? 'feat(x): add\n\n' : 'src/a.ts\nnode_modules/pkg/index.js\nb.ts'
    );
    await run(['node', 'fitness', '--check=semantic-commit']);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('excludes checks listed in disabledChecks from bundle defaultChecks', async () => {
    const dir = tempDir();
    runDir(dir, { '.fitnessrc.ts': 'export default { disabledChecks: ["cspell"] };' });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation((cmd: string) =>
      typeof cmd === 'string' && cmd.includes('--pretty') ? 'feat(pkg): init\n\n' : ''
    );
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    const eslintMock = async () => ({ errors: [], exitCode: 0, filesChecked: 0 });
    await run(['node', 'fitness'], { _eslintRunForTesting: eslintMock });
    process.chdir(origCwd);
    const out = logSpy.mock.calls.map((c) => c[0]).join('\n');
    expect(out).toMatch(/changelog/);
    expect(out).not.toMatch(/cspell/);
    logSpy.mockRestore();
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 with (none) when disabledChecks removes all configured checks', async () => {
    const dir = tempDir();
    runDir(dir, {
      '.fitnessrc.ts': 'export default { checks: ["changelog"], disabledChecks: ["changelog"] };',
    });
    const origCwd = process.cwd();
    process.chdir(dir);
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await expect(run(['node', 'fitness'])).rejects.toThrow('exit');
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(
      expect.stringContaining(enUS.UnknownCheckPrefix + enUS.UnknownCheckSpecNone)
    );
    process.chdir(origCwd);
    errSpy.mockRestore();
  });

  it('runs bundle defaultChecks when .fitnessrc has no checks list', async () => {
    const dir = tempDir();
    runDir(dir, { '.fitnessrc.ts': 'export default { skipTheseDirectories: [] };' });
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation((cmd: string) =>
      typeof cmd === 'string' && cmd.includes('--pretty') ? 'feat(pkg): init\n\n' : ''
    );
    const logSpy = vi.spyOn(console, 'log').mockImplementation(() => {});
    const eslintMock = async () => ({ errors: [], exitCode: 0, filesChecked: 0 });
    await run(['node', 'fitness'], { _eslintRunForTesting: eslintMock });
    process.chdir(origCwd);
    const out = logSpy.mock.calls.map((c) => c[0]).join('\n');
    expect(out).toMatch(/changelog/);
    expect(out).toMatch(/semantic-commit/);
    logSpy.mockRestore();
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('handles worker success when result has no meta (filesChecked -1)', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    workerMockRef.behavior = 'successNoMeta';
    const origVitest = process.env.VITEST;
    delete process.env.VITEST;
    await run(['node', 'fitness', '--check=changelog']);
    process.env.VITEST = origVitest;
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(0);
  });

  it('exits 1 when worker posts error message', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    workerMockRef.behavior = 'errorMessage';
    const origVitest = process.env.VITEST;
    delete process.env.VITEST;
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node', 'fitness', '--check=changelog']);
    process.env.VITEST = origVitest;
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('worker failed'));
    errSpy.mockRestore();
  });

  it('exits 1 when worker emits error', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    workerMockRef.behavior = 'errorEvent';
    const origVitest = process.env.VITEST;
    delete process.env.VITEST;
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node', 'fitness', '--check=changelog']);
    process.env.VITEST = origVitest;
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(expect.stringContaining('worker error'));
    errSpy.mockRestore();
  });

  it('exits 1 when worker times out', async () => {
    const dir = tempDir();
    runDir(dir);
    const origCwd = process.cwd();
    process.chdir(dir);
    execSyncMock.mockImplementation(() => '');
    workerMockRef.behavior = 'timeout';
    const origVitest = process.env.VITEST;
    delete process.env.VITEST;
    const errSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    await run(['node', 'fitness', '--check=changelog'], { _checkTimeoutMsForTesting: 10 });
    process.env.VITEST = origVitest;
    process.chdir(origCwd);
    expect(process.exit).toHaveBeenCalledWith(1);
    expect(errSpy).toHaveBeenCalledWith(expect.stringMatching(/Check timed out after .+s/));
    errSpy.mockRestore();
  });
});
