import type { Check, CheckName } from '@mayjournal/fitness';
import { parseDoc, runMermaidDocCheck, type CalloutTableBlock } from '@mayjournal/fitness-shared';

const WHY = /^why$/i;
const NUMBERED = /^\[?\d+\]?$/;

/** True when a row is numbered but leaves its `Why` cell blank. */
function isNumberedRowMissingWhy(row: string[], whyIndex: number): boolean {
  return NUMBERED.test(row[0] ?? '') && !(row[whyIndex] ?? '').trim();
}

/** Errors for one table: a missing `Why` column, or numbered rows with an empty `Why` cell. */
function whyErrorsForTable(file: string, table: CalloutTableBlock): string[] {
  const whyIndex = table.header.findIndex((cell) => WHY.test(cell));
  if (whyIndex === -1)
    return [`${file}: callout table at line ${table.line} is missing a "Why" column`];
  const errors: string[] = [];
  for (const row of table.rows)
    if (isNumberedRowMissingWhy(row, whyIndex))
      errors.push(
        `${file}: callout table row ${row[0]} has an empty "Why" cell (line ${table.line})`
      );
  return errors;
}

/**
 * Requires each numbered callout table to carry a `Why` column and warns when a
 * numbered row leaves its `Why` cell empty — so tables explain why each element
 * exists, not just what it is.
 */
export function validateDoc(file: string, content: string): string[] {
  const errors: string[] = [];
  for (const block of parseDoc(content))
    if (block.kind === 'table') errors.push(...whyErrorsForTable(file, block));
  return errors;
}

/** Callout tables must include a `Why` column. */
const mermaidCalloutWhyCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'mermaid-callout-why' as CheckName,
  run: (root = process.cwd()) => runMermaidDocCheck(root, validateDoc),
  runInProcess: true,
} satisfies Check;

export default mermaidCalloutWhyCheck;
