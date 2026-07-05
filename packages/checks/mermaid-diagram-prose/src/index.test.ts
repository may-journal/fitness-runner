import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import mermaidDiagramProseCheck, { validateDoc } from './index.js';

/** Wraps diagram + a callout table so the diagram is paired (prose only checked when paired). */
function doc(diagramBody: string): string {
  return [
    '```mermaid',
    diagramBody,
    '```',
    '',
    '| # | Description | Why |',
    '| --- | --- | --- |',
    '| 6 | uses | because |',
  ].join('\n');
}

describe('validateDoc', () => {
  it('passes when Rel labels are numbers only', () => {
    expect(validateDoc('doc.md', doc('Rel(app, api, "6")'))).toEqual([]);
  });

  it('flags a Rel label carrying prose beyond the number', () => {
    const errors = validateDoc('doc.md', doc('Rel(app, api, "6 Uses over HTTP")'));
    expect(
      errors.some((e) => e.includes('relationship label with prose ("6 Uses over HTTP")'))
    ).toBe(true);
  });

  it('flags a flowchart pipe edge label with prose', () => {
    const errors = validateDoc('doc.md', doc('app -->|6 calls the api| api'));
    expect(errors.some((e) => e.includes('with prose ("6 calls the api")'))).toBe(true);
  });

  it('flags a colon edge label with prose', () => {
    const errors = validateDoc('doc.md', doc('app --> api : 6 sends data'));
    expect(errors.some((e) => e.includes('with prose ("6 sends data")'))).toBe(true);
  });

  it('does not flag when the diagram has no paired callout table', () => {
    const content = ['```mermaid', 'Rel(app, api, "6 Uses")', '```'].join('\n');
    expect(validateDoc('doc.md', content)).toEqual([]);
  });

  it('does not flag node names (only relationship/edge labels)', () => {
    expect(validateDoc('doc.md', doc('Container(app, "6", "SwiftUI")'))).toEqual([]);
  });
});

describe('mermaidDiagramProseCheck', () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'mermaid-prose-'));
  });

  it('passes when there are no relevant files', async () => {
    const result = await mermaidDiagramProseCheck.run(dir);
    expect(result).toMatchObject({ ok: true, meta: { filesChecked: 0 } });
  });

  it('fails a doc with a prose relationship label', async () => {
    const full = join(dir, 'a.md');
    mkdirSync(join(full, '..'), { recursive: true });
    writeFileSync(full, doc('Rel(app, api, "6 Uses over HTTP")'));
    const result = await mermaidDiagramProseCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(1);
  });
});
