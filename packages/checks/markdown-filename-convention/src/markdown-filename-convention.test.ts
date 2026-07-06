import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import markdownFilenameConventionCheck, {
  isValidMarkdownBasename,
  validateMarkdownFile,
} from './index.js';

describe('isValidMarkdownBasename', () => {
  it('accepts kebab-case', () => {
    expect(isValidMarkdownBasename('api-design.md')).toBe(true);
  });

  it('accepts camelCase (with trailing digits)', () => {
    expect(isValidMarkdownBasename('releaseNotes.md')).toBe(true);
    expect(isValidMarkdownBasename('adr001.md')).toBe(true);
  });

  it('accepts standard root docs via the exception list', () => {
    expect(isValidMarkdownBasename('README.md')).toBe(true);
    expect(isValidMarkdownBasename('CHANGELOG.md')).toBe(true);
  });

  it('rejects snake_case', () => {
    expect(isValidMarkdownBasename('bad_name.md')).toBe(false);
  });

  it('rejects PascalCase', () => {
    expect(isValidMarkdownBasename('BadName.md')).toBe(false);
  });
});

describe('validateMarkdownFile', () => {
  it('returns no errors for a conforming path', () => {
    expect(validateMarkdownFile('plans/plan-checks-abstractions.md')).toEqual([]);
  });

  it('returns an error including the path for a non-conforming basename', () => {
    const errors = validateMarkdownFile('docs/Bad_Name.md');
    expect(errors).toHaveLength(1);
    expect(errors[0]).toBe('docs/Bad_Name.md: filename must be kebab-case or camelCase');
  });
});

describe('markdownFilenameConventionCheck.run', () => {
  it('passes when no markdown files exist', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-name-'));
    const result = await markdownFilenameConventionCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('passes for kebab-case, camelCase, and exception filenames', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-name-'));
    writeFileSync(join(dir, 'api-design.md'), '# ok');
    writeFileSync(join(dir, 'releaseNotes.md'), '# ok');
    writeFileSync(join(dir, 'README.md'), '# ok');
    const result = await markdownFilenameConventionCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(3);
  });

  it('fails for snake_case and PascalCase with clear messages', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-name-'));
    writeFileSync(join(dir, 'bad_name.md'), '# no');
    writeFileSync(join(dir, 'BadName.md'), '# no');
    writeFileSync(join(dir, 'good-name.md'), '# ok');
    const result = await markdownFilenameConventionCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(3);
    expect(result.errors).toContain('bad_name.md: filename must be kebab-case or camelCase');
    expect(result.errors).toContain('BadName.md: filename must be kebab-case or camelCase');
  });
});
