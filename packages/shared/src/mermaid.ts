/**
 * Shared parsing for the mermaid diagram + callout table fitness checks
 * (see may-journal/fitness-runner#28). Extracts, from a markdown document, the
 * mermaid diagrams and numbered callout tables and their callout numbers, so
 * each check can apply its own pairing and validation rules on top.
 */

/** Discriminant kind of a mermaid diagram block — the single source for the literal. */
export const DIAGRAM_KIND = 'diagram' as const;

/** Discriminant kind of a numbered callout table block. */
export const TABLE_KIND = 'table' as const;

/** A ```mermaid fenced block. */
export interface DiagramBlock {
  /** Raw mermaid source, fences stripped. */
  body: string;
  kind: typeof DIAGRAM_KIND;
  /** 1-based line of the opening fence. */
  line: number;
  /** Callout numbers referenced, in document order (may contain duplicates). */
  numbers: number[];
}

/** A GFM table whose first header cell marks it as a numbered callout table. */
export interface CalloutTableBlock {
  /** Trimmed header cells. */
  header: string[];
  kind: typeof TABLE_KIND;
  /** 1-based line of the header row. */
  line: number;
  /** Integers read from the first (`#`) column, in order (may contain duplicates). */
  numbers: number[];
  /** Trimmed body-row cells. */
  rows: string[][];
}

export type DocBlock = DiagramBlock | CalloutTableBlock;

/** Mermaid directive/style lines whose numbers (hex colors, stroke widths) are not callouts. */
const STYLE_LINE = /^\s*(classDef|style|linkStyle|%%)/;

/**
 * Patterns that locate a callout number as the leading integer of a label. The
 * digit's absolute position dedupes matches that overlap across patterns.
 */
