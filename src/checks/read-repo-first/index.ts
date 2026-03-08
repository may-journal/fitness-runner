import { join } from 'node:path';
import chalk from 'chalk';
import boxen from 'boxen';
import Table from 'cli-table3';
import { checkResult } from '../../utils/checkResult.js';
import { getColumns } from '../../utils/terminal.js';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

const NOTE_NO_VERIFY =
  ' NOTE: Do not under any circumstance use `--no-verify` as it will cause issues downstream, fixing locally is your best bet.';

/** Builds feedback text for CLI display (Agent/User context). checkFolderByName overrides folder when name differs. */
export function buildContextFeedback(
  enabledCheckNames: string[],
  checkFolderByName: Record<string, string> = {}
): string {
  const question = chalk.bold(
    'Did you familiarize yourself with the decisions logged in the repo,\n' +
      'specifically all "Fitness Checks" that are enabled via fitness-runner?'
  );
  const lines: string[] = [question];
  if (enabledCheckNames.length > 0) {
    const firstCol = 28;
    const maxTotal = Math.max(70, getColumns() - 8);
    const secondCol = Math.max(20, maxTotal - firstCol - 6);
    const table = new Table({
      colWidths: [firstCol, secondCol],
      head: [chalk.bold.white('Check'), chalk.bold.white('Src')],
      wordWrap: true,
    });
    for (const name of enabledCheckNames) {
      const folder = checkFolderByName[name] ?? name;
      const rel = join('src', 'checks', folder, 'README.md');
      table.push([name, rel]);
    }
    lines.push('');
    lines.push(table.toString());
  }
  lines.push('');
  lines.push(chalk.yellow(NOTE_NO_VERIFY.trim()));
  const content = lines.join('\n');
  return boxen(content, { borderColor: 'cyan', margin: 1, padding: 1 }) + '\n';
}

/** Builds read-repo-first feedback string from run context. */
function getFeedbackFromContext(context: Parameters<Check['run']>[1]): string {
  return buildContextFeedback(context?.enabledCheckNames ?? [], context?.checkFolderByName ?? {});
}

export const readRepoFirstCheck: Check = {
  name: CheckName.ReadRepoFirst,
  async run(_root = process.cwd(), context) {
    process.stdout.write(getFeedbackFromContext(context));
    return checkResult(true, [], 0);
  },
  runInProcess: true,
};
