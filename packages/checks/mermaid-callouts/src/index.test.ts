import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import mermaidCalloutsCheck from './index.js';

/** Wraps a mermaid diagram body in a fenced block without literal-backtick escaping. */
function mermaid(body: string): string {
  return ['```mermaid', body, '```'].join('\n');
}

/** Builds a numbered callout table from `#` → description row pairs. */
function calloutTable(rows: Array<[number | string, string]>): string {
  return [
    '| # | Description | Why |',
    '| --- | --- | --- |',
    ...rows.map(([n, desc]) => `| ${n} | ${desc} | because |`),
  ].join('\n');
}

describe('mermaidCalloutsCheck', () => {
  let dir: string;

  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'mermaid-callouts-'));
  });

  /** Writes a file at relPath under dir, creating parent dirs. */
  function write(relPath: string, content: string): void {
    const full = join(dir, relPath);
    mkdirSync(join(full, '..'), { recursive: true });
    writeFileSync(full, content);
  }

  it('passes when there are no markdown files', async () => {
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result).toMatchObject({ ok: true, errors: [], meta: { filesChecked: 0 } });
  });

  it('skips markdown with no diagram or callout table', async () => {
    write('doc.md', '# Just prose\n\nNothing to see here.');
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result).toMatchObject({ ok: true, meta: { filesChecked: 0 } });
  });

  it('passes when diagram and table callouts match 1-1', async () => {
    write(
      'architecture/01-context.md',
      [
        mermaid('Container(app, "1", "UI")\nContainer(api, "2", "API")\nRel(app, api, "3")'),
        '',
        calloutTable([
          [1, 'the app'],
          [2, 'the api'],
          [3, 'talks to'],
        ]),
      ].join('\n')
    );
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
    expect(result.meta?.filesChecked).toBe(1);
  });

  it('fails when a diagram callout has no matching table row', async () => {
    write(
      'doc.md',
      [mermaid('Rel(a, b, "1")\nRel(b, c, "2")'), '', calloutTable([[1, 'x']])].join('\n')
    );
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors).toContain('doc.md: diagram callout 2 has no matching table row (line 1)');
  });

  it('fails when a table row has no matching diagram callout', async () => {
    write(
      'doc.md',
      [
        mermaid('Rel(a, b, "1")'),
        '',
        calloutTable([
          [1, 'x'],
          [2, 'orphan'],
        ]),
      ].join('\n')
    );
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(
      result.errors.some((e) => e.includes('callout table row 2 has no matching diagram callout'))
    ).toBe(true);
  });

  it('fails when a numbered diagram has no associated table', async () => {
    write('doc.md', mermaid('Rel(a, b, "1")'));
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('no associated callout table'))).toBe(true);
  });

  it('fails when a callout table has no preceding diagram', async () => {
    write('doc.md', calloutTable([[1, 'x']]));
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.errors.some((e) => e.includes('has no preceding mermaid diagram'))).toBe(true);
  });

  it('fails on duplicate callout numbers on either side', async () => {
    write(
      'diagram-dup.md',
      [mermaid('Rel(a, b, "1")\nRel(a, c, "1")'), '', calloutTable([[1, 'x']])].join('\n')
    );
    write(
      'table-dup.md',
      [
        mermaid('Rel(a, b, "1")'),
        '',
        calloutTable([
          [1, 'x'],
          [1, 'again'],
        ]),
      ].join('\n')
    );
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(
      result.errors.some((e) => e.includes('diagram-dup.md: diagram callout 1 appears 2 times'))
    ).toBe(true);
    expect(
      result.errors.some((e) => e.includes('table-dup.md: callout table row 1 appears 2 times'))
    ).toBe(true);
  });

  it('passes for an un-numbered diagram with no table (nothing to pair)', async () => {
    write('doc.md', mermaid('Container(app, "SwiftUI")\nContainer(api, "Vapor")'));
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('passes when a legend diagram sits between the numbered diagram and its table', async () => {
    write(
      'architecture/01-context.md',
      [
        mermaid('Container(app, "1", "UI")\nRel(app, api, "2")'),
        '',
        mermaid('Person(p, "Person")\nSystem(s, "System")'), // legend: no callouts
        '',
        calloutTable([
          [1, 'the app'],
          [2, 'talks to'],
        ]),
      ].join('\n')
    );
    const result = await mermaidCalloutsCheck.run(dir);
    expect(result.ok).toBe(true);
    expect(result.errors).toHaveLength(0);
  });
});
