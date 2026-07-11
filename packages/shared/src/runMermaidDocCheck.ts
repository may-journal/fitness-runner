import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { checkResult } from './checkResult.js';
import { MD_EXT } from './fileNames.js';
import { findFilesByExtension } from './findFilesByExtension.js';
import { parseDoc } from './mermaid.js';
import type { CheckResult } from './types/check-result.types.js';

/**
 * Runs a per-document validator over every `.md` file under `root` that contains
 * at least one mermaid diagram or callout table; files with neither are skipped
 * and don't count toward `filesChecked`. Shared by the mermaid diagram + callout
 * table checks so each supplies only its own `validateDoc` rule.
 */
export async function runMermaidDocCheck(
  root: string,
  validateDoc: (file: string, content: string) => string[]
): Promise<CheckResult> {
  const errors: string[] = [];
  let filesChecked = 0;
  for (const file of await findFilesByExtension(root, MD_EXT)) {
    const content = readFileSync(join(root, file), 'utf8');
    if (parseDoc(content).length === 0) continue;
    filesChecked += 1;
    errors.push(...validateDoc(file, content));
  }
  return checkResult(errors.length === 0, errors, filesChecked);
}
