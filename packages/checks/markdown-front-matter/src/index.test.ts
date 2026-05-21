import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import markdownFrontMatterCheck, { getFrontMatterPaths, hasRequiredFrontMatter } from './index.js';

describe('getFrontMatterPaths', () => {
  it('returns [] when content has no front matter', () => {
    expect(getFrontMatterPaths('# No front matter')).toEqual([]);
    expect(getFrontMatterPaths('')).toEqual([]);
  });

  it('skips empty arrays and returns paths from non-empty arrays', () => {
    expect(
      getFrontMatterPaths('---\nfitnessFunctions: []\nrelatedConfigurations: ["./x"]\n---')
    ).toEqual(['./x']);
  });
});

describe('hasRequiredFrontMatter', () => {
  it('returns false when content has no front matter', () => {
    expect(hasRequiredFrontMatter('# No front matter')).toBe(false);
    expect(hasRequiredFrontMatter('')).toBe(false);
  });
});

describe('markdownFrontMatterCheck', () => {
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
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result).toMatchObject({ ok: true, errors: [], meta: { filesChecked: 0 } });
  });

  it('ignores markdown under node_modules, dist, coverage, .git, githooks', async () => {
    writeRule(
      'cspell.json',
      '{"ignorePaths":["node_modules","dist","coverage",".git","githooks"]}'
    );
    writeRule('README.md', '---\nfitnessFunctions: ["./package.json"]\n---\n# Root');
    writeRule('package.json', '{}');
    writeRule('node_modules/pkg/readme.md', '---\nfitnessFunctions: ["./nope"]\n---');
    writeRule('dist/docs.md', '# Doc');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.meta?.filesChecked).toBe(1);
    expect(result.ok).toBe(true);
  });

  it('fails when markdown has no front matter', async () => {
    writeRule('doc.md', '# Doc\n\nBody');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(
      'doc.md: missing front matter with fitnessFunctions or relatedConfigurations'
    );
  });

  it('fails when front matter has neither fitnessFunctions nor relatedConfigurations', async () => {
    writeRule('doc.md', '---\ntitle: Foo\n---\n# Doc');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(
      'doc.md: missing front matter with fitnessFunctions or relatedConfigurations'
    );
  });

  it('passes when file has required front matter with at least one path', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["./lib/bar.js"]\n---\n# Rule');
    writeRule('50-59Rules/lib/bar.js', '');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails when fitnessFunctions or relatedConfigurations is empty array', async () => {
    writeRule(
      '50-59Rules/01-foo.md',
      '---\nfitnessFunctions: []\nrelatedConfigurations: []\n---\n# Rule'
    );
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(
      '50-59Rules/01-foo.md: fitnessFunctions must not be an empty array'
    );
    expect(result.errors).toContain(
      '50-59Rules/01-foo.md: relatedConfigurations must not be an empty array'
    );
  });

  it('passes when only relatedConfigurations present with at least one path', async () => {
    writeRule('doc.md', '---\nrelatedConfigurations: ["config.json"]\n---\n# Doc');
    writeRule('config.json', '{}');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when fitnessFunctions path exists (relative to md file)', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["./lib/bar.js"]\n---\n# Rule');
    writeRule('50-59Rules/lib/bar.js', '');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when relatedConfigurations path exists (relative to md file)', async () => {
    writeRule(
      '50-59Rules/01-foo.md',
      '---\nrelatedConfigurations: ["config/bar.json"]\n---\n# Rule'
    );
    writeRule('50-59Rules/config/bar.json', '{}');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('skips http, #, and mailto paths', async () => {
    writeRule(
      '50-59Rules/01-foo.md',
      '---\nfitnessFunctions: ["https://x.com", "#anchor", "mailto:a@b.com"]\n---\n# Rule'
    );
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when fitnessFunctions or relatedConfigurations references a registered check name', async () => {
    writeRule(
      'src/checks/cspell/README.md',
      '---\nfitnessFunctions: ["cspell"]\nrelatedConfigurations: ["../../../cspell.json"]\n---\n# cspell'
    );
    writeRule('cspell.json', '{}');
    const result = await markdownFrontMatterCheck.run(dir, {
      registeredCheckNames: ['cspell', 'markdown-no-bold-italic', 'changelog'],
    });
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('fails when front matter path is missing', async () => {
    writeRule('50-59Rules/01-foo.md', '---\nfitnessFunctions: ["./missing.js"]\n---\n# Rule');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain(
      '50-59Rules/01-foo.md: front matter path missing: ./missing.js'
    );
  });

  it('fails when path escapes repo', async () => {
    writeRule(
      '50-59Rules/01-foo.md',
      '---\nfitnessFunctions: ["../../../etc/passwd"]\n---\n# Rule'
    );
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('escapes repo'))).toBe(true);
  });

  it('validates front matter paths in any markdown file', async () => {
    writeRule('other/readme.md', '---\nfitnessFunctions: ["./nope"]\n---\n');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('other/readme.md') && e.includes('./nope'))).toBe(
      true
    );
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('resolves paths relative to md file (e.g. ../../ from subdir)', async () => {
    writeRule(
      'src/checks/README.md',
      '---\nrelatedConfigurations: ["../../package.json"]\n---\n# Checks'
    );
    writeRule('package.json', '{}');
    const result = await markdownFrontMatterCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });
});
