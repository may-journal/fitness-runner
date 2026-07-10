import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import repeatedStringLiteralsCheck, {
  findDuplicates,
  scanStringLiterals,
  type Literal,
} from './index.js';

/** Collect just the values scanned from source, in order. */
function values(src: string): string[] {
  return scanStringLiterals(src).map((l) => l.value);
}

/** Build a fresh temp dir and write files (relPath -> content) into it. */
function tempRepo(files: Record<string, string>): string {
  const dir = mkdtempSync(join(tmpdir(), 'repeated-string-literals-'));
  for (const [rel, content] of Object.entries(files)) writeFileSync(join(dir, rel), content);
  return dir;
}

describe('scanStringLiterals', () => {
  it('captures single- and double-quoted literals with 1-based lines', () => {
    expect(scanStringLiterals('const a = \'pending\';\nconst b = "active";')).toEqual([
      { line: 1, value: 'pending' },
      { line: 2, value: 'active' },
    ] satisfies Literal[]);
  });

  it('ignores strings inside line and block comments', () => {
    expect(values("// 'pending' here\n/* 'active' too */\nconst x = 'kept';")).toEqual(['kept']);
  });

  it('skips module specifiers (import/require/from)', () => {
    const src = [
      "import x from 'node:fs';",
      "export { y } from 'node:path';",
      "const z = require('node:os');",
      "const d = await import('node:url');",
      "const keep = 'pending';",
    ].join('\n');
    expect(values(src)).toEqual(['pending']);
  });

  it('does not mistake a regex literal for a string', () => {
    // The quote chars live inside a regex character class, not a string.
    expect(values("const re = /['\"]/g;\nconst keep = 'active';")).toEqual(['active']);
  });

  it('treats / after a value as division, not a regex', () => {
    expect(values("const r = a / b / c;\nconst keep = 'active';")).toEqual(['active']);
  });

  it('skips template literals in v1', () => {
    expect(values("const t = `hello ${name}`;\nconst keep = 'active';")).toEqual(['active']);
  });

  it('honors backslash escapes inside strings', () => {
    expect(values("const a = 'it\\'s fine';")).toEqual(["it's fine"]);
  });

  it('drops literals shorter than MIN_LENGTH', () => {
    expect(values("const a = 'ok'; const b = 'yes';")).toEqual(['yes']);
  });

  it('drops idiomatic values (encodings, stdio modes, typeof results)', () => {
    const src = [
      "readFileSync(p, 'utf8');",
      "spawnSync(cmd, { stdio: 'inherit' });",
      "if (typeof x === 'object') noop();",
      "const keep = 'active';",
    ].join('\n');
    expect(values(src)).toEqual(['active']);
  });
});

describe('findDuplicates', () => {
  it('flags a value at or above the occurrence threshold and lists locations', () => {
    const errors = findDuplicates([
      {
        file: 'a.ts',
        literals: [
          { line: 1, value: 'active' },
          { line: 9, value: 'active' },
        ],
      },
      { file: 'b.ts', literals: [{ line: 2, value: 'active' }] },
    ]);
    expect(errors).toEqual([
      '"active" appears 3 times (a.ts:1, a.ts:9, b.ts:2) — extract a shared constant',
    ]);
  });

  it('does not flag a value below the threshold', () => {
    const errors = findDuplicates([
      {
        file: 'a.ts',
        literals: [
          { line: 1, value: 'active' },
          { line: 2, value: 'active' },
        ],
      },
    ]);
    expect(errors).toEqual([]);
  });

  it('caps listed locations and appends a +N more suffix', () => {
    const literals = Array.from({ length: 7 }, (_, i) => ({ line: i + 1, value: 'active' }));
    const [error] = findDuplicates([{ file: 'a.ts', literals }]);
    expect(error).toBe(
      '"active" appears 7 times (a.ts:1, a.ts:2, a.ts:3, a.ts:4, a.ts:5, +2 more) — extract a shared constant'
    );
  });

  it('orders most-repeated first', () => {
    const errors = findDuplicates([
      {
        file: 'a.ts',
        literals: [
          { line: 1, value: 'twice' },
          { line: 2, value: 'twice' },
          { line: 3, value: 'twice' },
          { line: 4, value: 'thrice' },
          { line: 5, value: 'thrice' },
          { line: 6, value: 'thrice' },
          { line: 7, value: 'thrice' },
        ],
      },
    ]);
    expect(errors[0]).toContain('"thrice" appears 4 times');
    expect(errors[1]).toContain('"twice" appears 3 times');
  });
});

describe('repeatedStringLiteralsCheck', () => {
  it('passes when no literal repeats past the threshold', async () => {
    const dir = tempRepo({ 'a.ts': "const a = 'alpha';\nconst b = 'beta';\n" });
    const result = await repeatedStringLiteralsCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toEqual([]);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails on a literal repeated across files', async () => {
    const dir = tempRepo({
      'a.ts': "const a = 'active';\nconst b = 'active';\n",
      'b.ts': "const c = 'active';\n",
    });
    const result = await repeatedStringLiteralsCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toEqual([
      '"active" appears 3 times (a.ts:1, a.ts:2, b.ts:1) — extract a shared constant',
    ]);
    expect(result.meta?.filesChecked).toBe(2);
  });

  it('excludes test/spec/bench files from the scan', async () => {
    const dir = tempRepo({
      'a.test.ts': "const a = 'active';\nconst b = 'active';\nconst c = 'active';\n",
      'b.bench.ts': "const a = 'active';\nconst b = 'active';\nconst c = 'active';\n",
    });
    const result = await repeatedStringLiteralsCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('never flags values allowed via .fitnessrc repeatedStringLiterals.allow', async () => {
    const dir = tempRepo({
      '.fitnessrc.js': "module.exports = { repeatedStringLiterals: { allow: ['active'] } };\n",
      'a.ts': "const a = 'active';\nconst b = 'active';\nconst c = 'active';\n",
    });
    const result = await repeatedStringLiteralsCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toEqual([]);
  });
});