const CALLOUT_PATTERNS = [
  /["'](\d+)(?=[\s"'])/g, // quoted label: "1 Check", "6"
  /[([{>]\s*(\d+)(?=\s)/g, // node-shape label: [2 Screen], ((1 Persona)), [(4 System)]
  /:\s*(\d+)(?=\s|$)/g, // edge/message label: A --> B : 11
  /\|\s*(\d+)(?=[\s|])/g, // flowchart pipe label: -->|5|, -->|6 calls the api|
  /--\s*(\d+)\s*--/g, // inline link label: A -- 5 --> B
];

/** Records each callout number's absolute digit position for one (non-style) line. */
function collectCalloutPositions(
  line: string,
  offset: number,
  byPosition: Map<number, number>
): void {
  for (const pattern of CALLOUT_PATTERNS) {
    for (const match of line.matchAll(pattern)) {
      const digitPos = offset + (match.index ?? 0) + match[0].indexOf(match[1]);
      byPosition.set(digitPos, Number(match[1]));
    }
  }
}

/**
 * Callout numbers referenced in a mermaid diagram body. Heuristic: the leading
 * integer of each node/edge label — quoted (`"1 Check"`), inside a node shape
 * (`[2 Screen]`, `((1 …))`), or as an edge label (`: 11`, `|5|`). Styling lines
 * are skipped so hex colors and stroke widths aren't mistaken for callouts.
 * Numbers are returned in document order, preserving duplicates.
 */
export function extractDiagramNumbers(body: string): number[] {
  const byPosition = new Map<number, number>();
  let offset = 0;
  for (const line of body.split('\n')) {
    if (!STYLE_LINE.test(line)) collectCalloutPositions(line, offset, byPosition);
    offset += line.length + 1; // +1 for the '\n' removed by split
  }
  return [...byPosition.entries()].sort(([a], [b]) => a - b).map(([, n]) => n);
}

/** True when the line is a GFM table separator row (`| --- | :--: |`). */
function isTableSeparator(line: string): boolean {
  return /^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$/.test(line);
}

/** Splits a GFM table row into trimmed cell strings, dropping the outer pipes. */
function splitRow(line: string): string[] {
  let inner = line.trim();
  if (inner.startsWith('|')) inner = inner.slice(1);
  if (inner.endsWith('|')) inner = inner.slice(0, -1);
  return inner.split('|').map((cell) => cell.trim());
}

/** True when a table's first header cell marks it as a numbered callout table. */
export function isCalloutHeader(cells: string[]): boolean {
  return cells.length > 0 && /^(#|no\.?|callout|ref)$/i.test(cells[0]);
}

/** Reads the leading integer from a callout table's first-column cells. */
function tableNumbers(rows: string[][]): number[] {
  const numbers: number[] = [];
  for (const row of rows) {
    const match = (row[0] ?? '').match(/^\[?(\d+)\]?$/);
    if (match) numbers.push(Number(match[1]));
  }
  return numbers;
}

/** Parses a mermaid fence opened at `start`; returns the block and the index after its closing fence. */
function parseDiagramBlock(
  lines: string[],
  start: number,
  marker: string
): { block: DiagramBlock; next: number } {
  const body: string[] = [];
  let i = start + 1;
  while (i < lines.length && !lines[i].trimStart().startsWith(marker)) {
    body.push(lines[i]);
    i += 1;
  }
  const joined = body.join('\n');
  return {
    block: {
      body: joined,
      kind: DIAGRAM_KIND,
      line: start + 1,
      numbers: extractDiagramNumbers(joined),
    },
    next: i + 1, // skip the closing fence
  };
}

/** Reads a GFM table at `start` (header at `start`, separator at `start + 1`); block is null when not a callout table. */
function parseTableBlock(
  lines: string[],
  start: number
): { block: CalloutTableBlock | null; next: number } {
  const header = splitRow(lines[start]);
  const rows: string[][] = [];
  let j = start + 2;
  while (j < lines.length && lines[j].includes('|') && lines[j].trim() !== '') {
    rows.push(splitRow(lines[j]));
    j += 1;
  }
  const block = isCalloutHeader(header)
    ? { header, kind: TABLE_KIND, line: start + 1, numbers: tableNumbers(rows), rows }
    : null;
  return { block, next: j };
}

/** True when `lines[i]` opens a GFM table (row followed by a separator row). */
function isTableStart(lines: string[], i: number): boolean {
  return lines[i].includes('|') && i + 1 < lines.length && isTableSeparator(lines[i + 1]);
}

/**
 * Parses a markdown document into ordered mermaid diagrams and numbered callout
 * tables. Non-callout tables and prose are ignored.
 */
export function parseDoc(content: string): DocBlock[] {
  const lines = content.split('\n');
  const blocks: DocBlock[] = [];
  let i = 0;
  while (i < lines.length) {
    const fence = lines[i].match(/^\s*(`{3,}|~{3,})\s*mermaid\b/i);
    if (fence) {
      const { block, next } = parseDiagramBlock(lines, i, fence[1]);
      blocks.push(block);
      i = next;
      continue;
    }
    if (isTableStart(lines, i)) {
      const { block, next } = parseTableBlock(lines, i);
      if (block) blocks.push(block);
      i = next;
      continue;
    }
    i += 1;
  }
  return blocks;
}

/** A mermaid diagram and the callout table that immediately follows it, if any. */
export interface DiagramTablePair {
  diagram: DiagramBlock;
  table: CalloutTableBlock | null;
}

/** Flushes an unpaired pending diagram (table: null) and returns `diagram` as the new pending. */
function startDiagram(
  pending: DiagramBlock | null,
  diagram: DiagramBlock,
  pairs: DiagramTablePair[]
): DiagramBlock {
  if (pending) pairs.push({ diagram: pending, table: null });
  return diagram;
}

/**
 * Associates each numbered diagram with the callout table that follows it before
 * the next numbered diagram. Legend diagrams (those with no callout numbers) are
 * exempt: they carry no callouts, so they neither require a table nor consume the
 * table that belongs to a preceding numbered diagram — the documented layout is
 * diagram, legend, then table. Tables appearing before any diagram are orphans.
 */
export function pairDiagramsWithTables(blocks: DocBlock[]): {
  orphanTables: CalloutTableBlock[];
  pairs: DiagramTablePair[];
} {
  const relevant = blocks.filter((b) => b.kind === TABLE_KIND || b.numbers.length > 0);
  const pairs: DiagramTablePair[] = [];
  const orphanTables: CalloutTableBlock[] = [];
  let pending: DiagramBlock | null = null;
  for (const block of relevant) {
    if (block.kind === DIAGRAM_KIND) {
      pending = startDiagram(pending, block, pairs);
      continue;
    }
    if (pending) {
      pairs.push({ diagram: pending, table: block });
      pending = null;
      continue;
    }
    orphanTables.push(block);
  }
  if (pending) pairs.push({ diagram: pending, table: null });
  return { orphanTables, pairs };
}
