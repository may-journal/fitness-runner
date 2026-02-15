import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import { rulesFrontMatterCheck } from './index.js';

describe('rulesFrontMatterCheck', () => {
  let dir: string;

  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'rules-fm-'));
  });

  /** Writes a file at relPath under dir, creating parent dirs. */
  function writeRule(relPath: string, content: string): void {
    const full = join(dir, relPath);
    mkdirSync(join(full, '..'), { recursive: true });
    writeFileSync(full, content);
  }

  it('passes when no 50-59Rules markdown files', async () => {
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result).toMatchObject({ ok: true, errors: [], meta: { filesChecked: 0 } });
  });

  it('passes when 50-59Rules file has no front matter paths', async () => {
    writeRule('50-59Rules/01-foo.md', '# Rule\n\nBody');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes when fitnessFunctions or relatedConfigurations is empty array', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: []\nrelatedConfigurations: []\n---\n# Rule');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when fitnessFunctions path exists', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["./lib/bar.js"]\n---\n# Rule');
    writeRule('lib/bar.js', '');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when relatedConfigurations path exists', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nrelatedConfigurations: ["config/bar.json"]\n---\n# Rule');
    writeRule('config/bar.json', '{}');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('skips http, #, and mailto paths', async () => {
    writeRule(
      '50-59Rules/01-foo.md',
      '---\nfitnessFunctions: ["https://x.com", "#anchor", "mailto:a@b.com"]\n---\n# Rule',
    );
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('fails when front matter path is missing', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["./missing.js"]\n---\n# Rule');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('50-59Rules/01-foo.md: front matter path missing: ./missing.js');
  });

  it('fails when path escapes repo', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["../../../etc/passwd"]\n---\n# Rule');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('escapes repo'))).toBe(true);
  });

  it('validates front matter paths in any markdown file', async () => {
    writeRule('other/readme.md', '---\nfitnessFunctions: ["./nope"]\n---\n');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('other/readme.md') && e.includes('./nope'))).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });
});
