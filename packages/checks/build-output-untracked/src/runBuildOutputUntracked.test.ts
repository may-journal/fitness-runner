import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import type { ExecSyncFn } from '@mayjournal/fitness-shared';
import { describe, it, expect } from 'vitest';
import {
  buildOutputUntrackedCheck,
  collectSourceFiles,
  distTrackingErrors,
  enUS,
  scanFileForDistImports,
} from './index.js';

function tempDir(): string {
  return mkdtempSync(join(tmpdir(), 'build-output-untracked-'));
}

function gitExec(opts: { direct?: string[]; ignored?: boolean; nested?: string[] }): ExecSyncFn {
  return ((command: string) => {
    if (command.includes('check-ignore')) {
      if (opts.ignored) return '';
      throw { status: 1, stdout: '' };
    }
    if (command === 'git ls-files -- dist') return `${(opts.direct ?? []).join('\n')}\n`;
    if (command === "git ls-files -- '**/dist/**'") return (opts.nested ?? []).join('\n');
    return '';
  }) as ExecSyncFn;
}

describe('top-level exports', () => {
  it('default export matches buildOutputUntrackedCheck', async () => {
    const mod = await import('./index.js');
    expect(mod.default).toBe(mod.buildOutputUntrackedCheck);
  });

  it('exposes message strings', () => {
    expect(enUS.NotIgnored).toContain('.gitignore');
    expect(enUS.RemovePrefix).toBe('Remove tracked files:');
  });

  it('has the opt-in check name', () => {
    expect(buildOutputUntrackedCheck.name).toBe('build-output-untracked');
    expect(buildOutputUntrackedCheck.runInProcess).toBe(true);
  });
});

describe('distTrackingErrors', () => {
  it('is empty when dist is ignored and untracked', () => {
    expect(distTrackingErrors('/repo', gitExec({ ignored: true }))).toEqual([]);
  });

  it('flags a missing gitignore rule', () => {
    expect(distTrackingErrors('/repo', gitExec({ ignored: false }))).toEqual([enUS.NotIgnored]);
  });

  it('flags tracked files and dedupes direct/nested, with a git rm hint', () => {
    const errors = distTrackingErrors(
      '/repo',
      gitExec({ direct: ['dist/a.js'], ignored: true, nested: ['dist/a.js', 'sub/dist/b.js'] })
    );
    expect(errors).toEqual(['Remove tracked files: dist/a.js, sub/dist/b.js (git rm --cached)']);
  });

  it('reports both rules at once when ignored is missing and files are tracked', () => {
    const errors = distTrackingErrors('/repo', gitExec({ direct: ['dist/a.js'], ignored: false }));
    expect(errors).toEqual([enUS.NotIgnored, 'Remove tracked files: dist/a.js (git rm --cached)']);
  });
});

describe('scanFileForDistImports', () => {
  it('flags from/import/require specifiers into dist with path:line', () => {
    const content = [
      "import ok from './local.js';",
      "import bad from '../dist/foo.js';",
      "const z = require('pkg/dist/z.js');",
      "export * from 'dist/root.js';",
      "const w = await import('normal-pkg');",
    ].join('\n');
    expect(scanFileForDistImports('src/x.ts', content)).toEqual([
      'src/x.ts:2 imports build output: "../dist/foo.js"',
      'src/x.ts:3 imports build output: "pkg/dist/z.js"',
      'src/x.ts:4 imports build output: "dist/root.js"',
    ]);
  });

  it('returns nothing for clean source', () => {
    expect(scanFileForDistImports('a.ts', "import x from './x.js';\n")).toEqual([]);
  });
});

describe('collectSourceFiles', () => {
  it('collects .ts/.mts/.cts and ignores other files, sorted', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'a.ts'), '');
    writeFileSync(join(dir, 'b.mts'), '');
    writeFileSync(join(dir, 'c.cts'), '');
    writeFileSync(join(dir, 'd.txt'), '');
    expect(await collectSourceFiles(dir)).toEqual(['a.ts', 'b.mts', 'c.cts']);
  });
});

describe('buildOutputUntrackedCheck.run', () => {
  it('passes on a clean repo and counts files + 1', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'clean.ts'), "import x from './local.js';\n");
    const result = await buildOutputUntrackedCheck.run(dir, {
      _execSync: gitExec({ ignored: true }),
    });
    expect(result.ok).toBe(true);
    expect(result.errors).toEqual([]);
    expect(result.meta?.filesChecked).toBe(2);
  });

  it('fails on a dist import in source', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'bad.ts'), "import x from '../dist/x.js';\n");
    const result = await buildOutputUntrackedCheck.run(dir, {
      _execSync: gitExec({ ignored: true }),
    });
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual(['bad.ts:1 imports build output: "../dist/x.js"']);
  });

  it('fails on a tracked dist file and a missing gitignore rule', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'clean.ts'), "import x from './local.js';\n");
    const result = await buildOutputUntrackedCheck.run(dir, {
      _execSync: gitExec({ direct: ['dist/build.js'], ignored: false }),
    });
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      enUS.NotIgnored,
      'Remove tracked files: dist/build.js (git rm --cached)',
    ]);
  });
});
