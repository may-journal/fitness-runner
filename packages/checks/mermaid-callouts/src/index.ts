import type { Check, CheckName } from '@mayjournal/fitness';
import {
  pairDiagramsWithTables,
  parseDoc,
  runMermaidDocCheck,
  type DiagramBlock,
  type CalloutTableBlock,
} from '@mayjournal/fitness-shared';

/** Counts each number's occurrences, preserving first-seen order. */
function counts(numbers: number[]): Map<number, number> {
  const map = new Map<number, number>();
  for (const n of numbers) map.set(n, (map.get(n) ?? 0) + 1);
  return map;
}

/** Errors for callout numbers appearing more than once. */
function duplicateErrors(
  byNumber: Map<number, number>,
  format: (n: number, count: number) => string
): string[] {
  const errors: string[] = [];
  for (const [n, count] of byNumber) if (count > 1) errors.push(format(n, count));
  return errors;
}

/** Errors for numbers present in `from` but absent from `other`. */
function missingErrors(
  from: Map<number, number>,
  other: Map<number, number>,
  format: (n: number) => string
): string[] {
  const errors: string[] = [];
  for (const n of from.keys()) if (!other.has(n)) errors.push(format(n));
  return errors;
}

/** Errors when a diagram and its table are not a 1-1 match on callout numbers. */
function compareCallouts(file: string, diagram: DiagramBlock, table: CalloutTableBlock): string[] {
  const inDiagram = counts(diagram.numbers);
  const inTable = counts(table.numbers);
  return [
    ...duplicateErrors(
      inDiagram,
      (n, c) => `${file}: diagram callout ${n} appears ${c} times (line ${diagram.line})`
    ),
    ...duplicateErrors(
      inTable,
      (n, c) => `${file}: callout table row ${n} appears ${c} times (line ${table.line})`
    ),
    ...missingErrors(
      inDiagram,
      inTable,
      (n) => `${file}: diagram callout ${n} has no matching table row (line ${diagram.line})`
    ),
    ...missingErrors(
      inTable,
      inDiagram,
      (n) => `${file}: callout table row ${n} has no matching diagram callout (line ${table.line})`
    ),
  ];
}

/**
 * Validates the 1-1 callout rule for one document: every numbered diagram has an
 * associated callout table with a bijective set of callout numbers. Un-numbered
 * diagrams need no table; a table with no preceding diagram is an error.
 */
export function validateDoc(file: string, content: string): string[] {
  const { pairs, orphanTables } = pairDiagramsWithTables(parseDoc(content));
  const errors: string[] = [];
  for (const table of orphanTables)
    errors.push(`${file}: callout table at line ${table.line} has no preceding mermaid diagram`);
  for (const { diagram, table } of pairs) {
    if (diagram.numbers.length === 0) continue;
    if (!table) {
      errors.push(
        `${file}: mermaid diagram at line ${diagram.line} has numbered callouts but no associated callout table`
      );
      continue;
    }
    errors.push(...compareCallouts(file, diagram, table));
  }
  return errors;
}

/**
 * Foundational check of the mermaid diagram + callout table set: every numbered
 * mermaid diagram must have an associated callout table, and their callout
 * numbers must be a 1-1 match (no orphan numbers, orphan rows, or duplicates).
 */
const mermaidCalloutsCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'mermaid-callouts' as CheckName,
  run: (root = process.cwd()) => runMermaidDocCheck(root, validateDoc),
  runInProcess: true,
} satisfies Check;

export default mermaidCalloutsCheck;
