import { join } from 'node:path';
import chalk from 'chalk';
import boxen from 'boxen';
import Table from 'cli-table3';
import { checkResult } from '../../utils/checkResult.js';
import { CheckName } from '../../types/index.types.js';
import type { Check } from '../../types/index.types.js';

const NOTE_NO_VERIFY =
  ' NOTE: Do not under any circumstance use `--no-verify` as it will cause issues downstream, fixing locally is your best bet.';

/** Check name → folder path when they differ (e.g. markdown-front-matter lives in rules-front-matter). */
const CHECK_TO_FOLDER: Record<string, string> = {
  [CheckName.MarkdownFrontMatter]: 'rules-front-matter',
};

/** Builds feedback text for CLI display (Agent/User context). */
export function buildContextFeedback(enabledCheckNames: string[]): string {
  const question = chalk.bold(
    'Did you familiarize yourself with the decisions logged in the repo,\n' +
      'specifically all "Fitness Checks" that are enabled via fitness-runner?'
  );
  const lines: string[] = [question];
  if (enabledCheckNames.length > 0) {
    const table = new Table({
      colWidths: [28, 42],
      head: [chalk.bold.white('Check'), chalk.bold.white('Src')],
    });
    for (const name of enabledCheckNames) {
      const folder = CHECK_TO_FOLDER[name] ?? name;
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

export const readRepoFirstCheck: Check = {
  name: CheckName.ReadRepoFirst,
  async run(root = process.cwd(), context) {
    const enabled = context?.enabledCheckNames ?? [];
    const feedback = buildContextFeedback(enabled);
    process.stdout.write(feedback);
    return checkResult(true, [], 0);
  },
};
