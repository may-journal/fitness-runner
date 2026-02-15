#!/usr/bin/env node
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

/** Returns whether current Node version satisfies .nvmrc and an optional error message. */
export function checkNodeVersion(root = process.cwd()): { ok: boolean; message?: string } {
  const nvmrcPath = join(root, '.nvmrc');
  const raw = readFileSync(nvmrcPath, 'utf8').trim();
  const requiredMajor = parseInt(raw.replace(/^v?(\d+).*$/, '$1'), 10);
  const currentMajor = parseInt(process.version.replace(/^v?(\d+).*$/, '$1'), 10);
  if (currentMajor >= requiredMajor) return { ok: true };
  return { ok: false, message: `Node ${process.version} does not satisfy .nvmrc (requires ${requiredMajor}.x). Run: nvm use` };
}

/** Runs the check and returns exit code (0 or 1); script entry point. */
export function main(root = process.cwd()): number {
  const r = checkNodeVersion(root);
  if (!r.ok && r.message) console.error(r.message);
  return r.ok ? 0 : 1;
}

/** Entry when run as script; calls process.exit(main()). */
export function runAsMain(): number {
  const code = main();
  process.exit(code);
  return code;
}

/** Called at load; runs runAsMain() only when this file is the process entry. */
export function runIfMain(): void {
  if (process.argv[1] === fileURLToPath(import.meta.url)) runAsMain();
}

runIfMain();
