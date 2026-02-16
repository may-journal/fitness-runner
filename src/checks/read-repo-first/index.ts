import chalk from 'chalk';
import boxen from 'boxen';
import wrapAnsi from 'wrap-ansi';
import type { Check } from '../../types/index.types.js';

const NOTE_NO_VERIFY = ' NOTE: Do not under any circumstance use `--no-verify` as it will cause issues downstream, fixing locally is your best bet.';

/** Exported for tests. */
export function getColumns(): number {
  const c = process.stdout.columns;
  return typeof c === 'number' && c > 0 ? c : 80;
}

/** Builds feedback text for CLI display (Agent/User context). */
export function buildContextFeedback(enabledCheckNames: string[]): string {
  const question = chalk.bold(
    'Did you familiarize yourself with the decisions logged in the repo,\n' +
    'specifically all "Fitness Checks" that are enabled via fitness-runner?',
  );
  const lines: string[] = [question];
  if (enabledCheckNames.length > 0) {
    const list = wrapAnsi('Enabled checks: ' + enabledCheckNames.join(', '), Math.max(getColumns() - 6, 40));
    lines.push('');
    lines.push(chalk.cyan(list));
  }
  lines.push('');
  lines.push(chalk.yellow(NOTE_NO_VERIFY.trim()));
  const content = lines.join('\n');
  return boxen(content, { padding: 1, borderColor: 'cyan', margin: 1 }) + '\n';
}

export const READ_REPO_FIRST_NAME = 'read-repo-first';

export const readRepoFirstCheck: Check = {
  name: READ_REPO_FIRST_NAME,
  async run(_root = process.cwd(), context) {
    const enabled = context?.enabledCheckNames ?? [];
    const feedback = buildContextFeedback(enabled);
    process.stdout.write(feedback);
    return { ok: true, errors: [], meta: { filesChecked: 0 } };
  },
};
