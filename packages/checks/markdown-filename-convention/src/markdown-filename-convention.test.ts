import { mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import kebabCheck, { KEBAB_RE } from './kebab-case.js';
import camelCheck, { CAMEL_RE } from './camel-case.js';

describe('markdown-filename-convention (two flavors from one package)', () => {
  it('exports two checks with distinct names', () => {
    expect(kebabCheck.name).toBe('markdown-filename-kebab-case');
    expect(camelCheck.name).toBe('markdown-filename-camel-case');
  });

  it('kebab-case matches hyphenated names and rejects camelCase', () => {
    expect(KEBAB_RE.test('api-design.md')).toBe(true);
    expect(KEBAB_RE.test('adr001.md')).toBe(true);
    expect(KEBAB_RE.test('releaseNotes.md')).toBe(false);
  });

  it('camelCase matches camel names and rejects hyphenated names', () => {
    expect(CAMEL_RE.test('releaseNotes.md')).toBe(true);
    expect(CAMEL_RE.test('adr001.md')).toBe(true);
    expect(CAMEL_RE.test('api-design.md')).toBe(false);
  });

  it('kebab flavor passes kebab + exempt files and fails camelCase', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-kebab-'));
    writeFileSync(join(dir, 'api-design.md'), '# ok');
    writeFileSync(join(dir, 'README.md'), '# ok');
    writeFileSync(join(dir, 'releaseNotes.md'), '# no');
    const result = await kebabCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(3);
    expect(result.errors).toContain('releaseNotes.md: filename must be kebab-case');
  });

  it('camel flavor passes camel + exempt files and fails kebab-case', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-camel-'));
    writeFileSync(join(dir, 'releaseNotes.md'), '# ok');
    writeFileSync(join(dir, 'README.md'), '# ok');
    writeFileSync(join(dir, 'api-design.md'), '# no');
    const result = await camelCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(3);
    expect(result.errors).toContain('api-design.md: filename must be camelCase');
  });
});
