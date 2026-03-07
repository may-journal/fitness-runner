import Table from 'cli-table3';
import chalk from 'chalk';
import { getColumns } from '../utils/terminal.js';
import { enUS } from './enUS.js';
import { interpolate } from '../utils/interpolate.js';

export type ResultRow = {
  errors?: string[];
  filesChecked: number;
  ms: number;
  name: string;
  ok: boolean;
};

/** Formats error block for colspan row. */
function formatErrorsBlock(checkName: string, errors: string[]): string {
  const header = chalk.red(`[${checkName}] ${enUS.PleaseFix}`);
  const bullets = errors.map((e) => chalk.red(enUS.ErrorBullet + e)).join('\n');
  return header + '\n' + bullets;
}

/** Returns green "passed" or red "failed" string. */
function formatStatus(ok: boolean): string {
  return ok ? chalk.green(enUS.StatusPassed) : chalk.red(enUS.StatusFailed);
}

/** Pushes colspan row with formatted errors to table. */
function pushErrorRow(table: InstanceType<typeof Table>, name: string, errors: string[]): void {
  table.push([{ colSpan: 4, content: formatErrorsBlock(name, errors), wordWrap: true }]);
}

/** Pushes one check row and optional error colspan row to table. */
function pushResultRow(table: InstanceType<typeof Table>, r: ResultRow): void {
  const files = r.filesChecked >= 0 ? String(r.filesChecked) : '-';
  table.push([r.name, formatStatus(r.ok), files, `${r.ms}ms`]);
  if (!r.ok && r.errors?.length) pushErrorRow(table, r.name, r.errors);
}

/** Builds table: check rows plus full-width colspan row per failed check with errors. */
export function buildTable(rows: ResultRow[]): string {
  const timeCol = Math.max(10, getColumns() - 51);
  const table = new Table({
    colWidths: [28, 10, 8, timeCol],
    head: [
      chalk.bold.white(enUS.TableCheck),
      chalk.bold.white(enUS.TableStatus),
      chalk.bold.white(enUS.TableFiles),
      chalk.bold.white(enUS.TableTime),
    ],
    wordWrap: true,
  });
  for (const r of rows) pushResultRow(table, r);
  return table.toString();
}

/** Builds total summary line. */
export function buildTotalLine(
  successCount: number,
  failureCount: number,
  totalFiles: number,
  totalMs: number
): string {
  return interpolate(enUS.TotalLine, {
    failure: failureCount,
    files: totalFiles,
    ms: totalMs,
    success: successCount,
  });
}
