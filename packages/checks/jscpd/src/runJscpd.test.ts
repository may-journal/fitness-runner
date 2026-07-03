import { mkdtempSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { enUS, jscpdCheck } from './index.js';

const PASS_OUTPUT = `No duplicates found.
┌────────┬────────────────┬─────────────┬──────────────┬──────────────┬──────────────────┬───────────────────┐
│ Format │ Files analyzed │ Total lines │ Total tokens │ Clones found │ Duplicated lines │ Duplicated tokens │
├────────┼────────────────┼─────────────┼──────────────┼──────────────┼──────────────────┼───────────────────┤
│ Total: │ 4              │ 120         │ 900          │ 0            │ 0 (0.00%)        │ 0 (0.00%)         │
└────────┴────────────────┴─────────────┴──────────────┴──────────────┴──────────────────┴───────────────────┘
Found 0 clones.
time: 4.042ms
`;

const FAIL_OUTPUT = `Clone found (typescript)
 - src/a.ts [1:1 - 8:2] (8 lines, 50 tokens)
   src/b.ts [1:1 - 8:2]
┌────────────┬────────────────┬─────────────┬──────────────┬──────────────┬──────────────────┬───────────────────┐
│ Format     │ Files analyzed │ Total lines │ Total tokens │ Clones found │ Duplicated lines │ Duplicated tokens │
├────────────┼────────────────┼─────────────┼──────────────┼──────────────┼──────────────────┼───────────────────┤
│ typescript │ 2              │ 16          │ 100          │ 1            │ 7 (43.75%)       │ 50 (50.00%)       │
├────────────┼────────────────┼─────────────┼──────────────┼──────────────┼──────────────────┼───────────────────┤
│ Total:     │ 2              │ 16          │ 100          │ 1            │ 7 (43.75%)       │ 50 (50.00%)       │
└────────────┴────────────────┴─────────────┴──────────────┴──────────────┴──────────────────┴───────────────────┘
Found 1 clones.
ERROR: jscpd found too many duplicates (43.8%) over threshold (1.0%)
time: 4.042ms
`;

const ESC = String.fromCharCode(27);

function tempDir(): string {
  return mkdtempSync(join(tmpdir(), 'jscpd-'));
}

describe('jscpd top-level exports', () => {
  it('default export matches jscpdCheck', async () => {
    const mod = await import('./index.js');
    expect(mod.default).toBe(mod.jscpdCheck);
  });

  it('enUS exposes FallbackRunHint', () => {
    expect(enUS.FallbackRunHint).toBe(
      'jscpd reported an error (run: npx jscpd --reporters console .)'
    );
  });

  describe('jscpdCheck', () => {
    it('has name jscpd', () => {
      expect(jscpdCheck.name).toBe('jscpd');
    });

    it('run() passes when under threshold', async () => {
      const dir = tempDir();
      const result = await jscpdCheck.run(dir, { _execSync: () => PASS_OUTPUT });
      expect(result.ok).toBe(true);
      expect(result.errors).toHaveLength(0);
      expect(result.meta?.filesChecked).toBe(4);
    });

    it('run() fails and parses the ERROR line when over threshold', async () => {
      const dir = tempDir();
      const result = await jscpdCheck.run(dir, {
        _execSync: () => {
          throw { status: 1, stdout: FAIL_OUTPUT };
        },
      });
      expect(result.ok).toBe(false);
      expect(result.errors).toEqual([
        'ERROR: jscpd found too many duplicates (43.8%) over threshold (1.0%)',
      ]);
      expect(result.meta?.filesChecked).toBe(2);
    });

    it('run() falls back to FallbackRunHint when exit is non-zero with no ERROR line', async () => {
      const dir = tempDir();
      const result = await jscpdCheck.run(dir, {
        _execSync: () => {
          throw { status: 1, stdout: 'jscpd crashed unexpectedly' };
        },
      });
      expect(result.ok).toBe(false);
      expect(result.errors).toEqual([enUS.FallbackRunHint]);
      expect(result.meta?.filesChecked).toBe(0);
    });

    it('run() strips ANSI color codes before parsing', async () => {
      const dir = tempDir();
      const colored = FAIL_OUTPUT.replace(
        'ERROR: jscpd found too many duplicates (43.8%) over threshold (1.0%)',
        `${ESC}[31mERROR: jscpd found too many duplicates (43.8%) over threshold (1.0%)${ESC}[39m`
      );
      const result = await jscpdCheck.run(dir, {
        _execSync: () => {
          throw { status: 1, stdout: colored };
        },
      });
      expect(result.errors).toEqual([
        'ERROR: jscpd found too many duplicates (43.8%) over threshold (1.0%)',
      ]);
    });
  });
});
