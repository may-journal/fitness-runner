import { describe, it, expect } from 'vitest';
import {
  extractDiagramNumbers,
  isCalloutHeader,
  parseDoc,
  pairDiagramsWithTables,
} from './mermaid.js';

/** Wraps a mermaid diagram body in a fenced block without literal-backtick escaping. */
function mermaid(body: string): string {
  return ['```mermaid', body, '```'].join('\n');
}

describe('extractDiagramNumbers', () => {
  it('pulls the leading integer from quoted labels', () => {
    expect(extractDiagramNumbers('Rel(a, b, "6")')).toEqual([6]);
    expect(extractDiagramNumbers('Rel(a, b, "6 Uses")')).toEqual([6]);
    expect(extractDiagramNumbers('Container(app, "1", "SwiftUI")')).toEqual([1]);
  });

  it('returns [] when no numbers are present', () => {
    expect(extractDiagramNumbers('Container(app, "SwiftUI")')).toEqual([]);
    expect(extractDiagramNumbers('')).toEqual([]);
  });

  it('reads unquoted node-shape labels (flowchart)', () => {
    const body = 'persona((1 Persona))\nscreen[2 Screen]\naction(3 Action)\nsystem[(4 System)]';
    expect(extractDiagramNumbers(body)).toEqual([1, 2, 3, 4]);
  });

  it('reads edge/message labels after a colon (classDiagram)', () => {
    expect(extractDiagramNumbers('getChecks --> resolveCheckNames : 11')).toEqual([11]);
  });

  it('reads flowchart pipe edge labels, with or without trailing prose', () => {
    expect(extractDiagramNumbers('app -->|5| api')).toEqual([5]);
    expect(extractDiagramNumbers('app -->|6 calls the api| api')).toEqual([6]);
  });

  it('ignores numbers in classDef/style lines (hex colors, stroke widths)', () => {
    const body = 'screen[2 Screen]:::screen\nclassDef screen fill:#6366f1,stroke-width:2px';
    expect(extractDiagramNumbers(body)).toEqual([2]);
  });

  it('reads mixed quoted labels and colon edge labels in one diagram', () => {
    const body = 'class Check["1 Check"]\nclass Run["2 Run"]\nRun --> Check : 3';
    expect(extractDiagramNumbers(body)).toEqual([1, 2, 3]);
  });
});

describe('isCalloutHeader', () => {
  it('accepts #/No/Callout/Ref as the first cell', () => {
    expect(isCalloutHeader(['#', 'Description'])).toBe(true);
    expect(isCalloutHeader(['No', 'Description'])).toBe(true);
    expect(isCalloutHeader(['Ref', 'Why'])).toBe(true);
  });

  it('rejects other first cells', () => {
    expect(isCalloutHeader(['Name', 'Value'])).toBe(false);
    expect(isCalloutHeader([])).toBe(false);
  });
});

describe('parseDoc', () => {
  it('returns diagram then callout table with body, header, and rows', () => {
    const content = [
      mermaid('Rel(a, b, "1")'),
      '',
      '| # | Description |',
      '| --- | --- |',
      '| 1 | an edge |',
    ].join('\n');
    const blocks = parseDoc(content);
    expect(blocks.map((b) => b.kind)).toEqual(['diagram', 'table']);
    expect(blocks[0]).toMatchObject({ kind: 'diagram', numbers: [1], body: 'Rel(a, b, "1")' });
    expect(blocks[1]).toMatchObject({ kind: 'table', header: ['#', 'Description'], numbers: [1] });
  });

  it('ignores non-callout tables', () => {
    const content = ['| Name | Value |', '| --- | --- |', '| a | 1 |'].join('\n');
    expect(parseDoc(content)).toEqual([]);
  });
});

describe('pairDiagramsWithTables', () => {
  it('pairs each diagram with the following table and flags orphan tables', () => {
    const blocks = parseDoc(
      [
        '| # | Description |',
        '| --- | --- |',
        '| 1 | orphan |',
        '',
        mermaid('Rel(a, b, "1")'),
        '',
        '| # | Description |',
        '| --- | --- |',
        '| 1 | paired |',
      ].join('\n')
    );
    const { pairs, orphanTables } = pairDiagramsWithTables(blocks);
    expect(orphanTables).toHaveLength(1);
    expect(pairs).toHaveLength(1);
    expect(pairs[0].diagram.numbers).toEqual([1]);
    expect(pairs[0].table?.numbers).toEqual([1]);
  });

  it('pairs a diagram with null when no table follows', () => {
    const blocks = parseDoc(mermaid('Rel(a, b, "1")'));
    const { pairs } = pairDiagramsWithTables(blocks);
    expect(pairs).toEqual([{ diagram: blocks[0], table: null }]);
  });

  it('skips a legend diagram (no callouts) so the numbered diagram pairs with the table', () => {
    const blocks = parseDoc(
      [
        mermaid('Rel(a, b, "1")'),
        '',
        mermaid('Person(p, "Person")\nSystem(s, "System")'), // legend: no callout numbers
        '',
        '| # | Description |',
        '| --- | --- |',
        '| 1 | paired past the legend |',
      ].join('\n')
    );
    const { pairs, orphanTables } = pairDiagramsWithTables(blocks);
    expect(orphanTables).toHaveLength(0);
    expect(pairs).toHaveLength(1);
    expect(pairs[0].diagram.numbers).toEqual([1]);
    expect(pairs[0].table?.numbers).toEqual([1]);
  });
});
