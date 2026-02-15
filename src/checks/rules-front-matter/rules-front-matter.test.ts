import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import { getFrontMatterPaths, rulesFrontMatterCheck } from './index.js';

describe('getFrontMatterPaths', () => {
  it('returns [] when content has no front matter', () => {
    expect(getFrontMatterPaths('# No front matter')).toEqual([]);
    expect(getFrontMatterPaths('')).toEqual([]);
  });
});

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

  it('passes when no markdown files', async () => {
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result).toMatchObject({ ok: true, errors: [], meta: { filesChecked: 0 } });
  });

  it('ignores markdown under node_modules, dist, coverage, .git, .husky', async () => {
    writeRule('README.md', '---\nfitnessFunctions: []\n---\n# Root');
    writeRule('node_modules/pkg/readme.md', '---\nfitnessFunctions: ["./nope"]\n---');
    writeRule('dist/docs.md', '# Doc');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.meta?.filesChecked).toBe(1);
    expect(result.ok).toBe(true);
  });

  it('fails when markdown has no front matter', async () => {
    writeRule('doc.md', '# Doc\n\nBody');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('doc.md: missing front matter with fitnessFunctions or relatedConfigurations');
  });

  it('fails when front matter has neither fitnessFunctions nor relatedConfigurations', async () => {
    writeRule('doc.md', '---\ntitle: Foo\n---\n# Doc');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('doc.md: missing front matter with fitnessFunctions or relatedConfigurations');
  });

  it('passes when file has required front matter but no paths to validate', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: []\n---\n# Rule');
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

  it('passes when only relatedConfigurations present (no fitnessFunctions)', async () => {
    writeRule('doc.md', '---\nrelatedConfigurations: []\n---\n# Doc');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when fitnessFunctions path exists (relative to md file)', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["./lib/bar.js"]\n---\n# Rule');
    writeRule('50-59Rules/lib/bar.js', '');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when relatedConfigurations path exists (relative to md file)', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nrelatedConfigurations: ["config/bar.json"]\n---\n# Rule');
    writeRule('50-59Rules/config/bar.json', '{}');
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

  it('resolves paths relative to md file (e.g. ../../ from subdir)', async () => {
    writeRule('src/checks/README.md', '---\nrelatedConfigurations: ["../../package.json"]\n---\n# Checks');
    writeRule('package.json', '{}');
    const result = await rulesFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });
});
