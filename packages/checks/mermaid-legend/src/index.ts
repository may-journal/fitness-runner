import type { Check, CheckName } from '@mayjournal/fitness';
import {
  DIAGRAM_KIND,
  parseDoc,
  runMermaidDocCheck,
  type DiagramBlock,
} from '@mayjournal/fitness-shared';

const FLOWCHART = /^\s*(flowchart|graph)\b/;
const CLASSDEF = /^\s*classDef\s+\w+/;
const NODE_DEF = /^\s*(\w[\w-]*)\s*(?:\(\(|\[\(|\[|\(|\{|>)\s*(\d+)\b/;
const CLASS_STMT = /^\s*class\s+([\w,\s-]+?)\s+\w+\s*$/;
const STYLE_STMT = /^\s*style\s+(\w[\w-]*)\b/;
const INLINE_CLASS = /:::\w+/;

interface NumberedNode {
  id: string;
  number: number;
  styledInline: boolean;
}

/** True when a diagram body is a flowchart/graph (per its first non-blank line). */
function isFlowchart(lines: string[]): boolean {
  const firstReal = lines.find((line) => line.trim());
  return firstReal != null && FLOWCHART.test(firstReal);
}

/** Node ids given a style class via `class`/`style` statements. */
function collectStyledIds(lines: string[]): Set<string> {
  const styledIds = new Set<string>();
  for (const line of lines) {
    const classStmt = line.match(CLASS_STMT);
    if (classStmt) for (const id of classStmt[1].split(',')) styledIds.add(id.trim());
    const styleStmt = line.match(STYLE_STMT);
    if (styleStmt) styledIds.add(styleStmt[1]);
  }
  return styledIds;
}

/** Numbered callout nodes defined in a flowchart body. */
function collectNumberedNodes(lines: string[]): NumberedNode[] {
  const nodes: NumberedNode[] = [];
  for (const line of lines) {
    const match = line.match(NODE_DEF);
    if (match)
      nodes.push({ id: match[1], number: Number(match[2]), styledInline: INLINE_CLASS.test(line) });
  }
  return nodes;
}

/** Errors for numbered nodes carrying no style class (inline or via a statement). */
function unstyledNodeErrors(
  file: string,
  block: DiagramBlock,
  nodes: NumberedNode[],
  styledIds: Set<string>
): string[] {
  const errors: string[] = [];
  for (const node of nodes)
    if (!node.styledInline && !styledIds.has(node.id))
      errors.push(
        `${file}: callout ${node.number} (${node.id}) has no style class (diagram at line ${block.line})`
      );
  return errors;
}

/** Legend/styling errors for one flowchart diagram that has numbered callouts. */
function legendErrorsForDiagram(file: string, block: DiagramBlock): string[] {
  const lines = block.body.split('\n');
  if (!isFlowchart(lines)) return [];
  const nodes = collectNumberedNodes(lines);
  if (nodes.length === 0) return [];
  const styledIds = collectStyledIds(lines);
  const errors: string[] = [];
  if (!lines.some((line) => CLASSDEF.test(line)))
    errors.push(
      `${file}: diagram at line ${block.line} has numbered callouts but no classDef legend`
    );
  errors.push(...unstyledNodeErrors(file, block, nodes, styledIds));
  return errors;
}

/**
 * Flags numbered flowchart callout nodes that carry no style class, and diagrams
 * with callouts but no `classDef` legend — so callouts render consistently and
 * map to a legend. C4/other diagram types (different styling model) are skipped.
 */
export function validateDoc(file: string, content: string): string[] {
  const errors: string[] = [];
  for (const block of parseDoc(content))
    if (block.kind === DIAGRAM_KIND) errors.push(...legendErrorsForDiagram(file, block));
  return errors;
}

/** Numbered callout nodes must have a legend entry and consistent styling. */
const mermaidLegendCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'mermaid-legend' as CheckName,
  run: (root = process.cwd()) => runMermaidDocCheck(root, validateDoc),
  runInProcess: true,
} satisfies Check;

export default mermaidLegendCheck;
