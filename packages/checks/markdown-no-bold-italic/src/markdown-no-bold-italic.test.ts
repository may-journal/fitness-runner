import { mkdirSync, mkdtempSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { describe, it, expect } from 'vitest';
import { findDisallowedEmphasis, markdownNoBoldItalicCheck } from './index.js';

describe('markdownNoBoldItalicCheck', () => {
  it('passes when no markdown files exist', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(0);
  });

  it('passes when markdown has no bold or italic', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'doc.md'), '# Title\n\nPlain text and code `*not*`.\n');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('passes for unordered list items (asterisk list markers)', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'list.md'), '* item one\n* item two\n* item three');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('ignores underscores inside markdown link URL or link text', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(
      join(dir, 'links.md'),
      'See [here](_dev/path/to_resource_). And _real_ emphasis.'
    );
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('_dev'))).toBe(false);
    expect(result.errors.some((e) => e.includes('_real_'))).toBe(true);
  });

  it('ignores emphasis inside inline code and fenced code blocks', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(
      join(dir, 'doc.md'),
      'Text with `**code**` and ```\n**block**\n``` and _real_ emphasis.'
    );
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.length).toBe(1);
    expect(result.errors[0]).toContain('_italic_');
  });

  it('fails when markdown contains **bold**', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'a.md'), 'Hello **world** here.');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('**bold**') && e.includes('a.md'))).toBe(true);
  });

  it('fails when markdown contains __bold__', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'b.md'), 'Say __hello__.');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('__bold__') && e.includes('b.md'))).toBe(true);
  });

  it('fails when markdown contains *italic*', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'c.md'), 'This is *italic* text.');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('*italic*') && e.includes('c.md'))).toBe(true);
  });

  it('fails when markdown contains _italic_', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'd.md'), 'And _italic_ here.');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('_italic_') && e.includes('d.md'))).toBe(true);
  });

  it('reports multiple violations in one file', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(join(dir, 'e.md'), '**B** and *i* and __u__.');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.length).toBeGreaterThanOrEqual(2);
  });

  it('skips node_modules and other excluded dirs', async () => {
    const dir = mkdtempSync(join(tmpdir(), 'md-no-bold-'));
    writeFileSync(
      join(dir, 'cspell.json'),
      '{"ignorePaths":["node_modules","dist","coverage",".git","githooks"]}'
    );
    writeFileSync(join(dir, 'ok.md'), 'Plain.');
    const nodeMod = join(dir, 'node_modules');
    mkdirSync(nodeMod, { recursive: true });
    writeFileSync(join(nodeMod, 'pkg.md'), '**bold**');
    const result = await markdownNoBoldItalicCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.meta?.filesChecked).toBe(1);
  });
});

describe('findDisallowedEmphasis', () => {
  it('returns empty for plain text', () => {
    expect(findDisallowedEmphasis('# Hi\n\nNo emphasis.')).toEqual([]);
  });

  it('detects **bold**', () => {
    const hits = findDisallowedEmphasis('x **y** z');
    expect(hits.some((h) => h.kind === '**bold**' && h.match === '**y**')).toBe(true);
  });

  it('detects __bold__', () => {
    const hits = findDisallowedEmphasis('a __b__ c');
    expect(hits.some((h) => h.kind === '__bold__' && h.match === '__b__')).toBe(true);
  });

  it('detects *italic* but not **', () => {
    const hits = findDisallowedEmphasis('*italic* and **bold**');
    expect(hits.some((h) => h.kind === '*italic*' && h.match === '*italic*')).toBe(true);
    expect(hits.some((h) => h.kind === '**bold**' && h.match === '**bold**')).toBe(true);
  });

  it('detects _italic_', () => {
    const hits = findDisallowedEmphasis('_italic_ word');
    expect(hits.some((h) => h.kind === '_italic_' && h.match === '_italic_')).toBe(true);
  });

  it('does not flag unordered list items (asterisk list markers)', () => {
    const hits = findDisallowedEmphasis('* item one\n* item two\n* item three');
    expect(hits.filter((h) => h.kind === '*italic*')).toHaveLength(0);
  });

  it('does not flag underscores inside link blocks [text](url)', () => {
    const hits = findDisallowedEmphasis('Link [_dev/path/to_resource_](path).');
    expect(hits.some((h) => h.match.includes('_dev'))).toBe(false);
  });
});
