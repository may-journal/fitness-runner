import type { Check, CheckName } from '@mayjournal/fitness';
import {
  pairDiagramsWithTables,
  parseDoc,
  runMermaidDocCheck,
  type DiagramBlock,
} from '@mayjournal/fitness-shared';

/** A callout number followed by prose, e.g. `6 Uses` — the duplication #20 targets. */
const NUMBER_THEN_PROSE = /^\d+\s+[A-Za-z]/;

/** C4 relationship calls whose label argument may carry prose. */
const REL_CALL = /\b(?:Bi)?Rel\w*\([^)]*\)/g;
/** Quoted argument content. */
const QUOTED = /["']([^"']*)["']/g;
/** Flowchart pipe edge label: `-->|6 uses|`. */
const PIPE_LABEL = /\|\s*([^|]*?)\s*\|/g;
/** classDef/style lines carry non-label numbers; skip them for colon labels. */
const STYLE_LINE = /^\s*(classDef|style|linkStyle|%%)/;
/** Edge label after a colon: `A --> B : 6 uses`. */
const COLON_LABEL = /:\s*(\d+\s+[A-Za-z][^|]*?)\s*$/;

/** Prose labels from C4 `Rel(...)` call arguments. */
function relCallProse(body: string): string[] {
  const found: string[] = [];
  for (const rel of body.matchAll(REL_CALL))
    for (const arg of rel[0].matchAll(QUOTED))
      if (NUMBER_THEN_PROSE.test(arg[1].trim())) found.push(arg[1].trim());
  return found;
}

/** Prose labels from a line's flowchart pipe edge labels (`-->|…|`). */
function pipeProse(line: string): string[] {
  const found: string[] = [];
  for (const pipe of line.matchAll(PIPE_LABEL))
    if (NUMBER_THEN_PROSE.test(pipe[1].trim())) found.push(pipe[1].trim());
  return found;
}

/** Prose label from a line's colon edge label (`A --> B : 6 uses`), if any. */
function colonProse(line: string): string[] {
  const colon = line.match(COLON_LABEL);
  return colon && NUMBER_THEN_PROSE.test(colon[1].trim()) ? [colon[1].trim()] : [];
}

/** Prose labels from a single diagram line's edge labels (styling lines skipped). */
function lineEdgeProse(line: string): string[] {
  if (STYLE_LINE.test(line)) return [];
  return [...pipeProse(line), ...colonProse(line)];
}

/** Relationship/edge labels in a diagram body that carry prose beyond a callout number. */
function proseLabels(diagram: DiagramBlock): string[] {
  const found = relCallProse(diagram.body);
  for (const line of diagram.body.split('\n')) found.push(...lineEdgeProse(line));
  return found;
}

/**
 * Flags relationship/edge labels that carry prose beyond a callout number when a
 * diagram is paired with a callout table — descriptions belong in the table.
 */
export function validateDoc(file: string, content: string): string[] {
  const errors: string[] = [];
  for (const { diagram, table } of pairDiagramsWithTables(parseDoc(content)).pairs) {
    if (!table || diagram.numbers.length === 0) continue;
    for (const label of proseLabels(diagram))
      errors.push(
        `${file}: diagram at line ${diagram.line} has a relationship label with prose ("${label}") — put descriptions in the callout table`
      );
  }
  return errors;
}

/** Diagram labels must not duplicate callout table prose (arrows carry numbers only). */
const mermaidDiagramProseCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'mermaid-diagram-prose' as CheckName,
  run: (root = process.cwd()) => runMermaidDocCheck(root, validateDoc),
  runInProcess: true,
} satisfies Check;

export default mermaidDiagramProseCheck;
