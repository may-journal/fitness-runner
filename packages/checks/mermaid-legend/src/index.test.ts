import { mkdtempSync, mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { beforeEach, describe, it, expect } from 'vitest';
import mermaidLegendCheck, { validateDoc } from './index.js';

/** A flowchart with the given node lines plus optional trailing lines. */
function flowchart(...body: string[]): string {
  return ['```mermaid', 'flowchart TB', ...body, '```'].join('\n');
}

describe('validateDoc', () => {
  it('passes when every callout node has an inline class and a classDef exists', () => {
    const content = flowchart(
      'persona((1 Persona)):::persona',
      'screen[2 Screen]:::screen',
      'classDef persona fill:#eef',
      'classDef screen fill:#f8f'
    );
    expect(validateDoc('doc.md', content)).toEqual([]);
  });

  it('flags a callout node with no style class', () => {
    const content = flowchart('persona((1 Persona))', 'classDef persona fill:#eef');
    expect(
      validateDoc('doc.md', content).some((e) =>
        e.includes('callout 1 (persona) has no style class')
      )
    ).toBe(true);
  });

  it('flags a diagram with callouts but no classDef legend', () => {
    const content = flowchart('persona((1 Persona)):::persona');
    expect(validateDoc('doc.md', content).some((e) => e.includes('no classDef legend'))).toBe(true);
  });

  it('accepts a separate class statement for styling', () => {
    const content = flowchart(
      'persona((1 Persona))',
      'class persona persona',
      'classDef persona fill:#eef'
    );
    expect(validateDoc('doc.md', content)).toEqual([]);
  });

  it('skips non-flowchart diagrams (e.g. classDiagram)', () => {
    const content = ['```mermaid', 'classDiagram', 'class Check["1 Check"]', '```'].join('\n');
    expect(validateDoc('doc.md', content)).toEqual([]);
  });
});

describe('mermaidLegendCheck', () => {
  let dir: string;
  beforeEach(() => {
    dir = mkdtempSync(join(tmpdir(), 'mermaid-legend-'));
  });

  it('passes when there are no relevant files', async () => {
    const result = await mermaidLegendCheck.run(dir);
    expect(result).toMatchObject({ ok: true, meta: { filesChecked: 0 } });
  });

  it('fails a flowchart with an unstyled callout node', async () => {
    const full = join(dir, 'a.md');
    mkdirSync(join(full, '..'), { recursive: true });
    writeFileSync(full, flowchart('persona((1 Persona))', 'classDef persona fill:#eef'));
    const result = await mermaidLegendCheck.run(dir);
    expect(result.ok).toBe(false);
    expect(result.meta?.filesChecked).toBe(1);
  });
});
