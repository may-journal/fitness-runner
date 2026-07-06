import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import mermaidCalloutWhyCheck, { validateDoc } from './index.js';

const diagram = ['```mermaid', 'Rel(a, b, "1")', '```'].join('\n');

describe('validateDoc', () => {
  it('passes when the callout table has a non-empty Why column', () => {
    const content = [
      diagram,
      '',
      '| # | Description | Why |',
      '| --- | --- | --- |',
      '| 1 | an edge | because it matters |',
    ].join('\n');
    expect(validateDoc('doc.md', content)).toEqual([]);
  });

  it('fails when the callout table is missing a Why column', () => {
    const content = [diagram, '', '| # | Description |', '| --- | --- |', '| 1 | an edge |'].join(
      '\n'
    );
    expect(validateDoc('doc.md', content)).toContain(
      'doc.md: callout table at line 5 is missing a "Why" column'
    );
  });

  it('flags a numbered row with an empty Why cell', () => {
    const content = [
      diagram,
      '',
      '| # | Description | Why |',
      '| --- | --- | --- |',
      '| 1 | an edge |  |',
    ].join('\n');
    expect(
      validateDoc('doc.md', content).some((e) => e.includes('row 1 has an empty "Why" cell'))
    ).toBe(true);
  });

  it('accepts Why case-insensitively', () => {
    const content = [
      diagram,
      '',
      '| # | Description | WHY |',
      '| --- | --- | --- |',
      '| 1 | an edge | reason |',
    ].join('\n');
    expect(validateDoc('doc.md', content)).toEqual([]);
  });
});

describe('mermaidCalloutWhyCheck', () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'mermaid-why-'));
  });
  function write(rel: string, content: string): void {
    const full = join(dir, rel);
    mkdirSync(join(full, '..'), { recursive: true });
    writeFileSync(full, content);
  }

  it('passes when there are no relevant files', async () => {
    const result = await mermaidCalloutWhyCheck.run(dir);
    expect(result).toMatchObject({ ok: true, meta: { filesChecked: 0 } });
  });

  it('fails a doc whose callout table lacks a Why column', async () => {
    write(
      'a.md',
      [diagram, '', '| # | Description |', '| --- | --- |', '| 1 | an edge |'].join('\n')
    );
    const result = await mermaidCalloutWhyCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(1);
  });
});
