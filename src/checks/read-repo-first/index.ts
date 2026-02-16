import * as readline from 'node:readline/promises';
import { stdin, stdout } from 'node:process';
import chalk from 'chalk';
import boxen from 'boxen';
import wrapAnsi from 'wrap-ansi';
import type { Check } from '../../types/index.types.js';

const ERROR_NOT_INTERACTIVE = 'Run fitness interactively to complete this check (requires TTY).';
const ERROR_DECLINED = 'You must familiarize yourself with the Fitness Checks before continuing.';

/** Builds check result from ok flag. */
function toResult(ok: boolean): { ok: boolean; errors: string[]; meta: { filesChecked: number } } {
  return { ok, errors: ok ? [] : [ERROR_DECLINED], meta: { filesChecked: 0 } };
}

/** True if trimmed answer is Y or y. */
function isYes(answer: string): boolean {
  return /^y$/i.test(answer.trim());
}

/** Asks user via questionFn or real readline; returns answer. */
async function askUser(prompt: string, questionFn?: (p: string) => Promise<string>): Promise<string> {
  if (questionFn) return questionFn(prompt);
  const rl = readline.createInterface({ input: stdin, output: stdout });
  return rl.question(prompt).finally(() => rl.close());
}

/** Returns override from context, or undefined. */
function getOverride(context: { _readAnswer?: string } | undefined): string | undefined {
  return context?._readAnswer;
}

/** Returns Y if env bypass set, else undefined. */
function getEnvBypass(): string | undefined {
  return process.env.FITNESS_READ_REPO_CONFIRMED === '1' ? 'Y' : undefined;
}

/** Prompts user when TTY; returns null when not TTY. */
async function promptIfInteractive(context: { _readlineQuestion?: (p: string) => Promise<string>; enabledCheckNames?: string[] } | undefined): Promise<string | null> {
  if (!stdin.isTTY) return null;
  const enabled = context?.enabledCheckNames ?? [];
  return askUser(buildPrompt(enabled), context?._readlineQuestion);
}

/** Resolves answer from context overrides, env, or interactive prompt. */
async function getAnswer(context: { _readAnswer?: string; _readlineQuestion?: (p: string) => Promise<string>; enabledCheckNames?: string[] } | undefined): Promise<string | null> {
  const override = getOverride(context);
  if (override !== undefined) return override;
  const envBypass = getEnvBypass();
  if (envBypass !== undefined) return envBypass;
  return promptIfInteractive(context);
}

/** Exported for tests. */
export function getColumns(): number {
  const c = process.stdout.columns;
  return typeof c === 'number' && c > 0 ? c : 80;
}

/** Exported for tests. */
export function buildPrompt(enabledCheckNames: string[]): string {
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
  lines.push(chalk.dim('Y | N: '));
  const content = lines.join('\n');
  return boxen(content, { padding: 1, borderColor: 'cyan', margin: 1 }) + '\n';
}

export const readRepoFirstCheck: Check = {
  name: 'read-repo-first',
  async run(_root = process.cwd(), context) {
    const answer = await getAnswer(context);
    if (answer === null) return { ok: false, errors: [ERROR_NOT_INTERACTIVE], meta: { filesChecked: 0 } };
    return toResult(isYes(answer));
  },
};
