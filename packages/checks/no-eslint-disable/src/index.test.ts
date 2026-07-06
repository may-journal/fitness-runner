import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import noEslintDisableCheck, {
  ESLINT_DISABLE_RE,
  findSourceFiles,
  scanContent,
  SOURCE_EXTENSIONS,
} from './index.js';

/** Directive strings kept out of comments so this test file never trips the check itself. */
const DIRECTIVES = {
  block: '/* eslint' + '-disable no-console */',
  file: '/* eslint' + '-disable */',
  line: 'const x = 1; // eslint' + '-disable-line',
  nextLine: '// eslint' + '-disable-next-line',
};

/** Creates a fresh temp directory for filesystem-backed cases. */
function tempDir(): string {
  return mkdtempSync(join(tmpdir(), 'no-eslint-disable-'));
}

describe('constants', () => {
  it('SOURCE_EXTENSIONS covers every documented extension', () => {
    expect(SOURCE_EXTENSIONS).toEqual(['.cjs', '.cts', '.js', '.mjs', '.mts', '.ts', '.tsx']);
  });

  it('ESLINT_DISABLE_RE matches every directive form', () => {
    for (const value of Object.values(DIRECTIVES)) {
      expect(ESLINT_DISABLE_RE.test(value)).toBe(true);
    }
  });
});

describe('scanContent', () => {
  it('returns no errors for a clean file', () => {
    expect(scanContent('a.ts', 'const clean = 1;\nexport default clean;\n')).toEqual([]);
  });

  it('reports path and 1-based line for each directive hit', () => {
    const content = ['const ok = 1;', DIRECTIVES.nextLine, DIRECTIVES.line].join('\n');
    expect(scanContent('src/b.ts', content)).toEqual([
      'src/b.ts:2: eslint-disable-next-line',
      'src/b.ts:3: eslint-disable-line',
    ]);
  });
});

describe('findSourceFiles', () => {
  it('finds and de-duplicates files across all extensions', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'a.ts'), 'const a = 1;\n');
    writeFileSync(join(dir, 'b.tsx'), 'const b = 2;\n');
    writeFileSync(join(dir, 'c.mjs'), 'export const c = 3;\n');
    writeFileSync(join(dir, 'ignore.md'), '# doc\n');
    expect(await findSourceFiles(dir)).toEqual(['a.ts', 'b.tsx', 'c.mjs']);
  });
});

describe('noEslintDisableCheck', () => {
  it('default export has the opt-in name and runs in process', () => {
    expect(noEslintDisableCheck.name).toBe('no-eslint-disable');
    expect(noEslintDisableCheck.runInProcess).toBe(true);
  });

  it('run() passes on a directory with no disable directives', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'a.ts'), 'const a = 1;\n');
    writeFileSync(join(dir, 'b.js'), 'const b = 2;\n');
    const result = await noEslintDisableCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toEqual([]);
    expect(result.meta?.filesChecked).toBe(2);
  });

  it('run() fails and reports one error per directive form', async () => {
    const dir = tempDir();
    writeFileSync(join(dir, 'file.ts'), DIRECTIVES.file + '\n');
    writeFileSync(join(dir, 'block.ts'), DIRECTIVES.block + '\n');
    writeFileSync(join(dir, 'line.ts'), DIRECTIVES.line + '\n');
    writeFileSync(join(dir, 'next.ts'), DIRECTIVES.nextLine + '\n');
    const result = await noEslintDisableCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      'block.ts:1: eslint-disable',
      'file.ts:1: eslint-disable',
      'line.ts:1: eslint-disable-line',
      'next.ts:1: eslint-disable-next-line',
    ]);
    expect(result.meta?.filesChecked).toBe(4);
  });
});
