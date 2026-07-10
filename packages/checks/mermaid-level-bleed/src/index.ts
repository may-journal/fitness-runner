import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { Check, CheckName } from '@mayjournal/fitness';
import {
  MD_EXT,
  TABLE_KIND,
  checkResult,
  findFilesByExtension,
  parseDoc,
  type CalloutTableBlock,
} from '@mayjournal/fitness-shared';

/** Numbered C4 level file, e.g. `architecture/02-containers.md`. */
const LEVEL_FILE = /(?:^|\/)architecture\/(\d+)[^/]*\.md$/;
const DESCRIPTION = /^description$/i;

interface Level {
  descriptions: Set<string>;
  file: string;
  level: number;
}

/** Adds normalized (lowercased, whitespace-collapsed) descriptions from one callout table. */
function addTableDescriptions(table: CalloutTableBlock, set: Set<string>): void {
  const headerIndex = table.header.findIndex((cell) => DESCRIPTION.test(cell));
  const index = headerIndex >= 0 ? headerIndex : 1;
  for (const row of table.rows) {
    const text = (row[index] ?? '').trim().toLowerCase().replace(/\s+/g, ' ');
    if (text) set.add(text);
  }
}

/** Normalized callout descriptions in a doc, for exact cross-level comparison. */
export function levelDescriptions(content: string): Set<string> {
  const set = new Set<string>();
  for (const block of parseDoc(content))
    if (block.kind === TABLE_KIND) addTableDescriptions(block, set);
  return set;
}

/** Reads the numbered C4 level files under `root`, ordered by level. */
async function collectLevels(root: string): Promise<Level[]> {
  const levels: Level[] = [];
  for (const file of await findFilesByExtension(root, MD_EXT)) {
    const match = file.match(LEVEL_FILE);
    if (match)
      levels.push({
        descriptions: levelDescriptions(readFileSync(join(root, file), 'utf8')),
        file,
        level: Number(match[1]),
      });
  }
  return levels.sort((a, b) => a.level - b.level);
}

/** Errors where a level repeats a description from the immediately-previous level verbatim. */
function bleedErrors(levels: Level[]): string[] {
  const errors: string[] = [];
  for (let i = 1; i < levels.length; i += 1) {
    const previous = levels[i - 1];
    for (const text of levels[i].descriptions)
      if (previous.descriptions.has(text))
        errors.push(
          `${levels[i].file}: description "${text}" repeats level ${previous.level} verbatim — lower levels should add level-specific rationale`
        );
  }
  return errors;
}

/**
 * Warns when a callout description at one C4 level repeats the previous level
 * verbatim — lower levels should add level-specific rationale, not restate the
 * parent. Compares only exact (normalized) matches to stay low-false-positive.
 */
const mermaidLevelBleedCheck = {
  // Not in the CheckName enum: opt-in-only, never joins defaultChecks (see check-name.ts).
  name: 'mermaid-level-bleed' as CheckName,
  async run(root = process.cwd()) {
    const levels = await collectLevels(root);
    const errors = bleedErrors(levels);
    return checkResult(errors.length === 0, errors, levels.length);
  },
  runInProcess: true,
} satisfies Check;

export default mermaidLevelBleedCheck;
