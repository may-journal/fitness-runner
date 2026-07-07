import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import {
  runMarkdownFilenameCheck,
  validateMarkdownFilename,
  type FilenameConvention,
} from './runMarkdownFilenameCheck.js';

const KEBAB: FilenameConvention = {
  label: 'kebab-case',
  pattern: /^[a-z0-9]+(-[a-z0-9]+)*\.md$/,
};
const CAMEL: FilenameConvention = { label: 'camelCase', pattern: /^[a-z][a-zA-Z0-9]*\.md$/ };

describe('validateMarkdownFilename', () => {
  it('accepts a path matching the convention pattern', () => {
    expect(validateMarkdownFilename('docs/api-design.md', KEBAB)).toEqual([]);
    expect(validateMarkdownFilename('docs/releaseNotes.md', CAMEL)).toEqual([]);
  });

  it('exempts standard root docs regardless of case or convention', () => {
    expect(validateMarkdownFilename('README.md', KEBAB)).toEqual([]);
    expect(validateMarkdownFilename('CHANGELOG.md', CAMEL)).toEqual([]);
  });

  it('validates only the basename, ignoring directory casing', () => {
    expect(validateMarkdownFilename('SomeDir/api-design.md', KEBAB)).toEqual([]);
  });

  it('rejects a path that does not match, with the label in the message', () => {
    expect(validateMarkdownFilename('docs/Bad_Name.md', KEBAB)).toEqual([
      'docs/Bad_Name.md: filename must be kebab-case',
    ]);
    // camelCase rejects kebab (the two flavors are mutually distinct)
    expect(validateMarkdownFilename('docs/api-design.md', CAMEL)).toEqual([
      'docs/api-design.md: filename must be camelCase',
    ]);
    expect(validateMarkdownFilename('docs/releaseNotes.md', KEBAB)).toEqual([
      'docs/releaseNotes.md: filename must be kebab-case',
    ]);
  });
});

describe('runMarkdownFilenameCheck', () => {
  it('passes when no markdown files exist', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-name-'));
    const result = await runMarkdownFilenameCheck(dir, KEBAB);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('passes conforming and exempt files, counting every scanned file', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-name-'));
    writeFileSync(join(dir, 'api-design.md'), '# ok');
    writeFileSync(join(dir, 'plan-checks.md'), '# ok');
    writeFileSync(join(dir, 'README.md'), '# ok');
    const result = await runMarkdownFilenameCheck(dir, KEBAB);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(3);
  });

  it('fails non-conforming files under the given convention', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-name-'));
    writeFileSync(join(dir, 'bad_name.md'), '# no');
    writeFileSync(join(dir, 'BadName.md'), '# no');
    writeFileSync(join(dir, 'good-name.md'), '# ok');
    const result = await runMarkdownFilenameCheck(dir, KEBAB);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(3);
    expect(result.errors).toContain('bad_name.md: filename must be kebab-case');
    expect(result.errors).toContain('BadName.md: filename must be kebab-case');
  });
});
